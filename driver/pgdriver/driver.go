package pgdriver

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

func init() {
	sql.Register("pg", NewDriver())
}

type logging interface {
	Printf(ctx context.Context, format string, v ...interface{})
}

type logger struct {
	log *log.Logger
}

func (l *logger) Printf(ctx context.Context, format string, v ...interface{}) {
	_ = l.log.Output(2, fmt.Sprintf(format, v...))
}

var Logger logging = &logger{
	log: log.New(os.Stderr, "pgdriver: ", log.LstdFlags|log.Lshortfile),
}

//------------------------------------------------------------------------------

type Driver struct {
	connector *Connector
}

var _ driver.DriverContext = (*Driver)(nil)

func NewDriver() Driver {
	return Driver{}
}

func (d Driver) OpenConnector(name string) (driver.Connector, error) {
	opts, err := parseDSN(name)
	if err != nil {
		return nil, err
	}
	return NewConnector(opts...), nil
}

func (d Driver) Open(name string) (driver.Conn, error) {
	connector, err := d.OpenConnector(name)
	if err != nil {
		return nil, err
	}
	return connector.Connect(context.TODO())
}

//------------------------------------------------------------------------------

type Connector struct {
	cfg *Config
}

func NewConnector(opts ...Option) *Connector {
	c := &Connector{cfg: newDefaultConfig()}
	for _, opt := range opts {
		opt(c.cfg)
	}
	return c
}

var _ driver.Connector = (*Connector)(nil)

func (c *Connector) Connect(ctx context.Context) (driver.Conn, error) {
	if err := c.cfg.verify(); err != nil {
		return nil, err
	}
	return newConn(ctx, c.cfg)
}

func (c *Connector) Driver() driver.Driver {
	return Driver{connector: c}
}

func (c *Connector) Config() *Config {
	return c.cfg
}

//------------------------------------------------------------------------------

type Conn struct {
	cfg *Config

	netConn net.Conn
	rd      *reader

	processID int32
	secretKey int32

	stmtCount int

	closed int32
}

func newConn(ctx context.Context, cfg *Config) (*Conn, error) {
	netConn, err := cfg.Dialer(ctx, cfg.Network, cfg.Addr)
	if err != nil {
		return nil, err
	}

	cn := &Conn{
		cfg:     cfg,
		netConn: netConn,
		rd:      newReader(netConn),
	}

	if cfg.TLSConfig != nil {
		if err := enableSSL(ctx, cn, cfg.TLSConfig); err != nil {
			return nil, err
		}
	}

	if err := startup(ctx, cn); err != nil {
		return nil, err
	}

	for k, v := range cfg.ConnParams {
		if v != nil {
			_, err = cn.ExecContext(ctx, fmt.Sprintf("SET %s TO $1", k), []driver.NamedValue{
				{Value: v},
			})
		} else {
			_, err = cn.ExecContext(ctx, fmt.Sprintf("SET %s TO DEFAULT", k), nil)
		}
		if err != nil {
			return nil, err
		}
	}

	return cn, nil
}

func (cn *Conn) reader(ctx context.Context, timeout time.Duration) *reader {
	cn.setReadDeadline(ctx, timeout)
	return cn.rd
}

func (cn *Conn) write(ctx context.Context, wb *writeBuffer) error {
	cn.setWriteDeadline(ctx, -1)

	n, err := cn.netConn.Write(wb.Bytes)
	wb.Reset()

	if err != nil {
		if n == 0 {
			Logger.Printf(ctx, "pgdriver: Conn.Write failed (zero-length): %s", err)
			return driver.ErrBadConn
		}
		return err
	}
	return nil
}

var _ driver.Conn = (*Conn)(nil)

func (cn *Conn) Prepare(query string) (driver.Stmt, error) {
	if cn.isClosed() {
		return nil, driver.ErrBadConn
	}

	ctx := context.TODO()

	name := fmt.Sprintf("pgdriver-%d", cn.stmtCount)
	cn.stmtCount++

	if err := writeParseDescribeSync(ctx, cn, name, query); err != nil {
		return nil, err
	}

	rowDesc, err := readParseDescribeSync(ctx, cn)
	if err != nil {
		return nil, err
	}

	return newStmt(cn, name, rowDesc), nil
}

func (cn *Conn) Close() error {
	if !atomic.CompareAndSwapInt32(&cn.closed, 0, 1) {
		return nil
	}
	return cn.netConn.Close()
}

func (cn *Conn) isClosed() bool {
	return atomic.LoadInt32(&cn.closed) == 1
}

func (cn *Conn) Begin() (driver.Tx, error) {
	return cn.BeginTx(context.Background(), driver.TxOptions{})
}

var _ driver.ConnBeginTx = (*Conn)(nil)

func (cn *Conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	// No need to check if the conn is closed. ExecContext below handles that.

	if sql.IsolationLevel(opts.Isolation) != sql.LevelDefault {
		return nil, errors.New("pgdriver: custom IsolationLevel is not supported")
	}
	if opts.ReadOnly {
		return nil, errors.New("pgdriver: ReadOnly transactions are not supported")
	}

	if _, err := cn.ExecContext(ctx, "BEGIN", nil); err != nil {
		return nil, err
	}
	return tx{cn: cn}, nil
}

var _ driver.ExecerContext = (*Conn)(nil)

func (cn *Conn) ExecContext(
	ctx context.Context, query string, args []driver.NamedValue,
) (driver.Result, error) {
	if cn.isClosed() {
		return nil, driver.ErrBadConn
	}
	res, err := cn.exec(ctx, query, args)
	if err != nil {
		return nil, cn.checkBadConn(err)
	}
	return res, nil
}

func (cn *Conn) exec(
	ctx context.Context, query string, args []driver.NamedValue,
) (driver.Result, error) {
	query, err := formatQuery(query, args)
	if err != nil {
		return nil, err
	}
	if err := writeQuery(ctx, cn, query); err != nil {
		return nil, err
	}
	return readQuery(ctx, cn)
}

var _ driver.QueryerContext = (*Conn)(nil)

func (cn *Conn) QueryContext(
	ctx context.Context, query string, args []driver.NamedValue,
) (driver.Rows, error) {
	if cn.isClosed() {
		return nil, driver.ErrBadConn
	}
	rows, err := cn.query(ctx, query, args)
	if err != nil {
		return nil, cn.checkBadConn(err)
	}
	return rows, nil
}

func (cn *Conn) query(
	ctx context.Context, query string, args []driver.NamedValue,
) (driver.Rows, error) {
	query, err := formatQuery(query, args)
	if err != nil {
		return nil, err
	}
	if err := writeQuery(ctx, cn, query); err != nil {
		return nil, err
	}
	return readQueryData(ctx, cn)
}

var _ driver.Pinger = (*Conn)(nil)

func (cn *Conn) Ping(ctx context.Context) error {
	_, err := cn.ExecContext(ctx, "SELECT 1", nil)
	return err
}

func (cn *Conn) setReadDeadline(ctx context.Context, timeout time.Duration) {
	if timeout == -1 {
		timeout = cn.cfg.ReadTimeout
	}
	_ = cn.netConn.SetReadDeadline(cn.deadline(ctx, timeout))
}

func (cn *Conn) setWriteDeadline(ctx context.Context, timeout time.Duration) {
	if timeout == -1 {
		timeout = cn.cfg.WriteTimeout
	}
	_ = cn.netConn.SetWriteDeadline(cn.deadline(ctx, timeout))
}

func (cn *Conn) deadline(ctx context.Context, timeout time.Duration) time.Time {
	deadline, ok := ctx.Deadline()
	if !ok {
		if timeout == 0 {
			return time.Time{}
		}
		return time.Now().Add(timeout)
	}

	if timeout == 0 {
		return deadline
	}
	if tm := time.Now().Add(timeout); tm.Before(deadline) {
		return tm
	}
	return deadline
}

var _ driver.Validator = (*Conn)(nil)

func (cn *Conn) IsValid() bool {
	return !cn.isClosed()
}

func (cn *Conn) checkBadConn(err error) error {
	if isBadConn(err, false) {
		// Close and return driver.ErrBadConn next time the conn is used.
		_ = cn.Close()
	}
	// Always return the original error.
	return err
}

//------------------------------------------------------------------------------

type rows struct {
	cn       *Conn
	rowDesc  *rowDescription
	reusable bool
	closed   bool
}

var _ driver.Rows = (*rows)(nil)

func newRows(cn *Conn, rowDesc *rowDescription, reusable bool) *rows {
	return &rows{
		cn:       cn,
		rowDesc:  rowDesc,
		reusable: reusable,
	}
}

func (r *rows) Columns() []string {
	if r.closed || r.rowDesc == nil {
		return nil
	}
	return r.rowDesc.names
}

func (r *rows) Close() error {
	if r.closed {
		return nil
	}
	defer r.close()

	for {
		switch err := r.Next(nil); err {
		case nil, io.EOF:
			return nil
		default: // unexpected error
			_ = r.cn.Close()
			return err
		}
	}
}

func (r *rows) close() {
	r.closed = true

	if r.rowDesc != nil {
		if r.reusable {
			rowDescPool.Put(r.rowDesc)
		}
		r.rowDesc = nil
	}
}

func (r *rows) Next(dest []driver.Value) error {
	if r.closed {
		return io.EOF
	}

	eof, err := r.next(dest)
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	} else if err != nil {
		return err
	}
	if eof {
		return io.EOF
	}
	return nil
}

func (r *rows) next(dest []driver.Value) (eof bool, _ error) {
	rd := r.cn.reader(context.TODO(), -1)
	var firstErr error
	for {
		c, msgLen, err := readMessageType(rd)
		if err != nil {
			return false, err
		}

		switch c {
		case dataRowMsg:
			return false, r.readDataRow(rd, dest)
		case commandCompleteMsg:
			if err := rd.Discard(msgLen); err != nil {
				return false, err
			}
		case readyForQueryMsg:
			r.close()

			if err := rd.Discard(msgLen); err != nil {
				return false, err
			}

			if firstErr != nil {
				return false, firstErr
			}
			return true, nil
		case parameterStatusMsg, noticeResponseMsg:
			if err := rd.Discard(msgLen); err != nil {
				return false, err
			}
		case errorResponseMsg:
			e, err := readError(rd)
			if err != nil {
				return false, err
			}
			if firstErr == nil {
				firstErr = e
			}
		default:
			return false, fmt.Errorf("pgdriver: Next: unexpected message %q", c)
		}
	}
}

func (r *rows) readDataRow(rd *reader, dest []driver.Value) error {
	numCol, err := readInt16(rd)
	if err != nil {
		return err
	}

	if len(dest) != int(numCol) {
		return fmt.Errorf("pgdriver: query returned %d columns, but Scan dest has %d items",
			numCol, len(dest))
	}

	for colIdx := int16(0); colIdx < numCol; colIdx++ {
		dataLen, err := readInt32(rd)
		if err != nil {
			return err
		}

		value, err := readColumnValue(rd, r.rowDesc.types[colIdx], int(dataLen))
		if err != nil {
			return err
		}

		if dest != nil {
			dest[colIdx] = value
		}
	}

	return nil
}

//------------------------------------------------------------------------------

func parseResult(b []byte) (driver.RowsAffected, error) {
	i := bytes.LastIndexByte(b, ' ')
	if i == -1 {
		return 0, nil
	}

	b = b[i+1 : len(b)-1]
	affected, err := strconv.ParseUint(bytesToString(b), 10, 64)
	if err != nil {
		return 0, nil
	}

	return driver.RowsAffected(affected), nil
}

//------------------------------------------------------------------------------

type tx struct {
	cn *Conn
}

var _ driver.Tx = (*tx)(nil)

func (tx tx) Commit() error {
	_, err := tx.cn.ExecContext(context.Background(), "COMMIT", nil)
	return err
}

func (tx tx) Rollback() error {
	_, err := tx.cn.ExecContext(context.Background(), "ROLLBACK", nil)
	return err
}

//------------------------------------------------------------------------------

type stmt struct {
	cn      *Conn
	name    string
	rowDesc *rowDescription
}

var (
	_ driver.Stmt             = (*stmt)(nil)
	_ driver.StmtExecContext  = (*stmt)(nil)
	_ driver.StmtQueryContext = (*stmt)(nil)
)

func newStmt(cn *Conn, name string, rowDesc *rowDescription) *stmt {
	return &stmt{
		cn:      cn,
		name:    name,
		rowDesc: rowDesc,
	}
}

func (stmt *stmt) Close() error {
	if stmt.rowDesc != nil {
		rowDescPool.Put(stmt.rowDesc)
		stmt.rowDesc = nil
	}

	ctx := context.TODO()
	if err := writeCloseStmt(ctx, stmt.cn, stmt.name); err != nil {
		return err
	}
	if err := readCloseStmtComplete(ctx, stmt.cn); err != nil {
		return err
	}
	return nil
}

func (stmt *stmt) NumInput() int {
	if stmt.rowDesc == nil {
		return -1
	}
	return int(stmt.rowDesc.numInput)
}

func (stmt *stmt) Exec(args []driver.Value) (driver.Result, error) {
	panic("not implemented")
}

func (stmt *stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if err := writeBindExecute(ctx, stmt.cn, stmt.name, args); err != nil {
		return nil, err
	}
	return readExtQuery(ctx, stmt.cn)
}

func (stmt *stmt) Query(args []driver.Value) (driver.Rows, error) {
	panic("not implemented")
}

func (stmt *stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if err := writeBindExecute(ctx, stmt.cn, stmt.name, args); err != nil {
		return nil, err
	}
	return readExtQueryData(ctx, stmt.cn, stmt.rowDesc)
}
