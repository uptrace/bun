package internal

import (
	"database/sql"
	"reflect"
)

var ifaceType = reflect.TypeOf((*interface{})(nil)).Elem()

type MapKey struct {
	iface interface{}
}

func NewMapKey(is []interface{}) MapKey {
	if maybeNullInt64, ok := newMapKey(is).(sql.NullInt64); ok {
		if maybeNullInt64.Valid {
			return MapKey{iface: maybeNullInt64.Int64}
		} else {
			return MapKey{iface: nil}
		}
	} else {
		return MapKey{iface: maybeNullInt64}
	}
}

func newMapKey(is []interface{}) interface{} {
	switch len(is) {
	case 1:
		ptr := new([1]interface{})
		copy((*ptr)[:], is)
		return *ptr
	case 2:
		ptr := new([2]interface{})
		copy((*ptr)[:], is)
		return *ptr
	case 3:
		ptr := new([3]interface{})
		copy((*ptr)[:], is)
		return *ptr
	case 4:
		ptr := new([4]interface{})
		copy((*ptr)[:], is)
		return *ptr
	case 5:
		ptr := new([5]interface{})
		copy((*ptr)[:], is)
		return *ptr
	case 6:
		ptr := new([6]interface{})
		copy((*ptr)[:], is)
		return *ptr
	case 7:
		ptr := new([7]interface{})
		copy((*ptr)[:], is)
		return *ptr
	case 8:
		ptr := new([8]interface{})
		copy((*ptr)[:], is)
		return *ptr
	case 9:
		ptr := new([9]interface{})
		copy((*ptr)[:], is)
		return *ptr
	case 10:
		ptr := new([10]interface{})
		copy((*ptr)[:], is)
		return *ptr
	default:
	}

	at := reflect.New(reflect.ArrayOf(len(is), ifaceType)).Elem()
	for i, v := range is {
		*(at.Index(i).Addr().Interface().(*interface{})) = v
	}
	return at.Interface()
}
