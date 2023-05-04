package bunbig_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/TommyLeng/bun/extra/bunbig"
)

func TestInt(t *testing.T) {
	a := big.NewInt(100)
	b := big.NewInt(200)

	t.Run("multiply", func(t *testing.T) {
		x := bunbig.FromMathBig(a)
		y := bunbig.FromMathBig(b)
		// 100 * 200 = 20000
		assert.Equal(t, bunbig.FromMathBig(big.NewInt(20000)), x.Mul(y))
	})

	t.Run("add", func(t *testing.T) {
		x := bunbig.FromMathBig(a)
		y := bunbig.FromMathBig(b)
		// 100 + 200 = 300
		assert.Equal(t, bunbig.FromMathBig(big.NewInt(300)), x.Add(y))
	})

	t.Run("sub", func(t *testing.T) {
		x := bunbig.FromMathBig(a)
		y := bunbig.FromMathBig(b)
		// 100 -200 = -100
		assert.Equal(t, bunbig.FromMathBig(big.NewInt(-100)), x.Sub(y))
	})

	t.Run("div", func(t *testing.T) {
		x := bunbig.FromMathBig(a)
		y := bunbig.FromMathBig(b)
		// 200 / 100 = 2
		assert.Equal(t, bunbig.FromMathBig(big.NewInt(2)), y.Div(x))
	})

	t.Run("negation", func(t *testing.T) {
		x := bunbig.FromMathBig(a)
		assert.Equal(t, bunbig.FromMathBig(big.NewInt(-100)), x.Neg())
	})

	t.Run("int64", func(t *testing.T) {
		x := bunbig.FromMathBig(a)
		assert.Equal(t, int64(-100), x.Neg().ToInt64())
	})

	t.Run("uint64", func(t *testing.T) {
		x := bunbig.FromMathBig(a)
		assert.Equal(t, uint64(100), x.ToUInt64())
	})

	t.Run("toString", func(t *testing.T) {
		x := bunbig.FromMathBig(a)
		assert.Equal(t, "100", x.String())
	})

	t.Run("fromString", func(t *testing.T) {
		x, err := bunbig.NewInt().FromString("100")
		assert.Nil(t, err)
		assert.Equal(t, "100", x.String())
	})

	t.Run("fromInt64", func(t *testing.T) {
		x := bunbig.FromInt64(100000000)
		assert.Equal(t, int64(100000000), x.ToInt64())
	})

	t.Run("fromUInt64", func(t *testing.T) {
		x := bunbig.FromUInt64(100000000)
		assert.Equal(t, int64(100000000), x.ToInt64())
	})

	t.Run("Abs", func(t *testing.T) {
		x := bunbig.FromMathBig(a)
		assert.Equal(t, x.Neg().Abs(), x)
	})

	t.Run("compare: ", func(t *testing.T) {
		x := bunbig.FromMathBig(a) // 100
		y := bunbig.FromMathBig(b) // 200

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
		x, err := bunbig.NewInt().FromString("")
		assert.Nil(t, err)
		assert.Equal(t, x.ToInt64(), int64(0))
	})

	t.Run("json", func(t *testing.T) {
		i := bunbig.FromInt64(1337)

		r, err := i.MarshalJSON()
		assert.Nil(t, err)
		assert.Equal(t, "1337", string(r))

		got := new(bunbig.Int)
		err = got.UnmarshalJSON(r)
		assert.Nil(t, err)
		assert.Equal(t, uint64(1337), got.ToUInt64())
	})
}

func TestFloat(t *testing.T) {

	cases := []struct {
		f1   float64
		f2   float64
		diff float64
		mul  float64
		sum  float64
		div  float64
		eq   bool
		geq  bool
		leq  bool
		lt   bool
		gt   bool
	}{
		{
			f1:   1.01,
			f2:   1.02,
			diff: 0.01,
			mul:  1.0302,
			sum:  2.03,
			div:  1,
			eq:   false,
			geq:  true,
			lt:   false,
			gt:   true,
			leq:  false,
		},
		{
			f1:   10.001,
			f2:   10.01,
			diff: 0.009,
			sum:  20.011,
			mul:  100.11001,
			div:  1,
			eq:   false,
			geq:  true,
			lt:   false,
			gt:   true,
			leq:  false,
		},
		{
			f1:   1,
			f2:   1,
			diff: 0,
			sum:  2,
			mul:  1,
			div:  1,
			eq:   true,
			geq:  true,
			leq:  true,
			lt:   false,
			gt:   false,
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%f , %f ", c.f1, c.f2), func(t *testing.T) {
			f1 := big.NewFloat(c.f1)
			f2 := big.NewFloat(c.f2)
			diff := big.NewFloat(c.diff)
			mul := big.NewFloat(c.mul)
			sum := big.NewFloat(c.sum)
			div := big.NewFloat(c.div)

			bunF1 := bunbig.NewFloat().FromMathFloat(f1)
			bunF2 := bunbig.NewFloat().FromMathFloat(f2)
			bunDiff := bunbig.NewFloat().FromMathFloat(diff)
			bunMul := bunbig.NewFloat().FromMathFloat(mul)
			bunSum := bunbig.NewFloat().FromMathFloat(sum)
			bunDiv := bunbig.NewFloat().FromMathFloat(div)

			cmp := bunF2.Cmp(bunF1)

			assert.Equal(t, bunDiff.String(), bunF2.Sub(bunF1).String())
			assert.Equal(t, bunMul.String(), bunF2.Mul(bunF1).String())
			assert.Equal(t, bunSum.String(), bunF2.Add(bunF1).String())
			assert.Equal(t, bunDiv.String(), bunF2.Div(bunF2).String())

			assert.Equal(t, cmp.Eq(), c.eq)
			assert.Equal(t, cmp.Geq(), c.geq)
			assert.Equal(t, cmp.Gt(), c.gt)
			assert.Equal(t, cmp.Leq(), c.leq)
			assert.Equal(t, cmp.Lt(), c.lt)
		})
	}

	f, err := bunbig.NewFloat().FromString("-100")
	assert.NoError(t, err)

	assert.Equal(t, f.Abs().String(), "100")

	f2 := bunbig.NewFloat().FromMathFloat(big.NewFloat(100))

	assert.Equal(t, f.String(), f2.Neg().String())
}

func TestFixture(t *testing.T) {
	data := `
- id: 1
  name: ethereum
  base: wei
  equals: 1000000000000000000	
- id: 2
  name: bitcoin
  base: satoshi
  equals: 1000000000
`
	type CryptoNetwork struct {
		ID     int
		Name   string
		Base   string
		Equals *bunbig.Int
	}

	cryptoNet := []CryptoNetwork{}

	err := yaml.Unmarshal([]byte(data), &cryptoNet)

	assert.NoError(t, err)

	// @Todo
	// we expect that the decoded values become convertible to bunbig.Int
}
