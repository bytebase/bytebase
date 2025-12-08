package advisor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pkg/errors"
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
	EnableSDL bool              `yaml:"enableSDL"`
	Want      []*storepb.Advice `yaml:"want,omitempty"`
}

// RunSQLReviewRuleTest helps to test the SQL review rule.
func RunSQLReviewRuleTest(t *testing.T, rule storepb.SQLReviewRule_Type, dbType storepb.Engine, needMetaData bool, record bool) {
	var tests []TestCase

	fileName := strings.Map(func(r rune) rune {
		switch r {
		case '.', '-':
			return '_'
		default:
			return r
		}
	}, strings.ToLower(rule.String()))
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

		// Use the schemaMetadata if available, otherwise use mock database for catalog creation
		catalogMetadata := schemaMetadata
		if catalogMetadata == nil {
			if dbType == storepb.Engine_POSTGRES {
				catalogMetadata = MockPostgreSQLDatabase
			} else {
				catalogMetadata = MockMySQLDatabase
			}
		}

		isCaseSensitive := false
		if dbType == storepb.Engine_POSTGRES {
			isCaseSensitive = true
		}

		// Create OriginalMetadata as DatabaseMetadata (read-only)
		// Clone to avoid mutations affecting future test cases
		originalCatalogClone, ok := proto.Clone(catalogMetadata).(*storepb.DatabaseSchemaMetadata)
		require.True(t, ok, "failed to clone catalog metadata")
		originalMetadata := model.NewDatabaseMetadata(originalCatalogClone, nil, nil, dbType, isCaseSensitive)

		// Create FinalMetadata as DatabaseMetadata (mutable for walk-through)
		// Clone to avoid mutations affecting future test cases
		finalCatalogClone, ok := proto.Clone(catalogMetadata).(*storepb.DatabaseSchemaMetadata)
		require.True(t, ok, "failed to clone catalog metadata")
		finalMetadata := model.NewDatabaseMetadata(finalCatalogClone, nil, nil, dbType, isCaseSensitive)

		payload, err := SetDefaultSQLReviewRulePayload(rule, dbType)
		require.NoError(t, err)

		ruleList := []*storepb.SQLReviewRule{
			{
				Type:    rule,
				Level:   storepb.SQLReviewRule_WARNING,
				Payload: string(payload),
			},
		}

		checkCtx := Context{
			Charset:                  "",
			Collation:                "",
			DBType:                   dbType,
			OriginalMetadata:         originalMetadata,
			FinalMetadata:            finalMetadata,
			Driver:                   nil,
			CurrentDatabase:          curDB,
			DBSchema:                 schemaMetadata,
			EnableSDL:                tc.EnableSDL,
			EnablePriorBackup:        true, // Enable backup for testing
			NoAppendBuiltin:          true,
			UsePostgresDatabaseOwner: true,
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

// SetDefaultSQLReviewRulePayload sets the default payload for this rule.
func SetDefaultSQLReviewRulePayload(ruleTp storepb.SQLReviewRule_Type, dbType storepb.Engine) (string, error) {
	var payload []byte
	var err error
	switch ruleTp {
	case storepb.SQLReviewRule_ENGINE_MYSQL_USE_INNODB,
		storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK,
		storepb.SQLReviewRule_NAMING_FULLY_QUALIFIED,
		storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT,
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE,
		storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_ON_DEL_CASCADE,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_RM_TBL_CASCADE,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY,
		storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE,
		storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN,
		storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND,
		storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_USING_FILESORT,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_USING_TEMPORARY,
		storepb.SQLReviewRule_TABLE_REQUIRE_PK,
		storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY,
		storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION,
		storepb.SQLReviewRule_TABLE_DISALLOW_TRIGGER,
		storepb.SQLReviewRule_TABLE_NO_DUPLICATE_INDEX,
		storepb.SQLReviewRule_COLUMN_NO_NULL,
		storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE,
		storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL,
		storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE,
		storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER,
		storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX,
		storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER,
		storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET,
		storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED,
		storepb.SQLReviewRule_COLUMN_ADD_NOT_NULL_REQUIRE_DEFAULT,
		storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT,
		storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT,
		storepb.SQLReviewRule_COLUMN_DEFAULT_DISALLOW_VOLATILE,
		storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY,
		storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE,
		storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN,
		storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_COLUMN_WITH_DEFAULT,
		storepb.SQLReviewRule_STATEMENT_NON_TRANSACTIONAL,
		storepb.SQLReviewRule_INDEX_CREATE_CONCURRENTLY,
		storepb.SQLReviewRule_STATEMENT_ADD_CHECK_NOT_VALID,
		storepb.SQLReviewRule_STATEMENT_ADD_FOREIGN_KEY_NOT_VALID,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_NOT_NULL,
		storepb.SQLReviewRule_STATEMENT_WHERE_NO_EQUAL_NULL,
		storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB,
		storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD,
		storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD,
		storepb.SQLReviewRule_SYSTEM_PROCEDURE_DISALLOW_CREATE,
		storepb.SQLReviewRule_SYSTEM_EVENT_DISALLOW_CREATE,
		storepb.SQLReviewRule_SYSTEM_VIEW_DISALLOW_CREATE,
		storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOW_CREATE,
		storepb.SQLReviewRule_STATEMENT_CREATE_SPECIFY_SCHEMA,
		storepb.SQLReviewRule_STATEMENT_CHECK_SET_ROLE_VARIABLE,
		storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS,
		storepb.SQLReviewRule_STATEMENT_JOIN_STRICT_COLUMN_ATTRS,
		storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME,
		storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION,
		storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION,
		storepb.SQLReviewRule_TABLE_DISALLOW_SET_CHARSET,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_CROSS_DB_QUERIES,
		storepb.SQLReviewRule_INDEX_NOT_REDUNDANT:
	case storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION:
		payload, err = json.Marshal(NamingRulePayload{
			Format: "_delete$",
		})
	case storepb.SQLReviewRule_NAMING_TABLE, storepb.SQLReviewRule_NAMING_COLUMN:
		format := "^[a-z]+(_[a-z]+)*$"
		maxLength := 64
		switch dbType {
		case storepb.Engine_SNOWFLAKE:
			format = "^[A-Z]+(_[A-Z]+)*$"
		case storepb.Engine_MSSQL:
			format = "^[A-Z]([_A-Za-z])*$"
		default:
			// Use default format for other databases
		}
		payload, err = json.Marshal(NamingRulePayload{
			Format:    format,
			MaxLength: maxLength,
		})
	case storepb.SQLReviewRule_NAMING_INDEX_IDX:
		payload, err = json.Marshal(NamingRulePayload{
			Format:    "^$|^idx_{{table}}_{{column_list}}$",
			MaxLength: 64,
		})
	case storepb.SQLReviewRule_NAMING_INDEX_PK:
		payload, err = json.Marshal(NamingRulePayload{
			Format:    "^$|^pk_{{table}}_{{column_list}}$",
			MaxLength: 64,
		})
	case storepb.SQLReviewRule_NAMING_INDEX_UK:
		payload, err = json.Marshal(NamingRulePayload{
			Format:    "^$|^uk_{{table}}_{{column_list}}$",
			MaxLength: 64,
		})
	case storepb.SQLReviewRule_NAMING_INDEX_FK:
		payload, err = json.Marshal(NamingRulePayload{
			Format:    "^$|^fk_{{referencing_table}}_{{referencing_column}}_{{referenced_table}}_{{referenced_column}}$",
			MaxLength: 64,
		})
	case storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT:
		payload, err = json.Marshal(NamingRulePayload{
			Format:    "^id$",
			MaxLength: 64,
		})
	case storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 5,
		})
	case storepb.SQLReviewRule_STATEMENT_MAXIMUM_JOIN_TABLE_COUNT:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 2,
		})
	case storepb.SQLReviewRule_STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 2,
		})
	case storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 1000,
		})
	case storepb.SQLReviewRule_TABLE_COMMENT, storepb.SQLReviewRule_COLUMN_COMMENT:
		payload, err = json.Marshal(CommentConventionRulePayload{
			Required:  true,
			MaxLength: 10,
		})
	case storepb.SQLReviewRule_COLUMN_REQUIRED:
		payload, err = json.Marshal(StringArrayTypeRulePayload{
			List: []string{
				"id",
				"created_ts",
				"updated_ts",
				"creator_id",
				"updater_id",
			},
		})
	case storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST:
		payload, err = json.Marshal(StringArrayTypeRulePayload{
			List: []string{"JSON", "BINARY_FLOAT"},
		})
	case storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 20,
		})
	case storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 2560,
		})
	case storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 20,
		})
	case storepb.SQLReviewRule_TABLE_DISALLOW_DDL:
		if dbType == storepb.Engine_MSSQL {
			payload, err = json.Marshal(StringArrayTypeRulePayload{
				List: []string{"MySchema.Identifier"},
			})
		} else {
			payload, err = json.Marshal(StringArrayTypeRulePayload{
				List: []string{"identifier"},
			})
		}
	case storepb.SQLReviewRule_TABLE_DISALLOW_DML:
		if dbType == storepb.Engine_MSSQL {
			payload, err = json.Marshal(StringArrayTypeRulePayload{
				List: []string{"MySchema.Identifier"},
			})
		} else {
			payload, err = json.Marshal(StringArrayTypeRulePayload{
				List: []string{"identifier"},
			})
		}
	case storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 5,
		})
	case storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST:
		payload, err = json.Marshal(StringArrayTypeRulePayload{
			List: []string{"utf8mb4", "UTF8"},
		})
	case storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST:
		payload, err = json.Marshal(StringArrayTypeRulePayload{
			List: []string{"utf8mb4_0900_ai_ci"},
		})
	case storepb.SQLReviewRule_SYSTEM_COMMENT_LENGTH:
		payload, err = json.Marshal(NumberTypeRulePayload{
			Number: 20,
		})
	case storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST:
		payload, err = json.Marshal(StringArrayTypeRulePayload{
			List: []string{"serial", "bigserial", "int", "bigint"},
		})
	case storepb.SQLReviewRule_INDEX_TYPE_ALLOW_LIST:
		payload, err = json.Marshal(StringArrayTypeRulePayload{
			List: []string{"BTREE", "HASH"},
		})
	case storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE:
		payload, err = json.Marshal(NamingCaseRulePayload{
			Upper: true,
		})
	case storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOWED_LIST:
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
