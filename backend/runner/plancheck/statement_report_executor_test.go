package plancheck

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// reportEngines derives the list from common.EngineSupportStatementReport so a
// newly admitted engine is covered without editing this test.
func reportEngines() []storepb.Engine {
	values := make([]int32, 0, len(storepb.Engine_name))
	for v := range storepb.Engine_name {
		values = append(values, v)
	}
	slices.Sort(values)

	var engines []storepb.Engine
	for _, v := range values {
		if engine := storepb.Engine(v); common.EngineSupportStatementReport(engine) {
			engines = append(engines, engine)
		}
	}
	return engines
}

func dedupeStatementTypes(types []storepb.StatementType) []storepb.StatementType {
	seen := make(map[storepb.StatementType]bool, len(types))
	var out []storepb.StatementType
	for _, t := range types {
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// The approval evaluator reads statement.sql_type from the report, so every
// report engine must classify the sheet (BYT-10136).
func TestSummaryStatementTypesCoverEveryReportEngine(t *testing.T) {
	const sheet = "CREATE TABLE t (id INT);\nUPDATE t SET id = 1 WHERE id = 2;\nDROP TABLE t;"
	want := []storepb.StatementType{
		storepb.StatementType_CREATE_TABLE,
		storepb.StatementType_UPDATE,
		storepb.StatementType_DROP_TABLE,
	}

	engines := reportEngines()
	require.Contains(t, engines, storepb.Engine_OCEANBASE, "OceanBase must be a report engine for this test to guard it")

	for _, engine := range engines {
		t.Run(engine.String(), func(t *testing.T) {
			stmts, err := parserbase.ParseStatements(engine, sheet)
			require.NoError(t, err)

			got, err := SummaryStatementTypes(engine, parserbase.ExtractASTs(stmts))
			require.NoError(t, err)
			require.ElementsMatch(t, want, dedupeStatementTypes(got))
		})
	}
}

// The MySQL-family classifier keeps STATEMENT_TYPE_UNSPECIFIED, so a DDL rule
// written as the negation of a DML list fails closed on an unclassified
// OceanBase statement.
func TestSummaryStatementTypesOceanBase(t *testing.T) {
	tests := []struct {
		name      string
		statement string
		want      []storepb.StatementType
	}{
		{"insert", "INSERT INTO t1 VALUES (1, 'a');", []storepb.StatementType{storepb.StatementType_INSERT}},
		{"insert with parallel hint", "INSERT /*+ PARALLEL(4) */ INTO t1 SELECT * FROM t2;", []storepb.StatementType{storepb.StatementType_INSERT}},
		{"update", "UPDATE t1 SET c1 = 'x' WHERE id = 1;", []storepb.StatementType{storepb.StatementType_UPDATE}},
		{"delete", "DELETE FROM t1 WHERE id = 1;", []storepb.StatementType{storepb.StatementType_DELETE}},
		{
			"create subpartitioned table",
			"CREATE TABLE t4 (id INT PRIMARY KEY, c1 INT) PARTITION BY RANGE(id) SUBPARTITION BY HASH(c1) SUBPARTITIONS 4 (PARTITION p0 VALUES LESS THAN (100), PARTITION p1 VALUES LESS THAN MAXVALUE);",
			[]storepb.StatementType{storepb.StatementType_CREATE_TABLE},
		},
		{
			"create range columns partitioned table",
			"CREATE TABLE t6 (id INT, dt DATE) PARTITION BY RANGE COLUMNS(dt) (PARTITION p0 VALUES LESS THAN ('2020-01-01'));",
			[]storepb.StatementType{storepb.StatementType_CREATE_TABLE},
		},
		{"alter table add column", "ALTER TABLE t1 ADD COLUMN c2 INT;", []storepb.StatementType{storepb.StatementType_ALTER_TABLE}},
		{"alter table modify column", "ALTER TABLE t1 MODIFY COLUMN c1 VARCHAR(20);", []storepb.StatementType{storepb.StatementType_ALTER_TABLE}},
		{"alter table add partition", "ALTER TABLE t1 ADD PARTITION (PARTITION p2 VALUES LESS THAN (200));", []storepb.StatementType{storepb.StatementType_ALTER_TABLE}},
		{"create index", "CREATE INDEX idx ON t1 (c1);", []storepb.StatementType{storepb.StatementType_CREATE_INDEX}},
		{"drop index", "DROP INDEX idx ON t1;", []storepb.StatementType{storepb.StatementType_DROP_INDEX}},
		{"truncate", "TRUNCATE TABLE t1;", []storepb.StatementType{storepb.StatementType_TRUNCATE}},
		{"drop table", "DROP TABLE t1;", []storepb.StatementType{storepb.StatementType_DROP_TABLE}},
		{
			"session variable is unclassified, not dropped",
			"SET SESSION ob_query_timeout = 10000000;",
			[]storepb.StatementType{storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED},
		},
		{
			"mixed sheet keeps every type",
			"UPDATE t1 SET c1 = 'x' WHERE id = 1;\nALTER TABLE t1 ADD COLUMN c2 INT;",
			[]storepb.StatementType{storepb.StatementType_UPDATE, storepb.StatementType_ALTER_TABLE},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmts, err := parserbase.ParseStatements(storepb.Engine_OCEANBASE, tt.statement)
			require.NoError(t, err)

			got, err := SummaryStatementTypes(storepb.Engine_OCEANBASE, parserbase.ExtractASTs(stmts))
			require.NoError(t, err)
			require.ElementsMatch(t, tt.want, dedupeStatementTypes(got))
		})
	}
}

// An engine without a registered classifier must error rather than yield an
// empty list, which would silently drop every statement.sql_type rule.
func TestSummaryStatementTypesRejectsEngineWithoutClassifier(t *testing.T) {
	_, err := SummaryStatementTypes(storepb.Engine_SNOWFLAKE, nil)
	require.Error(t, err)
}
