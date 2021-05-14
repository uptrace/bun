package pgdriver

import (
	"fmt"
	"net"
)

// Error represents an error returned by PostgreSQL server
// using PostgreSQL ErrorResponse protocol.
//
// https://www.postgresql.org/docs/10/static/protocol-message-formats.html
type Error struct {
	m map[byte]string
}

func (err Error) Field(k byte) string {
	return err.m[k]
}

func (err Error) IntegrityViolation() bool {
	switch err.Field('C') {
	case "23000", "23001", "23502", "23503", "23505", "23514", "23P01":
		return true
	default:
		return false
	}
}

func (err Error) Error() string {
	return fmt.Sprintf("%s #%s %s",
		err.Field('S'), err.Field('C'), err.Field('M'))
}

func isBadConn(err error, allowTimeout bool) bool {
	if err == nil {
		return false
	}
	if pgErr, ok := err.(Error); ok {
		return pgErr.Field('S') == "FATAL"
	}
	if allowTimeout {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return !netErr.Temporary()
		}
	}
	return true
}
