package tsql

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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
		result, err := ParseTSQL(test.Statement)
		a.NoError(err)
		sqlType, err := GetStatementTypes(result.Tree)
		a.NoError(err)
		a.Equal(test.Want, sqlType)
	}
}
