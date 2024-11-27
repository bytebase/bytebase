package advisor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
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
				Name: "bbdataarchive",
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
	Statement  string                                        `yaml:"statement"`
	ChangeType storepb.PlanCheckRunConfig_ChangeDatabaseType `yaml:"changeType"`
	Want       []*storepb.Advice                             `yaml:"want,omitempty"`
}

type testCatalog struct {
	finder *catalog.Finder
}

func (c *testCatalog) GetFinder() *catalog.Finder {
	return c.finder
}

// RunSQLReviewRuleTest helps to test the SQL review rule.
func RunSQLReviewRuleTest(t *testing.T, rule SQLReviewRuleType, dbType storepb.Engine, needMetaData bool, record bool) {
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

	sm := sheet.NewManager(nil)
	for i, tc := range tests {
		// Add engine types here for mocked database metadata.
		var schemaMetadata *storepb.DatabaseSchemaMetadata
		curDB := "TEST_DB"
		if needMetaData {
			switch dbType {
			case storepb.Engine_POSTGRES:
				schemaMetadata = MockPostgreSQLDatabase
			case storepb.Engine_MSSQL:
				curDB = "master"
				schemaMetadata = MockMSSQLDatabase
			case storepb.Engine_MYSQL:
				schemaMetadata = MockMySQLDatabase
			default:
				panic(fmt.Sprintf("%s doesn't have mocked metadata support", storepb.Engine_name[int32(dbType)]))
			}
		}

		database := MockMySQLDatabase
		if dbType == storepb.Engine_POSTGRES {
			database = MockPostgreSQLDatabase
		}
		finder := catalog.NewFinder(database, &catalog.FinderContext{CheckIntegrity: true, EngineType: dbType})

		payload, err := SetDefaultSQLReviewRulePayload(rule, dbType)
		require.NoError(t, err)

		ruleList := []*storepb.SQLReviewRule{
			{
				Type:    string(rule),
				Level:   storepb.SQLReviewRuleLevel_WARNING,
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
			CurrentDatabase: curDB,
			DBSchema:        schemaMetadata,
			ChangeType:      tc.ChangeType,
			PreUpdateBackupDetail: &storepb.PreUpdateBackupDetail{
				Database: "instances/instanceName/databases/bbdataarchive",
			},
			NoAppendBuiltin:          true,
			UsePostgresDatabaseOwner: true,
		}

		adviceList, err := SQLReviewCheck(sm, tc.Statement, ruleList, ctx)
		// Sort adviceList by (line, content)
		sort.Slice(adviceList, func(i, j int) bool {
			if adviceList[i].GetStartPosition() == nil || adviceList[j].GetStartPosition() == nil {
				return adviceList[i].GetStartPosition() == nil
			}
			if adviceList[i].GetStartPosition().Line != adviceList[j].GetStartPosition().Line {
				return adviceList[i].GetStartPosition().Line < adviceList[j].GetStartPosition().Line
			}
			return adviceList[i].Content < adviceList[j].Content
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

// SyncSlowQuery implements the Driver interface.
func (*MockDriver) SyncSlowQuery(_ context.Context, _ time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, nil
}

// CheckSlowQueryLogEnabled checks if slow query log is enabled.
func (*MockDriver) CheckSlowQueryLogEnabled(_ context.Context) error {
	return nil
}

// SetDefaultSQLReviewRulePayload sets the default payload for this rule.
func SetDefaultSQLReviewRulePayload(ruleTp SQLReviewRuleType, dbType storepb.Engine) (string, error) {
	var payload []byte
	var err error
	switch ruleTp {
	case SchemaRuleMySQLEngine,
		BuiltinRulePriorBackupCheck,
		SchemaRuleFullyQualifiedObjectName,
		SchemaRuleStatementNoSelectAll,
		SchemaRuleStatementRequireWhereForSelect,
		SchemaRuleStatementRequireWhereForUpdateDelete,
		SchemaRuleStatementNoLeadingWildcardLike,
		SchemaRuleStatementDisallowOnDelCascade,
		SchemaRuleStatementDisallowRemoveTblCascade,
		SchemaRuleStatementDisallowCommit,
		SchemaRuleStatementDisallowLimit,
		SchemaRuleStatementDisallowOrderBy,
		SchemaRuleStatementMergeAlterTable,
		SchemaRuleStatementInsertMustSpecifyColumn,
		SchemaRuleStatementInsertDisallowOrderByRand,
		SchemaRuleStatementDMLDryRun,
		SchemaRuleStatementDisallowUsingFilesort,
		SchemaRuleStatementDisallowUsingTemporary,
		SchemaRuleTableRequirePK,
		SchemaRuleTableNoFK,
		SchemaRuleTableDisallowPartition,
		SchemaRuleTableDisallowTrigger,
		SchemaRuleTableNoDuplicateIndex,
		SchemaRuleColumnNotNull,
		SchemaRuleColumnDisallowChangeType,
		SchemaRuleColumnSetDefaultForNotNull,
		SchemaRuleColumnDisallowChange,
		SchemaRuleColumnDisallowChangingOrder,
		SchemaRuleColumnDisallowDropInIndex,
		SchemaRuleColumnAutoIncrementMustInteger,
		SchemaRuleColumnDisallowSetCharset,
		SchemaRuleColumnAutoIncrementMustUnsigned,
		SchemaRuleAddNotNullColumnRequireDefault,
		SchemaRuleCurrentTimeColumnCountLimit,
		SchemaRuleColumnRequireDefault,
		SchemaRuleColumnDefaultDisallowVolatile,
		SchemaRuleSchemaBackwardCompatibility,
		SchemaRuleDropEmptyDatabase,
		SchemaRuleIndexNoDuplicateColumn,
		SchemaRuleIndexPKTypeLimit,
		SchemaRuleStatementDisallowAddColumnWithDefault,
		SchemaRuleStatementNonTransactional,
		SchemaRuleCreateIndexConcurrently,
		SchemaRuleStatementAddCheckNotValid,
		SchemaRuleStatementAddFKNotValid,
		SchemaRuleStatementDisallowAddNotNull,
		SchemaRuleStatementWhereNoEqualNull,
		SchemaRuleIndexTypeNoBlob,
		SchemaRuleIdentifierNoKeyword,
		SchemaRuleTableNameNoKeyword,
		SchemaRuleProcedureDisallowCreate,
		SchemaRuleEventDisallowCreate,
		SchemaRuleViewDisallowCreate,
		SchemaRuleFunctionDisallowCreate,
		SchemaRuleStatementCreateSpecifySchema,
		SchemaRuleStatementCheckSetRoleVariable,
		SchemaRuleStatementWhereDisallowFunctionsAndCaculations,
		SchemaRuleStatementDisallowMixInDDL,
		SchemaRuleStatementDisallowMixInDML,
		SchemaRuleStatementPriorBackupCheck,
		SchemaRuleStatementJoinStrictColumnAttrs,
		SchemaRuleStatementMaxExecutionTime,
		SchemaRuleStatementRequireAlgorithmOption,
		SchemaRuleStatementRequireLockOption,
		SchemaRuleTableDisallowSetCharset,
		SchemaRuleStatementDisallowCrossDBQueries,
		SchemaRuleIndexNotRedundant:
	case SchemaRuleTableDropNamingConvention:
		payload, err = json.Marshal(NamingRulePayload{
			Format: "_delete$",
		})
	case SchemaRuleTableNaming:
		fallthrough
	case SchemaRuleColumnNaming:
		format := "^[a-z]+(_[a-z]+)*$"
		maxLength := 64
		if dbType == storepb.Engine_SNOWFLAKE {
			format = "^[A-Z]+(_[A-Z]+)*$"
		} else if dbType == storepb.Engine_MSSQL {
			format = "^[A-Z]([_A-Za-z])*$"
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
	case SchemaRuleStatementMaximumJoinTableCount:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 2,
		})
	case SchemaRuleStatementWhereMaximumLogicalOperatorCount:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 2,
		})
	case SchemaRuleStatementMaximumLimitValue:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 1000,
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
	case SchemaRuleTableDisallowDDL:
		if dbType == storepb.Engine_MSSQL {
			payload, err = json.Marshal(StringArrayTypeRulePayload{
				List: []string{"MySchema.Identifier"},
			})
		} else {
			payload, err = json.Marshal(StringArrayTypeRulePayload{
				List: []string{"identifier"},
			})
		}
	case SchemaRuleTableDisallowDML:
		if dbType == storepb.Engine_MSSQL {
			payload, err = json.Marshal(StringArrayTypeRulePayload{
				List: []string{"MySchema.Identifier"},
			})
		} else {
			payload, err = json.Marshal(StringArrayTypeRulePayload{
				List: []string{"identifier"},
			})
		}
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
	case SchemaRuleIndexTypeAllowList:
		payload, err = json.Marshal(StringArrayTypeRulePayload{
			List: []string{"BTREE", "HASH"},
		})
	case SchemaRuleIdentifierCase:
		payload, err = json.Marshal(NamingCaseRulePayload{
			Upper: true,
		})
	case SchemaRuleFunctionDisallowList:
		payload, err = json.Marshal(StringArrayTypeRulePayload{
			List: []string{"rand", "uuid", "sleep"},
		})
	default:
		return "", errors.Errorf("unknown SQL review type for default payload: %s", ruleTp)
	}

	if err != nil {
		return "", err
	}
	return string(payload), nil
}
