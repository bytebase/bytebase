package mariadb

import (
	"strings"
	"unicode/utf8"

	"github.com/bytebase/omni/mariadb/ast"
	mariadbparser "github.com/bytebase/omni/mariadb/parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// OmniAST wraps an omni MariaDB AST node and implements the base.AST interface.
type OmniAST struct {
	Node ast.Node
	// Text is the original SQL text of this statement.
	Text string
	// StartPosition is the 1-based position where this statement starts.
	StartPosition *storepb.Position
}

// ASTSource returns the statement's own text, which the schema walk-through reads
// when it needs the statement as written.
func (a *OmniAST) ASTSource() string {
	return a.Text
}

// ASTStartPosition implements base.AST.
func (a *OmniAST) ASTStartPosition() *storepb.Position {
	return a.StartPosition
}

// ParseMariaDB parses SQL using omni's MariaDB parser and returns an ast.List.
func ParseMariaDB(sql string) (*ast.List, error) {
	return mariadbparser.Parse(sql)
}

// GetOmniNode extracts the omni AST node from a base.AST interface.
// Returns the node and true if it is an OmniAST, nil and false otherwise.
func GetOmniNode(a base.AST) (ast.Node, bool) {
	if a == nil {
		return nil, false
	}
	omniAST, ok := a.(*OmniAST)
	if !ok {
		return nil, false
	}
	return omniAST.Node, true
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

// endBeforeReturning clamps end to just before a DML statement's RETURNING
// clause, or returns end unchanged when there is none.
//
// RETURNING is MariaDB's and is the last clause of the statements that take it.
// Every rewrite here that turns a DML into a SELECT by copying its text through
// to the statement end must stop short of it: RETURNING means nothing in a
// SELECT and the server rejects the result. Route a new rewrite through this
// function rather than repeating the search.
//
// An error means the clause is present but its keyword could not be located.
// Callers decline to transform rather than emit SQL that fails when it runs.
func endBeforeReturning(returning []ast.Node, sql string, end int) (int, error) {
	if end <= 0 || end > len(sql) {
		end = len(sql)
	}
	if len(returning) == 0 {
		return end, nil
	}
	var first int
	switch e := returning[0].(type) {
	case *ast.ColumnRef:
		first = e.Loc.Start
	case *ast.ResTarget:
		first = e.Loc.Start
	case *ast.StarExpr:
		first = e.Loc.Start
	default:
		return 0, errors.Errorf("cannot locate RETURNING from %T", returning[0])
	}
	if first <= 0 || first > end {
		return 0, errors.New("cannot locate RETURNING")
	}
	// The keyword is the token before the clause's first item. A text search will
	// not do: a comment or string literal in or beside the clause may carry the
	// word, as in `RETURNING /* RETURNING */ b1`. The lexer skips comments and
	// gives a quoted RETURNING a token of its own, so neither is mistaken for it.
	tokens := mariadbparser.Tokenize(sql)
	for i, tok := range tokens {
		if tok.Loc != first {
			continue
		}
		if i == 0 {
			break
		}
		if prev := tokens[i-1]; strings.EqualFold(prev.Str, "RETURNING") {
			return prev.Loc, nil
		}
		break
	}
	return 0, errors.New("cannot locate the RETURNING keyword")
}
