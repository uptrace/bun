package pgdriver

import (
	"bufio"
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"

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

func enableSSL(ctx context.Context, cn *Conn, tlsConf *tls.Config) error {
	if err := writeSSLMsg(ctx, cn); err != nil {
		return err
	}

	c, err := cn.rd.ReadByte()
	if err != nil {
		return err
	}
	if c != 'S' {
		return errors.New("pgdriver: SSL is not enabled on the server")
	}

	cn.netConn = tls.Client(cn.netConn, tlsConf)
	cn.rd.Reset(cn.netConn)

	return nil
}

func writeSSLMsg(ctx context.Context, cn *Conn) error {
	wb := getWriteBuffer()
	defer putWriteBuffer(wb)

	wb.StartMessage(0)
	wb.WriteInt32(80877103)
	wb.FinishMessage()

	return cn.withWriter(ctx, -1, func(wr *bufio.Writer) error {
		if _, err := wr.Write(wb.Bytes); err != nil {
			return err
		}
		return wr.Flush()
	})
}

func startup(ctx context.Context, cn *Conn) error {
	if err := writeStartup(ctx, cn); err != nil {
		return err
	}

	for {
		c, msgLen, err := readMessageType(cn)
		if err != nil {
			return err
		}

		switch c {
		case backendKeyDataMsg:
			processID, err := readInt32(cn)
			if err != nil {
				return err
			}
			secretKey, err := readInt32(cn)
			if err != nil {
				return err
			}
			cn.processID = processID
			cn.secretKey = secretKey
		case authenticationOKMsg:
			if err := auth(ctx, cn); err != nil {
				return err
			}
		case readyForQueryMsg:
			return discard(cn, msgLen)
		case parameterStatusMsg, noticeResponseMsg:
			if err := discard(cn, msgLen); err != nil {
				return err
			}
		case errorResponseMsg:
			e, err := readError(cn)
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
	wb.WriteString(cn.driver.cfg.User)
	wb.WriteString("database")
	wb.WriteString(cn.driver.cfg.Database)
	if cn.driver.cfg.AppName != "" {
		wb.WriteString("application_name")
		wb.WriteString(cn.driver.cfg.AppName)
	}
	wb.WriteString("")
	wb.FinishMessage()

	return cn.withWriter(ctx, -1, func(wr *bufio.Writer) error {
		if _, err := wr.Write(wb.Bytes); err != nil {
			return err
		}
		return wr.Flush()
	})
}

//------------------------------------------------------------------------------

func auth(ctx context.Context, cn *Conn) error {
	num, err := readInt32(cn)
	if err != nil {
		return err
	}

	switch num {
	case authenticationOK:
		return nil
	case authenticationCleartextPassword:
		return authCleartext(ctx, cn)
	case authenticationMD5Password:
		return authMD5(ctx, cn)
	case authenticationSASL:
		return authSASL(ctx, cn)
	default:
		return fmt.Errorf("pgdriver: unknown authentication message: %q", num)
	}
}

func authCleartext(ctx context.Context, cn *Conn) error {
	if err := writePassword(ctx, cn, cn.driver.cfg.Password); err != nil {
		return err
	}
	return readAuthOK(cn)
}

func authMD5(ctx context.Context, cn *Conn) error {
	b, err := readN(cn, 4)
	if err != nil {
		return err
	}

	secret := "md5" + md5s(md5s(cn.driver.cfg.Password+cn.driver.cfg.User)+string(b))
	if err := writePassword(ctx, cn, secret); err != nil {
		return err
	}

	return readAuthOK(cn)
}

func writePassword(ctx context.Context, cn *Conn, password string) error {
	wb := getWriteBuffer()
	defer putWriteBuffer(wb)

	wb.StartMessage(passwordMessageMsg)
	wb.WriteString(password)
	wb.FinishMessage()

	return cn.withWriter(ctx, -1, func(wr *bufio.Writer) error {
		if _, err := wr.Write(wb.Bytes); err != nil {
			return err
		}
		return wr.Flush()
	})
}

func readAuthOK(cn *Conn) error {
	c, _, err := readMessageType(cn)
	if err != nil {
		return err
	}

	switch c {
	case authenticationOKMsg:
		num, err := readInt32(cn)
		if err != nil {
			return err
		}
		if num != 0 {
			return fmt.Errorf("pgdriver: unexpected authentication code: %q", num)
		}
		return nil
	case errorResponseMsg:
		e, err := readError(cn)
		if err != nil {
			return err
		}
		return e
	default:
		return fmt.Errorf("pgdriver: unknown password message: %q", c)
	}
}

func md5s(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

//------------------------------------------------------------------------------

func authSASL(ctx context.Context, cn *Conn) error {
	s, err := readString(cn)
	if err != nil {
		return err
	}
	if s != "SCRAM-SHA-256" {
		return fmt.Errorf("pgdriver: SASL: got %q, wanted %q", s, "SCRAM-SHA-256")
	}

	c0, err := cn.rd.ReadByte()
	if err != nil {
		return err
	}
	if c0 != 0 {
		return fmt.Errorf("pgdriver: SASL: got %q, wanted %q", c0, 0)
	}

	creds := sasl.Credentials(func() (Username, Password, Identity []byte) {
		return []byte(cn.driver.cfg.User), []byte(cn.driver.cfg.Password), nil
	})
	client := sasl.NewClient(sasl.ScramSha256, creds)

	_, resp, err := client.Step(nil)
	if err != nil {
		return err
	}

	if err := saslWriteInitialResponse(ctx, cn, resp); err != nil {
		return err
	}

	c, msgLen, err := readMessageType(cn)
	if err != nil {
		return err
	}

	switch c {
	case authenticationSASLContinueMsg:
		c11, err := readInt32(cn)
		if err != nil {
			return err
		}
		if c11 != 11 {
			return fmt.Errorf("pgdriver: SASL: got %q, wanted %q", c, 11)
		}

		b, err := readN(cn, msgLen-4)
		if err != nil {
			return err
		}

		_, resp, err = client.Step(b)
		if err != nil {
			return err
		}

		if err := saslWriteResponse(ctx, cn, resp); err != nil {
			return err
		}

		resp, err := saslReadAuthFinal(cn)
		if err != nil {
			return err
		}

		_, _, err = client.Step(resp)
		if err != nil {
			return err
		}

		if client.State() != sasl.ValidServerResponse {
			return fmt.Errorf("pgdriver: SASL: state=%q, wanted %q",
				client.State(), sasl.ValidServerResponse)
		}

		return nil
	case errorResponseMsg:
		e, err := readError(cn)
		if err != nil {
			return err
		}
		return e
	default:
		return fmt.Errorf("pgdriver: SASL: got %q, wanted %q", c, authenticationSASLContinueMsg)
	}
}

func saslWriteInitialResponse(ctx context.Context, cn *Conn, resp []byte) error {
	wb := getWriteBuffer()
	defer putWriteBuffer(wb)

	wb.StartMessage(saslInitialResponseMsg)
	wb.WriteString("SCRAM-SHA-256")
	wb.WriteInt32(int32(len(resp)))
	if _, err := wb.Write(resp); err != nil {
		return err
	}
	wb.FinishMessage()

	return cn.withWriter(ctx, -1, func(wr *bufio.Writer) error {
		if _, err := wr.Write(wb.Bytes); err != nil {
			return err
		}
		return wr.Flush()
	})
}

func saslWriteResponse(ctx context.Context, cn *Conn, resp []byte) error {
	wb := getWriteBuffer()
	defer putWriteBuffer(wb)

	wb.StartMessage(saslResponseMsg)
	if _, err := wb.Write(resp); err != nil {
		return err
	}
	wb.FinishMessage()

	return cn.withWriter(ctx, -1, func(wr *bufio.Writer) error {
		if _, err := wr.Write(wb.Bytes); err != nil {
			return err
		}
		return wr.Flush()
	})
}

func saslReadAuthFinal(cn *Conn) ([]byte, error) {
	c, msgLen, err := readMessageType(cn)
	if err != nil {
		return nil, err
	}

	switch c {
	case authenticationSASLFinalMsg:
		c12, err := readInt32(cn)
		if err != nil {
			return nil, err
		}
		if c12 != 12 {
			return nil, fmt.Errorf("pgdriver: SASL: got %q, wanted %q", c, 12)
		}

		resp, err := readN(cn, msgLen-4)
		if err != nil {
			return nil, err
		}

		return resp, readAuthOK(cn)
	case errorResponseMsg:
		e, err := readError(cn)
		if err != nil {
			return nil, err
		}
		return nil, e
	default:
		return nil, fmt.Errorf(
			"pgdriver: SASL: got %q, wanted %q", c, authenticationSASLFinalMsg)
	}
}

//------------------------------------------------------------------------------

func writeQuery(ctx context.Context, cn *Conn, query string) error {
	return cn.withWriter(ctx, -1, func(wr *bufio.Writer) error {
		if err := wr.WriteByte(queryMsg); err != nil {
			return err
		}

		binary.BigEndian.PutUint32(cn.buf, uint32(len(query)+5))
		if _, err := wr.Write(cn.buf[:4]); err != nil {
			return err
		}

		if _, err := wr.WriteString(query); err != nil {
			return err
		}
		if err := wr.WriteByte(0x0); err != nil {
			return err
		}

		return wr.Flush()
	})
}

var rowDescPool sync.Pool

type rowDescription struct {
	buf   []byte
	names []string
	types []int32
}

func newRowDescription(numCol int) *rowDescription {
	if numCol < 16 {
		numCol = 16
	}
	return &rowDescription{
		buf:   make([]byte, 0, 16*numCol),
		names: make([]string, 0, numCol),
		types: make([]int32, 0, numCol),
	}
}

func (d *rowDescription) reset(numCol int) {
	d.buf = make([]byte, 0, 16*numCol)
	d.names = d.names[:0]
	d.types = d.types[:0]
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

func readRowDescription(cn *Conn) (*rowDescription, error) {
	numCol, err := readInt16(cn)
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
		name, err := cn.rd.ReadSlice(0)
		if err != nil {
			return nil, err
		}
		rowDesc.addName(name[:len(name)-1])

		if _, err := readN(cn, 6); err != nil {
			return nil, err
		}

		dataType, err := readInt32(cn)
		if err != nil {
			return nil, err
		}
		rowDesc.addType(dataType)

		if _, err := readN(cn, 8); err != nil {
			return nil, err
		}
	}

	return rowDesc, nil
}

//------------------------------------------------------------------------------

func readNotification(cn *Conn) (channel, payload string, err error) {
	for {
		c, msgLen, err := readMessageType(cn)
		if err != nil {
			return "", "", err
		}

		switch c {
		case commandCompleteMsg, readyForQueryMsg, noticeResponseMsg:
			if err := discard(cn, msgLen); err != nil {
				return "", "", err
			}
		case errorResponseMsg:
			e, err := readError(cn)
			if err != nil {
				return "", "", err
			}
			return "", "", e
		case notificationResponseMsg:
			if err := discard(cn, 4); err != nil {
				return "", "", err
			}
			channel, err = readString(cn)
			if err != nil {
				return "", "", err
			}
			payload, err = readString(cn)
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

func readMessageType(cn *Conn) (byte, int, error) {
	c, err := cn.rd.ReadByte()
	if err != nil {
		return 0, 0, err
	}
	l, err := readInt32(cn)
	if err != nil {
		return 0, 0, err
	}
	return c, int(l) - 4, nil
}

func readInt16(cn *Conn) (int16, error) {
	b, err := readN(cn, 2)
	if err != nil {
		return 0, err
	}
	return int16(binary.BigEndian.Uint16(b)), nil
}

func readInt32(cn *Conn) (int32, error) {
	b, err := readN(cn, 4)
	if err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(b)), nil
}

func readString(cn *Conn) (string, error) {
	b, err := cn.rd.ReadSlice(0)
	if err != nil {
		return "", err
	}
	return string(b[:len(b)-1]), nil
}

func readError(cn *Conn) (error, error) {
	m := make(map[byte]string)
	for {
		c, err := cn.rd.ReadByte()
		if err != nil {
			return nil, err
		}
		if c == 0 {
			break
		}
		s, err := readString(cn)
		if err != nil {
			return nil, err
		}
		m[c] = s
	}
	return Error{m: m}, nil
}

func readN(cn *Conn, n int) ([]byte, error) {
	b := cn.buf[:n]
	if _, err := io.ReadFull(cn.rd, b); err != nil {
		return nil, err
	}
	return b, nil
}

func discard(cn *Conn, n int) error {
	if n <= len(cn.buf) {
		_, err := readN(cn, n)
		return err
	}

	b := make([]byte, n)
	_, err := io.ReadFull(cn.rd, b)
	return err
}
