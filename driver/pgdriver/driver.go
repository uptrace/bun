package pgdriver

import (
	"bufio"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

func init() {
	sql.Register("pg", NewDriver())
}

type Driver struct{}

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

type driverConnector struct {
	driver Driver

	network     string
	addr        string
	dialTimeout time.Duration
	dialer      func(ctx context.Context, network, addr string) (net.Conn, error)

	user     string
	password string
	database string
	appName  string
}

func NewConnector(opts ...DriverOption) driver.Connector {
	d := &driverConnector{
		network:     "tcp",
		addr:        "localhost:5432",
		dialTimeout: 5 * time.Second,

		user:     "postgres",
		database: "postgres",
	}
	d.dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
		netDialer := &net.Dialer{
			Timeout:   d.dialTimeout,
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
	if d.user == "" {
		return nil, errors.New("pgdriver: user name is required")
	}
	// fmt.Println("hello\n")
	return newConn(ctx, d)
}

func (d *driverConnector) Driver() driver.Driver {
	return &d.driver
}

//------------------------------------------------------------------------------

type Conn struct {
	driver *driverConnector

	netConn net.Conn

	rd  *bufio.Reader
	buf []byte

	processID int32
	secretKey int32
}

func newConn(ctx context.Context, driver *driverConnector) (*Conn, error) {
	netConn, err := driver.dialer(ctx, driver.network, driver.addr)
	if err != nil {
		return nil, err
	}

	cn := &Conn{
		driver:  driver,
		netConn: netConn,

		rd:  bufio.NewReader(netConn),
		buf: make([]byte, 64),
	}
	if err := startup(cn); err != nil {
		return nil, err
	}

	return cn, nil
}

func (cn *Conn) withWriter(fn func(wr *bufio.Writer) error) error {
	wr := getBufioWriter()
	wr.Reset(cn.netConn)
	err := fn(wr)
	putBufioWriter(wr)
	return err
}

var _ driver.Conn = (*Conn)(nil)

func (cn *Conn) Prepare(query string) (driver.Stmt, error) {
	panic("not implemented")
}

func (cn *Conn) Close() error {
	return cn.netConn.Close()
}

func (cn *Conn) Begin() (driver.Tx, error) {
	panic("not implemented")
}

var _ driver.ExecerContext = (*Conn)(nil)

func (cn *Conn) ExecContext(
	ctx context.Context, query string, args []driver.NamedValue,
) (driver.Result, error) {
	if err := writeQuery(cn, query); err != nil {
		return nil, err
	}

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
		case describeMsg,
			commandCompleteMsg,
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
	if err := writeQuery(cn, query); err != nil {
		return nil, err
	}
	return newRows(cn)
}

//------------------------------------------------------------------------------

type rows struct {
	cn *Conn

	rowDesc *rowDescription
	closed  bool
}

var _ driver.Rows = (*rows)(nil)

func newRows(cn *Conn) (*rows, error) {
	rows := &rows{
		cn: cn,
	}

	var firstErr error

	for {
		c, msgLen, err := readMessageType(cn)
		if err != nil {
			return nil, err
		}

		switch c {
		case rowDescriptionMsg:
			rowDesc, err := readRowDescription(cn)
			if err != nil {
				return nil, err
			}
			rows.rowDesc = rowDesc
			return rows, nil
		case readyForQueryMsg:
			if err := discard(cn, msgLen); err != nil {
				return nil, err
			}
			if firstErr != nil {
				return nil, firstErr
			}
			return nil, fmt.Errorf("pgdriver: newRows: unexpected message %q", c)
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
		case noticeResponseMsg, parameterStatusMsg:
			if err := discard(cn, msgLen); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("pgdriver: newRows: unexpected message %q", c)
		}
	}
}

func (r *rows) Columns() []string {
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
		default:
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

	for {
		c, msgLen, err := readMessageType(r.cn)
		if err != nil {
			return err
		}

		switch c {
		case dataRowMsg:
			return r.readDataRow(dest)
		case commandCompleteMsg:
			if err := discard(r.cn, msgLen); err != nil {
				return err
			}
		case readyForQueryMsg:
			if err := discard(r.cn, msgLen); err != nil {
				return err
			}
			r.close()
			return io.EOF
		default:
			return fmt.Errorf("pgdriver: Next: unexpected message %q", c)
		}
	}
}

func (r *rows) readDataRow(dest []driver.Value) error {
	numCol, err := readInt16(r.cn)
	if err != nil {
		return err
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

		dest[colIdx] = value
	}

	return nil
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
