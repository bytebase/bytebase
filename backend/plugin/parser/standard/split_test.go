package standard

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func generateOneMBInsert() string {
	var rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	letterList := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]byte, 1024*1024)
	for i := range b {
		b[i] = letterList[rand.Intn(len(letterList))]
	}
	return fmt.Sprintf("INSERT INTO t values('%s')", string(b))
}

func TestApplyMultiStatements(t *testing.T) {
	type testData struct {
		statement string
		total     int
	}
	tests := []testData{
		{
			statement: `
			CREATE TABLE t(
				a int,
				b int,
				c int);


			/* This is a comment */
			CREATE TABLE t1(
				a int, b int c)`,
			total: 2,
		},
		{
			statement: `
			CREATE TABLE t(
				a int,
				b int,
				c int);


			CREATE TABLE t1(
				a int, b int c);
			` + generateOneMBInsert(),
			total: 3,
		},
	}

	total := 0
	countStatements := func(string) error {
		total++
		return nil
	}

	for _, test := range tests {
		total = 0
		err := applyMultiStatements(strings.NewReader(test.statement), countStatements)
		require.NoError(t, err)
		require.Equal(t, test.total, total)
	}
}

func TestSplitSQL(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      []base.Statement
	}{
		{
			name:      "simple single statement",
			statement: "SELECT 1;",
			want: []base.Statement{
				{
					Text:     "SELECT 1;",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					End:      &storepb.Position{Line: 1, Column: 10},
					Range:    &storepb.Range{Start: 0, End: 9},
					Empty:    false,
				},
			},
		},
		{
			name:      "multi-line statement",
			statement: "SELECT\n  1;",
			want: []base.Statement{
				{
					Text:     "SELECT\n  1;",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					End:      &storepb.Position{Line: 2, Column: 5},
					Range:    &storepb.Range{Start: 0, End: 11},
					Empty:    false,
				},
			},
		},
		{
			name:      "multiple statements",
			statement: "SELECT 1;\nSELECT 2;",
			want: []base.Statement{
				{
					Text:     "SELECT 1;",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					End:      &storepb.Position{Line: 1, Column: 10},
					Range:    &storepb.Range{Start: 0, End: 9},
					Empty:    false,
				},
				{
					Text:     "SELECT 2;",
					BaseLine: 1,
					Start:    &storepb.Position{Line: 2, Column: 1},
					End:      &storepb.Position{Line: 2, Column: 10},
					Range:    &storepb.Range{Start: 10, End: 19},
					Empty:    false,
				},
			},
		},
		{
			name:      "multi-byte characters - Chinese",
			statement: "SELECT '中文';",
			want: []base.Statement{
				{
					Text:     "SELECT '中文';",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					// Column is 1-based character offset: S(1) E(2) L(3) E(4) C(5) T(6) ' '(7) '(8) 中(9) 文(10) '(11) ;(12) = 12 chars, End is exclusive so 13
					End:   &storepb.Position{Line: 1, Column: 13},
					Range: &storepb.Range{Start: 0, End: 16}, // byte length: 8 + 3 + 3 + 2 = 16
					Empty: false,
				},
			},
		},
		{
			name:      "multi-byte characters - emoji",
			statement: "SELECT '🎉';",
			want: []base.Statement{
				{
					Text:     "SELECT '🎉';",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					// Column is 1-based character offset: S(1) E(2) L(3) E(4) C(5) T(6) ' '(7) '(8) 🎉(9) '(10) ;(11) = 11 chars, End is exclusive so 12
					End:   &storepb.Position{Line: 1, Column: 12},
					Range: &storepb.Range{Start: 0, End: 14}, // byte length: 8 + 4 + 2 = 14
					Empty: false,
				},
			},
		},
		{
			name:      "multi-byte on second line",
			statement: "SELECT\n  '中文';",
			want: []base.Statement{
				{
					Text:     "SELECT\n  '中文';",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					// Line 2 (1-based): ' '(1) ' '(2) '(3) 中(4) 文(5) '(6) ;(7) = 7 chars, End is exclusive so 8
					End:   &storepb.Position{Line: 2, Column: 8},
					Range: &storepb.Range{Start: 0, End: 18}, // 7 + 3 + 3 + 3 + 2 = 18
					Empty: false,
				},
			},
		},
		{
			name:      "multiple statements with multi-byte on separate lines",
			statement: "SELECT '中';\nSELECT '文';",
			want: []base.Statement{
				{
					Text:     "SELECT '中';",
					BaseLine: 0,
					Start:    &storepb.Position{Line: 1, Column: 1},
					// S(1) E(2) L(3) E(4) C(5) T(6) ' '(7) '(8) 中(9) '(10) ;(11) = 11 chars, End is exclusive so 12
					End:   &storepb.Position{Line: 1, Column: 12},
					Range: &storepb.Range{Start: 0, End: 13}, // 8 + 3 + 2 = 13
					Empty: false,
				},
				{
					Text:     "SELECT '文';",
					BaseLine: 1,
					Start:    &storepb.Position{Line: 2, Column: 1},
					End:      &storepb.Position{Line: 2, Column: 12},
					Range:    &storepb.Range{Start: 14, End: 27}, // starts after newline
					Empty:    false,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SplitSQL(tc.statement)
			require.NoError(t, err)
			require.Equal(t, len(tc.want), len(got), "number of statements mismatch")
			for i, want := range tc.want {
				require.Equal(t, want.Text, got[i].Text, "Text mismatch at index %d", i)
				require.Equal(t, want.BaseLine, got[i].BaseLine, "BaseLine mismatch at index %d", i)
				require.Equal(t, want.Start, got[i].Start, "Start mismatch at index %d", i)
				require.Equal(t, want.End, got[i].End, "End mismatch at index %d", i)
				require.Equal(t, want.Range, got[i].Range, "Range mismatch at index %d", i)
				require.Equal(t, want.Empty, got[i].Empty, "Empty mismatch at index %d", i)
			}
		})
	}
}
