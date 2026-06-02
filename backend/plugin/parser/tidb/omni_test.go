package tidb

import (
	"testing"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestByteOffsetToRunePosition(t *testing.T) {
	tests := []struct {
		name       string
		sql        string
		byteOffset int
		want       *storepb.Position
	}{
		{
			name:       "empty input, offset 0",
			sql:        "",
			byteOffset: 0,
			want:       &storepb.Position{Line: 1, Column: 1},
		},
		{
			name:       "ascii start of line",
			sql:        "SELECT 1",
			byteOffset: 0,
			want:       &storepb.Position{Line: 1, Column: 1},
		},
		{
			name:       "ascii mid-line",
			sql:        "SELECT 1",
			byteOffset: 7,
			want:       &storepb.Position{Line: 1, Column: 8},
		},
		{
			name:       "ascii end of line",
			sql:        "SELECT 1",
			byteOffset: 8,
			want:       &storepb.Position{Line: 1, Column: 9},
		},
		{
			name:       "newline advances line, resets column",
			sql:        "SELECT 1\nSELECT 2",
			byteOffset: 9, // 'S' of second SELECT
			want:       &storepb.Position{Line: 2, Column: 1},
		},
		{
			name:       "multi-line, mid-line-2",
			sql:        "SELECT 1\nSELECT 2",
			byteOffset: 16, // '2'
			want:       &storepb.Position{Line: 2, Column: 8},
		},
		{
			name:       "multi-byte BMP rune (Chinese ideograph, 3 bytes)",
			sql:        "SELECT '中' AS x",
			byteOffset: 11, // byte AFTER the 3-byte ideograph — col should count it as 1 rune
			want:       &storepb.Position{Line: 1, Column: 10},
		},
		{
			name:       "surrogate-pair rune (emoji, 4 bytes) counted as ONE rune",
			sql:        "SELECT '😀' x",
			byteOffset: 12, // byte AFTER the emoji — storepb column is rune-based so emoji = 1
			want:       &storepb.Position{Line: 1, Column: 10},
		},
		{
			name:       "mid-rune offset treated as start-of-rune (BMP 3-byte)",
			sql:        "SELECT '中'",
			byteOffset: 9, // inside the 3-byte sequence (bytes 8,9,10 = E4 B8 AD)
			want:       &storepb.Position{Line: 1, Column: 9},
		},
		{
			name:       "mid-rune offset treated as start-of-rune (emoji)",
			sql:        "a😀b",
			byteOffset: 3, // inside emoji (emoji occupies bytes 1..4)
			want:       &storepb.Position{Line: 1, Column: 2},
		},
		{
			name:       "offset clamped to len(sql) when past end",
			sql:        "abc",
			byteOffset: 100,
			want:       &storepb.Position{Line: 1, Column: 4},
		},
		{
			name:       "negative offset treated as 0",
			sql:        "abc",
			byteOffset: -5,
			want:       &storepb.Position{Line: 1, Column: 1},
		},
		{
			name:       "offset at the newline character is still on the pre-newline line",
			sql:        "a\nb",
			byteOffset: 1, // the '\n' itself
			want:       &storepb.Position{Line: 1, Column: 2},
		},
		{
			name:       "multi-byte across a newline",
			sql:        "中\n中",
			byteOffset: 4, // byte of second '中' = line 2, col 1
			want:       &storepb.Position{Line: 2, Column: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ByteOffsetToRunePosition(tt.sql, tt.byteOffset)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGetOmniNode(t *testing.T) {
	a := require.New(t)

	// nil base.AST returns (nil, false).
	node, ok := GetOmniNode(nil)
	a.Nil(node)
	a.False(ok)

	// Non-OmniAST returns (nil, false). Use the native-tidb AST wrapper.
	nativeAST := &AST{StartPosition: &storepb.Position{Line: 1}}
	node, ok = GetOmniNode(nativeAST)
	a.Nil(node)
	a.False(ok)

	// OmniAST with non-nil node round-trips.
	list, err := ParseTiDBOmni("SELECT 1")
	a.NoError(err)
	a.NotNil(list)
	a.NotEmpty(list.Items)
	wrapped := &OmniAST{
		Node:          list.Items[0],
		Text:          "SELECT 1",
		StartPosition: &storepb.Position{Line: 1, Column: 1},
	}
	node, ok = GetOmniNode(wrapped)
	a.True(ok)
	a.Equal(list.Items[0], node)
}

func TestOmniASTStartPosition(t *testing.T) {
	a := require.New(t)
	pos := &storepb.Position{Line: 5, Column: 7}
	o := &OmniAST{StartPosition: pos}
	a.Equal(pos, o.ASTStartPosition())
}

// TestAsPingCapASTReturnsNativeAST pins the bridge contract: an OmniAST
// successfully parsed from omni-supported SQL also exposes a native pingcap
// AST via AsPingCapAST. Used by un-migrated advisors during the Phase 1.5
// migration window via tidbparser.GetTiDBAST.
func TestAsPingCapASTReturnsNativeAST(t *testing.T) {
	a := require.New(t)
	pos := &storepb.Position{Line: 1, Column: 1}
	o := &OmniAST{
		Text:          "SELECT 1",
		StartPosition: pos,
	}

	got, ok := o.AsPingCapAST()
	a.True(ok, "expected pingcap fallback to succeed for valid SQL")
	a.NotNil(got)
	a.Equal(pos, got.StartPosition)
	a.NotNil(got.Node, "pingcap StmtNode should be populated")
}

// TestAsPingCapASTCachesAcrossCalls pins the lazy+cached contract: the
// native parse runs once per OmniAST instance regardless of how many
// advisors call the bridge. Mirrors mysql/OmniAST.AsANTLRAST's antlrParsed
// flag pattern.
func TestAsPingCapASTCachesAcrossCalls(t *testing.T) {
	a := require.New(t)
	o := &OmniAST{
		Text:          "SELECT 1",
		StartPosition: &storepb.Position{Line: 1, Column: 1},
	}

	first, ok := o.AsPingCapAST()
	a.True(ok)
	a.NotNil(first)

	second, ok := o.AsPingCapAST()
	a.True(ok)

	// Same *AST pointer proves the cache hit; a fresh re-parse would
	// allocate a new wrapper.
	a.Same(first, second, "expected cached *AST; got fresh re-parse — bridge cache contract broken")
}

// TestAsPingCapASTCachesNegativeResult ensures a parse failure is also
// cached: a second call does not retry and continues to return (nil, false).
// Important so that an OmniAST wrapping unparseable-by-pingcap SQL doesn't
// repeatedly re-parse on every advisor call.
func TestAsPingCapASTCachesNegativeResult(t *testing.T) {
	a := require.New(t)
	o := &OmniAST{
		// Syntactically invalid SQL — pingcap should reject.
		Text:          "SELECT FROM WHERE;",
		StartPosition: &storepb.Position{Line: 1, Column: 1},
	}

	got, ok := o.AsPingCapAST()
	a.False(ok)
	a.Nil(got)
	a.True(o.pingcapParsed, "pingcapParsed flag should be set after first attempt")

	// Second call returns the same negative result without re-attempting.
	got2, ok2 := o.AsPingCapAST()
	a.False(ok2)
	a.Nil(got2)
}

// TestGetTiDBASTFallsBackToProvider pins the cross-cutting contract: when
// the dispatcher returns *OmniAST, un-migrated callers of GetTiDBAST get a
// native *AST through the PingCapASTProvider fallback. Without this, the
// dispatcher flip in §1.5.N+1 silently breaks every un-migrated advisor.
func TestGetTiDBASTFallsBackToProvider(t *testing.T) {
	a := require.New(t)
	o := &OmniAST{
		Text:          "SELECT 1",
		StartPosition: &storepb.Position{Line: 1, Column: 1},
	}

	got, ok := GetTiDBAST(o)
	a.True(ok, "GetTiDBAST should fall back to AsPingCapAST when handed an OmniAST")
	a.NotNil(got)
	a.NotNil(got.Node)
}

// TestAsPingCapASTLineTrackingMatchesCanonical pins that bridge-produced
// nodes have the same node.OriginTextPosition() as the canonical pre-flip
// path (ParseTiDBForSyntaxCheck). 49 of 51 tidb advisors call
// OriginTextPosition() to report advice line numbers; if the bridge omits
// the line-tracking work that the canonical path performs, post-flip
// un-migrated advisors silently regress to line 1 for every statement.
//
// This is the regression that PR review caught — the original bridge in
// this PR called bare ParseTiDB without applying applyTiDBLineTracking, so
// nodes came back with default OriginTextPosition (line 1 of the snippet,
// not line N of the original multi-statement input).
func TestAsPingCapASTLineTrackingMatchesCanonical(t *testing.T) {
	a := require.New(t)

	// Two-statement input. We compare the SECOND statement's line tracking
	// because that's where any drift would show: line 1 (default, broken)
	// vs line 2 (correct).
	multi := "CREATE TABLE foo (id INT);\nALTER TABLE foo ADD COLUMN x INT NOT NULL;"

	// Canonical pre-flip path: ParseTiDBForSyntaxCheck splits + parses +
	// applies applyTiDBLineTracking via the shared helper.
	canonical, err := ParseTiDBForSyntaxCheck(multi)
	a.NoError(err)
	a.Len(canonical, 2)
	canonicalSecond, ok := canonical[1].(*AST)
	a.True(ok)

	// Simulate what the dispatcher does post-flip: split, then for each
	// non-empty split build an OmniAST. The bridge must produce a node
	// whose OriginTextPosition matches the canonical path's.
	splits, err := base.SplitMultiSQL(storepb.Engine_TIDB, multi)
	a.NoError(err)
	a.GreaterOrEqual(len(splits), 2)
	secondSplit := splits[1]

	o := &OmniAST{
		Text: secondSplit.Text,
		// StartPosition is 1-based line in the original SQL; SplitMultiSQL
		// gives BaseLine() as 0-based, so +1 to convert.
		StartPosition: &storepb.Position{Line: int32(secondSplit.BaseLine()) + 1},
	}
	bridged, ok := o.AsPingCapAST()
	a.True(ok)

	a.Equal(
		canonicalSecond.Node.OriginTextPosition(),
		bridged.Node.OriginTextPosition(),
		"bridge OriginTextPosition must match ParseTiDBForSyntaxCheck — post-flip un-migrated advisors expect identical line numbers",
	)
	a.Equal(2, bridged.Node.OriginTextPosition(),
		"sanity: ALTER TABLE on line 2 of the input should report OriginTextPosition=2")
}

// TestAsPingCapASTLineTrackingMatchesCanonicalBlankLines exercises the
// leadingNewlinesStripped arithmetic in applyTiDBLineTracking (BYT-9381). With
// blank lines between statements, the native pingcap parser strips the leading
// newlines from the second statement's Text(), so the bridge must add them back
// to report the correct absolute OriginTextPosition. The single-newline sibling
// test above keeps leadingNewlinesStripped == 0 and so never covers this path;
// a regression in the arithmetic would otherwise land silently.
func TestAsPingCapASTLineTrackingMatchesCanonicalBlankLines(t *testing.T) {
	a := require.New(t)

	// Two blank lines between the statements: ALTER TABLE is on line 4.
	multi := "CREATE TABLE foo (id INT);\n\n\nALTER TABLE foo ADD COLUMN x INT NOT NULL;"

	canonical, err := ParseTiDBForSyntaxCheck(multi)
	a.NoError(err)
	a.Len(canonical, 2)
	canonicalSecond, ok := canonical[1].(*AST)
	a.True(ok)

	splits, err := base.SplitMultiSQL(storepb.Engine_TIDB, multi)
	a.NoError(err)
	a.GreaterOrEqual(len(splits), 2)
	secondSplit := splits[1]

	o := &OmniAST{
		Text:          secondSplit.Text,
		StartPosition: &storepb.Position{Line: int32(secondSplit.BaseLine()) + 1},
	}
	bridged, ok := o.AsPingCapAST()
	a.True(ok)

	a.Equal(
		canonicalSecond.Node.OriginTextPosition(),
		bridged.Node.OriginTextPosition(),
		"bridge OriginTextPosition must match ParseTiDBForSyntaxCheck across blank lines",
	)
	a.Equal(4, bridged.Node.OriginTextPosition(),
		"sanity: ALTER TABLE on line 4 (after two blank lines) should report OriginTextPosition=4")
}

// TestApplyTiDBLineTracking pins the leading-newline arithmetic directly across
// the edges the integration tests don't all exercise: no leading whitespace
// (line 1), single vs multiple leading newlines, CRLF line endings, a non-zero
// baseLine, and non-newline leading whitespace (BYT-9381).
func TestApplyTiDBLineTracking(t *testing.T) {
	nodes, err := ParseTiDB("SELECT 1", "", "")
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	node := nodes[0]

	cases := []struct {
		name     string
		baseLine int
		text     string
		want     int
	}{
		{"no leading whitespace is line 1", 0, "SELECT 1", 1},
		{"single leading newline", 0, "\nSELECT 1", 2},
		{"three leading newlines (the BYT-9381 bug)", 0, "\n\n\nSELECT 1", 4},
		{"CRLF leading newlines count once each", 0, "\r\n\r\n\r\nSELECT 1", 4},
		{"non-zero baseLine plus two newlines", 5, "\n\nSELECT 1", 8},
		{"leading spaces and tabs, no newline", 0, "  \tSELECT 1", 1},
		{"mixed leading whitespace with one newline", 0, "  \n  SELECT 1", 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := applyTiDBLineTracking(node, tc.baseLine, tc.text)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestCreateTableColumnLineTrackingBlankLines pins per-column line tracking for
// a CREATE TABLE preceded by blank lines (BYT-9381). The native parser may
// retain leading newlines in node.Text(); the column tokenizer counts from the
// start of that text, so its base line must exclude those leading newlines —
// otherwise the blank lines are double-counted into every column's line (the
// statement line is correct but columns land several lines too low).
func TestCreateTableColumnLineTrackingBlankLines(t *testing.T) {
	a := require.New(t)

	// CREATE TABLE on line 4 (after two blank lines); id on line 5, name on 6.
	multi := "SELECT 1;\n\n\nCREATE TABLE foo (\n  id INT,\n  name VARCHAR(50)\n);"
	asts, err := ParseTiDBForSyntaxCheck(multi)
	a.NoError(err)
	a.Len(asts, 2)
	ct, ok := asts[1].(*AST).Node.(*ast.CreateTableStmt)
	a.True(ok)

	a.Equal(4, ct.OriginTextPosition(), "CREATE TABLE statement line")
	a.Len(ct.Cols, 2)
	a.Equal(5, ct.Cols[0].OriginTextPosition(), "column id should be on line 5")
	a.Equal(6, ct.Cols[1].OriginTextPosition(), "column name should be on line 6")
}
