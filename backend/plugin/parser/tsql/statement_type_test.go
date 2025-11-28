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
		asts, err := base.Parse(storepb.Engine_MSSQL, test.Statement)
		a.NoError(err)
		a.NotEmpty(asts)
		sqlType, err := GetStatementTypes(asts)
		a.NoError(err)
		a.Equal(test.Want, sqlType)
	}
}
