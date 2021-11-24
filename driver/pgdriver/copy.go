package pgdriver

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/uptrace/bun"
)

// CopyFrom copies data from the reader to the query destination.
func CopyFrom(
	ctx context.Context, conn bun.Conn, r io.Reader, query string, args ...interface{},
) (res sql.Result, err error) {
	query, err = formatQueryArgs(query, args)
	if err != nil {
		return nil, err
	}

	if err := conn.Raw(func(driverConn interface{}) error {
		cn := driverConn.(*Conn)

		if err := writeQuery(ctx, cn, query); err != nil {
			return err
		}
		if err := readCopyIn(ctx, cn); err != nil {
			return err
		}
		if err := writeCopyData(ctx, cn, r); err != nil {
			return err
		}
		if err := writeCopyDone(ctx, cn); err != nil {
			return err
		}

		res, err = readQuery(ctx, cn)
		return err
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func readCopyIn(ctx context.Context, cn *Conn) error {
	rd := cn.reader(ctx, -1)
	var firstErr error
	for {
		c, msgLen, err := readMessageType(rd)
		if err != nil {
			return err
		}

		switch c {
		case errorResponseMsg:
			e, err := readError(rd)
			if err != nil {
				return err
			}
			if firstErr == nil {
				firstErr = e
			}
		case readyForQueryMsg:
			if err := rd.Discard(msgLen); err != nil {
				return err
			}
			return firstErr
		case copyInResponseMsg:
			if err := rd.Discard(msgLen); err != nil {
				return err
			}
			return firstErr
		case noticeResponseMsg, parameterStatusMsg:
			if err := rd.Discard(msgLen); err != nil {
				return err
			}
		default:
			return fmt.Errorf("pgdriver: readCopyIn: unexpected message %q", c)
		}
	}
}

func writeCopyData(ctx context.Context, cn *Conn, r io.Reader) error {
	wb := getWriteBuffer()
	defer putWriteBuffer(wb)

	for {
		wb.StartMessage(copyDataMsg)
		if _, err := wb.ReadFrom(r); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		wb.FinishMessage()

		if err := cn.write(ctx, wb); err != nil {
			return err
		}
	}

	return nil
}

func writeCopyDone(ctx context.Context, cn *Conn) error {
	wb := getWriteBuffer()
	defer putWriteBuffer(wb)

	wb.StartMessage(copyDoneMsg)
	wb.FinishMessage()

	return cn.write(ctx, wb)
}

//------------------------------------------------------------------------------

// CopyTo copies data from the query source to the writer.
func CopyTo(
	ctx context.Context, conn bun.Conn, w io.Writer, query string, args ...interface{},
) (res sql.Result, err error) {
	query, err = formatQueryArgs(query, args)
	if err != nil {
		return nil, err
	}

	if err := conn.Raw(func(driverConn interface{}) error {
		cn := driverConn.(*Conn)

		if err := writeQuery(ctx, cn, query); err != nil {
			return err
		}
		if err := readCopyOut(ctx, cn); err != nil {
			return err
		}

		res, err = readCopyData(ctx, cn, w)
		return err
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func readCopyOut(ctx context.Context, cn *Conn) error {
	rd := cn.reader(ctx, -1)
	var firstErr error
	for {
		c, msgLen, err := readMessageType(rd)
		if err != nil {
			return err
		}

		switch c {
		case errorResponseMsg:
			e, err := readError(rd)
			if err != nil {
				return err
			}
			if firstErr == nil {
				firstErr = e
			}
		case readyForQueryMsg:
			if err := rd.Discard(msgLen); err != nil {
				return err
			}
			return firstErr
		case copyOutResponseMsg:
			if err := rd.Discard(msgLen); err != nil {
				return err
			}
			return nil
		case noticeResponseMsg, parameterStatusMsg:
			if err := rd.Discard(msgLen); err != nil {
				return err
			}
		default:
			return fmt.Errorf("pgdriver: readCopyOut: unexpected message %q", c)
		}
	}
}

func readCopyData(ctx context.Context, cn *Conn, w io.Writer) (res sql.Result, err error) {
	rd := cn.reader(ctx, -1)
	var firstErr error
	for {
		c, msgLen, err := readMessageType(rd)
		if err != nil {
			return nil, err
		}

		switch c {
		case errorResponseMsg:
			e, err := readError(rd)
			if err != nil {
				return nil, err
			}
			if firstErr == nil {
				firstErr = e
			}
		case copyDataMsg:
			for msgLen > 0 {
				b, err := rd.ReadTemp(msgLen)
				if err != nil && err != bufio.ErrBufferFull {
					return nil, err
				}

				if _, err := w.Write(b); err != nil {
					if firstErr == nil {
						firstErr = err
					}
					break
				}

				msgLen -= len(b)
			}
		case copyDoneMsg:
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
		case commandCompleteMsg:
			tmp, err := rd.ReadTemp(msgLen)
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
		case readyForQueryMsg:
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
			return res, firstErr
		case noticeResponseMsg, parameterStatusMsg:
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("pgdriver: readCopyData: unexpected message %q", c)
		}
	}
}
