package common

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/google/go-cmp/cmp"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type fullPath []string

func (fp fullPath) string() string {
	return strings.Join(fp, ".")
}

type Pointer struct {
	p unsafe.Pointer
	t reflect.Type
}

func pointerOf(v reflect.Value) Pointer {
	return Pointer{
		unsafe.Pointer(v.Pointer()),
		v.Type(),
	}
}

type pointerPath struct {
	mapA map[Pointer]Pointer
	mapB map[Pointer]Pointer
}

// ref: https://github.com/google/go-cmp/blob/c3ad8435e7bef96af35732bc0789e5a2278c6d5f/cmp/path.go#L320
func (p *pointerPath) push(a, b reflect.Value) (eq, visited bool) {
	pa := pointerOf(a)
	pb := pointerOf(b)
	_, oka := p.mapA[pa]
	_, okb := p.mapB[pb]
	if oka || okb {
		eq = p.mapA[pa] == pb && p.mapB[pb] == pa
		return eq, true
	}
	p.mapA[pa] = pb
	p.mapB[pb] = pa
	return false, false
}

func (p *pointerPath) pop(a, b reflect.Value) {
	delete(p.mapA, pointerOf(a))
	delete(p.mapB, pointerOf(b))
}

type Diff struct {
	OldValue any
	NewValue any
}

type Action string

const (
	ActionAdd    Action = "ADD"
	ActionRemove Action = "REMOVE"
	ActionModify Action = "MODIFY"
)

type DiffKey struct {
	path   string
	action Action
}

type DiffMap map[DiffKey][]Diff

type diffFinder struct {
	curPath  fullPath
	curPtrs  pointerPath
	diffsMap DiffMap
	err      error
}

func (f *diffFinder) pushStep(fieldName string) {
	f.curPath = append(f.curPath, fieldName)
}

func (f *diffFinder) popStep() {
	f.curPath = f.curPath[:len(f.curPath)-1]
}

func (f *diffFinder) report(a, b any, diffType Action) {
	diff := Diff{
		OldValue: a,
		NewValue: b,
	}
	diffKey := DiffKey{
		path:   f.curPath.string(),
		action: diffType,
	}
	if diffs, ok := f.diffsMap[diffKey]; ok {
		diffs = append(diffs, diff)
		f.diffsMap[diffKey] = diffs
	} else {
		f.diffsMap[diffKey] = []Diff{diff}
	}
}

func (f *diffFinder) compareAny(a, b reflect.Value) {
	if f.compareNil(a, b) {
		return
	}

	t := a.Type()

	switch t.Kind() {
	case reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.String:
		if !cmp.Equal(a, b) {
			f.report(a, b, ActionModify)
		}
	case reflect.Struct:
		f.compareStruct(a, b)
	case reflect.Slice,
		reflect.Array:
		f.compareSlice(a, b)
	case reflect.Map:
		f.compareMap(a, b)
	case reflect.Ptr:
		f.comparePtr(a, b)
	case reflect.Interface:
		f.compareInterface(a, b)
	default:
	}
}

// only used by Slice, Pointer, Map.
// return true if either a or b is nil.
func (f *diffFinder) compareNil(a, b reflect.Value) bool {
	if !a.IsValid() && !b.IsValid() {
		return true
	}

	if a.IsValid() && !b.IsValid() {
		f.report(a, b, ActionRemove)
		return true
	} else if !a.IsValid() && b.IsValid() {
		f.report(a, b, ActionAdd)
		return true
	} else if a.Type() != b.Type() {
		// ignore value of different types.
		return true
	}

	return false
}

func (f *diffFinder) comparePtr(a, b reflect.Value) {
	// cycle-detection for pointers.
	if eq, visited := f.curPtrs.push(a, b); visited {
		if !eq {
			f.report(a, b, ActionModify)
		}
		return
	}
	defer f.curPtrs.pop(a, b)

	// dereference.
	f.compareAny(a.Elem(), b.Elem())
}

func (f *diffFinder) compareStruct(a, b reflect.Value) {
	t := a.Type()
	for i := 0; i < t.NumField(); i++ {
		fieldName := t.Field(i).Name
		// ignore unexported field.
		if !isExported(fieldName) {
			continue
		}
		f.pushStep(fieldName)
		f.compareAny(a.Field(i), b.Field(i))
		f.popStep()
	}
}

// unexported field name starts with a lowercase letter.
func isExported(id string) bool {
	r, _ := utf8.DecodeRuneInString(id)
	return unicode.IsUpper(r)
}

// this func won't recurssivly call 'compareAny()' so we don't need cycle-detection.
func (f *diffFinder) compareSlice(a, b reflect.Value) {
	msa, err := NewMultiSet(a.Interface())
	if err != nil {
		f.err = err
		return
	}
	msb, err := NewMultiSet(b.Interface())
	if err != nil {
		f.err = err
		return
	}

	delElems := Difference(msa, msb)
	delElems.Range(func(data any, count int) {
		for i := 0; i < count; i++ {
			f.report(data, nil, ActionRemove)
		}
	})
	newElems := Difference(msb, msa)
	newElems.Range(func(data any, count int) {
		for i := 0; i < count; i++ {
			f.report(nil, data, ActionAdd)
		}
	})
}

func (f *diffFinder) compareMap(a, b reflect.Value) {
	// cycle-detection for maps.
	if eq, visited := f.curPtrs.push(a, b); visited {
		if !eq {
			f.report(a, b, ActionModify)
		}
	}

	visited := make(map[any]bool)
	forEachMapKeys := func(keys []reflect.Value) {
		for _, key := range keys {
			if _, ok := visited[key.Interface()]; ok {
				continue
			}
			visited[key.Interface()] = true

			mapAVal := a.MapIndex(key)
			mapBVal := b.MapIndex(key)
			f.pushStep(key.String())
			f.compareAny(mapAVal, mapBVal)
			f.popStep()
		}
	}

	forEachMapKeys(a.MapKeys())
	forEachMapKeys(b.MapKeys())
}

func (f *diffFinder) compareInterface(a, b reflect.Value) {
	f.compareAny(a.Elem(), b.Elem())
}

// Used to compute differences between objects.
// NOTES:
//  1. Ignore comparisons between channels, functions and objects of different types.
//  2. Ignore unexported fields.
//  3. Unordered comparison on Slice and Array.
func FindDiff(a, b any) (DiffMap, error) {
	f := &diffFinder{
		diffsMap: make(DiffMap),
		curPtrs: pointerPath{
			mapA: make(map[Pointer]Pointer),
			mapB: make(map[Pointer]Pointer),
		},
	}
	f.compareAny(reflect.ValueOf(a), reflect.ValueOf(b))
	return f.diffsMap, f.err
}

func ConvertToV1pbDiffs(m DiffMap, maskedFieldPaths map[string]bool, ignFieldPaths map[string]bool) []*v1pb.Diff {
	v1pbDiffs := []*v1pb.Diff{}

	valToStr := func(v any) string {
		valStr := ""
		rflVal := reflect.ValueOf(v)
		if !rflVal.IsValid() {
			valStr = "<nil>"
		} else {
			valStr = fmt.Sprintf("%v", v)
		}
		return valStr
	}

	for k, v := range m {
		strValues := []string{}
		if ignFieldPaths != nil {
			if _, ok := ignFieldPaths[k.path]; ok {
				continue
			}
		}
		maskedField := false
		if maskedFieldPaths != nil {
			if _, ok := maskedFieldPaths[k.path]; ok && len(strValues) == 0 {
				strValues = append(strValues, "***")
				maskedField = true
			}
		}
		if !maskedField {
			switch k.action {
			case ActionModify:
				strValues = append(strValues, fmt.Sprintf("%v -> %v", valToStr(v[0].OldValue), valToStr(v[0].NewValue)))
			case ActionAdd:
				for _, diff := range v {
					strValues = append(strValues, valToStr(diff.NewValue))
				}
			case ActionRemove:
				for _, diff := range v {
					strValues = append(strValues, valToStr(diff.OldValue))
				}
			default:
			}
		}

		v1pbDiffs = append(v1pbDiffs, &v1pb.Diff{
			Action: string(k.action),
			Name:   k.path,
			Value:  strings.Join(strValues, ", "),
		})
	}

	// sort keys by paths.
	sort.Slice(v1pbDiffs, func(i, j int) bool {
		if v1pbDiffs[i].Action != v1pbDiffs[j].Action {
			return v1pbDiffs[i].Action < v1pbDiffs[j].Action
		}
		return v1pbDiffs[i].Name < v1pbDiffs[j].Name
	})
	return v1pbDiffs
}
