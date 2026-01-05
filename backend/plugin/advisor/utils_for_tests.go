package advisor

import (
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	database "github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ database.Driver = (*MockDriver)(nil)
)

const (
	// MockOldIndexName is the mock old index for test.
	MockOldIndexName = "old_index"
	// MockOldUKName is the mock old unique key for test.
	MockOldUKName = "old_uk"
	// MockOldMySQLPKName is the mock old primary key for MySQL test.
	MockOldMySQLPKName = "PRIMARY"
	// MockOldPostgreSQLPKName is the mock old primary key for PostgreSQL test.
	MockOldPostgreSQLPKName = "old_pk"
	// MockTableName is the mock table for test.
	MockTableName = "tech_book"
)

var (
	// MockIndexColumnList is the mock index column list for test.
	MockIndexColumnList = []string{"id", "name"}
	// MockMySQLDatabase is the mock MySQL database for test.
	MockMySQLDatabase = &storepb.DatabaseSchemaMetadata{
		Name: "test",
		Schemas: []*storepb.SchemaMetadata{
			{
				Tables: []*storepb.TableMetadata{
					{
						Name: MockTableName,
						Columns: []*storepb.ColumnMetadata{
							{
								Name: "id",
								Type: "int",
							},
							{
								Name: "name",
								Type: "varchar(255)",
							},
						},
						Indexes: []*storepb.IndexMetadata{
							{
								Name:        MockOldMySQLPKName,
								Expressions: []string{"id", "name"},
								Unique:      true,
								Primary:     true,
							},
							{
								Name:        MockOldUKName,
								Expressions: []string{"id", "name"},
								Unique:      true,
							},
							{
								Name:        MockOldIndexName,
								Expressions: []string{"id", "name"},
							},
						},
					},
				},
			},
		},
	}
	// MockPostgreSQLDatabase is the mock PostgreSQL database for test.
	MockPostgreSQLDatabase = &storepb.DatabaseSchemaMetadata{
		Name: "test",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*storepb.TableMetadata{
					{
						Name: MockTableName,
						Columns: []*storepb.ColumnMetadata{
							{Name: "id"},
							{Name: "name"},
						},
						Indexes: []*storepb.IndexMetadata{
							{
								Name:        MockOldPostgreSQLPKName,
								Expressions: []string{"id", "name"},
								Unique:      true,
								Primary:     true,
							},
							{
								Name:        MockOldUKName,
								Expressions: []string{"id", "name"},
								Unique:      true,
							},
							{
								Name:        MockOldIndexName,
								Expressions: []string{"id", "name"},
							},
						},
					},
				},
			},
			{
				Name: "bbdataarchive", // MySQL backup database for testing
			},
		},
	}
	MockMSSQLDatabase = &storepb.DatabaseSchemaMetadata{
		Name: "master",
		Schemas: []*storepb.SchemaMetadata{
			{
				Name: "dbo",
				Tables: []*storepb.TableMetadata{
					{
						Name: "pokes",
						Indexes: []*storepb.IndexMetadata{
							{
								Name:        "idx_0",
								Expressions: []string{"c1", "c2", "c3"},
							},
							{
								Name:        "idx_1",
								Expressions: []string{"c10", "c20"},
							},
						},
					},
					{Name: "pokes2"},
					{Name: "pokes3"},
				},
			},
		},
	}
)

// TestCase is the data struct for test.
type TestCase struct {
	Statement string            `yaml:"statement"`
	Want      []*storepb.Advice `yaml:"want,omitempty"`
}

// RunSQLReviewRuleTest helps to test the SQL review rule.
// The rule parameter should be a complete rule with Type, Level, and Payload already set.
func RunSQLReviewRuleTest(t *testing.T, rule *storepb.SQLReviewRule, dbType storepb.Engine, record bool) {
	require.NotNil(t, rule, "rule must not be nil")
	require.NotEqual(t, storepb.SQLReviewRule_TYPE_UNSPECIFIED, rule.Type, "rule type must be specified")

	var tests []TestCase

	fileName := strings.Map(func(r rune) rune {
		switch r {
		case '.', '-':
			return '_'
		default:
			return r
		}
	}, strings.ToLower(rule.Type.String()))
	filepath := filepath.Join("test", fileName+".yaml")
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err, rule.Type)

	sm := sheet.NewManager()
	for i, tc := range tests {
		// Set metadata and database-specific settings based on engine type
		var schemaMetadata *storepb.DatabaseSchemaMetadata
		curDB := "TEST_DB"
		isCaseSensitive := false

		switch dbType {
		case storepb.Engine_POSTGRES:
			schemaMetadata = MockPostgreSQLDatabase
			isCaseSensitive = true
		case storepb.Engine_MSSQL:
			schemaMetadata = MockMSSQLDatabase
			curDB = "master"
		case storepb.Engine_MYSQL:
			schemaMetadata = MockMySQLDatabase
		default:
			// Fallback to MySQL for engines without specific mock
			schemaMetadata = MockMySQLDatabase
		}

		// Create OriginalMetadata as DatabaseMetadata (read-only)
		// Clone to avoid mutations affecting future test cases
		metadata, ok := proto.Clone(schemaMetadata).(*storepb.DatabaseSchemaMetadata)
		require.True(t, ok, "failed to clone metadata")
		originalMetadata := model.NewDatabaseMetadata(metadata, nil, nil, dbType, isCaseSensitive)

		// Create FinalMetadata as DatabaseMetadata (mutable for walk-through)
		// Clone to avoid mutations affecting future test cases
		metadata, ok = proto.Clone(schemaMetadata).(*storepb.DatabaseSchemaMetadata)
		require.True(t, ok, "failed to clone metadata")
		finalMetadata := model.NewDatabaseMetadata(metadata, nil, nil, dbType, isCaseSensitive)

		// Use the rule provided by the caller (already has Type, Level, and Payload set)
		ruleList := []*storepb.SQLReviewRule{rule}

		checkCtx := Context{
			DBType:            dbType,
			OriginalMetadata:  originalMetadata,
			FinalMetadata:     finalMetadata,
			Driver:            nil,
			CurrentDatabase:   curDB,
			DBSchema:          schemaMetadata,
			EnablePriorBackup: true, // Enable backup for testing
			NoAppendBuiltin:   true,
			TenantMode:        true,
		}

		adviceList, err := SQLReviewCheck(t.Context(), sm, tc.Statement, ruleList, checkCtx)
		// Sort adviceList by (line, content)
		slices.SortFunc(adviceList, func(x, y *storepb.Advice) int {
			if x.GetStartPosition() == nil || y.GetStartPosition() == nil {
				if x.GetStartPosition() == nil && y.GetStartPosition() == nil {
					return 0
				} else if x.GetStartPosition() == nil {
					return -1
				}
				return 1
			}
			if x.GetStartPosition().Line != y.GetStartPosition().Line {
				if x.GetStartPosition().Line < y.GetStartPosition().Line {
					return -1
				}
				return 1
			}
			if x.Content < y.Content {
				return -1
			} else if x.Content > y.Content {
				return 1
			}
			return 0
		})

		require.NoError(t, err)
		if record {
			tests[i].Want = adviceList
		} else {
			require.Equalf(t, tc.Want, adviceList, "rule: %s, statements: %s", rule.Type, tc.Statement)
		}
	}

	if record {
		err := yamlFile.Close()
		require.NoError(t, err)
		byteValue, err := yaml.Marshal(tests)
		require.NoError(t, err)
		err = os.WriteFile(filepath, byteValue, 0644)
		require.NoError(t, err)
	}
}

// MockDriver is the driver for test only.
type MockDriver struct {
}

// Open implements the Driver interface.
func (d *MockDriver) Open(_ context.Context, _ storepb.Engine, _ database.ConnectionConfig) (database.Driver, error) {
	return d, nil
}

// Close implements the Driver interface.
func (*MockDriver) Close(_ context.Context) error {
	return nil
}

// Ping implements the Driver interface.
func (*MockDriver) Ping(_ context.Context) error {
	return nil
}

// GetDB gets the database.
func (*MockDriver) GetDB() *sql.DB {
	return nil
}

// Execute implements the Driver interface.
func (*MockDriver) Execute(_ context.Context, _ string, _ database.ExecuteOptions) (int64, error) {
	return 0, nil
}

// QueryConn queries a SQL statement in a given connection.
func (*MockDriver) QueryConn(_ context.Context, _ *sql.Conn, _ string, _ database.QueryContext) ([]*v1pb.QueryResult, error) {
	return nil, nil
}

// SyncInstance implements the Driver interface.
func (*MockDriver) SyncInstance(_ context.Context) (*database.InstanceMetadata, error) {
	return nil, nil
}

// SyncDBSchema implements the Driver interface.
func (*MockDriver) SyncDBSchema(_ context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	return nil, nil
}

// Dump implements the Driver interface.
func (*MockDriver) Dump(_ context.Context, _ io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	return nil
}
