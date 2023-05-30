package ast

import "strings"

// Serial is the struct for serial.
type Serial struct {
	numericType

	// the storage size(bytes)
	Size int
}

// EquivalentType implements the DataType interface.
func (s *Serial) EquivalentType(tp string) bool {
	tp = strings.ToLower(tp)
	switch s.Size {
	case 2:
		return tp == "smallserial" || tp == "serial2"
	case 4:
		return tp == "serial" || tp == "serial4"
	case 8:
		return tp == "bigserial" || tp == "serial8"
	}
	return false
}
