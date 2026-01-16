package tsql

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
	var (
		filepath = "test-data/test_statement_type.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	tests := []statementTypeTest{}
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for _, test := range tests {
		stmts, err := base.ParseStatements(storepb.Engine_MSSQL, test.Statement)
		a.NoError(err)
		asts := base.ExtractASTs(stmts)
		a.NotEmpty(asts)
		sqlType, err := GetStatementTypes(asts)
		a.NoError(err)

		// Convert enum to string for comparison
		sqlTypeStrings := make([]string, len(sqlType))
		for i, t := range sqlType {
			sqlTypeStrings[i] = t.String()
		}
		a.Equal(test.Want, sqlTypeStrings)
	}
}
