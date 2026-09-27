package mysql

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type statementTypeTest struct {
	Statement string
	Want      []string
}

// TestGetStatementType runs the fixture under every engine this package
// registers the classifier for, so MariaDB and OceanBase cannot silently
// diverge from MySQL (BYT-10136).
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

	engines := []storepb.Engine{
		storepb.Engine_MYSQL,
		storepb.Engine_MARIADB,
		storepb.Engine_OCEANBASE,
	}
	for _, engine := range engines {
		t.Run(engine.String(), func(t *testing.T) {
			a := require.New(t)
			for i, test := range tests {
				stmts, err := base.ParseStatements(engine, test.Statement)
				a.NoError(err)
				asts := base.ExtractASTs(stmts)

				sqlType, err := GetStatementTypes(asts)
				a.NoError(err)

				// Convert enum to string for comparison
				sqlTypeStrings := make([]string, len(sqlType))
				for j, t := range sqlType {
					sqlTypeStrings[j] = t.String()
				}

				// Record from the MySQL run only; the other engines must match it.
				if record && engine == storepb.Engine_MYSQL {
					tests[i].Want = sqlTypeStrings
				} else {
					a.Equal(test.Want, sqlTypeStrings)
				}
			}
		})
	}

	if record {
		yamltest.Record(t, filepath, tests)
	}
}
