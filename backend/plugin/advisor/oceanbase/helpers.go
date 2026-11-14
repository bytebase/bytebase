package oceanbase

import (
	"github.com/bytebase/omni/oceanbase/ast"

	oceanbaseparser "github.com/bytebase/bytebase/backend/plugin/parser/oceanbase"
)

func lineOf(baseLine int, text string, loc ast.Loc) int {
	if loc.Start < 0 {
		return baseLine + 1
	}
	return baseLine + int(oceanbaseparser.ByteOffsetToRunePosition(text, loc.Start).Line)
}
