package catalog

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gopkg.in/yaml.v3"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	// Register postgresql parser driver.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
)

type testData struct {
	Statement string
	// Use custom yaml tag to avoid generate field name `ignorecasesensitive`.
	IgnoreCaseSensitive bool `yaml:"ignore_case_sensitive"`
	Want                string
	Err                 *WalkThroughError
}

func TestMySQLWalkThrough(t *testing.T) {
	originDatabase := &storepb.DatabaseSchemaMetadata{
		Name: "test",
	}

	tests := []string{
		"mysql_walk_through",
	}

	for _, test := range tests {
		runWalkThroughTest(t, test, storepb.Engine_MYSQL, originDatabase, false /* record */)
	}
}

func TestMySQLV2WalkThrough(t *testing.T) {
	originDatabase := &storepb.DatabaseSchemaMetadata{
		Name: "test",
	}

	tests := []string{
		"mysqlv2_walk_through",
	}

	for _, test := range tests {
		runWalkThroughTest(t, test, storepb.Engine_ENGINE_UNSPECIFIED, originDatabase, false /* record */)
	}
}

func TestMySQLWalkThroughForIncomplete(t *testing.T) {
	tests := []string{
		"mysql_walk_through_for_incomplete",
	}

	for _, test := range tests {
		runWalkThroughTest(t, test, storepb.Engine_MYSQL, nil, false /* record */)
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
						DependentColumns: []*storepb.DependentColumn{
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
		runWalkThroughTest(t, test, storepb.Engine_POSTGRES, originDatabase, false /* record */)
	}
}

func convertInterfaceSliceToStringSlice(slice []any) []string {
	var res []string
	for _, item := range slice {
		res = append(res, item.(string))
	}
	return res
}

func runWalkThroughTest(t *testing.T, file string, engineType storepb.Engine, originDatabase *storepb.DatabaseSchemaMetadata, record bool) {
	tests := []testData{}
	filepath := filepath.Join("test", file+".yaml")
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err)

	for i, test := range tests {
		var state *DatabaseState
		if originDatabase != nil {
			state = newDatabaseState(originDatabase, &FinderContext{CheckIntegrity: true, EngineType: engineType, IgnoreCaseSensitive: test.IgnoreCaseSensitive})
		} else {
			finder := NewEmptyFinder(&FinderContext{CheckIntegrity: false, EngineType: engineType, IgnoreCaseSensitive: test.IgnoreCaseSensitive})
			state = finder.Origin
		}

		err := state.WalkThrough(test.Statement)
		if err != nil {
			if record {
				walkThroughError, ok := err.(*WalkThroughError)
				require.True(t, ok)
				tests[i].Err = walkThroughError
				tests[i].Want = ""
			} else {
				err, yes := err.(*WalkThroughError)
				require.True(t, yes)
				if err.Payload != nil {
					actualPayloadText, yes := err.Payload.([]string)
					require.True(t, yes)
					expectedPayloadText := convertInterfaceSliceToStringSlice(test.Err.Payload.([]any))
					err.Payload = nil
					test.Err.Payload = nil
					require.Equal(t, test.Err, err)
					require.Equal(t, expectedPayloadText, actualPayloadText)
				} else {
					require.Equal(t, test.Err, err)
				}
			}
			continue
		}
		require.NoError(t, err, test.Statement)

		if record {
			tests[i].Want = protojson.Format(state.convertToDatabaseMetadata())
			tests[i].Err = nil
		} else {
			want := &storepb.DatabaseSchemaMetadata{}
			err = protojson.Unmarshal([]byte(test.Want), want)
			require.NoError(t, err)
			result := state.convertToDatabaseMetadata()
			diff := cmp.Diff(want, result, protocmp.Transform())
			require.Equal(t, "", diff, test.Statement)
		}
	}

	if record {
		byteValue, err := yaml.Marshal(tests)
		require.NoError(t, err)
		err = os.WriteFile(filepath, byteValue, 0644)
		require.NoError(t, err)
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

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
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

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
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

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
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
			columnMeta.DefaultValue = &storepb.ColumnMetadata_Default{Default: &wrapperspb.StringValue{Value: *column.defaultValue}}
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

	sort.Slice(result, func(i, j int) bool {
		if result[i].Position != result[j].Position {
			return result[i].Position < result[j].Position
		}
		return result[i].Name < result[j].Name
	})

	return result
}
