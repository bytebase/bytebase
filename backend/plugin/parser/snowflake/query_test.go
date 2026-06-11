package snowflake

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/omni/snowflake/parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestValidateSQLForEditor(t *testing.T) {
	tests := []struct {
		statement   string
		valid       bool
		gotAllQuery bool
		err         bool
	}{
		{
			statement:   "SHOW TABLES;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			// Result-pipe: read-only source + trailing SELECT over $1 is a query.
			statement:   "SHOW TABLES ->> SELECT * FROM $1;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			// Result-pipe with a non-read-only source must not classify as a query.
			statement:   "CALL P() ->> SELECT * FROM $1;",
			valid:       false,
			gotAllQuery: false,
		},
		{
			// A non-read-only statement piped FROM a SHOW must not classify as a query.
			statement:   "SHOW TABLES ->> DELETE FROM T1;",
			valid:       false,
			gotAllQuery: false,
		},
		{
			statement:   "DESC TABLE bytebase;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   "SELECT * FROM t1 WHERE c1 = 1; SELECT * FROM t2;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   "CREATE TABLE t1 (c1 INT);",
			valid:       false,
			gotAllQuery: false,
		},
		{
			statement:   "UPDATE t1 SET c1 = 1;",
			valid:       false,
			gotAllQuery: false,
		},
		{
			statement:   "EXPLAIN SELECT * FROM t1;",
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   `select* from t`,
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   `explain select * from t;`,
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   "select * from t where a = 'klasjdfkljsa$tag$; -- lkjdlkfajslkdfj'",
			valid:       true,
			gotAllQuery: true,
		},
		{
			statement:   `create table t (a int);`,
			valid:       false,
			gotAllQuery: false,
		},
		{
			statement:   `SET max_execution_time = 1000; select * from t`,
			valid:       true,
			gotAllQuery: false,
		},
	}

	for _, test := range tests {
		gotValid, gotAllQuery, err := validateQuery(test.statement)
		if test.err {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, test.valid, gotValid, test.statement)
			require.Equal(t, test.gotAllQuery, gotAllQuery, test.statement)
		}
	}
}

// TestGetQueryType_CallExecuteAreDML locks the legacy classification of
// CALL / EXECUTE IMMEDIATE / EXECUTE TASK as DML (procedures can mutate data;
// the DDL fallback would map them to the wrong ACL bucket).
func TestGetQueryType_CallExecuteAreDML(t *testing.T) {
	for _, sql := range []string{
		"CALL P(1);",
		"EXECUTE IMMEDIATE 'SELECT 1';",
		"EXECUTE TASK T1;",
	} {
		file, err := parser.Parse(sql)
		if err != nil {
			t.Fatalf("parse %q: %v", sql, err)
		}
		if got := getQueryType(file.Stmts[0]); got != base.DML {
			t.Fatalf("getQueryType(%q) = %v, want base.DML", sql, got)
		}
	}
}

// TestGetQueryType_UnmappedStaysUnknown locks the fail-closed fallback: parsed
// statements the classifier does not deliberately map (legacy left them
// QueryTypeUnknown, which the SQL service denies) must NOT default to DDL.
func TestGetQueryType_UnmappedStaysUnknown(t *testing.T) {
	for _, tc := range []struct {
		sql  string
		want base.QueryType
	}{
		{"TRUNCATE TABLE T1;", base.QueryTypeUnknown},
		{"GRANT SELECT ON TABLE T1 TO ROLE R1;", base.QueryTypeUnknown},
		{"CREATE TABLE T1 (A INT);", base.DDL},
		{"ALTER TABLE T1 ADD COLUMN B INT;", base.DDL},
		{"DROP TABLE T1;", base.DDL},
	} {
		file, err := parser.Parse(tc.sql)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.sql, err)
		}
		if got := getQueryType(file.Stmts[0]); got != tc.want {
			t.Fatalf("getQueryType(%q) = %v, want %v", tc.sql, got, tc.want)
		}
	}
}

// TestGetQueryType_ShowPipeClassifiesByQuery locks that SHOW ... ->> <query>
// classifies as the piped query's type (so the trailing SELECT is
// permission-checked and masked as a query, not as info-schema metadata).
func TestGetQueryType_ShowPipeClassifiesByQuery(t *testing.T) {
	file, err := parser.Parse("SHOW TABLES ->> SELECT * FROM SENSITIVE_T;")
	if err != nil {
		t.Fatal(err)
	}
	if got := getQueryType(file.Stmts[0]); got != base.Select {
		t.Fatalf("got %v, want base.Select", got)
	}
	file, err = parser.Parse("SHOW TABLES;")
	if err != nil {
		t.Fatal(err)
	}
	if got := getQueryType(file.Stmts[0]); got != base.SelectInfoSchema {
		t.Fatalf("got %v, want base.SelectInfoSchema", got)
	}
}
