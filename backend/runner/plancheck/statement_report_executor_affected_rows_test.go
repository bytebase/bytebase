package plancheck

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type affectedRowTest struct {
	Statement string
	Want      int64
}

func TestGetAffectedRows(t *testing.T) {
	tests := []affectedRowTest{}

	const (
		record = false
	)

	var (
		filepath = "test/test_affected_rows.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, test := range tests {
		stmts, err := mysqlparser.ParseMySQL(test.Statement)
		a.NoError(err)

		if len(stmts) != 1 {
			t.Fatalf("the length of parse result of stmt %v is not one", test.Statement)
		}

		affectedRows, err := mysqlparser.GetAffectedRows(context.Background(), stmts[0], nil, buildGetTableDataSizeFuncForMySQL(getMetadataForAffectedRowsTest()))
		a.NoError(err)

		if record {
			tests[i].Want = affectedRows
		} else {
			a.Equal(test.Want, affectedRows)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

func getMetadataForAffectedRowsTest() *model.DBSchema {
	metadata := &storepb.DatabaseSchemaMetadata{
		Name: "testdb",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "",
				Tables: []*storepb.TableMetadata{
					{
						Name: "t1",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "c1",
							},
						},
						RowCount: 100,
					},
					{
						Name: "t2",
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "c1",
							},
							{
								Name: "c2",
							},
						},
						RowCount: 1000,
					},
				},
			},
		},
	}
	return model.NewDBSchema(metadata, nil /* schema */, nil /* config */)
}
