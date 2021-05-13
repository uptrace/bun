package pgdriver

import "fmt"

type pgError struct {
	m map[byte]string
}

func (err pgError) Field(k byte) string {
	return err.m[k]
}

func (err pgError) IntegrityViolation() bool {
	switch err.Field('C') {
	case "23000", "23001", "23502", "23503", "23505", "23514", "23P01":
		return true
	default:
		return false
	}
}

func (err pgError) Error() string {
	return fmt.Sprintf("%s #%s %s",
		err.Field('S'), err.Field('C'), err.Field('M'))
}
