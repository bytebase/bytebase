package doris

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestQuerySpanExtractor_ExplainUnwrap pins the behaviour that EXPLAIN
// statements still surface the tables the underlying query reads — omni's
// span walker only descends into top-level SELECT/SetOp, so without the
// EXPLAIN unwrap in the extractor an `EXPLAIN SELECT ... FROM t` would
// produce zero AccessTables and table-level ACL checks would not see `t`.
func TestQuerySpanExtractor_ExplainUnwrap(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected []base.ColumnResource
	}{
		{
			name:     "EXPLAIN SELECT exposes the SELECT's tables",
			sql:      "EXPLAIN SELECT * FROM users",
			expected: []base.ColumnResource{{Database: "test", Table: "users"}},
		},
		{
			name:     "EXPLAIN VERBOSE SELECT exposes tables",
			sql:      "EXPLAIN VERBOSE SELECT id FROM t1 JOIN t2 ON t1.id = t2.id",
			expected: []base.ColumnResource{{Database: "test", Table: "t1"}, {Database: "test", Table: "t2"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := newQuerySpanExtractor("test", base.GetQuerySpanContext{}, false)
			span, err := extractor.getQuerySpan(context.Background(), tt.sql)
			require.NoError(t, err)

			var got []base.ColumnResource
			for table := range span.SourceColumns {
				got = append(got, table)
			}
			require.Equal(t, len(tt.expected), len(got),
				"expected %d tables, got %d (%v)", len(tt.expected), len(got), got)
			for _, want := range tt.expected {
				found := false
				for _, g := range got {
					if g.Database == want.Database && g.Table == want.Table {
						found = true
						break
					}
				}
				require.True(t, found, "missing table %s.%s in %v", want.Database, want.Table, got)
			}
		})
	}
}
