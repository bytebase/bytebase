package common

import (
	"bytes"
	"encoding/gob"
	"reflect"

	"github.com/cockroachdb/errors"
)

type multiSetVal struct {
	value any
	count int
}

type MultiSet map[string]*multiSetVal

func assignableToNil(v reflect.Value) bool {
	switch v.Type().Kind() {
	case reflect.Slice,
		reflect.Struct,
		reflect.Map,
		reflect.Pointer,
		reflect.UnsafePointer,
		reflect.Func,
		reflect.Interface,
		reflect.Chan:
		return true
	default:
		return false
	}
}

func NewMultiSet(slice any) (MultiSet, error) {
	if k := reflect.TypeOf(slice).Kind(); k != reflect.Array && k != reflect.Slice {
		return nil, errors.Newf("unsupported type %s", k.String())
	}

	multiSet := make(MultiSet)
	rflSlice := reflect.ValueOf(slice)
	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)

	for i := 0; i < rflSlice.Len(); i++ {
		ele := rflSlice.Index(i)
		if assignableToNil(ele) && ele.IsNil() {
			// ignore nil value.
			continue
		}
		// serialize elements in gob format.
		if err := enc.Encode(ele.Interface()); err != nil {
			return nil, err
		}
		multiSet.insert(buf.String(), ele.Interface(), 1)
		buf.Reset()
	}

	return multiSet, nil
}

func (m MultiSet) Range(f func(any, int)) {
	for _, v := range m {
		f(v.value, v.count)
	}
}

func (m MultiSet) insert(key string, value any, count int) {
	if _, ok := m[key]; ok {
		m[key].count += count
	} else {
		m[key] = &multiSetVal{
			value: value,
			count: count,
		}
	}
}

func (m MultiSet) Clone() MultiSet {
	ret := make(MultiSet, len(m))
	for k, v := range m {
		ret[k] = v
	}
	return ret
}

func Union(a, b MultiSet) MultiSet {
	ret := make(MultiSet)
	for k, v := range a {
		ret.insert(k, v.value, v.count)
	}
	for k, v := range b {
		ret.insert(k, v.value, v.count)
	}
	return ret
}

func Intersection(a, b MultiSet) MultiSet {
	ret := make(MultiSet)
	for k, va := range a {
		if vb, ok := b[k]; ok {
			ret.insert(k, vb.value, min(va.count, vb.count))
		}
	}
	return ret
}

// remove the elements from A that are also present in B.
func Difference(a, b MultiSet) MultiSet {
	ret := a.Clone()
	// "a - b".
	for k, va := range ret {
		if vb, ok := b[k]; ok {
			if va.count <= vb.count {
				delete(ret, k)
			} else {
				ret[k].count = va.count - vb.count
			}
		}
	}
	return ret
}
