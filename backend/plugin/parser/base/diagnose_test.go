package base

import (
	"context"
	"encoding/json"
	"testing"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestDiagnoseNeverReturnsNil(t *testing.T) {
	a := require.New(t)

	// Engine diagnose funcs return a nil slice for valid statements.
	RegisterDiagnoseFunc(storepb.Engine_MONGODB, func(_ context.Context, _ DiagnoseContext, _ string) ([]Diagnostic, error) {
		return nil, nil
	})

	for _, engine := range []storepb.Engine{
		storepb.Engine_MONGODB,
		// Engine without a registered diagnose func.
		storepb.Engine_ENGINE_UNSPECIFIED,
	} {
		diagnostics, err := Diagnose(context.Background(), DiagnoseContext{}, engine, "db.users.find()")
		a.NoError(err)
		a.NotNil(diagnostics)
		a.Empty(diagnostics)

		// The LSP spec requires publishing an empty array (not null) to clear
		// previously published diagnostics on the client.
		payload, err := json.Marshal(lsp.PublishDiagnosticsParams{Diagnostics: diagnostics})
		a.NoError(err)
		a.Contains(string(payload), `"diagnostics":[]`)
	}
}

func TestConvertPositionToUTF16Position(t *testing.T) {
	testCases := []struct {
		description string
		text        string
		position    *storepb.Position
		want        *lsp.Position
	}{
		{
			description: "empty text",
			text:        "",
			position:    &storepb.Position{Line: 1, Column: 1},
			want:        &lsp.Position{Line: 0, Character: 0},
		},
		{
			description: "ascii",
			text:        "hello, world",
			position:    &storepb.Position{Line: 1, Column: 6}, // 1-based: 6th character = ','
			want:        &lsp.Position{Line: 0, Character: 5},  // 0-based UTF-16: character 5
		},
		{
			description: "surrogate pairs",
			// 𝄞 encoded in utf16 with 2 code units, and 4 code units in utf8.
			// "abc𝄞def" - characters: a(1) b(2) c(3) 𝄞(4) d(5) e(6) f(7)
			text:     "abc𝄞def",
			position: &storepb.Position{Line: 1, Column: 5}, // 1-based: 5th character = 'd'
			want: &lsp.Position{
				Line:      0,
				Character: 5, // 0-based UTF-16: a=0, b=1, c=2, 𝄞=3,4 (surrogate pair), d=5
			},
		},
		{
			description: "Chinese characters",
			// Chinese characters are 3 bytes in UTF-8, 1 code unit in UTF-16
			// "SELECT 你好 FROM" - characters: S(1) E(2) L(3) E(4) C(5) T(6) SPACE(7) 你(8) 好(9) SPACE(10) F(11)...
			text:     "SELECT 你好 FROM t",
			position: &storepb.Position{Line: 1, Column: 8}, // 1-based: 8th character = '你'
			want: &lsp.Position{
				Line:      0,
				Character: 7, // 0-based UTF-16: '你' is at character 7
			},
		},
		{
			description: "Mixed ASCII and Chinese",
			// "你好world" - characters: 你(1) 好(2) w(3) o(4) r(5) l(6) d(7)
			text:     "你好world",
			position: &storepb.Position{Line: 1, Column: 3}, // 1-based: 3rd character = 'w'
			want: &lsp.Position{
				Line:      0,
				Character: 2, // 0-based UTF-16: 你=0, 好=1, w=2
			},
		},
		{
			description: "Emoji",
			// Emoji like 😀 are 4 bytes in UTF-8, 2 code units in UTF-16 (surrogate pair)
			// "SELECT 😀 FROM" - characters: S(1) E(2) L(3) E(4) C(5) T(6) SPACE(7) 😀(8) SPACE(9) F(10)...
			text:     "SELECT 😀 FROM t",
			position: &storepb.Position{Line: 1, Column: 9}, // 1-based: 9th character = space after emoji
			want: &lsp.Position{
				Line:      0,
				Character: 9, // 0-based UTF-16: S=0...SPACE=6, 😀=7,8 (surrogate pair), SPACE=9
			},
		},
		{
			description: "Multi-line with Chinese",
			// Line 1: "SELECT 1;"
			// Line 2: "SELECT 你好"
			text:     "SELECT 1;\nSELECT 你好",
			position: &storepb.Position{Line: 2, Column: 8}, // 1-based: line 2, 8th character = '你'
			want: &lsp.Position{
				Line:      1,
				Character: 7, // 0-based UTF-16: line 1 (second line), character 7
			},
		},
	}

	a := require.New(t)
	for _, tc := range testCases {
		got := convertPositionToUTF16Position(tc.position, tc.text)
		a.Equalf(tc.want, got, "Test case: %s", tc.description)
	}
}
