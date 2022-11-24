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
	tests := []struct {
		statement        string
		fieldName        []string
		sensitiveDataMap db.SensitiveDataMap
		fieldMap         sensitiveFiledMap
	}{
		{
			statement: "select * from (select * from t) result LIMIT 100000;",
			fieldName: []string{"a", "b", "c", "d"},
			sensitiveDataMap: db.SensitiveDataMap{
				db.SensitiveData{
					Database: "db",
					Table:    "t",
					Column:   "a",
				}: db.SensitiveDataMaskTypeDefault,
				db.SensitiveData{
					Database: "db",
					Table:    "t",
					Column:   "d",
				}: db.SensitiveDataMaskTypeDefault,
			},
			fieldMap: sensitiveFiledMap{
				0: true,
				3: true,
			},
		},
		{
			statement: "select * from (select *, a, t.*, b from db.t) result LIMIT 100000;",
			fieldName: []string{"a", "b", "c", "a", "a", "b", "c", "b"},
			sensitiveDataMap: db.SensitiveDataMap{
				db.SensitiveData{
					Database: "db",
					Table:    "t",
					Column:   "a",
				}: db.SensitiveDataMaskTypeDefault,
				db.SensitiveData{
					Database: "db",
					Table:    "t",
					Column:   "c",
				}: db.SensitiveDataMaskTypeDefault,
			},
			fieldMap: sensitiveFiledMap{
				0: true,
				2: true,
				3: true,
				4: true,
				6: true,
			},
		},
	}

	for _, test := range tests {
		res, err := extractSensitiveField(db.MySQL, test.statement, "db", test.fieldName, test.sensitiveDataMap)
		require.NoError(t, err)
		require.Equal(t, test.fieldMap, res)
	}
}
