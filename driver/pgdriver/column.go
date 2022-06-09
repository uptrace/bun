package pgdriver

import (
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	pgBool = 16

	pgInt2 = 21
	pgInt4 = 23
	pgInt8 = 20

	pgFloat4 = 700
	pgFloat8 = 701

	pgText    = 25
	pgVarchar = 1043
	pgBytea   = 17

	pgDate        = 1082
	pgTimestamp   = 1114
	pgTimestamptz = 1184
)

func readColumnValue(rd *reader, dataType int32, dataLen int) (interface{}, error) {
	if dataLen == -1 {
		return nil, nil
	}

	switch dataType {
	case pgBool:
		return readBoolCol(rd, dataLen)
	case pgInt2:
		return readIntCol(rd, dataLen, 16)
	case pgInt4:
		return readIntCol(rd, dataLen, 32)
	case pgInt8:
		return readIntCol(rd, dataLen, 64)
	case pgFloat4:
		return readFloatCol(rd, dataLen, 32)
	case pgFloat8:
		return readFloatCol(rd, dataLen, 64)
	case pgTimestamp:
		return readTimeCol(rd, dataLen)
	case pgTimestamptz:
		return readTimeCol(rd, dataLen)
	case pgDate:
		// Return a string and let the scanner to convert string to time.Time if necessary.
		return readStringCol(rd, dataLen)
	case pgText, pgVarchar:
		return readStringCol(rd, dataLen)
	case pgBytea:
		return readBytesCol(rd, dataLen)
	}

	b := make([]byte, dataLen)
	if _, err := io.ReadFull(rd, b); err != nil {
		return nil, err
	}
	return b, nil
}

func readBoolCol(rd *reader, n int) (interface{}, error) {
	tmp, err := rd.ReadTemp(n)
	if err != nil {
		return nil, err
	}
	return len(tmp) == 1 && (tmp[0] == 't' || tmp[0] == '1'), nil
}

func readIntCol(rd *reader, n int, bitSize int) (interface{}, error) {
	if n <= 0 {
		return 0, nil
	}

	tmp, err := rd.ReadTemp(n)
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(bytesToString(tmp), 10, bitSize)
}

func readFloatCol(rd *reader, n int, bitSize int) (interface{}, error) {
	if n <= 0 {
		return 0, nil
	}

	tmp, err := rd.ReadTemp(n)
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(bytesToString(tmp), bitSize)
}

func readStringCol(rd *reader, n int) (interface{}, error) {
	if n <= 0 {
		return "", nil
	}

	b := make([]byte, n)

	if _, err := io.ReadFull(rd, b); err != nil {
		return nil, err
	}

	return bytesToString(b), nil
}

func readBytesCol(rd *reader, n int) (interface{}, error) {
	if n <= 0 {
		return []byte{}, nil
	}

	tmp, err := rd.ReadTemp(n)
	if err != nil {
		return nil, err
	}

	if len(tmp) < 2 || tmp[0] != '\\' || tmp[1] != 'x' {
		return nil, fmt.Errorf("pgdriver: can't parse bytea: %q", tmp)
	}
	tmp = tmp[2:] // Cut off "\x".

	b := make([]byte, hex.DecodedLen(len(tmp)))
	if _, err := hex.Decode(b, tmp); err != nil {
		return nil, err
	}
	return b, nil
}

func readTimeCol(rd *reader, n int) (interface{}, error) {
	if n <= 0 {
		return time.Time{}, nil
	}

	tmp, err := rd.ReadTemp(n)
	if err != nil {
		return time.Time{}, err
	}

	tm, err := ParseTime(bytesToString(tmp))
	if err != nil {
		return time.Time{}, err
	}
	return tm, nil
}

const (
	dateFormat         = "2006-01-02"
	timeFormat         = "15:04:05.999999999"
	timestampFormat    = "2006-01-02 15:04:05.999999999"
	timestamptzFormat  = "2006-01-02 15:04:05.999999999-07:00:00"
	timestamptzFormat2 = "2006-01-02 15:04:05.999999999-07:00"
	timestamptzFormat3 = "2006-01-02 15:04:05.999999999-07"
)

func ParseTime(s string) (time.Time, error) {
	switch l := len(s); {
	case l < len("15:04:05"):
		return time.Time{}, fmt.Errorf("pgdriver: can't parse time=%q", s)
	case l <= len(timeFormat):
		if s[2] == ':' {
			return time.ParseInLocation(timeFormat, s, time.UTC)
		}
		return time.ParseInLocation(dateFormat, s, time.UTC)
	default:
		if s[10] == 'T' {
			return time.Parse(time.RFC3339Nano, s)
		}
		if c := s[l-9]; c == '+' || c == '-' {
			return time.Parse(timestamptzFormat, s)
		}
		if c := s[l-6]; c == '+' || c == '-' {
			return time.Parse(timestamptzFormat2, s)
		}
		if c := s[l-3]; c == '+' || c == '-' {
			if strings.HasSuffix(s, "+00") {
				s = s[:len(s)-3]
				return time.ParseInLocation(timestampFormat, s, time.UTC)
			}
			return time.Parse(timestamptzFormat3, s)
		}
		return time.ParseInLocation(timestampFormat, s, time.UTC)
	}
}
