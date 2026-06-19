package mariadb

import (
	"unicode/utf8"

	"github.com/bytebase/omni/mariadb/ast"
	mariadbparser "github.com/bytebase/omni/mariadb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ParseMariaDBOmni parses SQL using omni's MariaDB parser and returns an ast.List.
func ParseMariaDBOmni(sql string) (*ast.List, error) {
	return mariadbparser.Parse(sql)
}

// ByteOffsetToRunePosition converts a byte offset in sql to a 1-based line:column
// where column is measured in Unicode code points (runes), matching storepb.Position semantics.
func ByteOffsetToRunePosition(sql string, byteOffset int) *storepb.Position {
	if byteOffset > len(sql) {
		byteOffset = len(sql)
	}

	line := int32(1)
	runeCol := int32(0) // 0-based rune count on current line
	i := 0
	for i < byteOffset {
		r, size := utf8.DecodeRuneInString(sql[i:])
		if r == '\n' {
			line++
			runeCol = 0
		} else {
			runeCol++
		}
		i += size
	}

	return &storepb.Position{
		Line:   line,
		Column: runeCol + 1, // convert to 1-based
	}
}
