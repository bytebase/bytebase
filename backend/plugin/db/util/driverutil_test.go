package util

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"

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
		gotUseSemanticVersion, gotVersion, gotSemanticVersionSuffix, err := FromStoredVersion(tc.storedVersion)
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
			sqlStatement: "  seLeCT * FROM test",
			limit:        123,
			want:         "WITH result AS (  seLeCT * FROM test) SELECT * FROM result LIMIT 123;",
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
			sqlStatement: "  seLeCT * FROM test",
			limit:        123,
			want:         "SELECT * FROM (  seLeCT * FROM test) result LIMIT 123;",
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
	var rand = rand.New(rand.NewSource(time.Now().UnixNano()))
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
		{
			// Test for EXPLAIN statements.
			statement:  "explain select 1;",
			schemaInfo: &db.SensitiveSchemaInfo{},
			fieldList:  nil,
		},
	}

	for _, test := range tests {
		res, err := extractSensitiveField(db.MySQL, test.statement, defaultDatabase, test.schemaInfo)
		require.NoError(t, err)
		require.Equal(t, test.fieldList, res, test.statement)
	}
}

func TestPostgreSQLExtractSensitiveField(t *testing.T) {
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
							Name: "public.t",
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
			// Test for Non-Recursive Common Table Expression with RECURSIVE key words.
			statement: `
				with recursive t1 as (
					select 1 as c1, 2 as c2, 3 as c3, 1 as n
					union
					select a, b, d, c from t
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
					Name:      "max",
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
			// Test for UNION.
			statement:  `select 1 as c1, 2 as c2, 3 as c3, 4 UNION ALL select * from t`,
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
					Sensitive: false,
				},
				{
					Name:      "?column?",
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
			// Test for explicit schema name.
			statement:  `select concat(public.t.a, public.t.b, public.t.c) from t`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "concat",
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
					Name:      "?column?",
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
			// Test for non-associated sub-query
			statement:  "select t.a, (select max(a) from t) from t as t1 join t on t.a = t1.b",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "a",
					Sensitive: true,
				},
				{
					Name:      "max",
					Sensitive: true,
				},
			},
		},
		{
			// Test for functions.
			statement:  `select max(a), a-b as c1, a=b as c2, a>b, b in (a, c, d) from (select * from t) result`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "max",
					Sensitive: true,
				},
				{
					Name:      "c1",
					Sensitive: true,
				},
				{
					Name:      "c2",
					Sensitive: true,
				},
				{
					Name:      "?column?",
					Sensitive: true,
				},
				{
					Name:      "?column?",
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
			// Test for sub-select.
			statement:  "select * from (select a, t.b, public.t.c, d as d1 from public.t) result LIMIT 100000;",
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
			// Test for field name.
			statement:  "select a, t.b, public.t.c, d as d1 from t",
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
			// Test for *.
			statement:  "select * from t",
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
			// Test for no FROM clause.
			statement:  "select 1;",
			schemaInfo: &db.SensitiveSchemaInfo{},
			fieldList:  []db.SensitiveField{{Name: "?column?", Sensitive: false}},
		},
		{
			// Test for EXPLAIN statements.
			statement:  "explain select 1;",
			schemaInfo: &db.SensitiveSchemaInfo{},
			fieldList:  nil,
		},
	}

	for _, test := range tests {
		res, err := extractSensitiveField(db.Postgres, test.statement, defaultDatabase, test.schemaInfo)
		require.NoError(t, err)
		require.Equal(t, test.fieldList, res, test.statement)
	}
}

func TestPLSQLExtractSensitiveField(t *testing.T) {
	const (
		defaultSchema = "ROOT"
	)
	var (
		defaultDatabaseSchema = &db.SensitiveSchemaInfo{
			DatabaseList: []db.DatabaseSchema{
				{
					Name: defaultSchema,
					TableList: []db.TableSchema{
						{
							Name: "T",
							ColumnList: []db.ColumnInfo{
								{
									Name:      "A",
									Sensitive: true,
								},
								{
									Name:      "B",
									Sensitive: false,
								},
								{
									Name:      "C",
									Sensitive: false,
								},
								{
									Name:      "D",
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
			// Test for Recursive Common Table Expression dependent closures.
			statement: `
				with t1(cc1, cc2, cc3, n) as (
					select a as c1, b as c2, c as c3, 1 as n from t
					union all
					select cc1 * cc2, cc2 + cc1, cc3 * cc2, n + 1 from t1 where n < 5
				)
				select * from t1;
			`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "CC1",
					Sensitive: true,
				},
				{
					Name:      "CC2",
					Sensitive: true,
				},
				{
					Name:      "CC3",
					Sensitive: true,
				},
				{
					Name:      "N",
					Sensitive: false,
				},
			},
		},
		{
			// Test for Recursive Common Table Expression.
			statement: `
				with t1 as (
					select 1 as c1, 2 as c2, 3 as c3, 1 as n from DUAL
					union all
					select c1 * a, c2 * b, c3 * d, n + 1 from t1, t where n < 5
				)
				select * from t1;
			`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "C1",
					Sensitive: true,
				},
				{
					Name:      "C2",
					Sensitive: false,
				},
				{
					Name:      "C3",
					Sensitive: true,
				},
				{
					Name:      "N",
					Sensitive: false,
				},
			},
		},
		{
			// Test that Common Table Expression rename field names.
			statement:  `with t1(d, c, b, a) as (select * from t) select * from t1`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "D",
					Sensitive: true,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "A",
					Sensitive: true,
				},
			},
		},
		{
			// Test for Common Table Expression with UNION.
			statement:  `with t1 as (select * from t), t2 as (select * from t1) select * from (select * from t1 union all select * from t2)`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "D",
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
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "D",
					Sensitive: true,
				},
			},
		},
		{
			// Test for multi-level Common Table Expression.
			statement:  `with tt2 as (with tt2 as (select * from t) select MAX(A) from tt2) select * from tt2`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "MAX(A)",
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
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "D",
					Sensitive: true,
				},
			},
		},
		{
			// Test for UNION.
			statement:  `select 1 as c1, 2 as c2, 3 as c3, 4 from DUAL UNION ALL select * from t`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "C1",
					Sensitive: true,
				},
				{
					Name:      "C2",
					Sensitive: false,
				},
				{
					Name:      "C3",
					Sensitive: false,
				},
				{
					Name:      "4",
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
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "D",
					Sensitive: true,
				},
			},
		},
		{
			// Test for explicit schema name.
			statement:  `select CONCAT(ROOT.T.A, ROOT.T.B) from T`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "CONCAT(ROOT.T.A,ROOT.T.B)",
					Sensitive: true,
				},
			},
		},
		{
			// Test for associated sub-query.
			statement:  `select a, (SELECT MAX(B) > Y.A FROM T X) from t y`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "(SELECTMAX(B)>Y.AFROMTX)",
					Sensitive: true,
				},
			},
		},
		{
			// Test for JOIN with ON clause.
			statement:  `select * from t t1 join t t2 on t1.a = t2.a`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "D",
					Sensitive: true,
				},
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "D",
					Sensitive: true,
				},
			},
		},
		{
			// Test for natural JOIN.
			statement:  `select * from t t1 natural join t t2`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "D",
					Sensitive: true,
				},
			},
		},
		{
			// Test for JOIN with USING clause.
			statement:  `select * from t t1 join t t2 using(a)`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "D",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "D",
					Sensitive: true,
				},
			},
		},
		{
			// Test for non-associated sub-query
			statement:  "select t.a, (SELECT MAX(A) FROM T) from t t1 join t on t.a = t1.b",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "(SELECTMAX(A)FROMT)",
					Sensitive: true,
				},
			},
		},
		{
			// Test for functions.
			statement:  `select A-B, B+C as c1 from (select * from t)`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A-B",
					Sensitive: true,
				},
				{
					Name:      "C1",
					Sensitive: false,
				},
			},
		},
		{
			// Test for functions.
			statement:  `select MAX(A), min(b) as c1 from (select * from t)`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "MAX(A)",
					Sensitive: true,
				},
				{
					Name:      "C1",
					Sensitive: false,
				},
			},
		},
		{
			// Test for sub-query
			statement:  "select * from (select * from t) where rownum <= 100000;",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "D",
					Sensitive: true,
				},
			},
		},
		{
			// Test for sub-select.
			statement:  "select * from (select a, t.b, root.t.c, d as d1 from root.t) where ROWNUM <= 100000;",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "D1",
					Sensitive: true,
				},
			},
		},
		{
			// Test for field name.
			statement:  "select a, t.b, root.t.c, d as d1 from t",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "D1",
					Sensitive: true,
				},
			},
		},
		{
			statement:  "SELECT * FROM ROOT.T;",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "D",
					Sensitive: true,
				},
			},
		},
		{
			// Test for EXPLAIN statements.
			statement:  "explain plan for select 1 from dual;",
			schemaInfo: &db.SensitiveSchemaInfo{},
			fieldList:  nil,
		},
		{
			// Test for no FROM DUAL.
			statement:  "select 1 from dual;",
			schemaInfo: &db.SensitiveSchemaInfo{},
			fieldList:  []db.SensitiveField{{Name: "1", Sensitive: false}},
		},
	}

	for _, test := range tests {
		res, err := extractSensitiveField(db.Oracle, test.statement, defaultSchema, test.schemaInfo)
		require.NoError(t, err)
		require.Equal(t, test.fieldList, res, test.statement)
	}
}

func TestSnowSQLExtractSensitiveField(t *testing.T) {
	var (
		defaultDatabase       = "SNOWFLAKE"
		defaultDatabaseSchema = &db.SensitiveSchemaInfo{
			DatabaseList: []db.DatabaseSchema{
				{
					Name: defaultDatabase,
					SchemaList: []db.SchemaSchema{
						{
							Name: "PUBLIC",
							TableList: []db.TableSchema{
								{
									Name: "T1",
									ColumnList: []db.ColumnInfo{
										{
											Name:      "A",
											Sensitive: true,
										},
										{
											Name:      "B",
											Sensitive: false,
										},
										{
											Name:      "C",
											Sensitive: false,
										},
										{
											Name:      "D",
											Sensitive: true,
										},
									},
								},
								{
									Name: "T2",
									ColumnList: []db.ColumnInfo{
										{
											Name:      "A",
											Sensitive: false,
										},
										{
											Name:      "E",
											Sensitive: false,
										},
									},
								},
								{
									Name: "T3",
									ColumnList: []db.ColumnInfo{
										{
											Name:      "E",
											Sensitive: true,
										},
										{
											Name:      "F",
											Sensitive: false,
										},
									},
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
			// Test for correlated sub-query.
			statement:  `SELECT A, (SELECT MAX(B) > Y.A FROM T1 X) FROM T1 Y`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "(SELECTMAX(B)>Y.AFROMT1X)",
					Sensitive: true,
				},
			},
		},
		{
			// Test for CTE in CTE.
			statement: `WITH TT1 (T1_COL1, T1_COL2) AS (
				WITH TT2 (T1_COL1, T1_COL2, T1_COL3) AS (
					SELECT A, B, C FROM T1
				)
				SELECT T1_COL1, T1_COL2 FROM TT2
			)
			SELECT * FROM TT1;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "T1_COL1",
					Sensitive: true,
				},
				{
					Name:      "T1_COL2",
					Sensitive: false,
				},
			},
		},
		{
			// Test for expression.
			statement:  `SELECT (SELECT A FROM T1 LIMIT 1), A + 1, 1, FUNCTIONCALL(D) FROM T1;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "(SELECTAFROMT1LIMIT1)",
					Sensitive: true,
				},
				{
					Name:      "A+1",
					Sensitive: true,
				},
				{
					Name:      "1",
					Sensitive: false,
				},
				{
					Name:      "FUNCTIONCALL(D)",
					Sensitive: true,
				},
			},
		},
		{
			// Test for multiple CTE
			statement: `
			WITH TT1 (T1_COL1, T1_COL2, T1_COL3, T1_COL4) AS (
				SELECT * FROM T1
			),
			TT2 (T2_COL1, T2_COL2) AS (
				SELECT * FROM T2
			)
			SELECT * FROM TT1 JOIN TT2;
			`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "T1_COL1",
					Sensitive: true,
				},
				{
					Name:      "T1_COL2",
					Sensitive: false,
				},
				{
					Name:      "T1_COL3",
					Sensitive: false,
				},
				{
					Name:      "T1_COL4",
					Sensitive: true,
				},
				{
					Name:      "T2_COL1",
					Sensitive: false,
				},
				{
					Name:      "T2_COL2",
					Sensitive: false,
				},
			},
		},
		{
			// Test for set operators(UNION, INTERSECT, ...)
			statement:  `SELECT A, B FROM T1 UNION SELECT * FROM T2 INTERSECT SELECT * FROM T3`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
			},
		},
		{
			// Test for subquery in from cluase with as alias.
			statement:  `SELECT T.A, A, B FROM (SELECT * FROM T1) AS T`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
			},
		},
		{
			// Test for field name.
			statement:  "SELECT $1, A, T.B AS N, T.C from T1 AS T",
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "N",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
			},
		},
		{
			statement:  `SELECT * FROM T1, T2, T3;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "D",
					Sensitive: true,
				},
				{
					Name:      "A",
					Sensitive: false,
				},
				{
					Name:      "E",
					Sensitive: false,
				},
				{
					Name:      "E",
					Sensitive: true,
				},
				{
					Name:      "F",
					Sensitive: false,
				},
			},
		},
		{
			statement:  `SELECT A, E, F FROM T1 NATURAL JOIN T2 NATURAL JOIN T3;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "E",
					Sensitive: true,
				},
				{
					Name:      "F",
					Sensitive: false,
				},
			},
		},
		{
			statement:  `SELECT A, B, D FROM T1;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "D",
					Sensitive: true,
				},
			},
		},
		{
			statement:  `SELECT * FROM T1;`,
			schemaInfo: defaultDatabaseSchema,
			fieldList: []db.SensitiveField{
				{
					Name:      "A",
					Sensitive: true,
				},
				{
					Name:      "B",
					Sensitive: false,
				},
				{
					Name:      "C",
					Sensitive: false,
				},
				{
					Name:      "D",
					Sensitive: true,
				},
			},
		},
	}

	for _, test := range tests {
		res, err := extractSensitiveField(db.Snowflake, test.statement, defaultDatabase, test.schemaInfo)
		require.NoError(t, err, test.statement)
		require.Equal(t, test.fieldList, res, test.statement)
	}
}
