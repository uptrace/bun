package pgdriver

import (
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/uptrace/bun/internal"
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

func readColumnValue(rd *reader, dataType int32, dataLen int) (any, error) {
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
		// Return a string and let the scanner convert the string to time.Time if necessary.
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

func readBoolCol(rd *reader, n int) (any, error) {
	tmp, err := rd.ReadTemp(n)
	if err != nil {
		return nil, err
	}
	return len(tmp) == 1 && (tmp[0] == 't' || tmp[0] == '1'), nil
}

func readIntCol(rd *reader, n int, bitSize int) (any, error) {
	if n <= 0 {
		return 0, nil
	}

	tmp, err := rd.ReadTemp(n)
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(internal.String(tmp), 10, bitSize)
}

func readFloatCol(rd *reader, n int, bitSize int) (any, error) {
	if n <= 0 {
		return 0, nil
	}

	tmp, err := rd.ReadTemp(n)
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(internal.String(tmp), bitSize)
}

func readStringCol(rd *reader, n int) (any, error) {
	if n <= 0 {
		return "", nil
	}

	b := make([]byte, n)

	if _, err := io.ReadFull(rd, b); err != nil {
		return nil, err
	}

	return internal.String(b), nil
}

func readBytesCol(rd *reader, n int) (any, error) {
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

func readTimeCol(rd *reader, n int) (any, error) {
	if n <= 0 {
		return time.Time{}, nil
	}

	tmp, err := rd.ReadTemp(n)
	if err != nil {
		return time.Time{}, err
	}

	tm, err := ParseTime(internal.String(tmp))
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
				return parseTimestampUTC(s[:len(s)-3])
			}
			return time.Parse(timestamptzFormat3, s)
		}
		return parseTimestampUTC(s)
	}
}

// parseTimestampUTC parses timestampFormat in UTC. It is a separate function,
// so that the fast path does not make ParseTime slower for other formats.
func parseTimestampUTC(s string) (time.Time, error) {
	if tm, ok := parseTimestampFast(s); ok {
		return tm, nil
	}
	return time.ParseInLocation(timestampFormat, s, time.UTC)
}

// parseTimestampFast is a fast path for "YYYY-MM-DD HH:MM:SS[.fraction]" in UTC.
// It returns false for any other input, so the caller can use time.Parse.
func parseTimestampFast(s string) (time.Time, bool) {
	if len(s) < len("2006-01-02 15:04:05") || len(s) > len(timestampFormat) {
		return time.Time{}, false
	}
	if s[4] != '-' || s[7] != '-' || s[10] != ' ' || s[13] != ':' || s[16] != ':' {
		return time.Time{}, false
	}

	// Check each field as soon as it is read, so that bad input leaves early.
	month, ok := atoi2(s[5:7])
	if !ok || month < 1 || month > 12 {
		return time.Time{}, false
	}
	hour, ok := atoi2(s[11:13])
	if !ok || hour > 23 {
		return time.Time{}, false
	}
	minute, ok := atoi2(s[14:16])
	if !ok || minute > 59 {
		return time.Time{}, false
	}
	sec, ok := atoi2(s[17:19])
	if !ok || sec > 59 {
		return time.Time{}, false
	}
	y1, ok1 := atoi2(s[0:2])
	y2, ok2 := atoi2(s[2:4])
	day, ok3 := atoi2(s[8:10])
	year := y1*100 + y2
	if !ok1 || !ok2 || !ok3 || day < 1 || day > daysInMonth(year, month) {
		return time.Time{}, false
	}

	var nsec int
	if len(s) > 19 {
		// Fraction: a dot and 1 to 9 digits.
		if s[19] != '.' || len(s) == 20 {
			return time.Time{}, false
		}
		for i := 20; i < len(s); i++ {
			c := s[i]
			if c < '0' || c > '9' {
				return time.Time{}, false
			}
			nsec = nsec*10 + int(c-'0')
		}
		for i := len(s); i < len(timestampFormat); i++ {
			nsec *= 10
		}
	}

	return time.Date(year, time.Month(month), day, hour, minute, sec, nsec, time.UTC), true
}

func atoi2(s string) (int, bool) {
	a, b := s[0]-'0', s[1]-'0'
	if a > 9 || b > 9 {
		return 0, false
	}
	return int(a)*10 + int(b), true
}

func daysInMonth(year, month int) int {
	switch month {
	case 2:
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			return 29
		}
		return 28
	case 4, 6, 9, 11:
		return 30
	}
	return 31
}
