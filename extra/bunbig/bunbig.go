package bunbig

// Cmp provides comparison operations for numeric types.
type Cmp interface {
	// equal
	Eq() bool
	// greater than
	Gt() bool
	// lower than
	Lt() bool
	// Greater or equal
	Geq() bool
	// Lower or equal
	Leq() bool
}
type (
	cmpInt struct {
		r int
	}
	cmpFloat struct {
		r int
	}
)
