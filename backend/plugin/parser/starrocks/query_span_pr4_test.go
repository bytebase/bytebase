package starrocks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// TestQuerySpanExtractor_PR4Features is the value-chain keystone for the PR4
// query features (omni #305 + #306): it proves that the new StarRocks query
// constructs deliver correct query-span lineage end-to-end through Bytebase
// (the extractor delegates to omni/starrocks/analysis). Before the omni dep
// bump these constructs do not parse, so lineage is empty/wrong; after, the
// physical sources resolve and the literal-derived relations contribute none.
func TestQuerySpanExtractor_PR4Features(t *testing.T) {
	tests := []struct {
		name           string
		sql            string
		expectedTables []base.ColumnResource
	}{
		{
			// PR4a: LATERAL + unnest table function — t is the only physical
			// source; the unnest relation must not leak a phantom table.
			name:           "lateral_unnest",
			sql:            "SELECT t.id, u.unnest FROM t, LATERAL unnest(t.arr) AS u",
			expectedTables: []base.ColumnResource{{Database: "test", Table: "t"}},
		},
		{
			// PR4a: VALUES inline-table is literal-derived — no physical table.
			name:           "values_inline_table",
			sql:            "SELECT * FROM (VALUES (1,'a'),(2,'b')) AS v(id,name)",
			expectedTables: nil,
		},
		{
			// PR4b: a column ref inside a map literal value flows to lineage.
			name:           "map_literal_column",
			sql:            "SELECT map<varchar,int>{'x':a_col} AS m FROM t",
			expectedTables: []base.ColumnResource{{Database: "test", Table: "t"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := newQuerySpanExtractor("test", base.GetQuerySpanContext{}, false)
			querySpan, err := extractor.getQuerySpan(context.Background(), tt.sql)
			require.NoError(t, err)

			var actualTables []base.ColumnResource
			for table := range querySpan.SourceColumns {
				actualTables = append(actualTables, table)
			}

			require.Equal(t, len(tt.expectedTables), len(actualTables),
				"expected %d source tables, got %d (%+v)", len(tt.expectedTables), len(actualTables), actualTables)
			for _, expected := range tt.expectedTables {
				found := false
				for _, actual := range actualTables {
					if actual.Database == expected.Database && actual.Table == expected.Table {
						found = true
						break
					}
				}
				require.True(t, found, "expected source table %s.%s not found in %+v",
					expected.Database, expected.Table, actualTables)
			}
		})
	}
}
