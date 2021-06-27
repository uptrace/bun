package pgdriver

import (
	"bufio"
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
	"sync"
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
	connector *driverConnector
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

type DriverStats struct {
	Queries uint64
	Errors  uint64
}

type driverConnector struct {
	cfg Config

	stats DriverStats
}

func NewConnector(opts ...DriverOption) driver.Connector {
	d := &driverConnector{
		cfg: newDefaultConfig(),
	}

	d.cfg.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
		netDialer := &net.Dialer{
			Timeout:   d.cfg.DialTimeout,
			KeepAlive: 5 * time.Minute,
		}
		return netDialer.DialContext(ctx, network, addr)
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

var _ driver.Connector = (*driverConnector)(nil)

func (d *driverConnector) Connect(ctx context.Context) (driver.Conn, error) {
	if d.cfg.User == "" {
		return nil, errors.New("pgdriver: user name is required")
	}
	return newConn(ctx, d)
}

func (d *driverConnector) Driver() driver.Driver {
	return Driver{connector: d}
}

func (d *driverConnector) Config() Config {
	return d.cfg
}

func (d *driverConnector) Stats() DriverStats {
	return DriverStats{
		Queries: atomic.LoadUint64(&d.stats.Queries),
		Errors:  atomic.LoadUint64(&d.stats.Errors),
	}
}

//------------------------------------------------------------------------------

type Conn struct {
	driver *driverConnector

	netConn net.Conn

	rd  *bufio.Reader
	buf []byte

	processID int32
	secretKey int32

	closed int32
}

func newConn(ctx context.Context, driver *driverConnector) (*Conn, error) {
	netConn, err := driver.cfg.Dialer(ctx, driver.cfg.Network, driver.cfg.Addr)
	if err != nil {
		return nil, err
	}

	cn := &Conn{
		driver:  driver,
		netConn: netConn,

		rd:  bufio.NewReader(netConn),
		buf: make([]byte, 64),
	}

	if cn.driver.cfg.TLSConfig != nil {
		if err := enableSSL(ctx, cn, cn.driver.cfg.TLSConfig); err != nil {
			return nil, err
		}
	}

	if err := startup(ctx, cn); err != nil {
		return nil, err
	}

	return cn, nil
}

func (cn *Conn) withWriter(
	ctx context.Context,
	timeout time.Duration,
	fn func(wr *bufio.Writer) error,
) error {
	wr := getBufioWriter()

	cn.setWriteDeadline(ctx, timeout)
	wr.Reset(cn.netConn)

	err := fn(wr)
	if err == nil {
		err = wr.Flush()
	}

	putBufioWriter(wr)

	return err
}

var _ driver.Conn = (*Conn)(nil)

func (cn *Conn) Prepare(query string) (driver.Stmt, error) {
	if cn.isClosed() {
		return nil, driver.ErrBadConn
	}

	name := query

	if err := writeParseDescribeSync(context.TODO(), cn, query, query); err != nil {
		return nil, err
	}

	rowDesc, err := readParseDescribeSync(cn)
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
	if err := writeQuery(ctx, cn, query); err != nil {
		return nil, err
	}

	cn.setReadDeadline(ctx, -1)

	var res driver.Result
	var firstErr error

	for {
		c, msgLen, err := readMessageType(cn)
		if err != nil {
			return nil, err
		}

		switch c {
		case errorResponseMsg:
			e, err := readError(cn)
			if err != nil {
				return nil, err
			}
			if firstErr == nil {
				firstErr = e
			}
		case emptyQueryResponseMsg:
			if firstErr == nil {
				firstErr = errEmptyQuery
			}
		case commandCompleteMsg:
			tmp, err := readN(cn, msgLen)
			if err != nil {
				firstErr = err
				break
			}

			r, err := parseResult(tmp)
			if err != nil {
				firstErr = err
			} else {
				res = r
			}
		case describeMsg,
			rowDescriptionMsg,
			noticeResponseMsg,
			parameterStatusMsg:
			if err := discard(cn, msgLen); err != nil {
				return nil, err
			}
		case readyForQueryMsg:
			if err := discard(cn, msgLen); err != nil {
				return nil, err
			}
			return res, firstErr
		default:
			return nil, fmt.Errorf("pgdriver: Exec: unexpected message %q", c)
		}
	}
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
	if err := writeQuery(ctx, cn, query); err != nil {
		return nil, err
	}
	return readQuery(ctx, cn)
}

var _ driver.Pinger = (*Conn)(nil)

func (cn *Conn) Ping(ctx context.Context) error {
	_, err := cn.ExecContext(ctx, "SELECT 1", nil)
	return err
}

func (cn *Conn) setReadDeadline(ctx context.Context, timeout time.Duration) {
	if timeout == -1 {
		timeout = cn.driver.cfg.ReadTimeout
	}
	_ = cn.netConn.SetReadDeadline(cn.deadline(ctx, timeout))
}

func (cn *Conn) setWriteDeadline(ctx context.Context, timeout time.Duration) {
	if timeout == -1 {
		timeout = cn.driver.cfg.WriteTimeout
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
	cn      *Conn
	rowDesc *rowDescription
	closed  bool
}

var _ driver.Rows = (*rows)(nil)

func newRows(cn *Conn, rowDesc *rowDescription) *rows {
	return &rows{
		cn:      cn,
		rowDesc: rowDesc,
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
		rowDescPool.Put(r.rowDesc)
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
	var firstErr error

	for {
		c, msgLen, err := readMessageType(r.cn)
		if err != nil {
			return false, err
		}

		switch c {
		case dataRowMsg:
			return false, r.readDataRow(dest)
		case commandCompleteMsg:
			if err := discard(r.cn, msgLen); err != nil {
				return false, err
			}
		case readyForQueryMsg:
			r.close()

			if err := discard(r.cn, msgLen); err != nil {
				return false, err
			}

			if firstErr != nil {
				return false, firstErr
			}
			return true, nil
		case errorResponseMsg:
			e, err := readError(r.cn)
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

func (r *rows) readDataRow(dest []driver.Value) error {
	numCol, err := readInt16(r.cn)
	if err != nil {
		return err
	}

	if len(dest) != int(numCol) {
		return fmt.Errorf("pgdriver: query returned %d columns, but dest has %d items",
			numCol, len(dest))
	}

	for colIdx := int16(0); colIdx < numCol; colIdx++ {
		dataLen, err := readInt32(r.cn)
		if err != nil {
			return err
		}

		value, err := readColumnValue(r.cn, r.rowDesc.types[colIdx], int(dataLen))
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
	return len(stmt.rowDesc.names)
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

//------------------------------------------------------------------------------

var bufioWriterPool = sync.Pool{
	New: func() interface{} {
		return bufio.NewWriter(nil)
	},
}

func getBufioWriter() *bufio.Writer {
	return bufioWriterPool.Get().(*bufio.Writer)
}

func putBufioWriter(wr *bufio.Writer) {
	bufioWriterPool.Put(wr)
}
