package util

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/db"
	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
)

func TestToStoredVersion(t *testing.T) {
	type test struct {
		useSemanticVersion    bool
		version               string
		semanticVersionSuffix string
		want                  string
		wantErr               string
	}
	tests := []test{
		{false, "hello", "", "0000.0000.0000-hello", ""},
		{false, "hello", "world", "0000.0000.0000-hello", ""},
		{true, "hello", "world", "", "No Major.Minor.Patch elements found"},
		{true, "v1.2.3", "world", "", "Invalid character(s) found in major number"},
		{true, "1.10000.3", "world", "", "major, minor, patch version should be < 10000"},
		{true, "1.2.3", "world", "0001.0002.0003-world", ""},
		{true, "2021.1.13", "world", "2021.0001.0013-world", ""},
	}
	for _, tc := range tests {
		got, err := ToStoredVersion(tc.useSemanticVersion, tc.version, tc.semanticVersionSuffix)
		if tc.wantErr != "" {
			require.Contains(t, err.Error(), tc.wantErr)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
}

func TestFromStoredVersion(t *testing.T) {
	type test struct {
		storedVersion             string
		wantUseSemanticVersion    bool
		wantVersion               string
		wantSemanticVersionSuffix string
		wantErr                   string
	}
	tests := []test{
		{"0000.0000.0000-hello", false, "hello", "", ""},
		{"0001.0001.0000-hello", true, "1.1.0", "hello", ""},
		{"2021.0001.0013-world", true, "2021.1.13", "world", ""},
		{"2021.0001.0013-hello-world", true, "2021.1.13", "hello-world", ""},
		{"2021.0001.0013", false, "", "", "should contain '-'"},
		{"2021.0001.0013.1234-hello", false, "", "", "should be in semantic version"},
		{"2021.0001-hello", false, "", "", "should be in semantic version"},
		{"2a21.0001.0013-hello", false, "", "", "should be in semantic version"},
		{"10000.0001.0000-hello", false, "", "", "should be < 10000"},
		{"", false, "", "", "should contain '-'"},
		{"hello", false, "", "", "should contain '-'"},
		{"1.2.3", false, "", "", "should contain '-'"},
	}
	for _, tc := range tests {
		gotUseSemanticVersion, gotVersion, gotSemanticVersionSuffix, err := fromStoredVersion(tc.storedVersion)
		if tc.wantErr != "" {
			require.Contains(t, err.Error(), tc.wantErr)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, tc.wantUseSemanticVersion, gotUseSemanticVersion)
		require.Equal(t, tc.wantVersion, gotVersion)
		require.Equal(t, tc.wantSemanticVersionSuffix, gotSemanticVersionSuffix)
	}
}

func TestGetStatementWithResultLimit(t *testing.T) {
	tests := []struct {
		sqlStatement string
		limit        int
		want         string
	}{
		{
			sqlStatement: "  seLeCT * FROM test;",
			limit:        123,
			want:         "WITH result AS (  seLeCT * FROM test) SELECT * FROM result LIMIT 123;",
		},
		{
			sqlStatement: "  seLeCT * FROM test;",
			limit:        0,
			want:         "WITH result AS (  seLeCT * FROM test) SELECT * FROM result;",
		},
		{
			sqlStatement: "  \n \r SELEct * from test ",
			limit:        100,
			want:         "WITH result AS (  \n \r SELEct * from test) SELECT * FROM result LIMIT 100;",
		},
		{
			sqlStatement: "SELECT\n*\nFROM\ntest  ;\n",
			limit:        100,
			want:         "WITH result AS (SELECT\n*\nFROM\ntest) SELECT * FROM result LIMIT 100;",
		},
		{
			sqlStatement: "SELECT\n*\nFROM\ntest  ;;;\n",
			limit:        100,
			want:         "WITH result AS (SELECT\n*\nFROM\ntest) SELECT * FROM result LIMIT 100;",
		},
		{
			sqlStatement: "SELECT\n*\nFROM\n`test;`  ;;;\n",
			limit:        100,
			want:         "WITH result AS (SELECT\n*\nFROM\n`test;`) SELECT * FROM result LIMIT 100;",
		},
		{
			sqlStatement: "EXPLAIN  \n \r SELEct * from test ",
			limit:        0,
			want:         "EXPLAIN  \n \r SELEct * from test",
		},
	}

	for _, test := range tests {
		got := getStatementWithResultLimit(test.sqlStatement, test.limit)
		if got != test.want {
			t.Errorf("trimSQLStatement %q: got result %v, want %v.", test.sqlStatement, got, test.want)
		}
	}
}

func TestGetMySQLStatementWithResultLimit(t *testing.T) {
	tests := []struct {
		sqlStatement string
		limit        int
		want         string
	}{
		{
			sqlStatement: "  seLeCT * FROM test;",
			limit:        123,
			want:         "SELECT * FROM (  seLeCT * FROM test) result LIMIT 123;",
		},
		{
			sqlStatement: "  seLeCT * FROM test;",
			limit:        0,
			want:         "SELECT * FROM (  seLeCT * FROM test) result;",
		},
		{
			sqlStatement: "  \n \r SELEct * from test ",
			limit:        100,
			want:         "SELECT * FROM (  \n \r SELEct * from test) result LIMIT 100;",
		},
		{
			sqlStatement: "SELECT\n*\nFROM\ntest  ;\n",
			limit:        100,
			want:         "SELECT * FROM (SELECT\n*\nFROM\ntest) result LIMIT 100;",
		},
		{
			sqlStatement: "SELECT\n*\nFROM\ntest  ;;;\n",
			limit:        100,
			want:         "SELECT * FROM (SELECT\n*\nFROM\ntest) result LIMIT 100;",
		},
		{
			sqlStatement: "SELECT\n*\nFROM\n`test;`  ;;;\n",
			limit:        100,
			want:         "SELECT * FROM (SELECT\n*\nFROM\n`test;`) result LIMIT 100;",
		},
		{
			sqlStatement: "EXPLAIN  \n \r SELEct * from test ",
			limit:        0,
			want:         "EXPLAIN  \n \r SELEct * from test",
		},
	}

	for _, test := range tests {
		got := getMySQLStatementWithResultLimit(test.sqlStatement, test.limit)
		if got != test.want {
			t.Errorf("trimSQLStatement %q: got result %v, want %v.", test.sqlStatement, got, test.want)
		}
	}
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
		err := ApplyMultiStatements(strings.NewReader(test.statement), countStatements)
		require.NoError(t, err)
		require.Equal(t, test.total, total)
	}
}

func generateOneMBInsert() string {
	rand.Seed(time.Now().UnixNano())
	letterList := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]byte, 1024*1024)
	for i := range b {
		b[i] = letterList[rand.Intn(len(letterList))]
	}
	return fmt.Sprintf("INSERT INTO t values('%s')", string(b))
}

func TestExtractSensitiveField(t *testing.T) {
	const (
		defaultDatabase = "db"
	)
	var (
		defaultDatabaseSchema = &db.SensitiveSchemaInfo{
			DatabaseList: []db.DatabaseSchema{
				{
					Name: defaultDatabase,
					TableList: []db.TableSchema{
						{
							Name: "t",
							ColumnList: []db.ColumnInfo{
								{
									Name:      "a",
									Sensitive: true,
								},
								{
									Name:      "b",
									Sensitive: false,
								},
								{
									Name:      "c",
									Sensitive: false,
								},
								{
									Name:      "d",
									Sensitive: true,
								},
							},
						},
					},
				},
			},
		}
	)
	tests := []struct {
		statement  string
		schemaInfo *db.SensitiveSchemaInfo
		fieldList  []db.SensitiveField
	}{
		{
			// Test for case-insensitive column names.
			statement:  `SELECT * FROM (select * from (select a from t) t1 join t as t2 using(A)) result LIMIT 10000;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "a",
					Sensitive: true,
				},
				{
					Name:      "b",
					Sensitive: false,
				},
				{
					Name:      "c",
					Sensitive: false,
				},
				{
					Name:      "d",
					Sensitive: true,
				},
			},
		},
		{
			// Test for explicit database name.
			statement:  `select concat(db.t.a, db.t.b, db.t.c) from t`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "concat(db.t.a, db.t.b, db.t.c)",
					Sensitive: true,
				},
			},
		},
		{
			// Test for Recursive Common Table Expression dependent closures.
			statement: `
				with recursive t1(cc1, cc2, cc3, n) as (
					select a as c1, b as c2, c as c3, 1 as n from t
					union
					select cc1 * cc2, cc2 + cc1, cc3 * cc2, n + 1 from t1 where n < 5
				)
				select * from t1;
			`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "cc1",
					Sensitive: true,
				},
				{
					Name:      "cc2",
					Sensitive: true,
				},
				{
					Name:      "cc3",
					Sensitive: true,
				},
				{
					Name:      "n",
					Sensitive: false,
				},
			},
		},
		{
			// Test for Recursive Common Table Expression.
			statement: `
				with recursive t1 as (
					select 1 as c1, 2 as c2, 3 as c3, 1 as n
					union
					select c1 * a, c2 * b, c3 * d, n + 1 from t1, t where n < 5
				)
				select * from t1;
			`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "c1",
					Sensitive: true,
				},
				{
					Name:      "c2",
					Sensitive: false,
				},
				{
					Name:      "c3",
					Sensitive: true,
				},
				{
					Name:      "n",
					Sensitive: false,
				},
			},
		},
		{
			// Test for Common Table Expression with UNION.
			statement:  `with t1 as (select * from t), t2 as (select * from t1) select * from t1 union all select * from t2`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "a",
					Sensitive: true,
				},
				{
					Name:      "b",
					Sensitive: false,
				},
				{
					Name:      "c",
					Sensitive: false,
				},
				{
					Name:      "d",
					Sensitive: true,
				},
			},
		},
		{
			// Test for Common Table Expression reference.
			statement:  `with t1 as (select * from t), t2 as (select * from t1) select * from t2`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "a",
					Sensitive: true,
				},
				{
					Name:      "b",
					Sensitive: false,
				},
				{
					Name:      "c",
					Sensitive: false,
				},
				{
					Name:      "d",
					Sensitive: true,
				},
			},
		},
		{
			// Test for multi-level Common Table Expression.
			statement:  `with tt2 as (with tt2 as (select * from t) select max(a) from tt2) select * from tt2;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "max(a)",
					Sensitive: true,
				},
			},
		},
		{
			// Test that Common Table Expression rename field names.
			statement:  `with t1(d, c, b, a) as (select * from t) select * from t1`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "d",
					Sensitive: true,
				},
				{
					Name:      "c",
					Sensitive: false,
				},
				{
					Name:      "b",
					Sensitive: false,
				},
				{
					Name:      "a",
					Sensitive: true,
				},
			},
		},
		{
			// Test for Common Table Expression.
			statement:  `with t1 as (select * from t) select * from t1`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "a",
					Sensitive: true,
				},
				{
					Name:      "b",
					Sensitive: false,
				},
				{
					Name:      "c",
					Sensitive: false,
				},
				{
					Name:      "d",
					Sensitive: true,
				},
			},
		},
		{
			// Test for associated sub-query.
			statement:  `select a, (select max(b) > y.a from t as x) from t as y`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "a",
					Sensitive: true,
				},
				{
					Name:      "(select max(b) > y.a from t as x)",
					Sensitive: true,
				},
			},
		},
		{
			// Test for UNION.
			statement:  `select * from t UNION ALL select * from t`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "a",
					Sensitive: true,
				},
				{
					Name:      "b",
					Sensitive: false,
				},
				{
					Name:      "c",
					Sensitive: false,
				},
				{
					Name:      "d",
					Sensitive: true,
				},
			},
		},
		{
			// Test for JOIN with ON clause.
			statement:  `select * from t as t1 join t as t2 on t1.a = t2.a`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "a",
					Sensitive: true,
				},
				{
					Name:      "b",
					Sensitive: false,
				},
				{
					Name:      "c",
					Sensitive: false,
				},
				{
					Name:      "d",
					Sensitive: true,
				},
				{
					Name:      "a",
					Sensitive: true,
				},
				{
					Name:      "b",
					Sensitive: false,
				},
				{
					Name:      "c",
					Sensitive: false,
				},
				{
					Name:      "d",
					Sensitive: true,
				},
			},
		},
		{
			// Test for natural JOIN.
			statement:  `select * from t as t1 natural join t as t2`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "a",
					Sensitive: true,
				},
				{
					Name:      "b",
					Sensitive: false,
				},
				{
					Name:      "c",
					Sensitive: false,
				},
				{
					Name:      "d",
					Sensitive: true,
				},
			},
		},
		{
			// Test for JOIN with USING clause.
			statement:  `select * from t as t1 join t as t2 using(a)`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "a",
					Sensitive: true,
				},
				{
					Name:      "b",
					Sensitive: false,
				},
				{
					Name:      "c",
					Sensitive: false,
				},
				{
					Name:      "d",
					Sensitive: true,
				},
				{
					Name:      "b",
					Sensitive: false,
				},
				{
					Name:      "c",
					Sensitive: false,
				},
				{
					Name:      "d",
					Sensitive: true,
				},
			},
		},
		{
			// Test for functions.
			statement:  `select max(a), a-b, a=b, a>b, b in (a, c, d) from (select * from t) result`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "max(a)",
					Sensitive: true,
				},
				{
					Name:      "a-b",
					Sensitive: true,
				},
				{
					Name:      "a=b",
					Sensitive: true,
				},
				{
					Name:      "a>b",
					Sensitive: true,
				},
				{
					Name:      "b in (a, c, d)",
					Sensitive: true,
				},
			},
		},
		{
			// Test for non-associated sub-query
			statement:  "select t.a, (select max(a) from t) from t as t1 join t on t.a = t1.b",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "a",
					Sensitive: true,
				},
				{
					Name:      "(select max(a) from t)",
					Sensitive: true,
				},
			},
		},
		{
			// Test for sub-query
			statement:  "select * from (select * from t) result LIMIT 100000;",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "a",
					Sensitive: true,
				},
				{
					Name:      "b",
					Sensitive: false,
				},
				{
					Name:      "c",
					Sensitive: false,
				},
				{
					Name:      "d",
					Sensitive: true,
				},
			},
		},
		{
			// Test for field name.
			statement:  "select * from (select a, t.b, db.t.c, d as d1 from db.t) result LIMIT 100000;",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "a",
					Sensitive: true,
				},
				{
					Name:      "b",
					Sensitive: false,
				},
				{
					Name:      "c",
					Sensitive: false,
				},
				{
					Name:      "d1",
					Sensitive: true,
				},
			},
		},
		{
			// Test for no FROM clause.
			statement:  "select 1;",
			schemaInfo: &db.SensitiveSchemaInfo{},
			fieldList:  []db.SensitiveField{{Name: "1", Sensitive: false}},
		},
	}

	for _, test := range tests {
		res, err := extractSensitiveField(db.MySQL, test.statement, defaultDatabase, test.schemaInfo)
		require.NoError(t, err)
		require.Equal(t, test.fieldList, res, test.statement)
	}
}
