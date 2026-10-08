package plancheck

import (
	"context"
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func TestCalculateAffectedRows(t *testing.T) {
	type estimate struct {
		statement string
		rows      int64
		err       error
	}
	repeat := func(count int, format string, rows int64) []estimate {
		var estimates []estimate
		for i := range count {
			estimates = append(estimates, estimate{fmt.Sprintf(format, i+1), rows, nil})
		}
		return estimates
	}
	for _, tc := range []struct {
		name         string
		statements   []estimate
		dmlCount     int
		insertCount  int
		wantRows     int64
		wantWarning  string
		wantExplains int
	}{
		{
			name:         "every_dml_estimated_sums_exactly",
			statements:   []estimate{{"u1", 10, nil}, {"u2", 17, nil}, {"u3", 34, nil}},
			dmlCount:     3,
			wantRows:     61,
			wantExplains: 3,
		},
		{
			name:         "dml_without_text_counts_as_the_average_estimate",
			statements:   []estimate{{"u1", 10, nil}, {"u2", 30, nil}},
			dmlCount:     5,
			wantRows:     100,
			wantWarning:  "Affected rows could not be estimated for 3 of 5 DML statements.",
			wantExplains: 2,
		},
		{
			// The ten one-row updates share a shape, so the delete after them is still estimated.
			name: "samples_cover_every_shape",
			statements: append(repeat(10, "UPDATE t SET v = 'y' WHERE id = %d;", 1),
				estimate{"DELETE FROM big WHERE s_id <= 50;", 5000, nil}),
			dmlCount:     11,
			wantRows:     5010,
			wantExplains: 10,
		},
		{
			// Five statements of each shape are estimated: 2 * 15 + 100 * 5.
			name: "shape_average_counts_for_its_unsampled_statements",
			statements: append(repeat(15, "UPDATE t SET c = c + 1 WHERE id = %d;", 2),
				repeat(5, "DELETE FROM big WHERE s_id = %d;", 100)...),
			dmlCount:     20,
			wantRows:     530,
			wantExplains: 10,
		},
		{
			// The first ten shapes are estimated at 10 rows each; the other two count as that average.
			name:         "shapes_beyond_the_samples_count_as_the_average_estimate",
			statements:   append(repeat(10, "UPDATE t%d SET c = 1;", 10), estimate{"DELETE FROM a;", 1000, nil}, estimate{"DELETE FROM b;", 1000, nil}),
			dmlCount:     12,
			wantRows:     120,
			wantWarning:  "Affected rows could not be estimated for 2 of 12 DML statements.",
			wantExplains: 10,
		},
		{
			name:         "failed_sample_warns",
			statements:   []estimate{{"u1", 3, nil}, {"d1", 0, errors.New("table t_new does not exist")}},
			dmlCount:     2,
			insertCount:  4,
			wantRows:     10,
			wantWarning:  "Affected rows could not be estimated for 1 of 2 DML statements: table t_new does not exist",
			wantExplains: 2,
		},
		{
			// The shape's other statement is estimated, so the failed one counts as that estimate.
			name:         "failed_sample_of_an_estimated_shape_warns",
			statements:   []estimate{{"UPDATE t SET c = 1;", 0, errors.New("deadlock detected")}, {"UPDATE t SET c = 2;", 5, nil}},
			dmlCount:     2,
			wantRows:     10,
			wantWarning:  "Affected rows could not be estimated for 1 of 2 DML statements: deadlock detected",
			wantExplains: 2,
		},
		{
			// One sampled shape fails and two shapes are never sampled; all three count as the average of the other nine.
			name:         "failures_and_unsampled_shapes_warn_together",
			statements:   append(append([]estimate{{"DELETE FROM missing;", 0, errors.New("table missing does not exist")}}, repeat(9, "UPDATE t%d SET c = 1;", 10)...), estimate{"DELETE FROM a;", 1000, nil}, estimate{"DELETE FROM b;", 1000, nil}),
			dmlCount:     12,
			wantRows:     120,
			wantWarning:  "Affected rows could not be estimated for 3 of 12 DML statements: table missing does not exist",
			wantExplains: 10,
		},
		{
			name:         "every_sample_failed",
			statements:   []estimate{{"d1", 0, errors.New("syntax error")}},
			dmlCount:     1,
			insertCount:  4,
			wantRows:     4,
			wantWarning:  "Affected rows could not be estimated for 1 of 1 DML statements: syntax error",
			wantExplains: 1,
		},
		{
			name:        "dml_without_text",
			dmlCount:    2,
			wantWarning: "Affected rows could not be estimated for 2 of 2 DML statements.",
		},
		{
			name:         "saturates_instead_of_overflowing",
			statements:   []estimate{{"u1", math.MaxInt64/2 + 1, nil}, {"u2", math.MaxInt64/2 + 1, nil}},
			dmlCount:     3,
			wantRows:     math.MaxInt64,
			wantWarning:  "Affected rows could not be estimated for 1 of 3 DML statements.",
			wantExplains: 2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changeSummary := &parserbase.ChangeSummary{
				ChangedResources: model.NewChangedResources(nil /* dbMetadata */),
				DMLCount:         tc.dmlCount,
				InsertCount:      tc.insertCount,
			}
			estimates := map[string]estimate{}
			for _, e := range tc.statements {
				changeSummary.DMLStatements = append(changeSummary.DMLStatements, e.statement)
				estimates[e.statement] = e
			}
			explains := 0
			explain := func(_ context.Context, statement string) (int64, error) {
				explains++
				e := estimates[statement]
				return e.rows, e.err
			}

			rows, warning := calculateAffectedRows(context.Background(), storepb.Engine_POSTGRES, changeSummary, explain)
			require.Equal(t, tc.wantRows, rows)
			require.Equal(t, tc.wantWarning, warning)
			require.Equal(t, tc.wantExplains, explains)
		})
	}
}

func TestShapeKey(t *testing.T) {
	for _, tc := range []struct {
		name        string
		statements  []string
		mysqlFamily bool
		want        string
	}{
		{
			name: "literals and spacing",
			statements: []string{
				"UPDATE t SET v = 'it''s' WHERE id = 1;",
				"UPDATE t SET v='x' WHERE id=20.5;",
				"UPDATE  t\n\tSET v = 'C:\\'  WHERE id = 3;",
			},
			want: "UPDATE t SET v=? WHERE id=?;",
		},
		{
			name: "MySQL family backslash escapes",
			statements: []string{
				"UPDATE t SET v = 'it\\'s' WHERE id = 1;",
				"UPDATE t SET v = 'C:\\\\' WHERE id = 2;",
			},
			mysqlFamily: true,
			want:        "UPDATE t SET v=? WHERE id=?;",
		},
		{
			name: "PostgreSQL escape strings",
			statements: []string{
				"UPDATE t SET v = E'it\\'s' WHERE id = 1;",
				"UPDATE t SET v = E'x' WHERE id = 2;",
			},
			want: "UPDATE t SET v=E? WHERE id=?;",
		},
		{
			name: "CockroachDB byte strings",
			statements: []string{
				"UPDATE t SET v = b'it\\'s' WHERE id = 1;",
				"UPDATE t SET v = b'x' WHERE id = 2;",
			},
			want: "UPDATE t SET v=b? WHERE id=?;",
		},
		{
			name: "ordinary comments",
			statements: []string{
				"-- row 1\nUPDATE t SET v = 1 WHERE id = 1;",
				"/* row 2 */ UPDATE t SET v = 2 /* it's */ WHERE id = 2; -- done",
				"UPDATE/**/t SET v = 3 WHERE id = 3;",
			},
			want: "UPDATE t SET v=? WHERE id=?;",
		},
		{
			name: "MySQL family # comments",
			statements: []string{
				"# row 1\nUPDATE t SET v = 1 WHERE id = 1;",
				"UPDATE t SET v = 2 WHERE id = 2; # row 2",
			},
			mysqlFamily: true,
			want:        "UPDATE t SET v=? WHERE id=?;",
		},
		{
			name: "MySQL family -- without whitespace is not a comment",
			statements: []string{
				"UPDATE t SET c = c--1 WHERE id = 1;",
				"UPDATE t SET c = c--2 WHERE id = 20;",
			},
			mysqlFamily: true,
			want:        "UPDATE t SET c=c--? WHERE id=?;",
		},
		{
			name: "lists of literals",
			statements: []string{
				"DELETE FROM t WHERE id IN (1, 2, 3)",
				"DELETE FROM t WHERE id IN ('a', 'b', 'c')",
			},
			want: "DELETE FROM t WHERE id IN(?,?,?)",
		},
		{
			name:       "numbers",
			statements: []string{"UPDATE t1 SET c2 = 0x1F WHERE t1.c3 > 1e3", "UPDATE t1 SET c2 = 7 WHERE t1.c3 > 1.5e-3", "UPDATE t1 SET c2 = 0B1010 WHERE t1.c3 > 0o17", "UPDATE t1 SET c2 = 0xFF_FF WHERE t1.c3 > 1_000.5"},
			want:       "UPDATE t1 SET c2=? WHERE t1.c3>?",
		},
		{
			name:       "leading-dot decimals",
			statements: []string{"UPDATE t SET v = .1 WHERE id > .5", "UPDATE t SET v = .10 WHERE id > .75e2"},
			want:       "UPDATE t SET v=.? WHERE id>.?",
		},
		{
			name:        "MySQL family numbers",
			statements:  []string{"UPDATE t1 SET c2 = 0x1F WHERE t1.c3 > 1e3", "UPDATE t1 SET c2 = 0b101 WHERE t1.c3 > 7"},
			mysqlFamily: true,
			want:        "UPDATE t1 SET c2=? WHERE t1.c3>?",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, statement := range tc.statements {
				require.Equal(t, tc.want, shapeKey(statement, tc.mysqlFamily), statement)
			}
		})
	}

	t.Run("statements that differ beyond literals keep separate shapes", func(t *testing.T) {
		for _, pair := range [][2]string{
			{`UPDATE "Orders" SET v = 1`, `UPDATE "orders" SET v = 1`},
			{`UPDATE "2024_orders" SET v = 1`, `UPDATE "2025_orders" SET v = 1`},
			{"UPDATE `Orders` SET v = 1", "UPDATE `orders` SET v = 1"},
			{"UPDATE Orders SET v = 1", "UPDATE orders SET v = 1"},
			{"UPDATE [2024_orders] SET v = 1", "UPDATE [2025_orders] SET v = 1"},
			{"UPDATE [1] SET v = 1", "UPDATE [2] SET v = 1"},
			{"UPDATE [t]]1] SET v = 1", "UPDATE [t]]2] SET v = 1"},
			{"UPDATE 2024_orders SET v = 1", "UPDATE 2025_orders SET v = 1"},
			{"DELETE FROM t WHERE id IN (1)", "DELETE FROM t WHERE id IN (1, 2)"},
			{"UPDATE t SET v = 1 /*!80000 WHERE id = 1 */", "UPDATE t SET v = 1"},
			{"UPDATE /*+ CARDINALITY(t 1) */ t SET v = 1", "UPDATE /*+ CARDINALITY(t 1000000) */ t SET v = 1"},
			{"UPDATE t SET v = 1 /*!80000 WHERE id = 1 */", "UPDATE t SET v = 1 /*!99999 WHERE id = 1 */"},
			{"UPDATE t SET v = 1 /*M!100100 WHERE id = 1 */", "UPDATE t SET v = 1 /*M!999999 WHERE id = 1 */"},
			{"UPDATE t SET v = 1 /*T![feature] WHERE id = 1 */", "UPDATE t SET v = 1"},
			{"UPDATE --+ INDEX(t i1)\nt SET v = 1", "UPDATE --+ INDEX(t i2)\nt SET v = 1"},
			{"UPDATE /*+ CARDINALITY(t 1000000) */ t SET v = 1", "UPDATE t SET v = 1"},
			{"UPDATE [t--x] SET v = 1 WHERE id = 1", "UPDATE [t--x] SET v = 1"},
			{"UPDATE t SET v = $$--$$ WHERE id = 1", "UPDATE t SET v = $$--$$"},
			{"UPDATE t SET v = 1 -- don't\nWHERE id = 1", "UPDATE t SET v = 1 -- don't\n"},
			{"UPDATE t SET v = v - -1 WHERE flag", "UPDATE t SET v = v --? WHERE flag"},
			{"UPDATE t SET v = 1 -- x\nWHERE id = 1", "UPDATE t SET v = 1 -- x WHERE id=?"},
			{"UPDATE t SET note = q'{it's}' WHERE id = 1 -- don't", "UPDATE t SET note = q'{it's}' WHERE id > 0 -- don't"},
			{"UPDATE t SET v = 1 /* a /* b */ it's */ WHERE id = 1 -- don't", "UPDATE t SET v = 1 /* a /* b */ it's */ WHERE id > 0 -- don't"},
			{"UPDATE t SET a = ARRAY[']'] WHERE id = 1 AND note <> 'a--b'", "UPDATE t SET a = ARRAY[']'] WHERE true AND note <> 'a--b'"},
			// The quote never closes, so each statement keeps its own shape.
			{"UPDATE t SET v = $$it's$$ WHERE id = 1", "UPDATE t SET v = $$it's$$"},
		} {
			require.NotEqual(t, shapeKey(pair[0], false), shapeKey(pair[1], false), pair[0])
			require.NotEqual(t, shapeKey(pair[0], true), shapeKey(pair[1], true), pair[0])
		}
		// On MySQL, a word such as 0X1F, 0o17, or 1_000 is an identifier.
		for _, pair := range [][2]string{{"UPDATE t SET v = 0X1F", "UPDATE t SET v = 0X20"}, {"UPDATE t SET v = 0o17", "UPDATE t SET v = 0o20"}, {"UPDATE t SET v = 1_000", "UPDATE t SET v = 1_001"}} {
			require.NotEqual(t, shapeKey(pair[0], true), shapeKey(pair[1], true), pair[0])
		}
		// -- starts a comment on PostgreSQL, but on MySQL only before whitespace.
		require.Equal(t, shapeKey("UPDATE t SET v = v--1 WHERE id = 1", false), shapeKey("UPDATE t SET v = v", false))
		require.NotEqual(t, shapeKey("UPDATE t SET v = v--1 WHERE id = 1", true), shapeKey("UPDATE t SET v = v", true))
		require.NotEqual(t, shapeKey("UPDATE t SET v = 'it\\'s' WHERE id = 1", true), shapeKey("UPDATE t SET v = 'it\\'s'", true))
		require.NotEqual(t, shapeKey(`UPDATE t SET v = "a\"b" WHERE id = 1`, true), shapeKey(`UPDATE t SET v = "a\"b"`, true))
		require.NotEqual(t, shapeKey("UPDATE t SET v = 1 # x\nWHERE id = 1", true), shapeKey("UPDATE t SET v = 1 # x WHERE id=?", true))
		require.NotEqual(t, shapeKey("UPDATE t SET note = $$it's$$ WHERE id = 1 -- don't", false), shapeKey("UPDATE t SET note = $$it's$$ WHERE id > 0 -- don't", false))
		require.NotEqual(t, shapeKey("UPDATE t SET note = $x$it's$x$ WHERE id = 1 -- don't", false), shapeKey("UPDATE t SET note = $x$it's$x$ WHERE id > 0 -- don't", false))
	})
}

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
