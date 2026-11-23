package base

import (
	"errors"
	"fmt"
	"strings"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ParseSegmentedStatements splits statement and parses each non-empty segment into an AST.
// Empty or whitespace-only segments are preserved as ParsedStatement entries with a nil AST.
func ParseSegmentedStatements[A AST](
	statement string,
	split func(string) ([]Statement, error),
	parseSegment func(Statement) (A, error),
) ([]ParsedStatement, error) {
	stmts, err := split(statement)
	if err != nil {
		return nil, err
	}
	result := make([]ParsedStatement, 0, len(stmts))
	for _, stmt := range stmts {
		ps := ParsedStatement{Statement: stmt}
		if stmt.Empty || strings.TrimSpace(stmt.Text) == "" {
			result = append(result, ps)
			continue
		}
		parsed, err := parseSegment(stmt)
		if err != nil {
			return nil, err
		}
		ps.AST = parsed
		result = append(result, ps)
	}
	return result, nil
}

// ParseNonEmptySegments splits statement and parses each non-empty segment, skipping empty or
// whitespace-only segments.
func ParseNonEmptySegments[A any](
	statement string,
	split func(string) ([]Statement, error),
	parseSegment func(Statement) (A, error),
) ([]A, error) {
	stmts, err := split(statement)
	if err != nil {
		return nil, err
	}
	var result []A
	for _, stmt := range stmts {
		if stmt.Empty || strings.TrimSpace(stmt.Text) == "" {
			continue
		}
		parsed, err := parseSegment(stmt)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}
	return result, nil
}

// ParseMultiNodeStatements splits statement and expands each segment into zero or more
// ParsedStatement entries. Segments producing no ASTs are preserved with a nil AST.
func ParseMultiNodeStatements(
	statement string,
	split func(string) ([]Statement, error),
	parseSegment func(Statement) ([]AST, error),
) ([]ParsedStatement, error) {
	stmts, err := split(statement)
	if err != nil {
		return nil, err
	}
	if len(stmts) == 0 {
		return nil, nil
	}
	result := make([]ParsedStatement, 0, len(stmts))
	for _, stmt := range stmts {
		asts, err := parseSegment(stmt)
		if err != nil {
			return nil, err
		}
		if len(asts) == 0 {
			result = append(result, ParsedStatement{Statement: stmt})
			continue
		}
		for _, a := range asts {
			result = append(result, ParsedStatement{Statement: stmt, AST: a})
		}
	}
	return result, nil
}

// ParseNonEmptyMultiNodeStatements splits statement and expands each non-empty
// segment into zero or more ParsedStatement entries, omitting empty segments
// and segments yielding no ASTs.
func ParseNonEmptyMultiNodeStatements(
	statement string,
	split func(string) ([]Statement, error),
	parseSegment func(Statement) ([]AST, error),
) ([]ParsedStatement, error) {
	stmts, err := split(statement)
	if err != nil {
		return nil, err
	}
	var result []ParsedStatement
	for _, stmt := range stmts {
		if stmt.Empty {
			continue
		}
		asts, err := parseSegment(stmt)
		if err != nil {
			return nil, err
		}
		for _, a := range asts {
			result = append(result, ParsedStatement{Statement: stmt, AST: a})
		}
	}
	return result, nil
}

// ConvertParseError converts an engine parse error matched via errors.As into a *SyntaxError
// with 1-based line:column coordinates shifted by basePos. Returns err unchanged if it does
// not match E.
func ConvertParseError[E error](
	statement string,
	err error,
	basePos *storepb.Position,
	extract func(E) (byteOffset int, message, rawMessage string),
) error {
	var pe E
	if !errors.As(err, &pe) {
		return err
	}
	offset, msg, rawMsg := extract(pe)
	return &SyntaxError{
		Position:   shiftPosition(ByteOffsetToRunePosition(statement, offset), basePos),
		Message:    msg,
		RawMessage: rawMsg,
	}
}

// ConvertSyntaxError converts an engine parse error matched via errors.As into
// a *SyntaxError whose 1-based line:column position is shifted by stmt.Start.
// When shiftColumn is false, only the line number is shifted.
func ConvertSyntaxError[E error](
	stmt Statement,
	err error,
	shiftColumn bool,
	extract func(E) (byteOffset int, rawMessage string),
	format func(E, *storepb.Position) string,
) error {
	var pe E
	if !errors.As(err, &pe) {
		return err
	}
	offset, rawMsg := extract(pe)
	pos := ByteOffsetToRunePosition(stmt.Text, offset)
	if stmt.Start != nil {
		if shiftColumn && pos.Line == 1 && stmt.Start.Column > 0 {
			pos.Column += stmt.Start.Column - 1
		}
		pos.Line += stmt.Start.Line - 1
	}
	return &SyntaxError{
		Position:   pos,
		Message:    format(pe, pos),
		RawMessage: rawMsg,
	}
}

// FormatLineColSyntaxError formats a syntax error message with 1-based line:column
// and optional related text.
func FormatLineColSyntaxError(pos *storepb.Position, message, relatedText string) string {
	msg := fmt.Sprintf("Syntax error at line %d:%d: %s", pos.Line, pos.Column, message)
	if relatedText != "" {
		msg += "\nrelated text: " + relatedText
	}
	return msg
}

// FormatLineSyntaxError formats a syntax error message with a 1-based line number.
func FormatLineSyntaxError(pos *storepb.Position, message string) string {
	return fmt.Sprintf("Syntax error at line %d: %s", pos.Line, message)
}

// CollectSingleStatementTypesWithPosition classifies each AST into at most one StatementType
// paired with its 1-based end line and SQL text. If includeUnspecified is false, statements
// classifying as STATEMENT_TYPE_UNSPECIFIED are omitted.
func CollectSingleStatementTypesWithPosition[A AST](
	asts []AST,
	typeErr string,
	includeUnspecified bool,
	classify func(A) (storepb.StatementType, string, bool),
) ([]StatementTypeWithPosition, error) {
	return CollectStatementTypesWithPosition(asts, func(a AST) ([]storepb.StatementType, string, bool, error) {
		wrapped, ok := a.(A)
		if !ok {
			return nil, "", false, errors.New(typeErr)
		}
		stmtType, text, ok := classify(wrapped)
		if !ok || (!includeUnspecified && stmtType == storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED) {
			return nil, "", false, nil
		}
		return []storepb.StatementType{stmtType}, text, true, nil
	})
}

// BuildPathAST links nodes in order into a PathAST, returning nil when nodes is empty.
func BuildPathAST(nodes []SelectorNode) *PathAST {
	if len(nodes) == 0 {
		return nil
	}
	pathAST := NewPathAST(nodes[0])
	current := pathAST.Root
	for _, next := range nodes[1:] {
		current.SetNext(next)
		current = next
	}
	return pathAST
}
