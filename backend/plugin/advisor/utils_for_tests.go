package advisor

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
	database "github.com/bytebase/bytebase/backend/plugin/db"
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

		payload, err := SetDefaultSQLReviewRulePayload(rule, dbType)
		require.NoError(t, err)

		ruleList := []*SQLReviewRule{
			{
				Type:    rule,
				Level:   SchemaRuleLevelWarning,
				Payload: string(payload),
			},
		}

		ctx := SQLReviewCheckContext{
			Charset:         "",
			Collation:       "",
			DbType:          dbType,
			Catalog:         &testCatalog{finder: finder},
			Driver:          nil,
			Context:         context.Background(),
			CurrentSchema:   "SYS",
			CurrentDatabase: "TEST_DB",
		}

		adviceList, err := SQLReviewCheck(tc.Statement, ruleList, ctx)
		// Sort adviceList by (line, content)
		slices.SortFunc[Advice](adviceList, func(i, j Advice) bool {
			if i.Line != j.Line {
				return i.Line < j.Line
			}
			return i.Content < j.Content
		})
		require.NoError(t, err)
		if record {
			tests[i].Want = adviceList
		} else {
			require.Equalf(t, tc.Want, adviceList, "rule: %s, statements: %s", rule, tc.Statement)
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

// GetType implements the Driver interface.
func (*MockDriver) GetType() database.Type {
	return database.Type("MOCK")
}

// GetDB gets the database.
func (*MockDriver) GetDB() *sql.DB {
	return nil
}

// Execute implements the Driver interface.
func (*MockDriver) Execute(_ context.Context, _ string, _ bool, _ database.ExecuteOptions) (int64, error) {
	return 0, nil
}

// QueryConn2 queries a SQL statement in a given connection.
func (*MockDriver) QueryConn2(_ context.Context, _ *sql.Conn, _ string, _ *database.QueryContext) ([]*v1pb.QueryResult, error) {
	return nil, nil
}

// RunStatement implements the Driver interface.
func (*MockDriver) RunStatement(_ context.Context, _ *sql.Conn, _ string) ([]*v1pb.QueryResult, error) {
	return nil, nil
}

// QueryConn implements the Driver interface.
func (*MockDriver) QueryConn(_ context.Context, _ *sql.Conn, statement string, _ *database.QueryContext) ([]any, error) {
	switch statement {
	// For TestStatementDMLDryRun
	case "EXPLAIN DELETE FROM tech_book":
		return nil, errors.Errorf("MockDriver disallows it")
	// For TestStatementAffectedRowLimit
	case "EXPLAIN UPDATE tech_book SET id = 1":
		return []any{
			nil,
			nil,
			[]any{
				[]any{nil, nil, nil, nil, nil, nil, nil, nil, nil, 1000, nil, nil},
			},
		}, nil
	// For TestInsertRowLimit
	case "EXPLAIN INSERT INTO tech_book SELECT * FROM tech_book":
		return []any{
			nil,
			nil,
			[]any{
				nil,
				[]any{nil, nil, nil, nil, nil, nil, nil, nil, nil, 1000, nil, nil},
			},
		}, nil
	}
	return []any{
		nil,
		nil,
		[]any{
			[]any{nil, nil, nil, nil, nil, nil, nil, nil, nil, 1, nil, nil},
		},
	}, nil
}

// SyncInstance implements the Driver interface.
func (*MockDriver) SyncInstance(_ context.Context) (*database.InstanceMetadata, error) {
	return nil, nil
}

// SyncDBSchema implements the Driver interface.
func (*MockDriver) SyncDBSchema(_ context.Context) (*storepb.DatabaseMetadata, error) {
	return nil, nil
}

// Dump implements the Driver interface.
func (*MockDriver) Dump(_ context.Context, _ io.Writer, _ bool) (string, error) {
	return "", nil
}

// Restore implements the Driver interface.
func (*MockDriver) Restore(_ context.Context, _ io.Reader) error {
	return nil
}

// CreateRole creates the role.
func (*MockDriver) CreateRole(_ context.Context, _ *database.DatabaseRoleUpsertMessage) (*database.DatabaseRoleMessage, error) {
	return nil, nil
}

// UpdateRole updates the role.
func (*MockDriver) UpdateRole(_ context.Context, _ string, _ *database.DatabaseRoleUpsertMessage) (*database.DatabaseRoleMessage, error) {
	return nil, nil
}

// FindRole finds the role by name.
func (*MockDriver) FindRole(_ context.Context, _ string) (*database.DatabaseRoleMessage, error) {
	return nil, nil
}

// ListRole lists the role.
func (*MockDriver) ListRole(_ context.Context) ([]*database.DatabaseRoleMessage, error) {
	return nil, nil
}

// DeleteRole deletes the role by name.
func (*MockDriver) DeleteRole(_ context.Context, _ string) error {
	return nil
}

// SyncSlowQuery implements the Driver interface.
func (*MockDriver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, nil
}

// CheckSlowQueryLogEnabled checks if slow query log is enabled.
func (*MockDriver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return nil
}

// SetDefaultSQLReviewRulePayload sets the default payload for this rule.
func SetDefaultSQLReviewRulePayload(ruleTp SQLReviewRuleType, dbType db.Type) (string, error) {
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
		SchemaRuleAddNotNullColumnRequireDefault,
		SchemaRuleCurrentTimeColumnCountLimit,
		SchemaRuleColumnRequireDefault,
		SchemaRuleSchemaBackwardCompatibility,
		SchemaRuleDropEmptyDatabase,
		SchemaRuleIndexNoDuplicateColumn,
		SchemaRuleIndexPKTypeLimit,
		SchemaRuleStatementDisallowAddColumnWithDefault,
		SchemaRuleCreateIndexConcurrently,
		SchemaRuleStatementAddCheckNotValid,
		SchemaRuleStatementDisallowAddNotNull,
		SchemaRuleIndexTypeNoBlob,
		SchemaRuleIdentifierNoKeyword,
		SchemaRuleTableNameNoKeyword:
	case SchemaRuleTableDropNamingConvention:
		payload, err = json.Marshal(NamingRulePayload{
			Format: "_delete$",
		})
	case SchemaRuleTableNaming:
		fallthrough
	case SchemaRuleColumnNaming:
		format := "^[a-z]+(_[a-z]+)*$"
		maxLength := 64
		if dbType == db.Snowflake {
			format = "^[A-Z]+(_[A-Z]+)*$"
		}
		payload, err = json.Marshal(NamingRulePayload{
			Format:    format,
			MaxLength: maxLength,
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
			List: []string{"JSON", "BINARY_FLOAT"},
		})
	case SchemaRuleColumnMaximumCharacterLength:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 20,
		})
	case SchemaRuleColumnMaximumVarcharLength:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 2560,
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
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 20,
		})
	case SchemaRuleIndexPrimaryKeyTypeAllowlist:
		payload, err = json.Marshal(StringArrayTypeRulePayload{
			List: []string{"serial", "bigserial", "int", "bigint"},
		})
	case SchemaRuleIdentifierCase:
		payload, err = json.Marshal(NamingCaseRulePayload{
			Upper: true,
		})
	default:
		return "", errors.Errorf("unknown SQL review type for default payload: %s", ruleTp)
	}

	if err != nil {
		return "", err
	}
	return string(payload), nil
}
