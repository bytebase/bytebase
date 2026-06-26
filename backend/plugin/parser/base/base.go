package base

import storepb "github.com/bytebase/bytebase/backend/generated-go/store"

// SyntaxError is a syntax error.
type SyntaxError struct {
	Position   *storepb.Position
	Message    string
	RawMessage string
}

// Error returns the error message.
func (e *SyntaxError) Error() string {
	return e.Message
}

func GetOffsetLength(total int) int {
	length := 1
	for {
		if total < 10 {
			return length
		}
		total /= 10
		length++
	}
}

// GetLineOffset returns the 0-based line offset from a StartPosition.
// This is useful for converting from the 1-based StartPosition to the 0-based offset
// needed for some calculations.
func GetLineOffset(startPosition *storepb.Position) int {
	if startPosition == nil {
		return 0
	}
	return int(startPosition.Line) - 1
}
