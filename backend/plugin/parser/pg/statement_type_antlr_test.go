package pg

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type statementTypeTest struct {
	Statement string
	Want      []string
}

func TestGetStatementType(t *testing.T) {
	tests := []statementTypeTest{}

	const (
		record = false
	)

	var (
		filepath = "test-data/test_statement_type.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, test := range tests {
		stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, test.Statement)
		a.NoError(err)
		asts := base.ExtractASTs(stmts)

		stmtsWithPos, err := GetStatementTypes(asts)
		a.NoError(err)

		// Convert enum to string for comparison
		sqlTypeStrings := make([]string, len(stmtsWithPos))
		for j, stmt := range stmtsWithPos {
			sqlTypeStrings[j] = stmt.Type.String()
		}

		if record {
			tests[i].Want = sqlTypeStrings
		} else {
			a.Equal(test.Want, sqlTypeStrings)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

func TestGetStatementTypesWithPositions(t *testing.T) {
	a := require.New(t)
	tests := []struct {
		name     string
		sql      string
		expected []StatementTypeWithPosition
	}{
		{
			name: "Single statement",
			sql:  "CREATE TABLE t1 (id INT);",
			expected: []StatementTypeWithPosition{
				{
					Type: storepb.StatementType_CREATE_TABLE,
					Line: 1,
					Text: "CREATE TABLE t1 (id INT);",
				},
			},
		},
		{
			name: "Multiple statements",
			sql: `CREATE TABLE t1 (id INT);
DROP TABLE t2;
INSERT INTO t1 VALUES (1);`,
			expected: []StatementTypeWithPosition{
				{
					Type: storepb.StatementType_CREATE_TABLE,
					Line: 1,
				},
				{
					Type: storepb.StatementType_DROP_TABLE,
					Line: 2,
				},
				{
					Type: storepb.StatementType_INSERT,
					Line: 3,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			stmts, err := base.ParseStatements(storepb.Engine_POSTGRES, tt.sql)
			a.NoError(err)
			asts := base.ExtractASTs(stmts)

			results, err := GetStatementTypes(asts)
			a.NoError(err)
			a.Len(results, len(tt.expected))

			for i, expected := range tt.expected {
				a.Equal(expected.Type, results[i].Type, "Statement %d type mismatch", i)
				a.Equal(expected.Line, results[i].Line, "Statement %d line mismatch", i)
				if expected.Text != "" {
					// Check that text contains expected content (may not include semicolon)
					a.Contains(results[i].Text, "CREATE TABLE t1", "Statement %d text mismatch", i)
				}
			}
		})
	}
}
