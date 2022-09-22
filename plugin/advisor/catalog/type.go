package catalog

type stateInt struct {
	defined bool
	value   int64
}

func newStateInt(value int64) stateInt {
	return stateInt{defined: true, value: value}
}

type stateString struct {
	defined bool
	value   string
}

func (s stateString) String() string {
	if s.defined {
		return s.value
	}
	return "Undefined String"
}

func newStateString(value string) stateString {
	return stateString{defined: true, value: value}
}

type stateStringSlice struct {
	defined bool
	value   []string
}

func (s stateStringSlice) len() int {
	if s.defined {
		return len(s.value)
	}
	return 0
}

func newStateStringSlice(value []string) stateStringSlice {
	return stateStringSlice{defined: true, value: value}
}

type stateStringPointer struct {
	defined bool
	value   *string
}

func newStateStringPointer(value *string) stateStringPointer {
	return stateStringPointer{defined: true, value: value}
}

type stateBool struct {
	defined bool
	value   bool
}

func (b stateBool) isTrue() bool {
	return b.defined && b.value
}

func newStateBool(value bool) stateBool {
	return stateBool{defined: true, value: value}
}
