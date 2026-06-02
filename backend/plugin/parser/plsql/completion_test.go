package plsql

import (
	"context"
	"io"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

type candidatesTest struct {
	Input string
	Want  []base.Candidate
}

func TestCompletion(t *testing.T) {
	tests := []candidatesTest{}

	const (
		record = false
	)
	var (
		filepath = "test-data/test_completion.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, t := range tests {
		text, caretLine, caretOffset := catchCaret(t.Input)
		result, err := base.Completion(context.Background(), storepb.Engine_ORACLE, base.CompletionContext{
			Scene:             base.SceneTypeAll,
			DefaultDatabase:   "SCHEMA1",
			Metadata:          getMetadataForTest,
			ListDatabaseNames: listDatabaseNamesForTest,
		}, text, caretLine, caretOffset)
		a.NoError(err)
		var filteredResult []base.Candidate
		for _, r := range result {
			switch r.Type {
			case base.CandidateTypeKeyword, base.CandidateTypeFunction:
				continue
			default:
				filteredResult = append(filteredResult, r)
			}
		}
		slices.SortFunc(filteredResult, func(a, b base.Candidate) int {
			if a.Type != b.Type {
				if a.Type < b.Type {
					return -1
				}
				return 1
			}
			if a.Text != b.Text {
				if a.Text < b.Text {
					return -1
				}
				return 1
			}
			if a.Definition < b.Definition {
				return -1
			}
			if a.Definition > b.Definition {
				return 1
			}
			return 0
		})

		if record {
			tests[i].Want = filteredResult
		} else {
			a.Equal(t.Want, filteredResult, t.Input)
		}
	}

	if record {
		yamltest.Record(t, filepath, tests)
	}
}

func TestCompletionDoesNotDependOnANTLR(t *testing.T) {
	content, err := os.ReadFile("completion.go")
	require.NoError(t, err)
	source := string(content)
	require.NotContains(t, source, "github.com/antlr4-go/antlr/v4")
	require.NotContains(t, source, "github.com/bytebase/parser/plsql")
	require.NotContains(t, source, "CodeCompletionCore")
}

func TestCompletionByteOffsetUsesUTF16Columns(t *testing.T) {
	input := "SELECT '😀' FROM t1 WHERE | AND c1 = 1"
	statement, caretLine, caretOffset := catchCaretUTF16(input)

	_, byteOffset := computeSQLAndByteOffset(statement, caretLine, caretOffset, false /* tricky */)

	require.Equal(t, strings.Index(input, "|"), byteOffset)
}

func TestCompletionLongTailTableSources(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantColumns []string
	}{
		{
			name:        "json table explicit columns",
			input:       "SELECT jt.| FROM t1, JSON_TABLE(t1.c1, '$.rows[*]' COLUMNS (id NUMBER PATH '$.id', name VARCHAR2(100) PATH '$.name')) jt",
			wantColumns: []string{"ID", "NAME"},
		},
		{
			name:        "xmltable explicit columns",
			input:       "SELECT xt.| FROM t1, XMLTABLE('/root/row' PASSING t1.c1 COLUMNS id NUMBER PATH 'id', name VARCHAR2(100) PATH 'name') xt",
			wantColumns: []string{"ID", "NAME"},
		},
		{
			name:        "inline external declared columns",
			input:       "SELECT ext.| FROM EXTERNAL ((a NUMBER, b VARCHAR2(10)) TYPE ORACLE_LOADER DEFAULT DIRECTORY data_dir LOCATION ('x.csv')) ext",
			wantColumns: []string{"A", "B"},
		},
		{
			name:        "lateral inline view columns",
			input:       "SELECT l.| FROM t1, LATERAL (SELECT c1 FROM t2 WHERE t2.c1 = t1.c1) l",
			wantColumns: []string{"C1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, caretLine, caretOffset := catchCaret(tt.input)
			result, err := base.Completion(context.Background(), storepb.Engine_ORACLE, base.CompletionContext{
				Scene:             base.SceneTypeAll,
				DefaultDatabase:   "SCHEMA1",
				Metadata:          getMetadataForTest,
				ListDatabaseNames: listDatabaseNamesForTest,
			}, text, caretLine, caretOffset)
			require.NoError(t, err)
			require.Equal(t, tt.wantColumns, completionColumnTexts(result))
		})
	}
}

func TestCompletionCoverageMatrix(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []completionCandidateSpec
		notWant []completionCandidateSpec
	}{
		{
			name:  "select list uses table columns",
			input: "SELECT | FROM t1",
			want:  columns("C1"),
		},
		{
			name:  "select list uses alias columns",
			input: "SELECT a.| FROM t1 a",
			want:  columns("C1"),
		},
		{
			name:  "select list uses schema qualified table columns",
			input: "SELECT | FROM schema1.t2",
			want:  columns("C1", "C2"),
		},
		{
			name:  "column prefix keeps matching column available",
			input: "SELECT C2| FROM t2",
			want:  columns("C2"),
		},
		{
			name:  "qualified column prefix keeps matching column available",
			input: "SELECT t2.C2| FROM t2",
			want:  columns("C2"),
		},
		{
			name:    "unknown qualifier offers no columns",
			input:   "SELECT missing.| FROM t1",
			notWant: columns("C1", "C2"),
		},
		{
			name:  "join on left alias columns",
			input: "SELECT * FROM t1 a JOIN t2 b ON a.|",
			want:  columns("C1"),
		},
		{
			name:  "join on right alias columns",
			input: "SELECT * FROM t1 a JOIN t2 b ON b.|",
			want:  columns("C1", "C2"),
		},
		{
			name:  "join where sees both table columns",
			input: "SELECT * FROM t1 a JOIN t2 b ON a.c1 = b.c1 WHERE |",
			want:  columns("C1", "C2"),
		},
		{
			name:    "left alias excludes right only column",
			input:   "SELECT a.| FROM t1 a JOIN t2 b ON a.c1 = b.c1",
			want:    columns("C1"),
			notWant: columns("C2"),
		},
		{
			name:  "join table reference offers tables",
			input: "SELECT * FROM t1 JOIN |",
			want:  tables("T1", "T2"),
		},
		{
			name:  "using clause offers joined columns",
			input: "SELECT * FROM t1 JOIN t2 USING (|)",
			want:  columns("C1", "C2"),
		},
		{
			name:  "cte explicit columns are available",
			input: "WITH x(x1, x2) AS (SELECT * FROM t2) SELECT x.| FROM x",
			want:  columns("X1", "X2"),
		},
		{
			name:  "cte star body expands through query span",
			input: "WITH x AS (SELECT * FROM t2) SELECT x.| FROM x",
			want:  columns("C1", "C2"),
		},
		{
			name:  "cte table candidate is available without schema qualifier",
			input: "WITH x AS (SELECT * FROM t2) SELECT * FROM |",
			want:  tables("X", "T1", "T2"),
		},
		{
			name:    "cte table candidate is excluded from schema qualified table reference",
			input:   "WITH x AS (SELECT * FROM t2) SELECT * FROM schema1.|",
			want:    tables("T1", "T2"),
			notWant: tables("X"),
		},
		{
			name:  "multiple ctes expose selected cte columns",
			input: "WITH a AS (SELECT c1 FROM t1), b AS (SELECT c1, c2 FROM t2) SELECT b.| FROM a JOIN b ON a.c1 = b.c1",
			want:  columns("C1", "C2"),
		},
		{
			name:  "derived table explicit projection columns",
			input: "SELECT x.| FROM (SELECT c1 FROM t1) x",
			want:  columns("C1"),
		},
		{
			name:  "derived table star body expands through query span",
			input: "SELECT x.| FROM (SELECT * FROM t2) x",
			want:  columns("C1", "C2"),
		},
		{
			name:    "unqualified select from derived table exposes derived columns",
			input:   "SELECT | FROM (SELECT c1 FROM t2) x",
			want:    columns("C1"),
			notWant: columns("C2"),
		},
		{
			name:  "lateral alias columns",
			input: "SELECT l.| FROM t1, LATERAL (SELECT c1 FROM t2 WHERE t2.c1 = t1.c1) l",
			want:  columns("C1"),
		},
		{
			name:    "lateral body sees left outer reference",
			input:   "SELECT * FROM t1, LATERAL (SELECT t1.| FROM t2) l",
			want:    columns("C1"),
			notWant: columns("C2"),
		},
		{
			name:  "json table qualified columns",
			input: "SELECT jt.| FROM t1, JSON_TABLE(t1.c1, '$.rows[*]' COLUMNS (id NUMBER PATH '$.id', name VARCHAR2(100) PATH '$.name')) jt",
			want:  columns("ID", "NAME"),
		},
		{
			name:  "json table unqualified columns",
			input: "SELECT | FROM t1, JSON_TABLE(t1.c1, '$.rows[*]' COLUMNS (id NUMBER PATH '$.id', name VARCHAR2(100) PATH '$.name')) jt",
			want:  columns("C1", "ID", "NAME"),
		},
		{
			name:  "xmltable qualified columns",
			input: "SELECT xt.| FROM t1, XMLTABLE('/root/row' PASSING t1.c1 COLUMNS id NUMBER PATH 'id', name VARCHAR2(100) PATH 'name') xt",
			want:  columns("ID", "NAME"),
		},
		{
			name:  "inline external qualified columns",
			input: "SELECT ext.| FROM EXTERNAL ((a NUMBER, b VARCHAR2(10)) TYPE ORACLE_LOADER DEFAULT DIRECTORY data_dir LOCATION ('x.csv')) ext",
			want:  columns("A", "B"),
		},
		{
			name:  "containers alias resolves base object columns",
			input: "SELECT c.| FROM CONTAINERS(schema1.t2) c",
			want:  columns("C1", "C2"),
		},
		{
			name:    "table function alias does not invent columns",
			input:   "SELECT x.| FROM TABLE(pkg.values()) x",
			notWant: columns("C1", "C2"),
		},
		{
			name:  "insert target table reference",
			input: "INSERT INTO |",
			want:  tables("T1", "T2"),
		},
		{
			name:  "insert column list uses target table columns",
			input: "INSERT INTO t2 (|) VALUES (1)",
			want:  columns("C1", "C2"),
		},
		{
			name:  "insert select uses source table columns",
			input: "INSERT INTO t1 SELECT | FROM t2",
			want:  columns("C1", "C2"),
		},
		{
			name:  "update target table reference",
			input: "UPDATE | SET c1 = 1",
			want:  tables("T1", "T2"),
		},
		{
			name:  "update set uses target table columns",
			input: "UPDATE t2 SET |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "update where uses target table columns",
			input: "UPDATE t2 SET c1 = 1 WHERE |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "delete target table reference",
			input: "DELETE FROM |",
			want:  tables("T1", "T2"),
		},
		{
			name:  "delete where uses target table columns",
			input: "DELETE FROM t2 WHERE |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "merge target table reference",
			input: "MERGE INTO |",
			want:  tables("T1", "T2"),
		},
		{
			name:    "merge source table reference offers real tables",
			input:   "MERGE INTO t1 target USING |",
			want:    tables("T1", "T2"),
			notWant: tables("TARGET"),
		},
		{
			name:  "merge on uses target and source columns",
			input: "MERGE INTO t1 target USING t2 source ON |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "schema qualified table reference offers schema tables",
			input: "SELECT * FROM schema1.|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "table prefix keeps matching table available",
			input: "SELECT * FROM t|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "quoted identifier context preserves candidate text",
			input: `SELECT "t2".| FROM t2`,
			want:  columns("C1", "C2"),
		},
		{
			name:  "select list uses t2 columns",
			input: "SELECT | FROM t2",
			want:  columns("C1", "C2"),
		},
		{
			name:  "where clause uses table columns",
			input: "SELECT * FROM t2 WHERE |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "order by clause uses table columns",
			input: "SELECT * FROM t2 ORDER BY |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "group by clause uses table columns",
			input: "SELECT * FROM t2 GROUP BY |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "having clause uses grouped table columns",
			input: "SELECT c1, COUNT(*) FROM t2 GROUP BY c1 HAVING |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "distinct select list uses table columns",
			input: "SELECT DISTINCT | FROM t2",
			want:  columns("C1", "C2"),
		},
		{
			name:  "table name qualifier uses table columns",
			input: "SELECT t2.| FROM t2",
			want:  columns("C1", "C2"),
		},
		{
			name:  "comparison expression uses table columns",
			input: "SELECT * FROM t2 WHERE c1 = |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "arithmetic expression uses table columns",
			input: "SELECT * FROM t2 WHERE c1 + |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "case expression uses table columns",
			input: "SELECT CASE WHEN | THEN c1 ELSE c2 END FROM t2",
			want:  columns("C1", "C2"),
		},
		{
			name:  "function argument uses table columns",
			input: "SELECT NVL(|, 0) FROM t2",
			want:  columns("C1", "C2"),
		},
		{
			name:  "between lower bound uses table columns",
			input: "SELECT * FROM t2 WHERE c1 BETWEEN | AND 10",
			want:  columns("C1", "C2"),
		},
		{
			name:  "between upper bound uses table columns",
			input: "SELECT * FROM t2 WHERE c1 BETWEEN 1 AND |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "like expression uses table columns",
			input: "SELECT * FROM t2 WHERE c1 LIKE |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "is null predicate continuation uses table columns",
			input: "SELECT * FROM t2 WHERE c1 IS NULL OR |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "select alias is available in order by",
			input: "SELECT c1 AS c_alias FROM t1 ORDER BY |",
			want:  columns("C1", "C_ALIAS"),
		},
		{
			name:  "select alias survives nested select in order by",
			input: "SELECT c1 AS c_alias FROM t1 WHERE EXISTS (SELECT c1 FROM t2) ORDER BY |",
			want:  columns("C1", "C_ALIAS"),
		},
		{
			name:    "select alias is not available in where",
			input:   "SELECT c1 AS c_alias FROM t1 WHERE |",
			want:    columns("C1"),
			notWant: columns("C_ALIAS"),
		},
		{
			name:  "select alias is available in group by",
			input: "SELECT c1 AS c_alias FROM t1 GROUP BY |",
			want:  columns("C1", "C_ALIAS"),
		},
		{
			name:  "comma join select list sees both tables",
			input: "SELECT | FROM t1 a, t2 b",
			want:  columns("C1", "C2"),
		},
		{
			name:  "comma join left alias columns",
			input: "SELECT a.| FROM t1 a, t2 b",
			want:  columns("C1"),
		},
		{
			name:  "comma join right alias columns",
			input: "SELECT b.| FROM t1 a, t2 b",
			want:  columns("C1", "C2"),
		},
		{
			name:  "left join select list sees both sides",
			input: "SELECT | FROM t1 LEFT JOIN t2 ON t1.c1 = t2.c1",
			want:  columns("C1", "C2"),
		},
		{
			name:  "right join select list sees both sides",
			input: "SELECT | FROM t1 RIGHT JOIN t2 ON t1.c1 = t2.c1",
			want:  columns("C1", "C2"),
		},
		{
			name:  "full outer join select list sees both sides",
			input: "SELECT | FROM t1 FULL OUTER JOIN t2 ON t1.c1 = t2.c1",
			want:  columns("C1", "C2"),
		},
		{
			name:  "cross join select list sees both sides",
			input: "SELECT | FROM t1 CROSS JOIN t2",
			want:  columns("C1", "C2"),
		},
		{
			name:  "natural join select list sees both sides",
			input: "SELECT | FROM t1 NATURAL JOIN t2",
			want:  columns("C1", "C2"),
		},
		{
			name:  "multi join final alias columns",
			input: "SELECT c.| FROM t1 a JOIN t2 b ON a.c1 = b.c1 JOIN t2 c ON b.c1 = c.c1",
			want:  columns("C1", "C2"),
		},
		{
			name:  "multi join where sees all aliases",
			input: "SELECT * FROM t1 a JOIN t2 b ON a.c1 = b.c1 JOIN t2 c ON b.c1 = c.c1 WHERE |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "schema qualified join alias columns",
			input: "SELECT s.| FROM t1 a JOIN schema2.t2 s ON a.c1 = s.c1",
			want:  columns("C1", "C2"),
		},
		{
			name:    "schema qualified alias excludes same table from other schema",
			input:   "SELECT a.| FROM schema1.t1 a JOIN schema4.t1 b ON a.c1 = b.c1",
			want:    columns("C1"),
			notWant: columns("SCHEMA4_ONLY"),
		},
		{
			name:  "join table prefix keeps matching tables",
			input: "SELECT * FROM t1 JOIN t|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "left join table reference offers tables",
			input: "SELECT * FROM t1 LEFT JOIN |",
			want:  tables("T1", "T2"),
		},
		{
			name:  "right join table reference offers tables",
			input: "SELECT * FROM t1 RIGHT JOIN |",
			want:  tables("T1", "T2"),
		},
		{
			name:  "cross join table reference offers tables",
			input: "SELECT * FROM t1 CROSS JOIN |",
			want:  tables("T1", "T2"),
		},
		{
			name:  "subquery select list uses inner table columns",
			input: "SELECT * FROM t1 WHERE EXISTS (SELECT | FROM t2)",
			want:  columns("C1", "C2"),
		},
		{
			name:  "subquery where uses inner table columns",
			input: "SELECT * FROM t1 WHERE EXISTS (SELECT * FROM t2 WHERE |)",
			want:  columns("C1", "C2"),
		},
		{
			name:  "correlated subquery can qualify outer table",
			input: "SELECT * FROM t1 outer_t WHERE EXISTS (SELECT * FROM t2 WHERE outer_t.|)",
			want:  columns("C1"),
		},
		{
			name:  "correlated subquery can qualify inner table",
			input: "SELECT * FROM t1 outer_t WHERE EXISTS (SELECT * FROM t2 inner_t WHERE inner_t.|)",
			want:  columns("C1", "C2"),
		},
		{
			name:  "scalar subquery select list uses inner table columns",
			input: "SELECT (SELECT | FROM t2) FROM t1",
			want:  columns("C1", "C2"),
		},
		{
			name:  "in subquery select list uses inner table columns",
			input: "SELECT * FROM t1 WHERE c1 IN (SELECT | FROM t2)",
			want:  columns("C1", "C2"),
		},
		{
			name:  "nested derived table alias columns",
			input: "SELECT y.| FROM (SELECT x.c1 FROM (SELECT c1 FROM t1) x) y",
			want:  columns("C1"),
		},
		{
			name:  "derived table with column alias",
			input: "SELECT x.| FROM (SELECT c1 AS renamed FROM t1) x",
			want:  columns("RENAMED"),
		},
		{
			name:  "derived table join exposes derived alias columns",
			input: "SELECT x.| FROM (SELECT c1 FROM t1) x JOIN t2 ON x.c1 = t2.c1",
			want:  columns("C1"),
		},
		{
			name:  "derived table join where sees derived and physical columns",
			input: "SELECT * FROM (SELECT c1 FROM t1) x JOIN t2 ON x.c1 = t2.c1 WHERE |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "cte selected through derived table",
			input: "WITH x AS (SELECT c1 FROM t1) SELECT | FROM (SELECT * FROM x) sub",
			want:  columns("C1"),
		},
		{
			name:  "cte joined with physical table",
			input: "WITH x AS (SELECT c1 FROM t1) SELECT | FROM x JOIN t2 ON x.c1 = t2.c1",
			want:  columns("C1", "C2"),
		},
		{
			name:  "cte alias joined with physical table",
			input: "WITH x AS (SELECT c1 FROM t1) SELECT xa.| FROM x xa JOIN t2 ON xa.c1 = t2.c1",
			want:  columns("C1"),
		},
		{
			name:  "cte in insert select",
			input: "WITH x AS (SELECT * FROM t2) INSERT INTO t1 SELECT | FROM x",
			want:  columns("C1", "C2"),
		},
		{
			name:  "cte in update scalar subquery",
			input: "WITH x AS (SELECT c1, c2 FROM t2) UPDATE t1 SET c1 = (SELECT | FROM x)",
			want:  columns("C1", "C2"),
		},
		{
			name:  "cte in delete subquery",
			input: "WITH x AS (SELECT * FROM t2) DELETE FROM t1 WHERE c1 IN (SELECT | FROM x)",
			want:  columns("C1", "C2"),
		},
		{
			name:  "cte set operation exposes positional columns",
			input: "WITH x AS (SELECT c1 FROM t1 UNION SELECT c1 FROM t2) SELECT x.| FROM x",
			want:  columns("C1"),
		},
		{
			name:  "lateral body uses inner alias columns",
			input: "SELECT * FROM t1, LATERAL (SELECT inner_t.| FROM t2 inner_t WHERE inner_t.c1 = t1.c1) l",
			want:  columns("C1", "C2"),
		},
		{
			name:  "left join lateral alias columns",
			input: "SELECT l.| FROM t1 LEFT JOIN LATERAL (SELECT c1 FROM t2 WHERE t2.c1 = t1.c1) l ON 1 = 1",
			want:  columns("C1"),
		},
		{
			name:  "lateral unqualified select list sees inner columns",
			input: "SELECT * FROM t1, LATERAL (SELECT | FROM t2 WHERE t2.c1 = t1.c1) l",
			want:  columns("C1", "C2"),
		},
		{
			name:  "json table table alias is offered as table candidate",
			input: "SELECT | FROM t1, JSON_TABLE(t1.c1, '$.rows[*]' COLUMNS (id NUMBER PATH '$.id')) jt",
			want:  tables("JT"),
		},
		{
			name:  "xmltable unqualified columns",
			input: "SELECT | FROM t1, XMLTABLE('/root/row' PASSING t1.c1 COLUMNS id NUMBER PATH 'id', name VARCHAR2(100) PATH 'name') xt",
			want:  columns("C1", "ID", "NAME"),
		},
		{
			name:  "inline external unqualified columns",
			input: "SELECT | FROM EXTERNAL ((a NUMBER, b VARCHAR2(10)) TYPE ORACLE_LOADER DEFAULT DIRECTORY data_dir LOCATION ('x.csv')) ext",
			want:  columns("A", "B"),
		},
		{
			name:  "containers without alias resolves base object columns",
			input: "SELECT t2.| FROM CONTAINERS(schema1.t2)",
			want:  columns("C1", "C2"),
		},
		{
			name:    "pivot does not invent transformed columns",
			input:   "SELECT | FROM t2 PIVOT (COUNT(*) FOR c1 IN (1 AS one))",
			notWant: columns("ONE"),
		},
		{
			name:    "unpivot does not invent transformed columns",
			input:   "SELECT | FROM t2 UNPIVOT (val FOR col IN (c1 AS 'C1', c2 AS 'C2'))",
			notWant: columns("VAL", "COL"),
		},
		{
			name:    "model does not invent transformed columns",
			input:   "SELECT | FROM t2 MODEL DIMENSION BY (c1) MEASURES (c2) RULES (c2[1] = 1)",
			notWant: columns("VAL", "COL"),
		},
		{
			name:  "insert schema qualified table reference",
			input: "INSERT INTO schema2.|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "insert schema qualified target columns",
			input: "INSERT INTO schema2.t2 (|) VALUES (1)",
			want:  columns("C1", "C2"),
		},
		{
			name:  "insert values expression uses target columns",
			input: "INSERT INTO t2 VALUES (|)",
			want:  columns("C1", "C2"),
		},
		{
			name:  "insert second column list position uses target columns",
			input: "INSERT INTO t2 (c1, |) VALUES (1, 2)",
			want:  columns("C1", "C2"),
		},
		{
			name:  "insert select with schema qualified source",
			input: "INSERT INTO t1 SELECT | FROM schema2.t2",
			want:  columns("C1", "C2"),
		},
		{
			name:  "update schema qualified target reference",
			input: "UPDATE schema2.|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "update schema qualified target set columns",
			input: "UPDATE schema2.t2 SET |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "update set expression uses target columns",
			input: "UPDATE t2 SET c1 = |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "update where continuation uses target columns",
			input: "UPDATE t2 SET c1 = 1 WHERE c1 > 0 AND |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "delete schema qualified table reference",
			input: "DELETE FROM schema2.|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "delete where continuation uses target columns",
			input: "DELETE FROM t2 WHERE c1 > 0 AND |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "merge target schema qualified table reference",
			input: "MERGE INTO schema2.|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "merge source schema qualified table reference",
			input: "MERGE INTO t1 target USING schema2.|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "merge qualified target columns",
			input: "MERGE INTO t1 target USING t2 source ON target.|",
			want:  columns("C1"),
		},
		{
			name:  "merge qualified source columns",
			input: "MERGE INTO t1 target USING t2 source ON source.|",
			want:  columns("C1", "C2"),
		},
		{
			name:  "merge update set uses target and source columns",
			input: "MERGE INTO t1 target USING t2 source ON target.c1 = source.c1 WHEN MATCHED THEN UPDATE SET target.c1 = |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "merge insert values uses target and source columns",
			input: "MERGE INTO t1 target USING t2 source ON target.c1 = source.c1 WHEN NOT MATCHED THEN INSERT (c1) VALUES (|)",
			want:  columns("C1", "C2"),
		},
		{
			name:  "select from root offers schemas",
			input: "SELECT * FROM |",
			want:  schemas("SCHEMA1", "SCHEMA2", "SCHEMA3"),
		},
		{
			name:  "select from root offers views",
			input: "SELECT * FROM |",
			want:  views("V1"),
		},
		{
			name:  "select from root offers sequences",
			input: "SELECT * FROM |",
			want:  sequences("SEQ1", "USER_ID_SEQ"),
		},
		{
			name:  "schema2 table reference offers schema2 tables",
			input: "SELECT * FROM schema2.|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "schema3 table reference offers schema3 tables",
			input: "SELECT * FROM schema3.|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "schema prefix keeps matching schema available",
			input: "SELECT * FROM schema|",
			want:  schemas("SCHEMA1", "SCHEMA2", "SCHEMA3"),
		},
		{
			name:  "view reference offers view candidate",
			input: "SELECT * FROM v|",
			want:  views("V1"),
		},
		{
			name:  "sequence reference offers sequence candidates",
			input: "DROP SEQUENCE |",
			want:  sequences("SEQ1", "USER_ID_SEQ"),
		},
		{
			name:  "sequence prefix keeps matching sequence",
			input: "DROP SEQUENCE user|",
			want:  sequences("USER_ID_SEQ"),
		},
		{
			name:  "alter table reference offers tables",
			input: "ALTER TABLE |",
			want:  tables("T1", "T2"),
		},
		{
			name:  "alter schema qualified table reference offers tables",
			input: "ALTER TABLE schema1.|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "alter drop column offers table columns",
			input: "ALTER TABLE t2 DROP COLUMN |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "drop table reference offers tables",
			input: "DROP TABLE |",
			want:  tables("T1", "T2"),
		},
		{
			name:  "drop schema qualified table reference offers tables",
			input: "DROP TABLE schema1.|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "drop view reference offers views",
			input: "DROP VIEW |",
			want:  views("V1"),
		},
		{
			name:  "truncate table reference offers tables",
			input: "TRUNCATE TABLE |",
			want:  tables("T1", "T2"),
		},
		{
			name:  "comment on table reference offers tables",
			input: "COMMENT ON TABLE |",
			want:  tables("T1", "T2"),
		},
		{
			name:  "comment on column offers table columns",
			input: "COMMENT ON COLUMN t2.|",
			want:  columns("C1", "C2"),
		},
		{
			name:  "grant on object offers tables",
			input: "GRANT SELECT ON |",
			want:  tables("T1", "T2"),
		},
		{
			name:  "revoke on object offers tables",
			input: "REVOKE SELECT ON |",
			want:  tables("T1", "T2"),
		},
		{
			name:  "create table datatype offers NUMBER",
			input: "CREATE TABLE t (c |)",
			want:  keywords("NUMBER"),
		},
		{
			name:  "create table references offers tables",
			input: "CREATE TABLE child (parent_id NUMBER REFERENCES |)",
			want:  tables("T1", "T2"),
		},
		{
			name:  "multiple statements limits to statement at caret insert",
			input: "SELECT * FROM t1; INSERT INTO |",
			want:  tables("T1", "T2"),
		},
		{
			name:  "multiple statements limits to statement at caret update",
			input: "SELECT * FROM t1; UPDATE | SET c1 = 1",
			want:  tables("T1", "T2"),
		},
		{
			name:  "multiple statements limits to statement at caret delete",
			input: "SELECT * FROM t1; DELETE FROM |",
			want:  tables("T1", "T2"),
		},
		{
			name:  "multiple statements select columns after invalid first statement",
			input: "SELECT FROM broken; SELECT | FROM t2",
			want:  columns("C1", "C2"),
		},
		{
			name:  "table prefix after multiple statements",
			input: "SELECT * FROM t1; SELECT * FROM t|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "quoted table alias columns",
			input: `SELECT "Alias".| FROM t2 "Alias"`,
			want:  columns("C1", "C2"),
		},
		{
			name:  "quoted cte alias columns",
			input: `WITH "X"(x1, x2) AS (SELECT * FROM t2) SELECT "X".| FROM "X"`,
			want:  columns("X1", "X2"),
		},
		{
			name:  "quoted derived alias columns",
			input: `SELECT "Sub".| FROM (SELECT c1 FROM t1) "Sub"`,
			want:  columns("C1"),
		},
		{
			name:  "quoted schema qualified table reference",
			input: `SELECT * FROM "SCHEMA1".|`,
			want:  tables("T1", "T2"),
		},
		{
			name:  "quoted schema qualified columns",
			input: `SELECT | FROM "SCHEMA1"."T2"`,
			want:  columns("C1", "C2"),
		},
		{
			name:  "reserved-like quoted alias columns",
			input: `SELECT "SELECT".| FROM t2 "SELECT"`,
			want:  columns("C1", "C2"),
		},
		{
			name:    "database link table keeps local table columns unavailable when metadata missing",
			input:   "SELECT remote_t.| FROM t2@remote remote_t",
			notWant: columns("C1", "C2"),
		},
		{
			name:    "schema qualified unknown table offers no known columns",
			input:   "SELECT | FROM schema1.missing_table",
			notWant: columns("C1", "C2"),
		},
		{
			name:    "unknown schema table reference offers no local tables",
			input:   "SELECT * FROM missing_schema.|",
			notWant: tables("T1", "T2"),
		},
		{
			name:    "unknown cte qualifier offers no columns",
			input:   "WITH x AS (SELECT c1 FROM t1) SELECT y.| FROM x",
			notWant: columns("C1", "C2"),
		},
		{
			name:    "unknown physical alias offers no columns",
			input:   "SELECT z.| FROM t1 a JOIN t2 b ON a.c1 = b.c1",
			notWant: columns("C1", "C2"),
		},
		{
			name:    "unknown lateral alias offers no columns",
			input:   "SELECT missing.| FROM t1, LATERAL (SELECT c1 FROM t2) l",
			notWant: columns("C1", "C2"),
		},
		{
			name:  "hierarchical start with uses from scope",
			input: "SELECT * FROM t2 START WITH | CONNECT BY PRIOR c1 = c2",
			want:  columns("C1", "C2"),
		},
		{
			name:  "hierarchical connect by uses from scope",
			input: "SELECT * FROM t2 START WITH c1 IS NULL CONNECT BY |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "analytic partition by uses table columns",
			input: "SELECT COUNT(*) OVER (PARTITION BY |) FROM t2",
			want:  columns("C1", "C2"),
		},
		{
			name:  "analytic order by uses table columns",
			input: "SELECT COUNT(*) OVER (ORDER BY |) FROM t2",
			want:  columns("C1", "C2"),
		},
		{
			name:  "for update select list keeps table columns",
			input: "SELECT | FROM t2 FOR UPDATE",
			want:  columns("C1", "C2"),
		},
		{
			name:  "subquery alias table candidate is offered",
			input: "SELECT | FROM (SELECT c1 FROM t1) sub1",
			want:  tables("SUB1"),
		},
		{
			name:  "physical alias table candidate is offered",
			input: "SELECT | FROM t1 x",
			want:  tables("X"),
		},
		{
			name:  "schema2 table prefix keeps matching tables",
			input: "SELECT * FROM schema2.t|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "quoted schema2 table prefix keeps matching tables",
			input: `SELECT * FROM "SCHEMA2".t|`,
			want:  tables("T1", "T2"),
		},
		{
			name:  "schema3 t1 select list uses columns",
			input: "SELECT | FROM schema3.t1",
			want:  columns("C1"),
		},
		{
			name:  "schema3 alias select list uses columns",
			input: "SELECT s.| FROM schema3.t2 s",
			want:  columns("C1", "C2"),
		},
		{
			name:  "schema3 alias where uses columns",
			input: "SELECT * FROM schema3.t2 s WHERE s.|",
			want:  columns("C1", "C2"),
		},
		{
			name:  "schema3 alias order by uses columns",
			input: "SELECT * FROM schema3.t2 s ORDER BY s.|",
			want:  columns("C1", "C2"),
		},
		{
			name:  "join schema3 table alias columns",
			input: "SELECT s.| FROM t1 a JOIN schema3.t2 s ON a.c1 = s.c1",
			want:  columns("C1", "C2"),
		},
		{
			name:  "insert schema3 qualified target columns",
			input: "INSERT INTO schema3.t2 (|) VALUES (1)",
			want:  columns("C1", "C2"),
		},
		{
			name:  "update schema3 qualified target set columns",
			input: "UPDATE schema3.t2 SET |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "delete schema3 qualified target where columns",
			input: "DELETE FROM schema3.t2 WHERE |",
			want:  columns("C1", "C2"),
		},
		{
			name:  "merge schema3 target qualified columns",
			input: "MERGE INTO schema3.t1 target USING schema2.t2 source ON source.|",
			want:  columns("C1", "C2"),
		},
		{
			name:  "cte table prefix keeps matching cte",
			input: "WITH x AS (SELECT * FROM t2) SELECT * FROM x|",
			want:  tables("X"),
		},
		{
			name:  "cte explicit column prefix keeps matching column",
			input: "WITH x(x1, x2) AS (SELECT * FROM t2) SELECT x.X1| FROM x",
			want:  columns("X1"),
		},
		{
			name:  "derived table column prefix keeps matching column",
			input: "SELECT x.C1| FROM (SELECT c1 FROM t1) x",
			want:  columns("C1"),
		},
		{
			name:  "json table column prefix keeps matching column",
			input: "SELECT jt.I| FROM t1, JSON_TABLE(t1.c1, '$.rows[*]' COLUMNS (id NUMBER PATH '$.id', name VARCHAR2(100) PATH '$.name')) jt",
			want:  columns("ID"),
		},
		{
			name:  "xmltable column prefix keeps matching column",
			input: "SELECT xt.N| FROM t1, XMLTABLE('/root/row' PASSING t1.c1 COLUMNS id NUMBER PATH 'id', name VARCHAR2(100) PATH 'name') xt",
			want:  columns("NAME"),
		},
		{
			name:  "inline external column prefix keeps matching column",
			input: "SELECT ext.A| FROM EXTERNAL ((a NUMBER, b VARCHAR2(10)) TYPE ORACLE_LOADER DEFAULT DIRECTORY data_dir LOCATION ('x.csv')) ext",
			want:  columns("A"),
		},
		{
			name:  "comment on schema qualified column offers columns",
			input: "COMMENT ON COLUMN schema1.t2.|",
			want:  columns("C1", "C2"),
		},
		{
			name:  "grant on schema qualified object offers tables",
			input: "GRANT SELECT ON schema1.|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "revoke on schema qualified object offers tables",
			input: "REVOKE SELECT ON schema1.|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "truncate schema qualified table reference offers tables",
			input: "TRUNCATE TABLE schema1.|",
			want:  tables("T1", "T2"),
		},
		{
			name:  "comment on schema qualified table reference offers tables",
			input: "COMMENT ON TABLE schema1.|",
			want:  tables("T1", "T2"),
		},
		{
			name:    "drop schema qualified view reference offers only views",
			input:   "DROP VIEW schema1.|",
			want:    views("V1"),
			notWant: append(tables("T1", "T2"), sequences("SEQ1", "USER_ID_SEQ")...),
		},
		{
			name:  "view schema qualified table reference offers views",
			input: "SELECT * FROM schema1.v|",
			want:  views("V1"),
		},
		{
			name:  "sequence schema qualified reference offers sequences",
			input: "DROP SEQUENCE schema1.|",
			want:  sequences("SEQ1", "USER_ID_SEQ"),
		},
		{
			name:    "sequence schema qualified reference uses non default schema",
			input:   "DROP SEQUENCE schema2.|",
			want:    sequences("SCHEMA2_SEQ"),
			notWant: sequences("SEQ1", "USER_ID_SEQ"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := completeOracleForTest(t, tt.input)
			for _, want := range tt.want {
				require.Truef(t, hasCompletionCandidate(result, want), "missing %s candidate %q in %v", want.typ, want.text, result)
			}
			for _, notWant := range tt.notWant {
				require.Falsef(t, hasCompletionCandidate(result, notWant), "unexpected %s candidate %q in %v", notWant.typ, notWant.text, result)
			}
		})
	}
}

func TestCompletionQuerySceneRestrictsTopLevelWrites(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []completionCandidateSpec
		notWant []completionCandidateSpec
	}{
		{
			name:    "statement start offers query keywords only",
			input:   "|",
			want:    keywords("SELECT", "WITH", "EXPLAIN"),
			notWant: keywords("INSERT", "UPDATE", "DELETE", "MERGE", "CREATE", "ALTER", "DROP", "TRUNCATE"),
		},
		{
			name:    "statement start prefix keeps matching query keyword",
			input:   "SEL|",
			want:    keywords("SELECT"),
			notWant: keywords("INSERT", "UPDATE", "CREATE", "DROP"),
		},
		{
			name:    "write statement does not offer object candidates",
			input:   "DROP TABLE |",
			notWant: tables("T1", "T2"),
		},
		{
			name:    "with prefixed write statement does not offer object candidates",
			input:   "WITH x AS (SELECT c1 FROM t1) INSERT INTO t1 SELECT * FROM |",
			notWant: tables("T1", "T2", "X"),
		},
		{
			name:  "select table reference still offers tables",
			input: "SELECT * FROM |",
			want:  tables("T1", "T2"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := completeOracleForTestWithScene(t, tt.input, base.SceneTypeQuery)
			for _, want := range tt.want {
				require.Truef(t, hasCompletionCandidate(result, want), "missing %s candidate %q in %v", want.typ, want.text, result)
			}
			for _, notWant := range tt.notWant {
				require.Falsef(t, hasCompletionCandidate(result, notWant), "unexpected %s candidate %q in %v", notWant.typ, notWant.text, result)
			}
		})
	}
}

func completionColumnTexts(candidates []base.Candidate) []string {
	var columns []string
	for _, candidate := range candidates {
		if candidate.Type == base.CandidateTypeColumn {
			columns = append(columns, candidate.Text)
		}
	}
	slices.Sort(columns)
	return columns
}

type completionCandidateSpec struct {
	text string
	typ  base.CandidateType
}

func completeOracleForTest(t *testing.T, input string) []base.Candidate {
	return completeOracleForTestWithScene(t, input, base.SceneTypeAll)
}

func completeOracleForTestWithScene(t *testing.T, input string, scene base.SceneType) []base.Candidate {
	t.Helper()
	text, caretLine, caretOffset := catchCaret(input)
	result, err := base.Completion(context.Background(), storepb.Engine_ORACLE, base.CompletionContext{
		Scene:             scene,
		DefaultDatabase:   "SCHEMA1",
		Metadata:          getMetadataForTest,
		ListDatabaseNames: listDatabaseNamesForTest,
	}, text, caretLine, caretOffset)
	require.NoError(t, err)
	return result
}

func hasCompletionCandidate(candidates []base.Candidate, want completionCandidateSpec) bool {
	for _, candidate := range candidates {
		if candidate.Type == want.typ && candidate.Text == want.text {
			return true
		}
	}
	return false
}

func columns(texts ...string) []completionCandidateSpec {
	return completionSpecs(base.CandidateTypeColumn, texts...)
}

func tables(texts ...string) []completionCandidateSpec {
	return completionSpecs(base.CandidateTypeTable, texts...)
}

func views(texts ...string) []completionCandidateSpec {
	return completionSpecs(base.CandidateTypeView, texts...)
}

func schemas(texts ...string) []completionCandidateSpec {
	return completionSpecs(base.CandidateTypeSchema, texts...)
}

func sequences(texts ...string) []completionCandidateSpec {
	return completionSpecs(base.CandidateTypeSequence, texts...)
}

func keywords(texts ...string) []completionCandidateSpec {
	return completionSpecs(base.CandidateTypeKeyword, texts...)
}

func completionSpecs(typ base.CandidateType, texts ...string) []completionCandidateSpec {
	specs := make([]completionCandidateSpec, 0, len(texts))
	for _, text := range texts {
		specs = append(specs, completionCandidateSpec{text: text, typ: typ})
	}
	return specs
}

func listDatabaseNamesForTest(_ context.Context, _ string) ([]string, error) {
	return []string{"SCHEMA1", "SCHEMA2", "SCHEMA3"}, nil
}

func getMetadataForTest(_ context.Context, _, databaseName string) (string, *model.DatabaseMetadata, error) {
	switch databaseName {
	case "SCHEMA1":
		return "SCHEMA1", model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
			Name: databaseName,
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "",
					Tables: []*storepb.TableMetadata{
						{
							Name: "T1",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "C1",
									Type: "int",
								},
							},
						},
						{
							Name: "T2",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "C1",
									Type: "int",
								},
								{
									Name: "C2",
									Type: "int",
								},
							},
						},
					},
					Views: []*storepb.ViewMetadata{
						{
							Name: "V1",
							Definition: `CREATE VIEW v1 AS
											SELECT *
											FROM t1
							`,
						},
					},
					Sequences: []*storepb.SequenceMetadata{
						{
							Name: "SEQ1",
						},
						{
							Name: "USER_ID_SEQ",
						},
					},
				},
			},
		}, nil, nil, storepb.Engine_ORACLE, true /* isObjectCaseSensitive */), nil
	case "SCHEMA2":
		return "SCHEMA2", model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
			Name: databaseName,
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "",
					Tables: []*storepb.TableMetadata{
						{
							Name: "T1",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "C1",
									Type: "int",
								},
							},
						},
						{
							Name: "T2",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "C1",
									Type: "int",
								},
								{
									Name: "C2",
									Type: "int",
								},
							},
						},
					},
					Views: []*storepb.ViewMetadata{
						{
							Name: "V1",
							Definition: `CREATE VIEW v1 AS
											SELECT *
											FROM t1
							`,
						},
					},
					Sequences: []*storepb.SequenceMetadata{
						{
							Name: "SCHEMA2_SEQ",
						},
					},
				},
			},
		}, nil, nil, storepb.Engine_ORACLE, true /* isObjectCaseSensitive */), nil
	case "SCHEMA3":
		return "SCHEMA3", model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
			Name: databaseName,
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "",
					Tables: []*storepb.TableMetadata{
						{
							Name: "T1",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "C1",
									Type: "int",
								},
							},
						},
						{
							Name: "T2",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "C1",
									Type: "int",
								},
								{
									Name: "C2",
									Type: "int",
								},
							},
						},
					},
					Views: []*storepb.ViewMetadata{
						{
							Name: "V1",
							Definition: `CREATE VIEW v1 AS
											SELECT *
											FROM t1
							`,
						},
					},
				},
			},
		}, nil, nil, storepb.Engine_ORACLE, true /* isObjectCaseSensitive */), nil
	case "SCHEMA4":
		return "SCHEMA4", model.NewDatabaseMetadata(&storepb.DatabaseSchemaMetadata{
			Name: databaseName,
			Schemas: []*storepb.SchemaMetadata{
				{
					Name: "",
					Tables: []*storepb.TableMetadata{
						{
							Name: "T1",
							Columns: []*storepb.ColumnMetadata{
								{
									Name: "C1",
									Type: "int",
								},
								{
									Name: "SCHEMA4_ONLY",
									Type: "int",
								},
							},
						},
					},
				},
			},
		}, nil, nil, storepb.Engine_ORACLE, true /* isObjectCaseSensitive */), nil
	default:
		return "", nil, nil
	}
}

func catchCaret(s string) (string, int, int) {
	line := 1
	column := 0
	for i, c := range s {
		switch c {
		case '|':
			return s[:i] + s[i+1:], line, column
		case '\n':
			line++
			column = 0
		default:
			column++
		}
	}
	return s, -1, -1
}

func catchCaretUTF16(s string) (string, int, int) {
	line := 1
	column := 0
	for i, c := range s {
		switch c {
		case '|':
			return s[:i] + s[i+1:], line, column
		case '\n':
			line++
			column = 0
		default:
			if c > 0xFFFF {
				column += 2
			} else {
				column++
			}
		}
	}
	return s, -1, -1
}
