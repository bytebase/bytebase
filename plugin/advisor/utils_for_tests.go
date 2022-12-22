package advisor

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/plugin/advisor/db"
	database "github.com/bytebase/bytebase/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
	MockMySQLDatabase = &storepb.DatabaseMetadata{
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
	MockPostgreSQLDatabase = &storepb.DatabaseMetadata{
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
		},
	}
)

// TestCase is the data struct for test.
type TestCase struct {
	Statement string   `yaml:"statement"`
	Want      []Advice `yaml:"want"`
}

type testCatalog struct {
	finder *catalog.Finder
}

func (c *testCatalog) GetFinder() *catalog.Finder {
	return c.finder
}

// RunSQLReviewRuleTest helps to test the SQL review rule.
func RunSQLReviewRuleTest(t *testing.T, rule SQLReviewRuleType, dbType db.Type, record bool) {
	var tests []TestCase

	fileName := strings.Map(func(r rune) rune {
		switch r {
		case '.', '-':
			return '_'
		default:
			return r
		}
	}, string(rule))
	filepath := filepath.Join("test", fileName+".yaml")
	yamlFile, err := os.Open(filepath)
	require.NoError(t, err)
	defer yamlFile.Close()

	byteValue, err := io.ReadAll(yamlFile)
	require.NoError(t, err)
	err = yaml.Unmarshal(byteValue, &tests)
	require.NoError(t, err, rule)

	for i, tc := range tests {
		database := MockMySQLDatabase
		if dbType == db.Postgres {
			database = MockPostgreSQLDatabase
		}
		finder := catalog.NewFinder(database, &catalog.FinderContext{CheckIntegrity: true, EngineType: dbType})

		payload, err := SetDefaultSQLReviewRulePayload(rule)
		require.NoError(t, err)

		ruleList := []*SQLReviewRule{
			{
				Type:    rule,
				Level:   SchemaRuleLevelWarning,
				Payload: string(payload),
			},
		}

		ctx := SQLReviewCheckContext{
			Charset:   "",
			Collation: "",
			DbType:    dbType,
			Catalog:   &testCatalog{finder: finder},
			Driver:    nil,
			Context:   context.Background(),
		}

		adviceList, err := SQLReviewCheck(tc.Statement, ruleList, ctx)
		require.NoError(t, err)
		if record {
			tests[i].Want = adviceList
		} else {
			require.Equal(t, tc.Want, adviceList, tc.Statement)
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

// RandomString returns random string with specific length.
func RandomString(length int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz")
	rand.Seed(time.Now().UnixNano())

	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// MockDriver is the driver for test only.
type MockDriver struct {
}

// Open implements the Driver interface.
func (d *MockDriver) Open(_ context.Context, _ database.Type, _ database.ConnectionConfig, _ database.ConnectionContext) (database.Driver, error) {
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

// GetDBConnection implements the Driver interface.
func (*MockDriver) GetDBConnection(_ context.Context, _ string) (*sql.DB, error) {
	return nil, nil
}

// Execute implements the Driver interface.
func (*MockDriver) Execute(_ context.Context, _ string, _ bool) (int64, error) {
	return 0, nil
}

// Query implements the Driver interface.
func (*MockDriver) Query(_ context.Context, statement string, _ *database.QueryContext) ([]interface{}, error) {
	switch statement {
	// For TestStatementDMLDryRun
	case "EXPLAIN DELETE FROM tech_book":
		return nil, errors.Errorf("MockDriver disallows it")
	// For TestStatementAffectedRowLimit
	case "EXPLAIN UPDATE tech_book SET id = 1":
		return []interface{}{
			nil,
			nil,
			[]interface{}{
				[]interface{}{nil, nil, nil, nil, nil, nil, nil, nil, nil, 1000, nil, nil},
			},
		}, nil
	// For TestInsertRowLimit
	case "EXPLAIN INSERT INTO tech_book SELECT * FROM tech_book":
		return []interface{}{
			nil,
			nil,
			[]interface{}{
				nil,
				[]interface{}{nil, nil, nil, nil, nil, nil, nil, nil, nil, 1000, nil, nil},
			},
		}, nil
	}
	return []interface{}{
		nil,
		nil,
		[]interface{}{
			[]interface{}{nil, nil, nil, nil, nil, nil, nil, nil, nil, 1, nil, nil},
		},
	}, nil
}

// SyncInstance implements the Driver interface.
func (*MockDriver) SyncInstance(_ context.Context) (*database.InstanceMeta, error) {
	return nil, nil
}

// SyncDBSchema implements the Driver interface.
func (*MockDriver) SyncDBSchema(_ context.Context, _ string) (*database.Schema, []*storepb.ForeignKeyMetadata, error) {
	return nil, nil, nil
}

// NeedsSetupMigration implements the Driver interface.
func (*MockDriver) NeedsSetupMigration(_ context.Context) (bool, error) {
	return false, nil
}

// SetupMigrationIfNeeded implements the Driver interface.
func (*MockDriver) SetupMigrationIfNeeded(_ context.Context) error {
	return nil
}

// ExecuteMigration implements the Driver interface.
func (*MockDriver) ExecuteMigration(_ context.Context, _ *database.MigrationInfo, _ string) (int64, string, error) {
	return 0, "", nil
}

// FindMigrationHistoryList implements the Driver interface.
func (*MockDriver) FindMigrationHistoryList(_ context.Context, _ *database.MigrationHistoryFind) ([]*database.MigrationHistory, error) {
	return nil, nil
}

// Dump implements the Driver interface.
func (*MockDriver) Dump(_ context.Context, _ string, _ io.Writer, _ bool) (string, error) {
	return "", nil
}

// Restore implements the Driver interface.
func (*MockDriver) Restore(_ context.Context, _ io.Reader) error {
	return nil
}

// CreateRole creates the role.
func (*MockDriver) CreateRole(_ context.Context, _ *v1pb.DatabaseRoleUpsert) (*v1pb.DatabaseRole, error) {
	return nil, nil
}

// UpdateRole updates the role.
func (*MockDriver) UpdateRole(_ context.Context, _ string, _ *v1pb.DatabaseRoleUpsert) (*v1pb.DatabaseRole, error) {
	return nil, nil
}

// FindRole finds the role by name.
func (*MockDriver) FindRole(_ context.Context, _ string) (*v1pb.DatabaseRole, error) {
	return nil, nil
}

// ListRole lists the role.
func (*MockDriver) ListRole(_ context.Context) ([]*v1pb.DatabaseRole, error) {
	return nil, nil
}

// DeleteRole deletes the role by name.
func (*MockDriver) DeleteRole(_ context.Context, _ string) error {
	return nil
}

// SetDefaultSQLReviewRulePayload sets the default payload for this rule.
func SetDefaultSQLReviewRulePayload(ruleTp SQLReviewRuleType) (string, error) {
	var payload []byte
	var err error
	switch ruleTp {
	case SchemaRuleMySQLEngine,
		SchemaRuleStatementNoSelectAll,
		SchemaRuleStatementRequireWhere,
		SchemaRuleStatementNoLeadingWildcardLike,
		SchemaRuleStatementDisallowCommit,
		SchemaRuleStatementDisallowLimit,
		SchemaRuleStatementDisallowOrderBy,
		SchemaRuleStatementMergeAlterTable,
		SchemaRuleStatementInsertMustSpecifyColumn,
		SchemaRuleStatementInsertDisallowOrderByRand,
		SchemaRuleStatementDMLDryRun,
		SchemaRuleTableRequirePK,
		SchemaRuleTableNoFK,
		SchemaRuleTableDisallowPartition,
		SchemaRuleColumnNotNull,
		SchemaRuleColumnDisallowChangeType,
		SchemaRuleColumnSetDefaultForNotNull,
		SchemaRuleColumnDisallowChange,
		SchemaRuleColumnDisallowChangingOrder,
		SchemaRuleColumnAutoIncrementMustInteger,
		SchemaRuleColumnDisallowSetCharset,
		SchemaRuleColumnAutoIncrementMustUnsigned,
		SchemaRuleCurrentTimeColumnCountLimit,
		SchemaRuleColumnRequireDefault,
		SchemaRuleSchemaBackwardCompatibility,
		SchemaRuleDropEmptyDatabase,
		SchemaRuleIndexNoDuplicateColumn,
		SchemaRuleIndexPKTypeLimit,
		SchemaRuleIndexTypeNoBlob:
	case SchemaRuleTableDropNamingConvention:
		payload, err = json.Marshal(NamingRulePayload{
			Format: "_delete$",
		})
	case SchemaRuleTableNaming:
		fallthrough
	case SchemaRuleColumnNaming:
		payload, err = json.Marshal(NamingRulePayload{
			Format:    "^[a-z]+(_[a-z]+)*$",
			MaxLength: 64,
		})
	case SchemaRuleIDXNaming:
		payload, err = json.Marshal(NamingRulePayload{
			Format:    "^$|^idx_{{table}}_{{column_list}}$",
			MaxLength: 64,
		})
	case SchemaRulePKNaming:
		payload, err = json.Marshal(NamingRulePayload{
			Format:    "^$|^pk_{{table}}_{{column_list}}$",
			MaxLength: 64,
		})
	case SchemaRuleUKNaming:
		payload, err = json.Marshal(NamingRulePayload{
			Format:    "^$|^uk_{{table}}_{{column_list}}$",
			MaxLength: 64,
		})
	case SchemaRuleFKNaming:
		payload, err = json.Marshal(NamingRulePayload{
			Format:    "^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$",
			MaxLength: 64,
		})
	case SchemaRuleAutoIncrementColumnNaming:
		payload, err = json.Marshal(NamingRulePayload{
			Format:    "^id$",
			MaxLength: 64,
		})
	case SchemaRuleStatementInsertRowLimit, SchemaRuleStatementAffectedRowLimit:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 5,
		})
	case SchemaRuleTableCommentConvention, SchemaRuleColumnCommentConvention:
		payload, err = json.Marshal(CommentConventionRulePayload{
			Required:  true,
			MaxLength: 10,
		})
	case SchemaRuleRequiredColumn:
		payload, err = json.Marshal(StringArrayTypeRulePayload{
			List: []string{
				"id",
				"created_ts",
				"updated_ts",
				"creator_id",
				"updater_id",
			},
		})
	case SchemaRuleColumnTypeDisallowList:
		payload, err = json.Marshal(StringArrayTypeRulePayload{
			List: []string{"JSON"},
		})
	case SchemaRuleColumnMaximumCharacterLength:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 20,
		})
	case SchemaRuleColumnAutoIncrementInitialValue:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 20,
		})
	case SchemaRuleIndexKeyNumberLimit, SchemaRuleIndexTotalNumberLimit:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 5,
		})
	case SchemaRuleCharsetAllowlist:
		payload, err = json.Marshal(StringArrayTypeRulePayload{
			List: []string{"utf8mb4", "UTF8"},
		})
	case SchemaRuleCollationAllowlist:
		payload, err = json.Marshal(StringArrayTypeRulePayload{
			List: []string{"utf8mb4_0900_ai_ci"},
		})
	case SchemaRuleCommentLength:
		payload, err = json.Marshal(CommentConventionRulePayload{
			Required:  true,
			MaxLength: 20,
		})
	default:
		return "", errors.Errorf("unknown SQL review type for default payload: %s", ruleTp)
	}

	if err != nil {
		return "", err
	}
	return string(payload), nil
}
