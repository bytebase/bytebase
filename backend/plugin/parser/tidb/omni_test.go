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
