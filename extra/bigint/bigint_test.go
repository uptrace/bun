package bigint_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun/extra/buntypes/bigint"
)

func TestMathOperations(t *testing.T) {
	a := big.NewInt(100)
	b := big.NewInt(200)

	t.Run("multiply", func(t *testing.T) {
		x := bigint.FromMathBig(a)
		y := bigint.FromMathBig(b)
		// 100 * 200 = 20000
		assert.Equal(t, bigint.FromMathBig(big.NewInt(20000)), x.Mul(y))
	})
	t.Run("add", func(t *testing.T) {
		x := bigint.FromMathBig(a)
		y := bigint.FromMathBig(b)
		// 100 + 200 = 300
		assert.Equal(t, bigint.FromMathBig(big.NewInt(300)), x.Add(y))
	})

	t.Run("sub", func(t *testing.T) {
		x := bigint.FromMathBig(a)
		y := bigint.FromMathBig(b)
		// 100 -200 = -100
		assert.Equal(t, bigint.FromMathBig(big.NewInt(-100)), x.Sub(y))
	})

	t.Run("div", func(t *testing.T) {
		x := bigint.FromMathBig(a)
		y := bigint.FromMathBig(b)
		// 200 / 100 = 2
		assert.Equal(t, bigint.FromMathBig(big.NewInt(2)), y.Div(x))
	})

	t.Run("negation", func(t *testing.T) {
		x := bigint.FromMathBig(a)
		assert.Equal(t, bigint.FromMathBig(big.NewInt(-100)), x.Neg())
	})

	t.Run("int64", func(t *testing.T) {
		x := bigint.FromMathBig(a)
		assert.Equal(t, int64(-100), x.Neg().ToInt64())
	})

	t.Run("uint64", func(t *testing.T) {
		x := bigint.FromMathBig(a)
		assert.Equal(t, uint64(100), x.ToUInt64())
	})
	t.Run("toString", func(t *testing.T) {
		x := bigint.FromMathBig(a)
		assert.Equal(t, "100", x.String())
	})
	t.Run("fromString", func(t *testing.T) {
		x, err := bigint.FromString("100")
		assert.Nil(t, err)
		assert.Equal(t, "100", x.String())
	})
	t.Run("fromInt64", func(t *testing.T) {
		x := bigint.FromInt64(100000000)
		assert.Equal(t, int64(100000000), x.ToInt64())
	})

	t.Run("Abs", func(t *testing.T) {
		x := bigint.FromMathBig(a)

		assert.Equal(t, x.Neg().Abs(), x)
	})
	t.Run("compare: ", func(t *testing.T) {
		x := bigint.FromMathBig(a) // 100
		y := bigint.FromMathBig(b) // 200

		cmp := x.Cmp(y)

		t.Run("eq ?", func(t *testing.T) {
			assert.Equal(t, cmp.Eq(), false)
		})
		t.Run("lt ?", func(t *testing.T) {
			assert.Equal(t, cmp.Lt(), true)
		})
		t.Run("gt ?", func(t *testing.T) {
			assert.Equal(t, cmp.Gt(), false)
		})
		t.Run("leq ?", func(t *testing.T) {
			assert.Equal(t, cmp.Leq(), true)
		})
		t.Run("geq ?", func(t *testing.T) {
			assert.Equal(t, cmp.Geq(), false)
		})
	})

	t.Run("empty string ", func(t *testing.T) {

		x, err := bigint.FromString("")
		assert.Nil(t, err)
		assert.Equal(t, x.ToInt64(), int64(0))
	})

}
