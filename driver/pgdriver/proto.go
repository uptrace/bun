package pgdriver

import (
	"bufio"
	"context"
	"crypto/md5"
	"crypto/tls"
	"database/sql"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"mellium.im/sasl"
)

// https://www.postgresql.org/docs/current/protocol-message-formats.html
//nolint:deadcode,varcheck,unused
const (
	commandCompleteMsg  = 'C'
	errorResponseMsg    = 'E'
	noticeResponseMsg   = 'N'
	parameterStatusMsg  = 'S'
	authenticationOKMsg = 'R'
	backendKeyDataMsg   = 'K'
	noDataMsg           = 'n'
	passwordMessageMsg  = 'p'
	terminateMsg        = 'X'

	saslInitialResponseMsg        = 'p'
	authenticationSASLContinueMsg = 'R'
	saslResponseMsg               = 'p'
	authenticationSASLFinalMsg    = 'R'

	authenticationOK                = 0
	authenticationCleartextPassword = 3
	authenticationMD5Password       = 5
	authenticationSASL              = 10

	notificationResponseMsg = 'A'

	describeMsg             = 'D'
	parameterDescriptionMsg = 't'

	queryMsg              = 'Q'
	readyForQueryMsg      = 'Z'
	emptyQueryResponseMsg = 'I'
	rowDescriptionMsg     = 'T'
	dataRowMsg            = 'D'

	parseMsg         = 'P'
	parseCompleteMsg = '1'

	bindMsg         = 'B'
	bindCompleteMsg = '2'

	executeMsg = 'E'

	syncMsg  = 'S'
	flushMsg = 'H'

	closeMsg         = 'C'
	closeCompleteMsg = '3'

	copyInResponseMsg  = 'G'
	copyOutResponseMsg = 'H'
	copyDataMsg        = 'd'
	copyDoneMsg        = 'c'
)

var errEmptyQuery = errors.New("pgdriver: query is empty")

type reader struct {
	*bufio.Reader
	buf []byte
}

func newReader(r io.Reader) *reader {
	return &reader{
		Reader: bufio.NewReader(r),
		buf:    make([]byte, 128),
	}
}

func (r *reader) ReadTemp(n int) ([]byte, error) {
	if n <= len(r.buf) {
		b := r.buf[:n]
		_, err := io.ReadFull(r.Reader, b)
		return b, err
	}

	b := make([]byte, n)
	_, err := io.ReadFull(r.Reader, b)
	return b, err
}

func (r *reader) Discard(n int) error {
	_, err := r.ReadTemp(n)
	return err
}

func enableSSL(ctx context.Context, cn *Conn, tlsConf *tls.Config) error {
	if err := writeSSLMsg(ctx, cn); err != nil {
		return err
	}

	rd := cn.reader(ctx, -1)

	c, err := rd.ReadByte()
	if err != nil {
		return err
	}
	if c != 'S' {
		return errors.New("pgdriver: SSL is not enabled on the server")
	}

	tlsCN := tls.Client(cn.netConn, tlsConf)
	if err := tlsCN.HandshakeContext(ctx); err != nil {
		return fmt.Errorf("pgdriver: TLS handshake failed: %w", err)
	}
	cn.netConn = tlsCN
	rd.Reset(cn.netConn)

	return nil
}

func writeSSLMsg(ctx context.Context, cn *Conn) error {
	wb := getWriteBuffer()
	defer putWriteBuffer(wb)

	wb.StartMessage(0)
	wb.WriteInt32(80877103)
	wb.FinishMessage()

	return cn.write(ctx, wb)
}

//------------------------------------------------------------------------------

func startup(ctx context.Context, cn *Conn) error {
	if err := writeStartup(ctx, cn); err != nil {
		return err
	}

	rd := cn.reader(ctx, -1)

	for {
		c, msgLen, err := readMessageType(rd)
		if err != nil {
			return err
		}

		switch c {
		case backendKeyDataMsg:
			processID, err := readInt32(rd)
			if err != nil {
				return err
			}
			secretKey, err := readInt32(rd)
			if err != nil {
				return err
			}
			cn.processID = processID
			cn.secretKey = secretKey
		case authenticationOKMsg:
			if err := auth(ctx, cn, rd); err != nil {
				return err
			}
		case readyForQueryMsg:
			return rd.Discard(msgLen)
		case parameterStatusMsg, noticeResponseMsg:
			if err := rd.Discard(msgLen); err != nil {
				return err
			}
		case errorResponseMsg:
			e, err := readError(rd)
			if err != nil {
				return err
			}
			return e
		default:
			return fmt.Errorf("pgdriver: unexpected startup message: %q", c)
		}
	}
}

func writeStartup(ctx context.Context, cn *Conn) error {
	wb := getWriteBuffer()
	defer putWriteBuffer(wb)

	wb.StartMessage(0)
	wb.WriteInt32(196608)
	wb.WriteString("user")
	wb.WriteString(cn.cfg.User)
	wb.WriteString("database")
	wb.WriteString(cn.cfg.Database)
	if cn.cfg.AppName != "" {
		wb.WriteString("application_name")
		wb.WriteString(cn.cfg.AppName)
	}
	wb.WriteString("")
	wb.FinishMessage()

	return cn.write(ctx, wb)
}

//------------------------------------------------------------------------------

func auth(ctx context.Context, cn *Conn, rd *reader) error {
	num, err := readInt32(rd)
	if err != nil {
		return err
	}

	switch num {
	case authenticationOK:
		return nil
	case authenticationCleartextPassword:
		return authCleartext(ctx, cn, rd)
	case authenticationMD5Password:
		return authMD5(ctx, cn, rd)
	case authenticationSASL:
		if err := authSASL(ctx, cn, rd); err != nil {
			return fmt.Errorf("pgdriver: SASL: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("pgdriver: unknown authentication message: %q", num)
	}
}

func authCleartext(ctx context.Context, cn *Conn, rd *reader) error {
	if err := writePassword(ctx, cn, cn.cfg.Password); err != nil {
		return err
	}
	return readAuthOK(cn, rd)
}

func readAuthOK(cn *Conn, rd *reader) error {
	c, _, err := readMessageType(rd)
	if err != nil {
		return err
	}

	switch c {
	case authenticationOKMsg:
		num, err := readInt32(rd)
		if err != nil {
			return err
		}
		if num != 0 {
			return fmt.Errorf("pgdriver: unexpected authentication code: %q", num)
		}
		return nil
	case errorResponseMsg:
		e, err := readError(rd)
		if err != nil {
			return err
		}
		return e
	default:
		return fmt.Errorf("pgdriver: unknown password message: %q", c)
	}
}

//------------------------------------------------------------------------------

func authMD5(ctx context.Context, cn *Conn, rd *reader) error {
	b, err := rd.ReadTemp(4)
	if err != nil {
		return err
	}

	secret := "md5" + md5s(md5s(cn.cfg.Password+cn.cfg.User)+string(b))
	if err := writePassword(ctx, cn, secret); err != nil {
		return err
	}

	return readAuthOK(cn, rd)
}

func writePassword(ctx context.Context, cn *Conn, password string) error {
	wb := getWriteBuffer()
	defer putWriteBuffer(wb)

	wb.StartMessage(passwordMessageMsg)
	wb.WriteString(password)
	wb.FinishMessage()

	return cn.write(ctx, wb)
}

func md5s(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

//------------------------------------------------------------------------------

func authSASL(ctx context.Context, cn *Conn, rd *reader) error {
	var saslMech sasl.Mechanism

loop:
	for {
		s, err := readString(rd)
		if err != nil {
			return err
		}

		switch s {
		case "":
			break loop
		case sasl.ScramSha256.Name:
			saslMech = sasl.ScramSha256
		case sasl.ScramSha256Plus.Name:
			// ignore
		default:
			return fmt.Errorf("got %q, wanted %q", s, sasl.ScramSha256.Name)
		}
	}

	creds := sasl.Credentials(func() (Username, Password, Identity []byte) {
		return []byte(cn.cfg.User), []byte(cn.cfg.Password), nil
	})
	client := sasl.NewClient(saslMech, creds)

	_, resp, err := client.Step(nil)
	if err != nil {
		return fmt.Errorf("client.Step 1 failed: %w", err)
	}

	if err := saslWriteInitialResponse(ctx, cn, saslMech, resp); err != nil {
		return err
	}

	c, msgLen, err := readMessageType(rd)
	if err != nil {
		return err
	}

	switch c {
	case authenticationSASLContinueMsg:
		c11, err := readInt32(rd)
		if err != nil {
			return err
		}
		if c11 != 11 {
			return fmt.Errorf("got %q, wanted %q", c, 11)
		}

		b, err := rd.ReadTemp(msgLen - 4)
		if err != nil {
			return err
		}

		_, resp, err = client.Step(b)
		if err != nil {
			return fmt.Errorf("client.Step 2 failed: %w", err)
		}

		if err := saslWriteResponse(ctx, cn, resp); err != nil {
			return err
		}

		resp, err = saslReadAuthFinal(cn, rd)
		if err != nil {
			return err
		}

		if _, _, err := client.Step(resp); err != nil {
			return fmt.Errorf("client.Step 3 failed: %w", err)
		}

		if client.State() != sasl.ValidServerResponse {
			return fmt.Errorf("got state=%q, wanted %q", client.State(), sasl.ValidServerResponse)
		}

		return nil
	case errorResponseMsg:
		e, err := readError(rd)
		if err != nil {
			return err
		}
		return e
	default:
		return fmt.Errorf("got %q, wanted %q", c, authenticationSASLContinueMsg)
	}
}

func saslWriteInitialResponse(
	ctx context.Context, cn *Conn, saslMech sasl.Mechanism, resp []byte,
) error {
	wb := getWriteBuffer()
	defer putWriteBuffer(wb)

	wb.StartMessage(saslInitialResponseMsg)
	wb.WriteString(saslMech.Name)
	wb.WriteInt32(int32(len(resp)))
	if _, err := wb.Write(resp); err != nil {
		return err
	}
	wb.FinishMessage()

	return cn.write(ctx, wb)
}

func saslWriteResponse(ctx context.Context, cn *Conn, resp []byte) error {
	wb := getWriteBuffer()
	defer putWriteBuffer(wb)

	wb.StartMessage(saslResponseMsg)
	if _, err := wb.Write(resp); err != nil {
		return err
	}
	wb.FinishMessage()

	return cn.write(ctx, wb)
}

func saslReadAuthFinal(cn *Conn, rd *reader) ([]byte, error) {
	c, msgLen, err := readMessageType(rd)
	if err != nil {
		return nil, err
	}

	switch c {
	case authenticationSASLFinalMsg:
		c12, err := readInt32(rd)
		if err != nil {
			return nil, err
		}
		if c12 != 12 {
			return nil, fmt.Errorf("got %q, wanted %q", c, 12)
		}

		resp := make([]byte, msgLen-4)
		if _, err := io.ReadFull(rd, resp); err != nil {
			return nil, err
		}

		if err := readAuthOK(cn, rd); err != nil {
			return nil, err
		}

		return resp, nil
	case errorResponseMsg:
		e, err := readError(rd)
		if err != nil {
			return nil, err
		}
		return nil, e
	default:
		return nil, fmt.Errorf("got %q, wanted %q", c, authenticationSASLFinalMsg)
	}
}

//------------------------------------------------------------------------------

func writeQuery(ctx context.Context, cn *Conn, query string) error {
	wb := getWriteBuffer()
	defer putWriteBuffer(wb)

	wb.StartMessage(queryMsg)
	wb.WriteString(query)
	wb.FinishMessage()

	return cn.write(ctx, wb)
}

func readQuery(ctx context.Context, cn *Conn) (sql.Result, error) {
	rd := cn.reader(ctx, -1)

	var res driver.Result
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
		case emptyQueryResponseMsg:
			if firstErr == nil {
				firstErr = errEmptyQuery
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
		case describeMsg,
			rowDescriptionMsg,
			noticeResponseMsg,
			parameterStatusMsg:
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
		case readyForQueryMsg:
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
			return res, firstErr
		default:
			return nil, fmt.Errorf("pgdriver: Exec: unexpected message %q", c)
		}
	}
}

func readQueryData(ctx context.Context, cn *Conn) (*rows, error) {
	rd := cn.reader(ctx, -1)
	var firstErr error
	for {
		c, msgLen, err := readMessageType(rd)
		if err != nil {
			return nil, err
		}

		switch c {
		case rowDescriptionMsg:
			rowDesc, err := readRowDescription(rd)
			if err != nil {
				return nil, err
			}
			return newRows(cn, rowDesc, true), nil
		case commandCompleteMsg:
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
		case readyForQueryMsg:
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
			if firstErr != nil {
				return nil, firstErr
			}
			return &rows{closed: true}, nil
		case errorResponseMsg:
			e, err := readError(rd)
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
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("pgdriver: newRows: unexpected message %q", c)
		}
	}
}

//------------------------------------------------------------------------------

var rowDescPool sync.Pool

type rowDescription struct {
	buf      []byte
	names    []string
	types    []int32
	numInput int16
}

func newRowDescription(numCol int) *rowDescription {
	if numCol < 16 {
		numCol = 16
	}
	return &rowDescription{
		buf:      make([]byte, 0, 16*numCol),
		names:    make([]string, 0, numCol),
		types:    make([]int32, 0, numCol),
		numInput: -1,
	}
}

func (d *rowDescription) reset(numCol int) {
	d.buf = make([]byte, 0, 16*numCol)
	d.names = d.names[:0]
	d.types = d.types[:0]
	d.numInput = -1
}

func (d *rowDescription) addName(name []byte) {
	if len(d.buf)+len(name) > cap(d.buf) {
		d.buf = make([]byte, 0, cap(d.buf))
	}

	i := len(d.buf)
	d.buf = append(d.buf, name...)
	d.names = append(d.names, bytesToString(d.buf[i:]))
}

func (d *rowDescription) addType(dataType int32) {
	d.types = append(d.types, dataType)
}

func readRowDescription(rd *reader) (*rowDescription, error) {
	numCol, err := readInt16(rd)
	if err != nil {
		return nil, err
	}

	rowDesc, ok := rowDescPool.Get().(*rowDescription)
	if !ok {
		rowDesc = newRowDescription(int(numCol))
	} else {
		rowDesc.reset(int(numCol))
	}

	for i := 0; i < int(numCol); i++ {
		name, err := rd.ReadSlice(0)
		if err != nil {
			return nil, err
		}
		rowDesc.addName(name[:len(name)-1])

		if _, err := rd.ReadTemp(6); err != nil {
			return nil, err
		}

		dataType, err := readInt32(rd)
		if err != nil {
			return nil, err
		}
		rowDesc.addType(dataType)

		if _, err := rd.ReadTemp(8); err != nil {
			return nil, err
		}
	}

	return rowDesc, nil
}

//------------------------------------------------------------------------------

func readNotification(ctx context.Context, rd *reader) (channel, payload string, err error) {
	for {
		c, msgLen, err := readMessageType(rd)
		if err != nil {
			return "", "", err
		}

		switch c {
		case commandCompleteMsg, readyForQueryMsg, noticeResponseMsg:
			if err := rd.Discard(msgLen); err != nil {
				return "", "", err
			}
		case errorResponseMsg:
			e, err := readError(rd)
			if err != nil {
				return "", "", err
			}
			return "", "", e
		case notificationResponseMsg:
			if err := rd.Discard(4); err != nil {
				return "", "", err
			}
			channel, err = readString(rd)
			if err != nil {
				return "", "", err
			}
			payload, err = readString(rd)
			if err != nil {
				return "", "", err
			}
			return channel, payload, nil
		default:
			return "", "", fmt.Errorf("pgdriver: readNotification: unexpected message %q", c)
		}
	}
}

//------------------------------------------------------------------------------

func writeParseDescribeSync(ctx context.Context, cn *Conn, name, query string) error {
	wb := getWriteBuffer()
	defer putWriteBuffer(wb)

	wb.StartMessage(parseMsg)
	wb.WriteString(name)
	wb.WriteString(query)
	wb.WriteInt16(0)
	wb.FinishMessage()

	wb.StartMessage(describeMsg)
	wb.WriteByte('S')
	wb.WriteString(name)
	wb.FinishMessage()

	wb.StartMessage(syncMsg)
	wb.FinishMessage()

	return cn.write(ctx, wb)
}

func readParseDescribeSync(ctx context.Context, cn *Conn) (*rowDescription, error) {
	rd := cn.reader(ctx, -1)
	var numParam int16
	var rowDesc *rowDescription
	var firstErr error
	for {
		c, msgLen, err := readMessageType(rd)
		if err != nil {
			return nil, err
		}

		switch c {
		case parseCompleteMsg:
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
		case rowDescriptionMsg: // response to DESCRIBE message.
			rowDesc, err = readRowDescription(rd)
			if err != nil {
				return nil, err
			}
			rowDesc.numInput = numParam
		case parameterDescriptionMsg: // response to DESCRIBE message.
			numParam, err = readInt16(rd)
			if err != nil {
				return nil, err
			}

			for i := 0; i < int(numParam); i++ {
				if _, err := readInt32(rd); err != nil {
					return nil, err
				}
			}
		case noDataMsg: // response to DESCRIBE message.
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
		case readyForQueryMsg:
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
			if firstErr != nil {
				return nil, firstErr
			}
			return rowDesc, err
		case errorResponseMsg:
			e, err := readError(rd)
			if err != nil {
				return nil, err
			}
			if firstErr == nil {
				firstErr = e
			}
		case noticeResponseMsg, parameterStatusMsg:
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("pgdriver: readParseDescribeSync: unexpected message %q", c)
		}
	}
}

func writeBindExecute(ctx context.Context, cn *Conn, name string, args []driver.NamedValue) error {
	wb := getWriteBuffer()
	defer putWriteBuffer(wb)

	wb.StartMessage(bindMsg)
	wb.WriteString("")
	wb.WriteString(name)
	wb.WriteInt16(0)
	wb.WriteInt16(int16(len(args)))
	for i := range args {
		wb.StartParam()
		bytes, err := appendStmtArg(wb.Bytes, args[i].Value)
		if err != nil {
			return err
		}
		if bytes != nil {
			wb.Bytes = bytes
			wb.FinishParam()
		} else {
			wb.FinishNullParam()
		}
	}
	wb.WriteInt16(0)
	wb.FinishMessage()

	wb.StartMessage(executeMsg)
	wb.WriteString("")
	wb.WriteInt32(0)
	wb.FinishMessage()

	wb.StartMessage(syncMsg)
	wb.FinishMessage()

	return cn.write(ctx, wb)
}

func readExtQuery(ctx context.Context, cn *Conn) (driver.Result, error) {
	rd := cn.reader(ctx, -1)
	var res driver.Result
	var firstErr error
	for {
		c, msgLen, err := readMessageType(rd)
		if err != nil {
			return nil, err
		}

		switch c {
		case bindCompleteMsg, dataRowMsg:
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
		case commandCompleteMsg: // response to EXECUTE message.
			tmp, err := rd.ReadTemp(msgLen)
			if err != nil {
				return nil, err
			}

			r, err := parseResult(tmp)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
			} else {
				res = r
			}
		case readyForQueryMsg: // Response to SYNC message.
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
			if firstErr != nil {
				return nil, firstErr
			}
			return res, nil
		case errorResponseMsg:
			e, err := readError(rd)
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
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("pgdriver: readExtQuery: unexpected message %q", c)
		}
	}
}

func readExtQueryData(ctx context.Context, cn *Conn, rowDesc *rowDescription) (*rows, error) {
	rd := cn.reader(ctx, -1)
	var firstErr error
	for {
		c, msgLen, err := readMessageType(rd)
		if err != nil {
			return nil, err
		}

		switch c {
		case bindCompleteMsg:
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
			return newRows(cn, rowDesc, false), nil
		case commandCompleteMsg: // response to EXECUTE message.
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
		case readyForQueryMsg: // Response to SYNC message.
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
			if firstErr != nil {
				return nil, firstErr
			}
			return &rows{closed: true}, nil
		case errorResponseMsg:
			e, err := readError(rd)
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
			if err := rd.Discard(msgLen); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("pgdriver: readExtQueryData: unexpected message %q", c)
		}
	}
}

func writeCloseStmt(ctx context.Context, cn *Conn, name string) error {
	wb := getWriteBuffer()
	defer putWriteBuffer(wb)

	wb.StartMessage(closeMsg)
	wb.WriteByte('S') //nolint
	wb.WriteString(name)
	wb.FinishMessage()

	wb.StartMessage(flushMsg)
	wb.FinishMessage()

	return cn.write(ctx, wb)
}

func readCloseStmtComplete(ctx context.Context, cn *Conn) error {
	rd := cn.reader(ctx, -1)
	for {
		c, msgLen, err := readMessageType(rd)
		if err != nil {
			return err
		}

		switch c {
		case closeCompleteMsg:
			return rd.Discard(msgLen)
		case errorResponseMsg:
			e, err := readError(rd)
			if err != nil {
				return err
			}
			return e
		case noticeResponseMsg, parameterStatusMsg:
			if err := rd.Discard(msgLen); err != nil {
				return err
			}
		default:
			return fmt.Errorf("pgdriver: readCloseCompleteMsg: unexpected message %q", c)
		}
	}
}

//------------------------------------------------------------------------------

func readMessageType(rd *reader) (byte, int, error) {
	c, err := rd.ReadByte()
	if err != nil {
		return 0, 0, err
	}
	l, err := readInt32(rd)
	if err != nil {
		return 0, 0, err
	}
	return c, int(l) - 4, nil
}

func readInt16(rd *reader) (int16, error) {
	b, err := rd.ReadTemp(2)
	if err != nil {
		return 0, err
	}
	return int16(binary.BigEndian.Uint16(b)), nil
}

func readInt32(rd *reader) (int32, error) {
	b, err := rd.ReadTemp(4)
	if err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(b)), nil
}

func readString(rd *reader) (string, error) {
	b, err := rd.ReadSlice(0)
	if err != nil {
		return "", err
	}
	return string(b[:len(b)-1]), nil
}

func readError(rd *reader) (error, error) {
	m := make(map[byte]string)
	for {
		c, err := rd.ReadByte()
		if err != nil {
			return nil, err
		}
		if c == 0 {
			break
		}
		s, err := readString(rd)
		if err != nil {
			return nil, err
		}
		m[c] = s
	}
	return Error{m: m}, nil
}

//------------------------------------------------------------------------------

func appendStmtArg(b []byte, v driver.Value) ([]byte, error) {
	switch v := v.(type) {
	case nil:
		return nil, nil
	case int64:
		return strconv.AppendInt(b, v, 10), nil
	case float64:
		switch {
		case math.IsNaN(v):
			return append(b, "NaN"...), nil
		case math.IsInf(v, 1):
			return append(b, "Infinity"...), nil
		case math.IsInf(v, -1):
			return append(b, "-Infinity"...), nil
		default:
			return strconv.AppendFloat(b, v, 'f', -1, 64), nil
		}
	case bool:
		if v {
			return append(b, "TRUE"...), nil
		}
		return append(b, "FALSE"...), nil
	case []byte:
		if v == nil {
			return nil, nil
		}

		b = append(b, `\x`...)

		s := len(b)
		b = append(b, make([]byte, hex.EncodedLen(len(v)))...)
		hex.Encode(b[s:], v)

		return b, nil
	case string:
		for _, r := range v {
			if r == 0 {
				continue
			}
			if r < utf8.RuneSelf {
				b = append(b, byte(r))
				continue
			}
			l := len(b)
			if cap(b)-l < utf8.UTFMax {
				b = append(b, make([]byte, utf8.UTFMax)...)
			}
			n := utf8.EncodeRune(b[l:l+utf8.UTFMax], r)
			b = b[:l+n]
		}
		return b, nil
	case time.Time:
		if v.IsZero() {
			return nil, nil
		}
		return v.UTC().AppendFormat(b, "2006-01-02 15:04:05.999999-07:00"), nil
	default:
		return nil, fmt.Errorf("pgdriver: unexpected arg: %T", v)
	}
}
