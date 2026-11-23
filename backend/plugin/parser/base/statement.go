package base

import (
	omnianalysis "github.com/bytebase/omni/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Statement is the result of splitting SQL (text + positions, no AST).
type Statement struct {
	// Text is the SQL text content.
	Text string
	// Empty indicates if the sql is empty, such as `/* comments */;` or just `;`.
	Empty bool

	// Start is the inclusive start position of the SQL in the original SQL (1-based).
	Start *storepb.Position
	// End is the exclusive end position of the SQL in the original SQL (1-based).
	// It points to the position AFTER the last character of the statement.
	End *storepb.Position

	// Range is the byte offset range of the SQL in the original SQL.
	// This field may not be present for every engine.
	// Range is intended for sql execution log display. It may not represent the actual sql that is sent to the database.
	Range *storepb.Range
}

// BaseLine returns the 0-based line number of the first line of the SQL.
// This is derived from Start.Line - 1 and is used for advisor code compatibility.
func (s *Statement) BaseLine() int {
	if s.Start == nil {
		return 0
	}
	return int(s.Start.Line) - 1
}

// TruncateAtSemicolonToken returns sql[start:end], where end is the Loc of
// the first token at or after start whose type is ';' (the end of the
// statement containing start), or len(sql) when there is none. Scanning the
// token stream rather than raw bytes means a ';' inside a string literal,
// comment, or quoted identifier never counts as a statement boundary.
// Completers use this to bound clause fragments to the current statement so
// fragment re-parsing cannot scale with the size of the whole sheet
// (BYT-9886). loc and typ adapt the engine-specific token type.
func TruncateAtSemicolonToken[T any](sql string, start int, tokens []T, loc func(T) int, typ func(T) int) string {
	for _, tok := range tokens {
		if loc(tok) >= start && typ(tok) == ';' {
			return sql[start:loc(tok)]
		}
	}
	return sql[start:]
}

// NewStatementFromRange builds a Statement from a split segment range, including an immediately following semicolon.
func NewStatementFromRange(sql string, mapper *ByteOffsetPositionMapper, byteStart, byteEnd int, empty bool) Statement {
	if byteEnd < len(sql) && sql[byteEnd] == ';' {
		byteEnd++
	}
	return NewExactStatement(sql, mapper, byteStart, byteEnd, empty)
}

// NewExactStatement builds a Statement from an exact [byteStart, byteEnd) slice of sql.
func NewExactStatement(sql string, mapper *ByteOffsetPositionMapper, byteStart, byteEnd int, empty bool) Statement {
	return Statement{
		Text:  sql[byteStart:byteEnd],
		Empty: empty,
		Start: mapper.Position(byteStart),
		End:   mapper.Position(byteEnd),
		Range: &storepb.Range{
			Start: int32(byteStart),
			End:   int32(byteEnd),
		},
	}
}

// BuildStatements converts split segments into []Statement using a single linear pass for positions.
// When includeTrailingSemi is true, segments ending right before ';' extend by one byte to include it.
func BuildStatements[T any](sql string, segs []T, bounds func(T) (start, end int, empty bool), includeTrailingSemi bool) []Statement {
	if len(segs) == 0 {
		return nil
	}
	result := make([]Statement, 0, len(segs))
	mapper := NewByteOffsetPositionMapper(sql)
	for _, seg := range segs {
		start, end, empty := bounds(seg)
		if includeTrailingSemi {
			result = append(result, NewStatementFromRange(sql, mapper, start, end, empty))
		} else {
			result = append(result, NewExactStatement(sql, mapper, start, end, empty))
		}
	}
	return result
}

// OmniStatements converts omni split statements into []Statement with 1-based line and rune column positions.
func OmniStatements(sql string, stmts []omnianalysis.Statement) []Statement {
	return BuildStatements(sql, stmts, func(s omnianalysis.Statement) (int, int, bool) {
		return s.Start, s.End, s.Empty
	}, false)
}

// ParsedStatement is the result of parsing SQL (Statement + AST).
// AST is guaranteed to be non-nil after successful parsing.
type ParsedStatement struct {
	Statement     // embedded - access fields directly like ps.Text, ps.Start
	AST       AST // always non-nil after parsing
}

// FilterEmptyStatements removes empty statements from the list.
func FilterEmptyStatements(list []Statement) []Statement {
	var result []Statement
	for _, stmt := range list {
		if !stmt.Empty {
			result = append(result, stmt)
		}
	}
	return result
}

// ExtractASTs extracts non-nil ASTs from a slice of ParsedStatements.
// Empty statements (with nil AST) are skipped.
// Returns nil if no ASTs are found (preserves nil-check compatibility).
// This is useful for backward compatibility when migrating from []AST to []ParsedStatement.
func ExtractASTs(stmts []ParsedStatement) []AST {
	var asts []AST
	for _, stmt := range stmts {
		if stmt.AST != nil {
			asts = append(asts, stmt.AST)
		}
	}
	return asts
}
