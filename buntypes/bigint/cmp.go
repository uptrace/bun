package bigint

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
type cmp struct {
	r int
}

func (c *cmp) Eq() bool {
	return c.r == 0
}

func (c *cmp) Lt() bool {
	return c.r < 0
}

func (c *cmp) Gt() bool {
	return c.r > 0
}

func (c *cmp) Leq() bool {
	return c.r == 0 || c.r < 0
}

func (c *cmp) Geq() bool {
	return c.r == 0 || c.r > 0
}
