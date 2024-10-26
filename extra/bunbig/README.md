
**bunbig** is a wrapper around math/big package to let us use big.int type in bun

Disclaimer: math/big does not implement database/sql scan/value methods and it can't be used in bun. This package uses math/big in its heart and extends its usefulness even into postgres.
## Types

* Int
  - Int wraps around big.Int and supports its base functionalities
* Float
  - Float is a bun counterpart of big.Float

## Example use

```

	type TableWithBigint struct {
		ID        uint64
		Name      string
		Deposit   *bunbig.Int
		Residue *bunbig.Float
	}

```

### Mathematical Operations: 

This package supports basic mathematical operations such as addition, subtraction, division, negation etc.

Example : 

```
	x := bunbig.FromInt64(100)
	y , err := bunbig.FromString("9999999999999999999999999999999999999999")

	if err!=nil {
		panic(err)
	}

	y.Add(x) // 9999999999999999999999999999999999999999 + 100
	y.Sub(x) // 9999999999999999999999999999999999999999 - 100
	y.Neg() //  -9999999999999999999999999999999999999999
	// on the fly operation
	c:= bunbig.FromInt64(100).Mul(y) // 100 * 100 = 10000
	c.Div(x) // 10000/100 = 100
	c.Neg().Abs() // |-10000| = 10000

```

For extracting math/big's Int you can simply do as follows:

```
    d:= bunbig.ToMathBig(x)
```

Now you can do your calculations and convert it back to bunbig with:

```
   x = bunbig.FromBigint(d)
```

### comparisons:

let we have x , y as two bigint.Bigint numbers in buntypes. 
```
   x:= bunbig.FromInt64(100)
   y:= bunbig.FromInt64(90)
```

For comparing the above numbers, we can do as follows:

```
   cmp:=x.Cmp(y)
```

**cmp** has the following methods: 

```
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

```

Thus, we have the following results:

```
  x.Cmp(y).Eq() // 100 == 90 : false
  x.Cmp(y).Geq() // 100 >= 90 : true
  x.Cmp(y).Gt() // 100 > 90 : true
  x.Cmp(y).Lt() // 100 < 90 : false
  x.Cmp(y).Leq() // 100 <= 90 : false
```