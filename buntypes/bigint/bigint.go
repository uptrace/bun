package bigint

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"math/big"
)

type Bigint big.Int

func newBigint(x *big.Int) *Bigint {
	return (*Bigint)(x)
}

// same as NewBigint()
func FromMathBig(x *big.Int) *Bigint {
	return (*Bigint)(x)
}

func FromInt64(x int64) *Bigint {
	return FromMathBig(big.NewInt(x))
}

func FromString(x string) (*Bigint, error) {
	if x == "" {
		return FromInt64(0), nil
	}
	a := big.NewInt(0)
	b, ok := a.SetString(x, 10)

	if !ok {
		return nil, fmt.Errorf("cannot create Bigint from string")
	}

	return newBigint(b), nil
}

func (b *Bigint) Value() (driver.Value, error) {
	return (*big.Int)(b).String(), nil
}

func (b *Bigint) Scan(value interface{}) error {

	var i sql.NullString

	if err := i.Scan(value); err != nil {
		return err
	}

	if _, ok := (*big.Int)(b).SetString(i.String, 10); ok {
		return nil
	}

	return fmt.Errorf("Error converting type %T into Bigint", value)
}

func (b *Bigint) ToMathBig() *big.Int {
	return (*big.Int)(b)
}

func (b *Bigint) Sub(x *Bigint) *Bigint {
	return (*Bigint)(big.NewInt(0).Sub(b.ToMathBig(), x.ToMathBig()))
}

func (b *Bigint) Add(x *Bigint) *Bigint {
	return (*Bigint)(big.NewInt(0).Add(b.ToMathBig(), x.ToMathBig()))
}

func (b *Bigint) Mul(x *Bigint) *Bigint {
	return (*Bigint)(big.NewInt(0).Mul(b.ToMathBig(), x.ToMathBig()))
}

func (b *Bigint) Div(x *Bigint) *Bigint {
	return (*Bigint)(big.NewInt(0).Div(b.ToMathBig(), x.ToMathBig()))
}

func (b *Bigint) Neg() *Bigint {
	return (*Bigint)(big.NewInt(0).Neg(b.ToMathBig()))
}

func (b *Bigint) ToUInt64() uint64 {
	return b.ToMathBig().Uint64()
}

func (b *Bigint) ToInt64() int64 {
	return b.ToMathBig().Int64()
}

func (b *Bigint) String() string {
	return b.ToMathBig().String()
}

func (b *Bigint) Cmp(target *Bigint) Cmp {
	return &cmp{r: b.ToMathBig().Cmp(target.ToMathBig())}
}

func (b *Bigint) Abs() *Bigint {
	return (*Bigint)(new(big.Int).Abs(b.ToMathBig()))
}
