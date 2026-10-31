package starrocks

import (
	"testing"

	"github.com/stretchr/testify/require"

	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// changedTables flattens the built resources into "database.table" strings.
func changedTables(t *testing.T, statement string) []string {
	t.Helper()
	summary := extract(t, statement, nil)
	var result []string
	for _, database := range summary.ChangedResources.Build().GetDatabases() {
		for _, schema := range database.GetSchemas() {
			for _, table := range schema.GetTables() {
				result = append(result, database.GetName()+"."+table.GetName())
			}
		}
	}
	return result
}

func extract(t *testing.T, statement string, dbMetadata *model.DatabaseMetadata) *base.ChangeSummary {
	t.Helper()
	stmts, err := base.ParseStatements(storepb.Engine_STARROCKS, statement)
	require.NoError(t, err)
	summary, err := base.ExtractChangedResources(storepb.Engine_STARROCKS, "db", "", dbMetadata, base.ExtractASTs(stmts), statement)
	require.NoError(t, err)
	return summary
}

func TestExtractChangedResourcesTables(t *testing.T) {
	for _, tc := range []struct {
		name      string
		statement string
		want      []string
	}{
		{"create table takes the request database", "CREATE TABLE t (a INT) DISTRIBUTED BY HASH(a) BUCKETS 1;", []string{"db.t"}},
		{"a qualified name keeps its own database", "CREATE TABLE other.t (a INT) DISTRIBUTED BY HASH(a) BUCKETS 1;", []string{"other.t"}},
		{"a catalog qualifier is dropped", "CREATE TABLE cat.other.t (a INT) DISTRIBUTED BY HASH(a) BUCKETS 1;", []string{"other.t"}},
		{"drop table", "DROP TABLE t;", []string{"db.t"}},
		{"truncate table", "TRUNCATE TABLE t;", []string{"db.t"}},
		{"alter table", "ALTER TABLE t ADD COLUMN b INT;", []string{"db.t"}},
		{"create index names its table", "CREATE INDEX idx ON t (a);", []string{"db.t"}},
		{"drop index names its table", "DROP INDEX idx ON t;", []string{"db.t"}},
		{"insert names its target", "INSERT INTO t VALUES (1);", []string{"db.t"}},
		{"update names its target, not its source", "UPDATE t SET a = 1 WHERE a IN (SELECT a FROM s);", []string{"db.t"}},
		{"delete names its target, not its source", "DELETE FROM t WHERE a = 1;", []string{"db.t"}},
		{"a view records no table", "CREATE VIEW v AS SELECT 1;", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.ElementsMatch(t, tc.want, changedTables(t, tc.statement))
		})
	}
}

func TestExtractChangedResourcesDMLAccounting(t *testing.T) {
	for _, tc := range []struct {
		name        string
		statement   string
		wantDML     int
		wantInserts int
		wantTexts   []string
	}{
		{name: "insert values counts rows, not statements", statement: "INSERT INTO t VALUES (1), (2), (3);", wantInserts: 3},
		{name: "insert select is a DML statement", statement: "INSERT INTO t SELECT a FROM s;", wantDML: 1, wantTexts: []string{"INSERT INTO t SELECT a FROM s;"}},
		{name: "update is a DML statement", statement: "UPDATE t SET a = 1 WHERE a = 2;", wantDML: 1, wantTexts: []string{"UPDATE t SET a = 1 WHERE a = 2;"}},
		{name: "delete is a DML statement", statement: "DELETE FROM t WHERE a = 1;", wantDML: 1, wantTexts: []string{"DELETE FROM t WHERE a = 1;"}},
		{name: "DDL is neither", statement: "CREATE TABLE t (a INT) DISTRIBUTED BY HASH(a) BUCKETS 1;"},
		{
			name:        "a script accumulates both",
			statement:   "INSERT INTO t VALUES (1), (2);\nUPDATE t SET a = 1;\nDELETE FROM t;",
			wantDML:     2,
			wantInserts: 2,
			wantTexts:   []string{"UPDATE t SET a = 1;", "DELETE FROM t;"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			summary := extract(t, tc.statement, nil)
			require.Equal(t, tc.wantDML, summary.DMLCount)
			require.Equal(t, tc.wantInserts, summary.InsertCount)
			require.Equal(t, tc.wantTexts, summary.DMLStatements)
		})
	}
}

func TestExtractChangedResourcesDatabaseOnlyTargets(t *testing.T) {
	for _, tc := range []struct {
		name      string
		statement string
		want      []string
	}{
		{"a qualified view records its database", "CREATE VIEW other.v AS SELECT 1;", []string{"other"}},
		{"an unqualified view records nothing", "CREATE VIEW v AS SELECT 1;", []string{}},
		{"alter view", "ALTER VIEW other.v AS SELECT 1;", []string{"other"}},
		{"drop view", "DROP VIEW other.v;", []string{"other"}},
		{"a qualified materialized view records its database", "DROP MATERIALIZED VIEW other.mv;", []string{"other"}},
		{"drop procedure", "DROP PROCEDURE other.p;", []string{"other"}},
		{"a table DDL records no database-only target", "CREATE TABLE other.t (a INT) DISTRIBUTED BY HASH(a) BUCKETS 1;", []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			summary := extract(t, tc.statement, nil)
			require.Equal(t, tc.want, summary.ChangedResources.GetDatabaseOnlyTargets())
		})
	}
}

func TestExtractChangedResourcesAffectedRows(t *testing.T) {
	dbMetadata := model.NewDatabaseMetadata(&metadatapb.DatabaseSchemaMetadata{
		Name:    "db",
		Schemas: []*metadatapb.SchemaMetadata{{Tables: []*metadatapb.TableMetadata{{Name: "t", RowCount: 42}}}},
	}, []byte{}, &storepb.DatabaseConfig{}, storepb.Engine_STARROCKS, false /* caseSensitive */)

	for _, tc := range []struct {
		name      string
		statement string
		want      int64
	}{
		{"drop discards existing rows", "DROP TABLE t;", 42},
		{"truncate discards existing rows", "TRUNCATE TABLE t;", 42},
		{"alter rewrites existing rows", "ALTER TABLE t ADD COLUMN b INT;", 42},
		{"create has no existing rows", "CREATE TABLE t (a INT) DISTRIBUTED BY HASH(a) BUCKETS 1;", 0},
		{"insert leaves existing rows alone", "INSERT INTO t VALUES (1);", 0},
		{"create index leaves existing rows alone", "CREATE INDEX idx ON t (a);", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			summary := extract(t, tc.statement, dbMetadata)
			require.Equal(t, tc.want, summary.ChangedResources.CountAffectedTableRows())
		})
	}
}

// foreignAST stands in for another engine's AST, which reached this extractor
// through the shared registry before StarRocks had one of its own.
type foreignAST struct{}

func (*foreignAST) ASTStartPosition() *storepb.Position { return nil }

func TestExtractChangedResourcesRejectsForeignAST(t *testing.T) {
	_, err := extractChangedResources("db", "", nil, []base.AST{&foreignAST{}}, "")
	require.ErrorContains(t, err, "expected omniAST for StarRocks")
}
