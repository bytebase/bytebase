package tidb

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
