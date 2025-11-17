package catalog

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/store/model"
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
		var proto *storepb.DatabaseSchemaMetadata
		if originDatabase != nil {
			// Make a deep copy to avoid mutation across tests
			proto = cloneDatabaseSchemaMetadata(originDatabase)
		} else {
			proto = &storepb.DatabaseSchemaMetadata{}
		}

		// Create DatabaseMetadata for walk-through
		state := model.NewDatabaseMetadata(proto, !test.IgnoreCaseSensitive, !test.IgnoreCaseSensitive)

		asts, _ := sm.GetASTsForChecks(engineType, test.Statement)
		err := WalkThrough(state, engineType, asts)
		if err != nil {
			err, yes := err.(*WalkThroughError)
			require.True(t, yes)
			require.Equal(t, test.Err, err)
			continue
		}
		require.NoError(t, err, test.Statement)

		// Skip comparison if want is empty (error cases)
		if test.Want == "" {
			continue
		}

		want := &storepb.DatabaseSchemaMetadata{}
		err = common.ProtojsonUnmarshaler.Unmarshal([]byte(test.Want), want)
		require.NoError(t, err)
		// Sort proto for deterministic comparison
		state.SortProto()
		result := state.GetProto()
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
		var proto *storepb.DatabaseSchemaMetadata
		if originDatabase != nil {
			// Make a deep copy to avoid mutation across tests
			proto = cloneDatabaseSchemaMetadata(originDatabase)
		} else {
			proto = &storepb.DatabaseSchemaMetadata{}
		}

		// Create DatabaseMetadata for walk-through
		state := model.NewDatabaseMetadata(proto, !test.IgnoreCaseSensitive, !test.IgnoreCaseSensitive)

		// Parse using ANTLR parser instead of legacy parser
		parseResult, parseErr := pgparser.ParsePostgreSQL(test.Statement)
		if parseErr != nil {
			t.Fatalf("Failed to parse SQL with ANTLR: %v\nSQL: %s", parseErr, test.Statement)
		}

		// Call WalkThrough with ANTLR tree
		err := WalkThrough(state, engineType, parseResult)
		if err != nil {
			err, yes := err.(*WalkThroughError)
			require.True(t, yes)
			require.Equal(t, test.Err, err)
			continue
		}
		require.NoError(t, err, test.Statement)

		// Skip comparison if want is empty (error cases)
		if test.Want == "" {
			continue
		}

		want := &storepb.DatabaseSchemaMetadata{}
		err = common.ProtojsonUnmarshaler.Unmarshal([]byte(test.Want), want)
		require.NoError(t, err)
		// Sort proto for deterministic comparison
		state.SortProto()
		result := state.GetProto()
		diff := cmp.Diff(want, result, protocmp.Transform())
		require.Empty(t, diff)
	}
}

// cloneDatabaseSchemaMetadata creates a deep copy of the database schema metadata.
func cloneDatabaseSchemaMetadata(original *storepb.DatabaseSchemaMetadata) *storepb.DatabaseSchemaMetadata {
	if original == nil {
		return nil
	}

	clone := &storepb.DatabaseSchemaMetadata{
		Name:         original.Name,
		CharacterSet: original.CharacterSet,
		Collation:    original.Collation,
		Owner:        original.Owner,
		SearchPath:   original.SearchPath,
		Schemas:      make([]*storepb.SchemaMetadata, 0, len(original.Schemas)),
	}

	for _, schema := range original.Schemas {
		cloneSchema := &storepb.SchemaMetadata{
			Name:   schema.Name,
			Tables: make([]*storepb.TableMetadata, 0, len(schema.Tables)),
			Views:  make([]*storepb.ViewMetadata, 0, len(schema.Views)),
		}

		for _, table := range schema.Tables {
			cloneTable := &storepb.TableMetadata{
				Name:        table.Name,
				Engine:      table.Engine,
				Collation:   table.Collation,
				Comment:     table.Comment,
				Columns:     make([]*storepb.ColumnMetadata, 0, len(table.Columns)),
				Indexes:     make([]*storepb.IndexMetadata, 0, len(table.Indexes)),
				ForeignKeys: make([]*storepb.ForeignKeyMetadata, 0, len(table.ForeignKeys)),
			}

			for _, col := range table.Columns {
				cloneCol := &storepb.ColumnMetadata{
					Name:         col.Name,
					Position:     col.Position,
					Default:      col.Default,
					Nullable:     col.Nullable,
					Type:         col.Type,
					CharacterSet: col.CharacterSet,
					Collation:    col.Collation,
					Comment:      col.Comment,
				}
				cloneTable.Columns = append(cloneTable.Columns, cloneCol)
			}

			for _, idx := range table.Indexes {
				cloneIdx := &storepb.IndexMetadata{
					Name:        idx.Name,
					Expressions: append([]string{}, idx.Expressions...),
					Type:        idx.Type,
					Unique:      idx.Unique,
					Primary:     idx.Primary,
					Visible:     idx.Visible,
					Comment:     idx.Comment,
				}
				cloneTable.Indexes = append(cloneTable.Indexes, cloneIdx)
			}

			cloneSchema.Tables = append(cloneSchema.Tables, cloneTable)
		}

		for _, view := range schema.Views {
			cloneView := &storepb.ViewMetadata{
				Name:       view.Name,
				Definition: view.Definition,
			}
			if len(view.DependencyColumns) > 0 {
				cloneView.DependencyColumns = make([]*storepb.DependencyColumn, 0, len(view.DependencyColumns))
				for _, dep := range view.DependencyColumns {
					cloneDep := &storepb.DependencyColumn{
						Schema: dep.Schema,
						Table:  dep.Table,
						Column: dep.Column,
					}
					cloneView.DependencyColumns = append(cloneView.DependencyColumns, cloneDep)
				}
			}
			cloneSchema.Views = append(cloneSchema.Views, cloneView)
		}

		clone.Schemas = append(clone.Schemas, cloneSchema)
	}

	return clone
}
