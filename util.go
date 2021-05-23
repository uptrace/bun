package bun

import "reflect"

func indirect(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Interface:
		return indirect(v.Elem())
	case reflect.Ptr:
		return v.Elem()
	default:
		return v
	}
}

func walk(v reflect.Value, index []int, fn func(reflect.Value)) {
	v = reflect.Indirect(v)
	switch v.Kind() {
	case reflect.Slice:
		sliceLen := v.Len()
		for i := 0; i < sliceLen; i++ {
			visitField(v.Index(i), index, fn)
		}
	default:
		visitField(v, index, fn)
	}
}

func visitField(v reflect.Value, index []int, fn func(reflect.Value)) {
	v = reflect.Indirect(v)
	if len(index) > 0 {
		v = v.Field(index[0])
		if v.Kind() == reflect.Ptr && v.IsNil() {
			return
		}
		walk(v, index[1:], fn)
	} else {
		fn(v)
	}
}

func typeByIndex(t reflect.Type, index []int) reflect.Type {
	for _, x := range index {
		switch t.Kind() {
		case reflect.Ptr:
			t = t.Elem()
		case reflect.Slice:
			t = indirectType(t.Elem())
		}
		t = t.Field(x).Type
	}
	return indirectType(t)
}

func indirectType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func sliceElemType(v reflect.Value) reflect.Type {
	elemType := v.Type().Elem()
	if elemType.Kind() == reflect.Interface && v.Len() > 0 {
		return indirect(v.Index(0).Elem()).Type()
	}
	return indirectType(elemType)
}

func makeSliceNextElemFunc(v reflect.Value) func() reflect.Value {
	if v.Kind() == reflect.Array {
		var pos int
		return func() reflect.Value {
			v := v.Index(pos)
			pos++
			return v
		}
	}

	sliceType := v.Type()
	elemType := sliceType.Elem()

	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
		return func() reflect.Value {
			if v.Len() < v.Cap() {
				v.Set(v.Slice(0, v.Len()+1))
				elem := v.Index(v.Len() - 1)
				if elem.IsNil() {
					elem.Set(reflect.New(elemType))
				}
				return elem.Elem()
			}

			elem := reflect.New(elemType)
			v.Set(reflect.Append(v, elem))
			return elem.Elem()
		}
	}

	zero := reflect.Zero(elemType)
	return func() reflect.Value {
		l := v.Len()
		c := v.Cap()

		if l < c {
			v.Set(v.Slice(0, l+1))
			return v.Index(l)
		}

		v.Set(reflect.Append(v, zero))
		return v.Index(l)
	}
}
