package catalog

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

type testData struct {
	Statement string
	// Use custom yaml tag to avoid generate field name `ignorecasesensitive`.
	IgnoreCaseSensitive bool `yaml:"ignore_case_sensitive"`
	Want                string
	Err                 *WalkThroughError
}

func TestTiDBWalkThrough(t *testing.T) {
	originDatabase := &storepb.DatabaseSchemaMetadata{
		Name: "test",
	}

	tests := []string{
		"tidb_walk_through",
	}

	for _, test := range tests {
		runWalkThroughTest(t, test, storepb.Engine_TIDB, originDatabase)
	}
}

func TestMySQLWalkThrough(t *testing.T) {
	originDatabase := &storepb.DatabaseSchemaMetadata{
		Name: "test",
	}

	tests := []string{
		"mysql_walk_through",
	}

	for _, test := range tests {
		runWalkThroughTest(t, test, storepb.Engine_MYSQL, originDatabase)
	}
}

func TestPostgreSQLWalkThrough(t *testing.T) {
	originDatabase := &storepb.DatabaseSchemaMetadata{
		Name: "postgres",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "test",
						Columns: []*storepb.ColumnMetadata{
							{
								Name:     "id",
								Type:     "int",
								Nullable: false,
							},
							{
								Name:     "name",
								Type:     "varchar(20)",
								Nullable: true,
							},
						},
					},
				},
				Views: []*storepb.ViewMetadata{
					{
						Name:       "v1",
						Definition: "SELECT id, name FROM test",
						DependencyColumns: []*storepb.DependencyColumn{
							{
								Schema: "public",
								Table:  "test",
								Column: "id",
							},
							{
								Schema: "public",
								Table:  "test",
								Column: "name",
							},
						},
					},
				},
			},
		},
	}

	tests := []string{
		"pg_walk_through",
	}

	for _, test := range tests {
		runWalkThroughTest(t, test, storepb.Engine_POSTGRES, originDatabase)
	}
}

func TestPostgreSQLANTLRWalkThrough(t *testing.T) {
	originDatabase := &storepb.DatabaseSchemaMetadata{
		Name: "postgres",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: "test",
						Columns: []*storepb.ColumnMetadata{
							{
								Name:     "id",
								Type:     "int",
								Nullable: false,
							},
							{
								Name:     "name",
								Type:     "varchar(20)",
								Nullable: true,
							},
						},
					},
				},
				Views: []*storepb.ViewMetadata{
					{
						Name:       "v1",
						Definition: "SELECT id, name FROM test",
						DependencyColumns: []*storepb.DependencyColumn{
							{
								Schema: "public",
								Table:  "test",
								Column: "id",
							},
							{
								Schema: "public",
								Table:  "test",
								Column: "name",
							},
						},
					},
				},
			},
		},
	}

	tests := []string{
		"pg_walk_through",
	}

	for _, test := range tests {
		runANTLRWalkThroughTest(t, test, storepb.Engine_POSTGRES, originDatabase)
	}
}

func runWalkThroughTest(t *testing.T, file string, engineType storepb.Engine, originDatabase *storepb.DatabaseSchemaMetadata) {
	tests := []testData{}
	filepath := filepath.Join("test", file+".yaml")
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err)
	sm := sheet.NewManager(nil)

	for _, test := range tests {
		var state *DatabaseState
		if originDatabase != nil {
			state = NewDatabaseState(originDatabase, test.IgnoreCaseSensitive, engineType)
		} else {
			state = NewDatabaseState(&storepb.DatabaseSchemaMetadata{}, test.IgnoreCaseSensitive, engineType)
		}

		asts, _ := sm.GetASTsForChecks(engineType, test.Statement)
		err := WalkThrough(state, asts)
		if err != nil {
			err, yes := err.(*WalkThroughError)
			require.True(t, yes)
			require.Equal(t, test.Err, err)
			continue
		}
		require.NoError(t, err, test.Statement)

		want := &storepb.DatabaseSchemaMetadata{}
		err = common.ProtojsonUnmarshaler.Unmarshal([]byte(test.Want), want)
		require.NoError(t, err)
		result := state.convertToDatabaseMetadata()
		diff := cmp.Diff(want, result, protocmp.Transform())
		require.Empty(t, diff)
	}
}

func runANTLRWalkThroughTest(t *testing.T, file string, engineType storepb.Engine, originDatabase *storepb.DatabaseSchemaMetadata) {
	tests := []testData{}
	filepath := filepath.Join("test", file+".yaml")
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err)

	for _, test := range tests {
		var state *DatabaseState
		if originDatabase != nil {
			state = NewDatabaseState(originDatabase, test.IgnoreCaseSensitive, engineType)
		} else {
			state = NewDatabaseState(&storepb.DatabaseSchemaMetadata{}, test.IgnoreCaseSensitive, engineType)
		}

		// Parse using ANTLR parser instead of legacy parser
		parseResult, parseErr := pgparser.ParsePostgreSQL(test.Statement)
		if parseErr != nil {
			t.Fatalf("Failed to parse SQL with ANTLR: %v\nSQL: %s", parseErr, test.Statement)
		}

		// Call WalkThrough with ANTLR tree
		err := WalkThrough(state, parseResult)
		if err != nil {
			err, yes := err.(*WalkThroughError)
			require.True(t, yes)
			require.Equal(t, test.Err, err)
			continue
		}
		require.NoError(t, err, test.Statement)

		want := &storepb.DatabaseSchemaMetadata{}
		err = common.ProtojsonUnmarshaler.Unmarshal([]byte(test.Want), want)
		require.NoError(t, err)
		result := state.convertToDatabaseMetadata()
		diff := cmp.Diff(want, result, protocmp.Transform())
		require.Empty(t, diff)
	}
}

// convertToDatabaseMetadata only used for tests.
func (d *DatabaseState) convertToDatabaseMetadata() *storepb.DatabaseSchemaMetadata {
	if d.deleted {
		return nil
	}
	return &storepb.DatabaseSchemaMetadata{
		Name:         d.name,
		CharacterSet: d.characterSet,
		Collation:    d.collation,
		Schemas:      d.convertToSchemaMetadataList(),
		Extensions:   []*storepb.ExtensionMetadata{},
	}
}

// convertToSchemaMetadataList only used for tests.
func (d *DatabaseState) convertToSchemaMetadataList() []*storepb.SchemaMetadata {
	result := []*storepb.SchemaMetadata{}

	for _, schema := range d.schemaSet {
		schemaMeta := &storepb.SchemaMetadata{
			Name:   schema.name,
			Tables: schema.convertToTableMetadataList(),
			// TODO(rebelice): convert views if needed.
			Views:     []*storepb.ViewMetadata{},
			Functions: []*storepb.FunctionMetadata{},
			Streams:   []*storepb.StreamMetadata{},
			Tasks:     []*storepb.TaskMetadata{},
		}

		result = append(result, schemaMeta)
	}

	slices.SortFunc(result, func(x, y *storepb.SchemaMetadata) int {
		if x.Name < y.Name {
			return -1
		} else if x.Name > y.Name {
			return 1
		}
		return 0
	})

	return result
}

// convertToTableMetadataList only used for tests.
func (s *SchemaState) convertToTableMetadataList() []*storepb.TableMetadata {
	result := []*storepb.TableMetadata{}

	for _, table := range s.tableSet {
		tableMeta := &storepb.TableMetadata{
			Name:        table.name,
			Columns:     table.convertToColumnMetadataList(),
			Indexes:     table.convertToIndexMetadataList(),
			ForeignKeys: []*storepb.ForeignKeyMetadata{},
		}

		if table.engine != nil {
			tableMeta.Engine = *table.engine
		}

		if table.collation != nil {
			tableMeta.Collation = *table.collation
		}

		if table.comment != nil {
			tableMeta.Comment = *table.comment
		}

		result = append(result, tableMeta)
	}

	slices.SortFunc(result, func(x, y *storepb.TableMetadata) int {
		if x.Name < y.Name {
			return -1
		} else if x.Name > y.Name {
			return 1
		}
		return 0
	})

	return result
}

// convertToIndexMetadataList only used for tests.
func (t *TableState) convertToIndexMetadataList() []*storepb.IndexMetadata {
	result := []*storepb.IndexMetadata{}

	for _, index := range t.indexSet {
		indexMeta := &storepb.IndexMetadata{
			Name:        index.name,
			Expressions: index.ExpressionList(),
			Unique:      index.Unique(),
			Primary:     index.Primary(),
		}

		if index.indexType != nil {
			indexMeta.Type = *index.indexType
		}

		if index.visible != nil {
			indexMeta.Visible = *index.visible
		}

		if index.comment != nil {
			indexMeta.Comment = *index.comment
		}

		result = append(result, indexMeta)
	}

	slices.SortFunc(result, func(x, y *storepb.IndexMetadata) int {
		if x.Name < y.Name {
			return -1
		} else if x.Name > y.Name {
			return 1
		}
		return 0
	})

	return result
}

// convertToColumnMetadataList only used for tests.
func (t *TableState) convertToColumnMetadataList() []*storepb.ColumnMetadata {
	result := []*storepb.ColumnMetadata{}

	for _, column := range t.columnSet {
		columnMeta := &storepb.ColumnMetadata{
			Name:     column.name,
			Nullable: column.Nullable(),
			Type:     column.Type(),
		}

		if column.defaultValue != nil {
			columnMeta.Default = *column.defaultValue
		}

		if column.characterSet != nil {
			columnMeta.CharacterSet = *column.characterSet
		}

		if column.collation != nil {
			columnMeta.Collation = *column.collation
		}

		if column.comment != nil {
			columnMeta.Comment = *column.comment
		}

		if column.position != nil {
			columnMeta.Position = int32(*column.position)
		}

		result = append(result, columnMeta)
	}

	slices.SortFunc(result, func(x, y *storepb.ColumnMetadata) int {
		if x.Position != y.Position {
			if x.Position < y.Position {
				return -1
			}
			return 1
		}
		if x.Name < y.Name {
			return -1
		} else if x.Name > y.Name {
			return 1
		}
		return 0
	})

	return result
}
