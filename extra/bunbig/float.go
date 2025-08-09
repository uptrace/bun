package bunbig

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"math/big"
)

type (
	Float big.Float
)

func NewFloat() *Float {
	return new(Float)
}

func (f *Float) FromMathFloat(fl *big.Float) *Float {
	return (*Float)(fl)
}

func (f *Float) Value() (driver.Value, error) {
	return (*big.Float)(f).String(), nil
}

func (f *Float) Scan(value any) error {

	var i sql.NullString

	if err := i.Scan(value); err != nil {
		return err
	}

	if _, ok := (*big.Float)(f).SetString(i.String); ok {
		return nil
	}

	return fmt.Errorf("Error converting type %T into Float", value)
}

func (f *Float) toMathFloat() *big.Float {
	return (*big.Float)(f)
}

func (f *Float) Add(target *Float) *Float {
	return f.FromMathFloat(new(big.Float).Add(f.toMathFloat(), target.toMathFloat()))
}
func (f *Float) Sub(target *Float) *Float {
	return f.FromMathFloat(new(big.Float).Sub(f.toMathFloat(), target.toMathFloat()))
}
func (f *Float) Mul(target *Float) *Float {
	return f.FromMathFloat(new(big.Float).Mul(f.toMathFloat(), target.toMathFloat()))
}
func (f *Float) Div(target *Float) *Float {
	return f.FromMathFloat(new(big.Float).Quo(f.toMathFloat(), target.toMathFloat()))
}
func (f *Float) Neg() *Float {
	return f.FromMathFloat(new(big.Float).Neg(f.toMathFloat()))
}
func (f *Float) Abs() *Float {
	return f.FromMathFloat(new(big.Float).Abs(f.toMathFloat()))
}

func (f *Float) String() string {
	return (f.toMathFloat()).String()
}
func (f *Float) ToFloat64() (float64, int8) {
	x, a := f.toMathFloat().Float64()
	return x, int8(a)
}

func (f *Float) FromString(inp string) (*Float, error) {
	_f, _, err := big.ParseFloat(inp, 10, 10, big.ToZero)
	if err != nil {
		return nil, err
	}
	return f.FromMathFloat(_f), nil
}

func (f *Float) Cmp(target *Float) Cmp {
	return &cmpFloat{r: f.toMathFloat().Cmp(target.toMathFloat())}
}

func (c *cmpFloat) Eq() bool {
	return c.r == 0
}

func (c *cmpFloat) Lt() bool {
	return c.r < 0
}

func (c *cmpFloat) Gt() bool {
	return c.r > 0
}

func (c *cmpFloat) Leq() bool {
	return c.r == 0 || c.r < 0
}

func (c *cmpFloat) Geq() bool {
	return c.r == 0 || c.r > 0
}
