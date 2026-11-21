package base

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestByteOffsetToRunePositionAndShift(t *testing.T) {
	sql := "ab\n€d"
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, ByteOffsetToRunePosition(sql, -5))
	// Offset 4 lands mid-rune inside '€' (3 bytes at offset 3..6).
	require.Equal(t, &storepb.Position{Line: 2, Column: 1}, ByteOffsetToRunePosition(sql, 4))
	require.Equal(t, &storepb.Position{Line: 2, Column: 2}, ByteOffsetToRunePosition(sql, 6))
	require.Equal(t, &storepb.Position{Line: 2, Column: 3}, ByteOffsetToRunePosition(sql, 999))

	basePos := &storepb.Position{Line: 10, Column: 5}
	require.Equal(t, &storepb.Position{Line: 10, Column: 7}, ShiftPosition(&storepb.Position{Line: 1, Column: 3}, basePos))
	require.Equal(t, &storepb.Position{Line: 11, Column: 3}, ShiftPosition(&storepb.Position{Line: 2, Column: 3}, basePos))
	require.Nil(t, ShiftPosition(nil, basePos))
}

func TestBuildStatementsAndMapByteRangesToLSP(t *testing.T) {
	sql := "SELECT 1; SELECT '🚀';"
	type seg struct {
		start, end int
		empty      bool
	}
	segs := []seg{{0, 8, false}, {9, len(sql) - 1, false}}

	withSemi := BuildStatements(sql, segs, func(s seg) (int, int, bool) { return s.start, s.end, s.empty }, true)
	require.Len(t, withSemi, 2)
	require.Equal(t, "SELECT 1;", withSemi[0].Text)
	require.Equal(t, " SELECT '🚀';", withSemi[1].Text)

	exact := BuildStatements(sql, segs, func(s seg) (int, int, bool) { return s.start, s.end, s.empty }, false)
	require.Equal(t, "SELECT 1", exact[0].Text)
	require.Nil(t, BuildStatements(sql, []seg(nil), func(s seg) (int, int, bool) { return s.start, s.end, s.empty }, false))

	require.Nil(t, MapByteRangesToLSP(sql, []seg(nil), func(s seg) (int, int) { return s.start, s.end }))
	lspRanges := MapByteRangesToLSP(sql, []seg{{10, len(sql) - 1, false}, {0, 8, false}, {5, 2, false}}, func(s seg) (int, int) { return s.start, s.end })
	require.Len(t, lspRanges, 2)
	require.Equal(t, uint32(10), lspRanges[0].Start.Character)
	// Surrogate pair '🚀' adds 2 UTF-16 code units.
	require.Equal(t, uint32(21), lspRanges[0].End.Character)
	require.Equal(t, uint32(0), lspRanges[1].Start.Character)
	require.Equal(t, uint32(8), lspRanges[1].End.Character)
}

func TestSpanHelpers(t *testing.T) {
	require.Equal(t, Explain, ConvertQueryType(2, 1, 2, 3, 4, 5))
	require.Equal(t, QueryTypeUnknown, ConvertQueryType(99, 1, 2, 3, 4, 5))

	require.Nil(t, ConvertColumnSet[string](nil, func(s string) ColumnResource { return ColumnResource{Column: s} }))
	cols := ConvertColumnSet(map[string]bool{"c1": true}, func(s string) ColumnResource { return ColumnResource{Column: s} })
	require.Equal(t, SourceColumnSet{{Column: "c1"}: true}, cols)

	require.Nil(t, ConvertQuerySpanResults[int](nil, func(_ int) QuerySpanResult { return QuerySpanResult{} }))
	res := ConvertQuerySpanResults([]int{1}, func(_ int) QuerySpanResult { return QuerySpanResult{Name: "a"} })
	require.Equal(t, []QuerySpanResult{{Name: "a"}}, res)

	cause := errors.New("mix")
	mixErr := &MixedTablesError{Cause: cause}
	require.ErrorIs(t, mixErr, MixUserSystemTablesError)
	require.ErrorIs(t, mixErr, cause)
}

func TestCaretToByteOffsetAndCandidates(t *testing.T) {
	sql := "SELECT '🚀'\nFROM tbl"
	// Line 2, col 5 -> after "FROM "
	require.Equal(t, 19, CaretToByteOffset(sql, 2, 5))

	cands := DeduplicateCandidates([]string{"b", "a", "b"}, func(s string) (Candidate, bool) {
		return Candidate{Type: CandidateTypeColumn, Text: s}, true
	}, CompareCandidateByTypeText)
	require.Equal(t, []Candidate{
		{Type: CandidateTypeColumn, Text: "a"},
		{Type: CandidateTypeColumn, Text: "b"},
	}, cands)
}

func TestExplainAndDiagnoseHelpers(t *testing.T) {
	sentinel := errors.New("executes")
	require.ErrorIs(t, ConvertExplainAnalyzeError(sentinel, sentinel), ErrExplainAnalyzeExecutes)
	other := errors.New("other")
	require.Equal(t, other, ConvertExplainAnalyzeError(other, sentinel))

	require.Empty(t, DiagnoseSyntaxError("SELECT 1", nil))
	diags := ConvertOffsetDiagnostics("SELECT\n  *", []int{9}, func(off int) (int, string) {
		return off, "bad token"
	})
	require.Len(t, diags, 1)
	require.Equal(t, uint32(1), diags[0].Range.Start.Line)
	require.Equal(t, uint32(2), diags[0].Range.Start.Character)
}
