package tidb

import (
	"context"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestExplainAnalyzeAccessTables(t *testing.T) {
	var metadata []*metadatapb.DatabaseSchemaMetadata
	for _, database := range []string{"db", "other"} {
		metadata = append(metadata, &metadatapb.DatabaseSchemaMetadata{
			Name: database,
			Schemas: []*metadatapb.SchemaMetadata{{
				Name: "",
				Tables: []*metadatapb.TableMetadata{
					{Name: "t", Columns: []*metadatapb.ColumnMetadata{{Name: "id"}}},
					{Name: "u", Columns: []*metadatapb.ColumnMetadata{{Name: "id"}}},
				},
			}},
		})
	}
	getter, lister := buildMockDatabaseMetadataGetter(metadata)
	for _, tc := range []struct {
		name      string
		statement string
		queryType base.QueryType
		sources   base.SourceColumnSet
	}{
		{
			name:      "count with predicate",
			statement: "EXPLAIN ANALYZE SELECT count(*) FROM t WHERE id = 839195265",
			queryType: base.Select,
			sources:   base.SourceColumnSet{{Database: "db", Table: "t"}: true},
		},
		{
			name:      "join and cross database subquery",
			statement: "EXPLAIN ANALYZE SELECT a.id FROM t a JOIN u b ON a.id = b.id WHERE a.id IN (SELECT id FROM other.t)",
			queryType: base.Select,
			sources:   base.SourceColumnSet{{Database: "db", Table: "t"}: true, {Database: "db", Table: "u"}: true, {Database: "other", Table: "t"}: true},
		},
		{
			name:      "derived join retains every table",
			statement: "EXPLAIN ANALYZE SELECT * FROM (SELECT t.id FROM t JOIN u ON t.id = u.id) x",
			queryType: base.Select,
			sources:   base.SourceColumnSet{{Database: "db", Table: "t"}: true, {Database: "db", Table: "u"}: true},
		},
		{
			name:      "nested derived join across databases",
			statement: "EXPLAIN ANALYZE SELECT * FROM (SELECT * FROM (SELECT t.id FROM t JOIN other.t u ON t.id = u.id) x) y",
			queryType: base.Select,
			sources:   base.SourceColumnSet{{Database: "db", Table: "t"}: true, {Database: "other", Table: "t"}: true},
		},
		{
			name:      "derived predicate subquery",
			statement: "EXPLAIN ANALYZE SELECT * FROM (SELECT id FROM t WHERE id IN (SELECT id FROM other.u)) x",
			queryType: base.Select,
			sources:   base.SourceColumnSet{{Database: "db", Table: "t"}: true, {Database: "other", Table: "u"}: true},
		},
		{
			name:      "subqueries inside compound expressions",
			statement: "EXPLAIN ANALYZE SELECT COALESCE((SELECT max(id) FROM other.t), 0) FROM t WHERE id > 0 AND id IN (SELECT id FROM u)",
			queryType: base.Select,
			sources:   base.SourceColumnSet{{Database: "db", Table: "t"}: true, {Database: "db", Table: "u"}: true, {Database: "other", Table: "t"}: true},
		},
		{
			name:      "qualified table",
			statement: "EXPLAIN ANALYZE SELECT count(*) FROM other.t",
			queryType: base.Select,
			sources:   base.SourceColumnSet{{Database: "other", Table: "t"}: true},
		},
		{
			name:      "plain explain",
			statement: "EXPLAIN SELECT * FROM t",
			queryType: base.Explain,
			sources:   base.SourceColumnSet{},
		},
		{
			name:      "analyzed write remains dml",
			statement: "EXPLAIN ANALYZE DELETE FROM t WHERE id = 1",
			queryType: base.DML,
			sources:   base.SourceColumnSet{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			span, err := GetQuerySpan(context.Background(), base.GetQuerySpanContext{
				GetDatabaseMetadataFunc: getter,
				ListDatabaseNamesFunc:   lister,
			}, base.Statement{Text: tc.statement}, "db", "", false)
			require.NoError(t, err)
			require.Equal(t, tc.queryType, span.Type)
			require.Equal(t, tc.sources, span.SourceColumns)
			require.Empty(t, span.Results)
		})
	}
}
