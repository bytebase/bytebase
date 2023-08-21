package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	"github.com/bytebase/bytebase/backend/store"
)

func TestGetStatementsFromSchemaGroups(t *testing.T) {
	a := require.New(t)

	tcs := []struct {
		name                     string
		statement                string
		parserEngineType         parser.EngineType
		schemaGroupParent        string
		schemaGroups             []*store.SchemaGroupMessage
		schemaGroupMatchedTables map[string][]string

		expectedStatements       []string
		expectedSchemaGroupNames []string
	}{
		{
			name:              "simple",
			statement:         "ALTER TABLE salary ADD COLUMN num INT;",
			parserEngineType:  parser.MySQL,
			schemaGroupParent: "projects/p2/databaseGroups/g1",
			schemaGroups: []*store.SchemaGroupMessage{
				{
					ResourceID:  "schema group 1",
					Placeholder: "salary",
				},
			},
			schemaGroupMatchedTables: map[string][]string{
				"schema group 1": {"salary_01"},
			},
			expectedStatements:       []string{"ALTER TABLE salary_01 ADD COLUMN num INT\n;\n"},
			expectedSchemaGroupNames: []string{"projects/p2/databaseGroups/g1/schemaGroups/schema group 1"},
		},
		{
			name:              "matched but has no matched tables",
			statement:         "ALTER TABLE salary ADD COLUMN num INT;",
			parserEngineType:  parser.MySQL,
			schemaGroupParent: "projects/p2/databaseGroups/g1",
			schemaGroups: []*store.SchemaGroupMessage{
				{
					ResourceID:  "schema group 1",
					Placeholder: "salary",
				},
			},
			schemaGroupMatchedTables: map[string][]string{
				"schema group 1": nil,
			},
			expectedStatements:       nil,
			expectedSchemaGroupNames: nil,
		},
		{
			name: "complex 1",
			statement: `ALTER TABLE salary ADD COLUMN num INT;
CREATE INDEX salary_num_idx ON salary (num);
CREATE TABLE singleton(id INT);
ALTER TABLE person ADD COLUMN name VARCHAR(30);
ALTER TABLE partpartially ADD COLUMN num INT;
ALTER TABLE singleton ADD COLUMN num INT;`,
			parserEngineType:  parser.MySQL,
			schemaGroupParent: "projects/p2/databaseGroups/g1",
			schemaGroups: []*store.SchemaGroupMessage{
				{
					ResourceID:  "schema group 1",
					Placeholder: "salary",
				},
			},
			schemaGroupMatchedTables: map[string][]string{
				"schema group 1": {"salary_01", "salary_02"},
			},
			expectedStatements: []string{
				"ALTER TABLE salary_01 ADD COLUMN num INT;\n\nCREATE INDEX salary_01_num_idx ON salary_01 (num);\n",
				"ALTER TABLE salary_02 ADD COLUMN num INT;\n\nCREATE INDEX salary_02_num_idx ON salary_02 (num);\n",
				"\nCREATE TABLE singleton(id INT);\n\nALTER TABLE person ADD COLUMN name VARCHAR(30);\n\nALTER TABLE partpartially ADD COLUMN num INT;\n\nALTER TABLE singleton ADD COLUMN num INT\n;\n",
			},
			expectedSchemaGroupNames: []string{
				"projects/p2/databaseGroups/g1/schemaGroups/schema group 1",
				"projects/p2/databaseGroups/g1/schemaGroups/schema group 1",
				"",
			},
		},
		{
			name: "complex 2",
			statement: `ALTER TABLE salary ADD COLUMN num INT;
CREATE INDEX salary_num_idx ON salary (num);
CREATE TABLE singleton(id INT);
ALTER TABLE person ADD COLUMN name VARCHAR(30);
ALTER TABLE partpartially ADD COLUMN num INT;
ALTER TABLE singleton ADD COLUMN num INT;`,
			parserEngineType:  parser.MySQL,
			schemaGroupParent: "projects/p2/databaseGroups/g1",
			schemaGroups: []*store.SchemaGroupMessage{
				{
					ResourceID:  "schema group 1",
					Placeholder: "salary",
				},
				{
					ResourceID:  "schema group 2",
					Placeholder: "singleton",
				},
			},
			schemaGroupMatchedTables: map[string][]string{
				"schema group 1": {"salary_01", "salary_02"},
				"schema group 2": {"singleton_00"},
			},
			expectedStatements: []string{
				"ALTER TABLE salary_01 ADD COLUMN num INT;\n\nCREATE INDEX salary_01_num_idx ON salary_01 (num);\n",
				"ALTER TABLE salary_02 ADD COLUMN num INT;\n\nCREATE INDEX salary_02_num_idx ON salary_02 (num);\n",
				"\nCREATE TABLE singleton_00(id INT);\n",
				"\nALTER TABLE person ADD COLUMN name VARCHAR(30);\n\nALTER TABLE partpartially ADD COLUMN num INT;\n",
				"\nALTER TABLE singleton_00 ADD COLUMN num INT\n;\n",
			},
			expectedSchemaGroupNames: []string{
				"projects/p2/databaseGroups/g1/schemaGroups/schema group 1",
				"projects/p2/databaseGroups/g1/schemaGroups/schema group 1",
				"projects/p2/databaseGroups/g1/schemaGroups/schema group 2",
				"",
				"projects/p2/databaseGroups/g1/schemaGroups/schema group 2",
			},
		},
	}

	for _, tc := range tcs {
		statements, schemaGroupNames, err := GetStatementsAndSchemaGroupsFromSchemaGroups(tc.statement, tc.parserEngineType, tc.schemaGroupParent, tc.schemaGroups, tc.schemaGroupMatchedTables)
		a.NoError(err, tc.name)
		a.Equal(tc.expectedStatements, statements, tc.name)
		a.Equal(tc.expectedSchemaGroupNames, schemaGroupNames, tc.name)
	}
}
