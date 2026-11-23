package base

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type stubNodeAST struct {
	text  string
	start *storepb.Position
	kind  storepb.StatementType
	valid bool
}

func (s *stubNodeAST) ASTStartPosition() *storepb.Position { return s.start }

type otherStubAST struct{}

func (otherStubAST) ASTStartPosition() *storepb.Position { return nil }

type sampleParseErr struct {
	pos int
	msg string
}

func (e *sampleParseErr) Error() string { return fmt.Sprintf("parse error at %d: %s", e.pos, e.msg) }

func TestParseSegmentedStatementsAndNonEmptySegments(t *testing.T) {
	split := func(_ string) ([]Statement, error) {
		return []Statement{
			{Text: "SELECT 1;", Start: &storepb.Position{Line: 1, Column: 1}},
			{Text: "  ; ", Empty: true, Start: &storepb.Position{Line: 1, Column: 10}},
			{Text: "   \n\t ", Start: &storepb.Position{Line: 1, Column: 14}},
			{Text: "SELECT 2;", Start: &storepb.Position{Line: 2, Column: 1}},
		}, nil
	}
	parseSeg := func(stmt Statement) (*stubNodeAST, error) {
		return &stubNodeAST{text: stmt.Text, start: stmt.Start}, nil
	}

	parsed, err := ParseSegmentedStatements("ignored", split, parseSeg)
	require.NoError(t, err)
	require.Len(t, parsed, 4)
	require.NotNil(t, parsed[0].AST)
	require.Nil(t, parsed[1].AST)
	require.Nil(t, parsed[2].AST)
	require.NotNil(t, parsed[3].AST)
	require.Equal(t, "SELECT 2;", parsed[3].AST.(*stubNodeAST).text)

	nonEmpty, err := ParseNonEmptySegments("ignored", split, parseSeg)
	require.NoError(t, err)
	require.Len(t, nonEmpty, 2)
	require.Equal(t, "SELECT 1;", nonEmpty[0].text)
	require.Equal(t, "SELECT 2;", nonEmpty[1].text)

	// Error propagation from split and parseSegment.
	splitErr := errors.New("split failed")
	_, err = ParseSegmentedStatements("x", func(string) ([]Statement, error) { return nil, splitErr }, parseSeg)
	require.ErrorIs(t, err, splitErr)
	_, err = ParseNonEmptySegments("x", func(string) ([]Statement, error) { return nil, splitErr }, parseSeg)
	require.ErrorIs(t, err, splitErr)

	parseErr := errors.New("parse failed")
	failParse := func(Statement) (*stubNodeAST, error) { return nil, parseErr }
	_, err = ParseSegmentedStatements("x", split, failParse)
	require.ErrorIs(t, err, parseErr)
	_, err = ParseNonEmptySegments("x", split, failParse)
	require.ErrorIs(t, err, parseErr)
}

func TestParseMultiNodeStatements(t *testing.T) {
	emptySplit := func(string) ([]Statement, error) { return nil, nil }
	res, err := ParseMultiNodeStatements("", emptySplit, func(Statement) ([]AST, error) { return nil, nil })
	require.NoError(t, err)
	require.Nil(t, res)

	split := func(string) ([]Statement, error) {
		return []Statement{
			{Text: "-- comment", Empty: true},
			{Text: "SELECT 1; SELECT 2;"},
		}, nil
	}
	res, err = ParseMultiNodeStatements("input", split, func(stmt Statement) ([]AST, error) {
		if stmt.Empty {
			return nil, nil
		}
		return []AST{
			&stubNodeAST{text: "first"},
			&stubNodeAST{text: "second"},
		}, nil
	})
	require.NoError(t, err)
	require.Len(t, res, 3)
	require.Nil(t, res[0].AST)
	require.Equal(t, "first", res[1].AST.(*stubNodeAST).text)
	require.Equal(t, "second", res[2].AST.(*stubNodeAST).text)

	parseErr := errors.New("bad syntax")
	_, err = ParseMultiNodeStatements("input", split, func(Statement) ([]AST, error) { return nil, parseErr })
	require.ErrorIs(t, err, parseErr)

	// ParseNonEmptyMultiNodeStatements omits empty segments and segments yielding zero ASTs.
	nonEmpty, err := ParseNonEmptyMultiNodeStatements("input", split, func(stmt Statement) ([]AST, error) {
		if stmt.Text == "skip" {
			return nil, nil
		}
		return []AST{&stubNodeAST{text: "a"}, &stubNodeAST{text: "b"}}, nil
	})
	require.NoError(t, err)
	require.Len(t, nonEmpty, 2)
	require.Equal(t, "a", nonEmpty[0].AST.(*stubNodeAST).text)
	require.Equal(t, "b", nonEmpty[1].AST.(*stubNodeAST).text)
}

func TestConvertParseError(t *testing.T) {
	extract := func(pe *sampleParseErr) (int, string, string) {
		return pe.pos, pe.Error(), pe.msg
	}

	// Non-matching error passes through unchanged.
	other := errors.New("io error")
	require.Equal(t, other, ConvertParseError("SELECT 1", other, nil, extract))

	// Wrapped matching error converts with rune column and basePos shift.
	wrapped := errors.Wrap(&sampleParseErr{pos: 10, msg: "unexpected token"}, "wrap")
	converted := ConvertParseError("SELECT '中' x", wrapped, &storepb.Position{Line: 3, Column: 5}, extract)
	var synErr *SyntaxError
	require.ErrorAs(t, converted, &synErr)
	// Offset 10 is right after "'中" (8 ASCII + 3-byte '中' => mid-rune clamp at col 9; +4 shift = 13).
	require.Equal(t, &storepb.Position{Line: 3, Column: 13}, synErr.Position)
	require.Equal(t, "parse error at 10: unexpected token", synErr.Message)
	require.Equal(t, "unexpected token", synErr.RawMessage)

	// Multiline segment shifts line only (column stays segment-relative).
	mlErr := ConvertParseError("SELECT 1\nFROM", &sampleParseErr{pos: 11, msg: "bad from"}, &storepb.Position{Line: 4, Column: 7}, extract)
	require.ErrorAs(t, mlErr, &synErr)
	require.Equal(t, &storepb.Position{Line: 5, Column: 3}, synErr.Position)

	// ConvertSyntaxError formats using the shifted position and honors shiftColumn.
	stmt := Statement{Text: "SELECT FROM", Start: &storepb.Position{Line: 3, Column: 5}}
	errLineCol := ConvertSyntaxError(stmt, &sampleParseErr{pos: 4, msg: "bad"}, true,
		func(pe *sampleParseErr) (int, string) { return pe.pos, pe.msg },
		func(pe *sampleParseErr, pos *storepb.Position) string {
			return FormatLineColSyntaxError(pos, pe.msg, "near FROM")
		},
	)
	require.ErrorAs(t, errLineCol, &synErr)
	require.Equal(t, &storepb.Position{Line: 3, Column: 9}, synErr.Position)
	require.Equal(t, "Syntax error at line 3:9: bad\nrelated text: near FROM", synErr.Message)
	require.Equal(t, "bad", synErr.RawMessage)

	errLineOnly := ConvertSyntaxError(stmt, &sampleParseErr{pos: 4, msg: "bad"}, false,
		func(pe *sampleParseErr) (int, string) { return pe.pos, pe.msg },
		func(pe *sampleParseErr, pos *storepb.Position) string {
			return FormatLineSyntaxError(pos, pe.msg)
		},
	)
	require.ErrorAs(t, errLineOnly, &synErr)
	require.Equal(t, &storepb.Position{Line: 3, Column: 5}, synErr.Position)
	require.Equal(t, "Syntax error at line 3: bad", synErr.Message)

	// Non-matching error passes through unchanged.
	require.Equal(t, other, ConvertSyntaxError(stmt, other, true,
		func(pe *sampleParseErr) (int, string) { return pe.pos, pe.msg },
		func(_ *sampleParseErr, pos *storepb.Position) string { return FormatLineSyntaxError(pos, "") },
	))
}

func TestCollectSingleStatementTypesWithPosition(t *testing.T) {
	asts := []AST{
		&stubNodeAST{
			text:  "CREATE TABLE t (\n  id INT\n);",
			start: &storepb.Position{Line: 2, Column: 1},
			kind:  storepb.StatementType_CREATE_TABLE,
			valid: true,
		},
		&stubNodeAST{
			text:  "SELECT 1;",
			start: &storepb.Position{Line: 5, Column: 1},
			kind:  storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED,
			valid: true,
		},
		&stubNodeAST{
			text:  "",
			start: &storepb.Position{Line: 6, Column: 1},
			valid: false,
		},
	}
	classify := func(a *stubNodeAST) (storepb.StatementType, string, bool) {
		return a.kind, a.text, a.valid
	}

	included, err := CollectSingleStatementTypesWithPosition(asts, "wrong type", true, classify)
	require.NoError(t, err)
	require.Equal(t, []StatementTypeWithPosition{
		{Type: storepb.StatementType_CREATE_TABLE, Line: 4, Text: "CREATE TABLE t (\n  id INT\n);"},
		{Type: storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED, Line: 5, Text: "SELECT 1;"},
	}, included)

	filtered, err := CollectSingleStatementTypesWithPosition(asts, "wrong type", false, classify)
	require.NoError(t, err)
	require.Equal(t, []StatementTypeWithPosition{
		{Type: storepb.StatementType_CREATE_TABLE, Line: 4, Text: "CREATE TABLE t (\n  id INT\n);"},
	}, filtered)

	_, err = CollectSingleStatementTypesWithPosition([]AST{otherStubAST{}}, "expected stub AST", true, classify)
	require.EqualError(t, err, "expected stub AST")
}

func TestBuildPathAST(t *testing.T) {
	require.Nil(t, BuildPathAST(nil))
	require.Nil(t, BuildPathAST([]SelectorNode{}))

	single := BuildPathAST([]SelectorNode{NewItemSelector("email")})
	require.NotNil(t, single)
	s, err := single.String()
	require.NoError(t, err)
	require.Equal(t, "email", s)

	chain := BuildPathAST([]SelectorNode{
		NewItemSelector("user"),
		NewArraySelector("phones", 0),
		NewItemSelector("number"),
	})
	require.NotNil(t, chain)
	s, err = chain.String()
	require.NoError(t, err)
	require.Equal(t, "user.phones[0].number", s)
}
