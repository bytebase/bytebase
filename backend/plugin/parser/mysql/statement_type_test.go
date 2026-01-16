package mysql

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
		stmts, err := base.ParseStatements(storepb.Engine_MYSQL, test.Statement)
		a.NoError(err)
		asts := base.ExtractASTs(stmts)

		sqlType, err := GetStatementTypes(asts)
		a.NoError(err)

		// Convert enum to string for comparison
		sqlTypeStrings := make([]string, len(sqlType))
		for j, t := range sqlType {
			sqlTypeStrings[j] = t.String()
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
