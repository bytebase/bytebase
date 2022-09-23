package catalog

type stateInt struct {
	valid bool
	value int64
}

func newStateInt(value int64) stateInt {
	return stateInt{valid: true, value: value}
}

type stateString struct {
	valid bool
	value string
}

func (s stateString) String() string {
	if s.valid {
		return s.value
	}
	return "Undefined String"
}

func newStateString(value string) stateString {
	return stateString{valid: true, value: value}
}

type stateStringSlice struct {
	valid bool
	value []string
}

func (s stateStringSlice) copy() stateStringSlice {
	var list []string
	list = append(list, s.value...)
	return stateStringSlice{
		valid: s.valid,
		value: list,
	}
}

func (s stateStringSlice) len() int {
	if s.valid {
		return len(s.value)
	}
	return -1
}

func newStateStringSlice(value []string) stateStringSlice {
	return stateStringSlice{valid: true, value: value}
}

type stateStringPointer struct {
	valid bool
	value *string
}

func (p stateStringPointer) copy() stateStringPointer {
	if p.value != nil {
		s := *p.value
		return stateStringPointer{
			valid: p.valid,
			value: &s,
		}
	}
	return stateStringPointer{
		valid: p.valid,
		value: nil,
	}
}

func newStateStringPointer(value *string) stateStringPointer {
	return stateStringPointer{valid: true, value: value}
}

type stateBool struct {
	valid bool
	value bool
}

func (b stateBool) isTrue() bool {
	return b.valid && b.value
}

func newStateBool(value bool) stateBool {
	return stateBool{valid: true, value: value}
}
