package base

import storepb "github.com/bytebase/bytebase/backend/generated-go/store"

type RestoreContext struct {
	InstanceID              string
	GetDatabaseMetadataFunc GetDatabaseMetadataFunc
	ListDatabaseNamesFunc   ListDatabaseNamesFunc
	IsCaseSensitive         bool
}

// StatementRangeIndices returns the inclusive [start, end] indices in list covering
// [startPos, endPos]. When exclusiveEnd is true, a statement whose Start >= endPos
// ends the range at i - 1; otherwise it ends at i.
func StatementRangeIndices(list []Statement, startPos, endPos *storepb.Position, exclusiveEnd bool) (start, end int) {
	start = 0
	end = len(list) - 1
	for i, item := range list {
		if EqualOrLessPosition(item.Start, startPos) {
			start = i
		}
	}
	for i := len(list) - 1; i >= 0; i-- {
		if EqualOrGreaterPosition(list[i].Start, endPos) {
			if exclusiveEnd {
				end = i - 1
			} else {
				end = i
			}
		}
	}
	return start, end
}

// EqualOrLessPosition reports whether a is at or before b in line:column order.
func EqualOrLessPosition(a, b *storepb.Position) bool {
	if a.Line < b.Line {
		return true
	}
	return a.Line == b.Line && a.Column <= b.Column
}

// EqualOrGreaterPosition reports whether a is at or after b in line:column order.
func EqualOrGreaterPosition(a, b *storepb.Position) bool {
	if a.Line > b.Line {
		return true
	}
	return a.Line == b.Line && a.Column >= b.Column
}
