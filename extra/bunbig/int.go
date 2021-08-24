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

// same as NewBigint()
func FromMathBig(x *big.Int) *Int {
	return (*Int)(x)
}

func FromInt64(x int64) *Int {
	return FromMathBig(big.NewInt(x))
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

func (b *Int) Value() (driver.Value, error) {
	return (*big.Int)(b).String(), nil
}

func (b *Int) Scan(value interface{}) error {

	var i sql.NullString

	if err := i.Scan(value); err != nil {
		return err
	}

	if _, ok := (*big.Int)(b).SetString(i.String, 10); ok {
		return nil
	}

	return fmt.Errorf("Error converting type %T into Bigint", value)
}

func (b *Int) ToMathBig() *big.Int {
	return (*big.Int)(b)
}

func (b *Int) Sub(x *Int) *Int {
	return (*Int)(big.NewInt(0).Sub(b.ToMathBig(), x.ToMathBig()))
}

func (b *Int) Add(x *Int) *Int {
	return (*Int)(big.NewInt(0).Add(b.ToMathBig(), x.ToMathBig()))
}

func (b *Int) Mul(x *Int) *Int {
	return (*Int)(big.NewInt(0).Mul(b.ToMathBig(), x.ToMathBig()))
}

func (b *Int) Div(x *Int) *Int {
	return (*Int)(big.NewInt(0).Div(b.ToMathBig(), x.ToMathBig()))
}

func (b *Int) Neg() *Int {
	return (*Int)(big.NewInt(0).Neg(b.ToMathBig()))
}

func (b *Int) ToUInt64() uint64 {
	return b.ToMathBig().Uint64()
}

func (b *Int) ToInt64() int64 {
	return b.ToMathBig().Int64()
}

func (b *Int) String() string {
	return b.ToMathBig().String()
}

func (b *Int) Abs() *Int {
	return (*Int)(new(big.Int).Abs(b.ToMathBig()))
}

var _ yaml.Unmarshaler = (*Int)(nil)

func (b *Int) UnmarshalYAML(value *yaml.Node) error {
	var str string
	if err := value.Decode(&str); err != nil {
		return err
	}

	var err error
	b, err = NewInt().FromString(str)
	return err
}

func (b *Int) Cmp(target *Int) Cmp {
	return &cmpInt{r: b.ToMathBig().Cmp(target.ToMathBig())}
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
