package mysql

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type statementTypeTest struct {
	Statement string
	Want      string
}

func TestGetStatementType(t *testing.T) {
	tests := []statementTypeTest{}

	const (
		record = true
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
		t.Log(test.Statement)
		stmts, err := ParseMySQL(test.Statement)
		a.NoError(err)

		if len(stmts) != 1 {
			t.Fatalf("the length of parse result of stmt %v is not one", test.Statement)
		}

		sqlType := GetStatementType(stmts[0])

		if record {
			tests[i].Want = sqlType
		} else {
			a.Equal(test.Want, sqlType)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		a.NoError(err)
		err = os.WriteFile(filepath, byteValue, 0644)
		a.NoError(err)
	}
}

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
		filepath = "test-data/test_affected_rows.yaml"
	)

	a := require.New(t)
	yamlFile, err := os.Open(filepath)
	a.NoError(err)

	byteValue, err := io.ReadAll(yamlFile)
	a.NoError(yamlFile.Close())
	a.NoError(err)
	a.NoError(yaml.Unmarshal(byteValue, &tests))

	for i, test := range tests {
		stmts, err := ParseMySQL(test.Statement)
		a.NoError(err)

		if len(stmts) != 1 {
			t.Fatalf("the length of parse result of stmt %v is not one", test.Statement)
		}

		affectedRows, err := GetAffectedRows(context.Background(), nil, getMetadataForAffectedRowsTest(), stmts[0])
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

func getMetadataForAffectedRowsTest() *storepb.DatabaseSchemaMetadata {
	return &storepb.DatabaseSchemaMetadata{
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
}
