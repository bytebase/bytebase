package redshift

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	lsp "github.com/bytebase/lsp-protocol"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestGetStatementRanges(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      []base.Range
	}{
		{
			name:      "multi statement",
			statement: "SELECT 1;\n  SELECT 2;",
			want: []base.Range{
				{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 9},
				},
				{
					Start: lsp.Position{Line: 1, Character: 2},
					End:   lsp.Position{Line: 1, Character: 11},
				},
			},
		},
		{
			name:      "utf16 comment prefix",
			statement: "/*🙂*/ SELECT 1;",
			want: []base.Range{
				{
					Start: lsp.Position{Line: 0, Character: 7},
					End:   lsp.Position{Line: 0, Character: 16},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GetStatementRanges(context.Background(), base.StatementRangeContext{}, tc.statement)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestGetStatementRangesScalesLinearlyForLargeScript(t *testing.T) {
	smallDuration := timeGetStatementRanges(t, redshiftStatementRangeScript(1000))
	largeDuration := timeGetStatementRanges(t, redshiftStatementRangeScript(10000))

	require.Lessf(t, largeDuration, smallDuration*40, "10x more statements should not take %s vs %s", largeDuration, smallDuration)
}

func TestGetStatementRangesToleratesIncompleteSQL(t *testing.T) {
	got, err := GetStatementRanges(context.Background(), base.StatementRangeContext{}, "SELECT * FROM")
	require.NoError(t, err)
	require.Equal(t, []base.Range{
		{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 0, Character: 13},
		},
	}, got)
}

func redshiftStatementRangeScript(count int) string {
	var builder strings.Builder
	for i := range count {
		if i > 0 {
			builder.WriteByte('\n')
		}
		table := fmt.Sprintf("perf_redshift_%d", i)
		switch i % 4 {
		case 0:
			fmt.Fprintf(&builder, "CREATE TABLE %s (id INT, payload VARCHAR(255))", table)
		case 1:
			fmt.Fprintf(&builder, "INSERT INTO %s (id, payload) VALUES (%d, 'payload-%d')", table, i, i)
		case 2:
			fmt.Fprintf(&builder, "UPDATE %s SET payload = 'updated-%d' WHERE id = %d", table, i, i)
		default:
			fmt.Fprintf(&builder, "DROP TABLE IF EXISTS %s", table)
		}
		builder.WriteByte(';')
	}
	return builder.String()
}

func timeGetStatementRanges(t *testing.T, statement string) time.Duration {
	t.Helper()
	start := time.Now()
	ranges, err := GetStatementRanges(context.Background(), base.StatementRangeContext{}, statement)
	require.NoError(t, err)
	require.NotEmpty(t, ranges)
	return time.Since(start)
}
