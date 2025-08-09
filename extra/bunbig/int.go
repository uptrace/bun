package bunbig

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"math/big"

	"gopkg.in/yaml.v3"
)

type Int big.Int

func NewInt() *Int {
	return new(Int)
}
func newBigint(x *big.Int) *Int {
	return (*Int)(x)
}

// FromMathBig is same as NewBigint()
func FromMathBig(x *big.Int) *Int {
	return (*Int)(x)
}

func (i *Int) ToMathBig() *big.Int {
	return (*big.Int)(i)
}

func FromInt64(x int64) *Int {
	return FromMathBig(big.NewInt(x))
}

func (i *Int) ToUInt64() uint64 {
	return i.ToMathBig().Uint64()
}

func FromUInt64(x uint64) *Int {
	return FromMathBig(new(big.Int).SetUint64(x))
}

func (i *Int) ToInt64() int64 {
	return i.ToMathBig().Int64()
}

func (i *Int) FromString(x string) (*Int, error) {
	if x == "" {
		return FromInt64(0), nil
	}
	a := big.NewInt(0)
	b, ok := a.SetString(x, 10)

	if !ok {
		return nil, fmt.Errorf("cannot create Int from string")
	}

	return newBigint(b), nil
}

func (i *Int) String() string {
	return i.ToMathBig().String()
}

func (i *Int) Value() (driver.Value, error) {
	return (*big.Int)(i).String(), nil
}

func (i *Int) Scan(value any) error {
	var x sql.NullString

	if err := x.Scan(value); err != nil {
		return err
	}

	if _, ok := (*big.Int)(i).SetString(x.String, 10); ok {
		return nil
	}

	return fmt.Errorf("error converting type %T into Int", value)
}

func (i *Int) MarshalJSON() ([]byte, error) {
	return []byte(i.String()), nil
}

func (i *Int) UnmarshalJSON(p []byte) error {
	if string(p) == "null" {
		return nil
	}
	var z big.Int
	_, ok := z.SetString(string(p), 10)
	if !ok {
		return fmt.Errorf("not a valid big integer: %s", p)
	}
	*i = (Int)(z)
	return nil
}

func (i *Int) Sub(x *Int) *Int {
	return (*Int)(big.NewInt(0).Sub(i.ToMathBig(), x.ToMathBig()))
}

func (i *Int) Add(x *Int) *Int {
	return (*Int)(big.NewInt(0).Add(i.ToMathBig(), x.ToMathBig()))
}

func (i *Int) Mul(x *Int) *Int {
	return (*Int)(big.NewInt(0).Mul(i.ToMathBig(), x.ToMathBig()))
}

func (i *Int) Div(x *Int) *Int {
	return (*Int)(big.NewInt(0).Div(i.ToMathBig(), x.ToMathBig()))
}

func (i *Int) Neg() *Int {
	return (*Int)(big.NewInt(0).Neg(i.ToMathBig()))
}

func (i *Int) Abs() *Int {
	return (*Int)(new(big.Int).Abs(i.ToMathBig()))
}

var _ yaml.Unmarshaler = (*Int)(nil)

// @todo , this part needs to be fixed
func (i *Int) UnmarshalYAML(value *yaml.Node) error {
	var str string
	if err := value.Decode(&str); err != nil {
		return err
	}
	// ineffassign
	// ignored to be fixed later
	// b, err := NewInt().FromString(str)

	return nil
}

func (i *Int) Cmp(target *Int) Cmp {
	return &cmpInt{r: i.ToMathBig().Cmp(target.ToMathBig())}
}

func (c *cmpInt) Eq() bool {
	return c.r == 0
}

func (c *cmpInt) Lt() bool {
	return c.r < 0
}

func (c *cmpInt) Gt() bool {
	return c.r > 0
}

func (c *cmpInt) Leq() bool {
	return c.r == 0 || c.r < 0
}

func (c *cmpInt) Geq() bool {
	return c.r == 0 || c.r > 0
}
