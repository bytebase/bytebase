package tidb

import (
	"context"
	"slices"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// rowIDTestMetadata mirrors what schema sync stores: information_schema never
// lists _tidb_rowid, so no table carries it. v_shadow is a view whose real
// column is named _tidb_rowid, which TiDB allows for views but not base tables.
func rowIDTestMetadata() []*metadatapb.DatabaseSchemaMetadata {
	other := &metadatapb.DatabaseSchemaMetadata{
		Name: "other",
		Schemas: []*metadatapb.SchemaMetadata{
			{Tables: []*metadatapb.TableMetadata{
				{Name: "logs", Columns: []*metadatapb.ColumnMetadata{{Name: "phone"}}},
			}},
		},
	}
	app := &metadatapb.DatabaseSchemaMetadata{
		Name: "app",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "",
				Tables: []*metadatapb.TableMetadata{
					{Name: "logs", Columns: []*metadatapb.ColumnMetadata{{Name: "site_code"}, {Name: "phone"}}},
					{Name: "clustered_logs", Columns: []*metadatapb.ColumnMetadata{{Name: "id"}, {Name: "site_code"}}},
				},
				Views: []*metadatapb.ViewMetadata{
					{Name: "v_plain", Definition: "SELECT site_code FROM logs"},
					{Name: "v_shadow", Definition: "SELECT phone AS _tidb_rowid FROM logs"},
					{Name: "v_rowid", Definition: "SELECT _tidb_rowid AS rid, phone FROM logs"},
				},
			},
		},
	}
	return []*metadatapb.DatabaseSchemaMetadata{app, other}
}

func getRowIDTestSpan(t *testing.T, statement string) *base.QuerySpan {
	t.Helper()
	getter, lister := buildMockDatabaseMetadataGetter(rowIDTestMetadata())
	span, err := GetQuerySpan(
		context.TODO(),
		base.GetQuerySpanContext{GetDatabaseMetadataFunc: getter, ListDatabaseNamesFunc: lister},
		base.Statement{Text: statement},
		"app",
		"",
		false,
	)
	require.NoError(t, err, statement)
	require.NotNil(t, span, statement)
	return span
}

type rowIDResult struct {
	name string
	// lineage lists the source columns as "database.table.column", sorted.
	lineage []string
}

func flattenRowIDResults(span *base.QuerySpan) []rowIDResult {
	var got []rowIDResult
	for _, r := range span.Results {
		lineage := []string{}
		for c := range r.SourceColumns {
			lineage = append(lineage, c.Database+"."+c.Table+"."+c.Column)
		}
		slices.Sort(lineage)
		got = append(got, rowIDResult{name: r.Name, lineage: lineage})
	}
	return got
}

func TestGetQuerySpanTiDBRowIDResolves(t *testing.T) {
	none := []string{}
	phone := []string{"app.logs.phone"}
	tests := []struct {
		description string
		statement   string
		want        []rowIDResult
	}{
		{
			description: "unqualified",
			statement:   "SELECT _tidb_rowid FROM logs",
			want:        []rowIDResult{{"_tidb_rowid", none}},
		},
		{
			description: "table-qualified beside an ordinary column",
			statement:   "SELECT logs._tidb_rowid, phone FROM logs",
			want:        []rowIDResult{{"_tidb_rowid", none}, {"phone", phone}},
		},
		{
			description: "database-qualified",
			statement:   "SELECT app.logs._tidb_rowid FROM app.logs",
			want:        []rowIDResult{{"_tidb_rowid", none}},
		},
		{
			description: "through a table alias",
			statement:   "SELECT m._tidb_rowid FROM logs m",
			want:        []rowIDResult{{"_tidb_rowid", none}},
		},
		{
			description: "backticked and upper case",
			statement:   "SELECT `_TIDB_ROWID` FROM logs",
			want:        []rowIDResult{{"_TIDB_ROWID", none}},
		},
		{
			description: "inside aggregates beside an ordinary column",
			statement:   "SELECT MIN(_tidb_rowid) AS start_key, MAX(phone) AS p FROM logs",
			want:        []rowIDResult{{"start_key", none}, {"p", phone}},
		},
		{
			description: "BYT-10230 customer query",
			statement: `SELECT
    FLOOR((t.row_num - 1) / 500) + 1 AS page_num,
    MIN(t._tidb_rowid) AS start_key,
    MAX(t._tidb_rowid) AS end_key,
    COUNT(*) AS page_size
FROM (
    SELECT _tidb_rowid, ROW_NUMBER() OVER (ORDER BY _tidb_rowid) AS row_num
    FROM app.logs WHERE site_code = "3129"
) t
GROUP BY page_num
ORDER BY page_num`,
			want: []rowIDResult{{"page_num", none}, {"start_key", none}, {"end_key", none}, {"page_size", none}},
		},
		{
			description: "qualified in a join of base tables",
			statement:   "SELECT l._tidb_rowid, c.site_code FROM logs l JOIN clustered_logs c ON l.site_code = c.site_code",
			want:        []rowIDResult{{"_tidb_rowid", none}, {"site_code", []string{"app.clustered_logs.site_code"}}},
		},
		{
			description: "unqualified over a join of base tables",
			statement:   "SELECT _tidb_rowid FROM logs l JOIN clustered_logs c ON l.site_code = c.site_code",
			want:        []rowIDResult{{"_tidb_rowid", none}},
		},
		{
			description: "scalar subquery whose outer scope is a base table",
			statement:   "SELECT (SELECT _tidb_rowid FROM logs LIMIT 1) AS r FROM clustered_logs",
			want:        []rowIDResult{{"r", none}},
		},
		{
			description: "subquery qualified to an aliased outer base table",
			statement:   "SELECT (SELECT n._tidb_rowid FROM clustered_logs LIMIT 1) AS r FROM logs n",
			want:        []rowIDResult{{"r", none}},
		},
		{
			description: "subquery under an outer join of base tables",
			statement:   "SELECT (SELECT _tidb_rowid FROM logs LIMIT 1) AS r FROM logs l JOIN clustered_logs c ON l.site_code = c.site_code",
			want:        []rowIDResult{{"r", none}},
		},
		{
			description: "same table name in two databases, qualified to the second",
			statement:   "SELECT other.logs._tidb_rowid, other.logs.phone FROM app.logs, other.logs",
			want:        []rowIDResult{{"_tidb_rowid", none}, {"phone", []string{"other.logs.phone"}}},
		},
		{
			description: "view that projects the hidden column",
			statement:   "SELECT rid, phone FROM v_rowid",
			want:        []rowIDResult{{"rid", none}, {"phone", phone}},
		},
		{
			description: "CTE that projects the hidden column",
			statement:   "WITH c AS (SELECT _tidb_rowid AS rid, phone FROM logs) SELECT rid, phone FROM c",
			want:        []rowIDResult{{"rid", none}, {"phone", phone}},
		},
		{
			description: "qualified to the base table while a view shares the scope",
			statement:   "SELECT logs._tidb_rowid FROM logs, v_plain",
			want:        []rowIDResult{{"_tidb_rowid", none}},
		},
		{
			description: "SELECT * still excludes the hidden column",
			statement:   "SELECT * FROM logs",
			want:        []rowIDResult{{"site_code", []string{"app.logs.site_code"}}, {"phone", phone}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			span := getRowIDTestSpan(t, tc.statement)
			require.NoError(t, span.NotFoundError)
			require.Equal(t, tc.want, flattenRowIDResults(span))
		})
	}
}

func TestGetQuerySpanTiDBRowIDRealColumnWins(t *testing.T) {
	phone := []string{"app.logs.phone"}
	tests := []struct {
		description string
		statement   string
		want        []rowIDResult
	}{
		{
			description: "view column",
			statement:   "SELECT _tidb_rowid FROM v_shadow",
			want:        []rowIDResult{{"_tidb_rowid", phone}},
		},
		{
			description: "derived-table column",
			statement:   "SELECT x._tidb_rowid FROM (SELECT phone AS _tidb_rowid FROM logs) x",
			want:        []rowIDResult{{"_tidb_rowid", phone}},
		},
		{
			description: "CTE column, with the CTE named like a base table in another case",
			statement:   "WITH LOGS AS (SELECT phone AS _tidb_rowid FROM logs) SELECT _tidb_rowid FROM logs",
			want:        []rowIDResult{{"_tidb_rowid", phone}},
		},
		{
			description: "qualified rowid falls outward to a derived column when the local table lacks it",
			// TiDB binds x._tidb_rowid to the outer derived x because a clustered
			// base table has no _tidb_rowid. The outer column is real and masked.
			statement: "SELECT (SELECT x._tidb_rowid FROM clustered_logs x LIMIT 1) AS r FROM (SELECT phone AS _tidb_rowid FROM logs) x",
			want:      []rowIDResult{{"r", phone}},
		},
		{
			description: "view column and hidden column side by side",
			statement:   "SELECT v_shadow._tidb_rowid, logs._tidb_rowid FROM logs, v_shadow",
			want:        []rowIDResult{{"_tidb_rowid", phone}, {"_tidb_rowid", []string{}}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			span := getRowIDTestSpan(t, tc.statement)
			require.NoError(t, span.NotFoundError)
			require.Equal(t, tc.want, flattenRowIDResults(span))
		})
	}
}

func TestGetQuerySpanTiDBRowIDKeepsNotFound(t *testing.T) {
	tests := []struct {
		description string
		statement   string
		wantColumn  string
	}{
		{"view that does not project it", "SELECT _tidb_rowid FROM v_plain", "_tidb_rowid"},
		{"aliased view", "SELECT v.`_tidb_rowid` FROM v_plain v", "_tidb_rowid"},
		{"derived table that does not project it", "SELECT _tidb_rowid FROM (SELECT site_code FROM logs) x", "_tidb_rowid"},
		{"qualified to such a derived table", "SELECT x._tidb_rowid FROM (SELECT site_code FROM logs) x", "_tidb_rowid"},
		{"CTE that does not project it", "WITH c AS (SELECT site_code FROM logs) SELECT _tidb_rowid FROM c", "_tidb_rowid"},
		{"unqualified with a view in the FROM scope", "SELECT _tidb_rowid FROM clustered_logs, v_plain", "_tidb_rowid"},
		{"unqualified with a derived table in the FROM scope", "SELECT _tidb_rowid FROM logs JOIN (SELECT site_code FROM logs) x ON 1 = 1", "_tidb_rowid"},
		{"qualifier bound only to an outer view", "SELECT (SELECT x._tidb_rowid FROM logs LIMIT 1) FROM v_plain x", "_tidb_rowid"},
		{"qualifier shadowed in the subquery by a view", "SELECT (SELECT x._tidb_rowid FROM v_plain x LIMIT 1) FROM logs x", "_tidb_rowid"},
		{"qualified rowid with the same alias on an outer view", "SELECT (SELECT x._tidb_rowid FROM logs x LIMIT 1) AS r FROM v_plain x", "_tidb_rowid"},
		{"unqualified with a view in the outer scope", "SELECT (SELECT _tidb_rowid FROM clustered_logs LIMIT 1) FROM v_plain", "_tidb_rowid"},
		{"no FROM clause", "SELECT _tidb_rowid", "_tidb_rowid"},
		{"other unknown column on a base table", "SELECT nope FROM logs", "nope"},
		{"name that only starts with the hidden column's", "SELECT _tidb_rowid_x FROM logs", "_tidb_rowid_x"},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			span := getRowIDTestSpan(t, tc.statement)
			require.Error(t, span.NotFoundError)
			var resourceNotFound *base.ResourceNotFoundError
			require.True(t, errors.As(span.NotFoundError, &resourceNotFound))
			require.NotNil(t, resourceNotFound.Column)
			require.Equal(t, tc.wantColumn, *resourceNotFound.Column)
		})
	}
}
