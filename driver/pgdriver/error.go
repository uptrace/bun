package pgdriver

import (
	"database/sql/driver"
	"fmt"
	"net"
)

// Error represents an error returned by PostgreSQL server
// using PostgreSQL ErrorResponse protocol.
//
// https://www.postgresql.org/docs/current/static/protocol-message-formats.html
type Error struct {
	m map[byte]string
}

// Field returns a string value associated with an error field.
//
// https://www.postgresql.org/docs/current/static/protocol-error-fields.html
func (err Error) Field(k byte) string {
	return err.m[k]
}

// IntegrityViolation reports whether an error is a part of
// Integrity Constraint Violation class of errors.
//
// https://www.postgresql.org/docs/current/static/errcodes-appendix.html
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

	if err, ok := err.(Error); ok {
		switch err.Field('V') {
		case "FATAL", "PANIC":
			return true
		}
		switch err.Field('C') {
		case "25P02", // current transaction is aborted
			"57014": // canceling statement due to user request
			return true
		}
		return false
	}

	if allowTimeout {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return !err.Temporary()
		}
	}

	return true
}

func checkBadConn(err error) error {
	if isBadConn(err, false) {
		return driver.ErrBadConn
	}
	return err
}
