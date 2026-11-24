package advisor

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"

	"github.com/bytebase/bytebase/backend/common/yamltest"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store/model"
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
	// MockMySQLDatabase is the mock MySQL database for test.
	MockMySQLDatabase = &metadatapb.DatabaseSchemaMetadata{
		Name: "test",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Tables: []*metadatapb.TableMetadata{
					{
						Name: MockTableName,
						Columns: []*metadatapb.ColumnMetadata{
							{
								Name: "id",
								Type: "int",
							},
							{
								Name: "name",
								Type: "varchar(255)",
							},
							{
								Name: "creator",
								Type: "varchar(255)",
							},
						},
						Indexes: []*metadatapb.IndexMetadata{
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
					{
						Name: "orders",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "order_id", Type: "int"},
							{Name: "customer_name", Type: "varchar(255)"},
							{Name: "amount", Type: "decimal(10,2)"},
							{Name: "note", Type: "varchar(255)"},
							// name also exists here (non-indexed) so tests can verify
							// qualified lookups distinguish tech_book.name (indexed) from orders.name.
							{Name: "name", Type: "varchar(255)"},
						},
						Indexes: []*metadatapb.IndexMetadata{
							{
								Name:        "PRIMARY",
								Expressions: []string{"order_id"},
								Unique:      true,
								Primary:     true,
							},
							{
								Name:        "idx_customer",
								Expressions: []string{"customer_name"},
							},
							{
								Name:        "idx_amount",
								Expressions: []string{"amount"},
							},
						},
					},
				},
			},
		},
	}
	// MockPostgreSQLDatabase is the mock PostgreSQL database for test.
	MockPostgreSQLDatabase = &metadatapb.DatabaseSchemaMetadata{
		Name: "test",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "public",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: MockTableName,
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer", Position: 1},
							{Name: "name", Type: "text", Position: 2},
							{Name: "creator", Type: "text", Position: 3},
						},
						Indexes: []*metadatapb.IndexMetadata{
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
					{
						Name: "orders",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "order_id", Type: "integer", Position: 1},
							{Name: "customer_name", Type: "text", Position: 2},
							{Name: "amount", Type: "numeric", Position: 3},
							{Name: "note", Type: "text", Position: 4},
							{Name: "name", Type: "text", Position: 5},
						},
						Indexes: []*metadatapb.IndexMetadata{
							{Name: "orders_pkey", Expressions: []string{"order_id"}, Unique: true, Primary: true},
							{Name: "idx_orders_customer", Expressions: []string{"customer_name"}},
							{Name: "idx_orders_amount", Expressions: []string{"amount"}},
						},
					},
					{
						Name: "products",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "product_id", Type: "integer", Position: 1},
							{Name: "title", Type: "text", Position: 2},
							{Name: "price", Type: "numeric", Position: 3},
						},
						Indexes: []*metadatapb.IndexMetadata{
							{Name: "products_pkey", Expressions: []string{"product_id"}, Unique: true, Primary: true},
							{Name: "idx_products_price", Expressions: []string{"price"}},
						},
					},
					{
						Name: "no_index_table",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "col_a", Type: "integer", Position: 1},
							{Name: "col_b", Type: "text", Position: 2},
						},
					},
					{
						Name: "insert_target",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "id", Type: "integer", Position: 1},
							{Name: "name", Type: "text", Position: 2},
						},
					},
				},
			},
			{
				Name: "bbdataarchive", // MySQL backup database for testing
			},
		},
	}
	MockMSSQLDatabase = &metadatapb.DatabaseSchemaMetadata{
		Name: "master",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "dbo",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "pokes",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "c1", Type: "int"},
							{Name: "c2", Type: "int"},
							{Name: "c3", Type: "int"},
							{Name: "c10", Type: "int"},
							{Name: "c20", Type: "int"},
							{Name: "foo", Type: "int"},
							{Name: "bar", Type: "int"},
						},
						Indexes: []*metadatapb.IndexMetadata{
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
					{
						Name: "pokes2",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "foo", Type: "int"},
							{Name: "bar", Type: "int"},
							// c1 also exists here (non-indexed) so tests can verify
							// qualified lookups distinguish pokes.c1 (indexed) from pokes2.c1.
							{Name: "c1", Type: "int"},
						},
					},
					{Name: "pokes3"},
				},
			},
			{
				Name: "sales",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "orders",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "order_id", Type: "int"},
							{Name: "customer_id", Type: "int"},
							{Name: "note", Type: "varchar"},
						},
						Indexes: []*metadatapb.IndexMetadata{
							{
								Name:        "PRIMARY",
								Expressions: []string{"order_id"},
								Unique:      true,
								Primary:     true,
							},
							{
								Name:        "idx_customer",
								Expressions: []string{"customer_id"},
							},
						},
					},
				},
			},
		},
	}
	// MockOracleDatabase is the mock Oracle database for test. Oracle's sync
	// (omni's oracle/driver) stores SchemaMetadata{Name: ""} — a single
	// anonymous schema containing every object. The mock mirrors this so
	// advisors resolve tables the same way in tests and production.
	MockOracleDatabase = &metadatapb.DatabaseSchemaMetadata{
		Name: "TEST_DB",
		Schemas: []*metadatapb.SchemaMetadata{
			{
				Name: "",
				Tables: []*metadatapb.TableMetadata{
					{
						Name: "TECH_BOOK",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "ID", Type: "NUMBER"},
							{Name: "NAME", Type: "VARCHAR2(255)"},
							{Name: "CREATOR", Type: "VARCHAR2(255)"},
						},
						Indexes: []*metadatapb.IndexMetadata{
							{Name: "TECH_BOOK_PK", Expressions: []string{"ID", "NAME"}, Unique: true, Primary: true},
							{Name: "IDX_TECH_BOOK_NAME", Expressions: []string{"ID", "NAME"}},
						},
					},
					{
						Name: "ORDERS",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "ORDER_ID", Type: "NUMBER"},
							{Name: "CUSTOMER_NAME", Type: "VARCHAR2(255)"},
							{Name: "AMOUNT", Type: "NUMBER"},
							{Name: "NOTE", Type: "VARCHAR2(255)"},
							{Name: "NAME", Type: "VARCHAR2(255)"},
						},
						Indexes: []*metadatapb.IndexMetadata{
							{Name: "ORDERS_PK", Expressions: []string{"ORDER_ID"}, Unique: true, Primary: true},
							{Name: "IDX_ORDERS_CUSTOMER", Expressions: []string{"CUSTOMER_NAME"}},
							{Name: "IDX_ORDERS_AMOUNT", Expressions: []string{"AMOUNT"}},
						},
					},
					{
						Name: "PRODUCTS",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "PRODUCT_ID", Type: "NUMBER"},
							{Name: "TITLE", Type: "VARCHAR2(255)"},
							{Name: "PRICE", Type: "NUMBER"},
						},
						Indexes: []*metadatapb.IndexMetadata{
							{Name: "PRODUCTS_PK", Expressions: []string{"PRODUCT_ID"}, Unique: true, Primary: true},
							{Name: "IDX_PRODUCTS_PRICE", Expressions: []string{"PRICE"}},
						},
					},
					{
						Name: "NO_INDEX_TABLE",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "COL_A", Type: "NUMBER"},
							{Name: "COL_B", Type: "VARCHAR2(255)"},
						},
					},
					{
						Name: "INSERT_TARGET",
						Columns: []*metadatapb.ColumnMetadata{
							{Name: "ID", Type: "NUMBER"},
							{Name: "NAME", Type: "VARCHAR2(255)"},
						},
					},
				},
			},
		},
	}
)

// testCase is the data struct for advisor YAML test fixtures.
type testCase struct {
	Statement string            `yaml:"statement"`
	Want      []*storepb.Advice `yaml:"want,omitempty"`
}

// RunSQLReviewRuleTest helps to test the SQL review rule.
// The rule parameter should be a complete rule with Type, Level, and Payload already set.
func RunSQLReviewRuleTest(t *testing.T, rule *storepb.SQLReviewRule, dbType storepb.Engine, record bool) {
	require.NotNil(t, rule, "rule must not be nil")
	require.NotEqual(t, storepb.SQLReviewRule_TYPE_UNSPECIFIED, rule.Type, "rule type must be specified")

	var tests []testCase

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
		var schemaMetadata *metadatapb.DatabaseSchemaMetadata
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
		case storepb.Engine_ORACLE:
			schemaMetadata = MockOracleDatabase
			curDB = "TEST_DB"
			// Match production: store.IsObjectCaseSensitive(Oracle) returns
			// true via the default branch in backend/store/instance.go. Test
			// fixtures use upper-case identifiers like the real Oracle sync.
			isCaseSensitive = true
		default:
			// Fallback to MySQL for engines without specific mock
			schemaMetadata = MockMySQLDatabase
		}

		// Create OriginalMetadata as DatabaseMetadata (read-only)
		// Clone to avoid mutations affecting future test cases
		metadata, ok := proto.Clone(schemaMetadata).(*metadatapb.DatabaseSchemaMetadata)
		require.True(t, ok, "failed to clone metadata")
		originalMetadata := model.NewDatabaseMetadata(metadata, nil, nil, dbType, isCaseSensitive)

		// Create FinalMetadata as DatabaseMetadata (mutable for walk-through)
		// Clone to avoid mutations affecting future test cases
		metadata, ok = proto.Clone(schemaMetadata).(*metadatapb.DatabaseSchemaMetadata)
		require.True(t, ok, "failed to clone metadata")
		finalMetadata := model.NewDatabaseMetadata(metadata, nil, nil, dbType, isCaseSensitive)

		// Use the rule provided by the caller (already has Type, Level, and Payload set).
		// Old test fixtures omit Engine for brevity; production rule configs must set it.
		testRule, ok := proto.Clone(rule).(*storepb.SQLReviewRule)
		require.True(t, ok, "failed to clone rule")
		if testRule.Engine == storepb.Engine_ENGINE_UNSPECIFIED {
			testRule.Engine = dbType
		}
		ruleList := []*storepb.SQLReviewRule{testRule}

		checkCtx := Context{
			DBType:                dbType,
			OriginalMetadata:      originalMetadata,
			FinalMetadata:         finalMetadata,
			CurrentDatabase:       curDB,
			DBSchema:              schemaMetadata,
			IsObjectCaseSensitive: isCaseSensitive,
			EnablePriorBackup:     true, // Enable backup for testing
			NoAppendBuiltin:       true,
			TenantMode:            true,
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
		yamltest.Record(t, filepath, tests)
	}
}

// ProbeCall records how a rule asked the connection to probe.
type ProbeCall struct {
	PreExecutions []string
	Autocommit    bool
	Statement     string
}

// ColumnLookupCall records one catalogue lookup a rule asked for.
type ColumnLookupCall struct {
	Database string
	Table    string
	Column   string
}

// ProbeRecorder is a db.Prober for tests. It records the probes, estimates and
// catalogue lookups a rule asks for, so a rule's statements can be asserted
// without a server, and answers from the fields a test sets.
type ProbeRecorder struct {
	Probes    []ProbeCall
	Estimates []string
	Lookups   []ColumnLookupCall

	// ProbeFunc, when set, answers each probe instead of ProbeErr and Result.
	ProbeFunc func(pCtx db.ProbeContext, statement string) (db.ProbeResult, error)
	// ProbeErr makes every probe fail, as a refused statement does.
	ProbeErr error
	// Result is what a probe answers. Empty means no rows.
	Result db.ProbeResult
	// Rows is the row count an estimate answers.
	Rows int64
	// EstimateErr makes every estimate fail.
	EstimateErr error
	// Extras is what a catalogue lookup answers, by "database.table.column".
	Extras map[string]string
	// LookupErr makes every catalogue lookup fail.
	LookupErr error
}

// Probe runs the probe and records it.
func (r *ProbeRecorder) Probe(_ context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	r.Probes = append(r.Probes, ProbeCall{PreExecutions: pCtx.PreExecutions, Autocommit: pCtx.Autocommit, Statement: statement})
	if r.ProbeFunc != nil {
		return r.ProbeFunc(pCtx, statement)
	}
	if r.ProbeErr != nil {
		return db.ProbeResult{}, r.ProbeErr
	}
	return r.Result, nil
}

// CountAffectedRows records the statement and answers Rows.
func (r *ProbeRecorder) CountAffectedRows(_ context.Context, statement string) (int64, error) {
	r.Estimates = append(r.Estimates, statement)
	if r.EstimateErr != nil {
		return 0, r.EstimateErr
	}
	return r.Rows, nil
}

// ColumnExtra records the lookup and answers Extras.
func (r *ProbeRecorder) ColumnExtra(_ context.Context, database, table, column string) (string, error) {
	r.Lookups = append(r.Lookups, ColumnLookupCall{Database: database, Table: table, Column: column})
	if r.LookupErr != nil {
		return "", r.LookupErr
	}
	return r.Extras[database+"."+table+"."+column], nil
}

// ProbeOnly is the seam with neither capability: a connection that can run a
// statement but cannot answer what a driver must answer itself.
type ProbeOnly struct {
	Recorder *ProbeRecorder
}

// Probe runs the probe and records it.
func (p ProbeOnly) Probe(ctx context.Context, pCtx db.ProbeContext, statement string) (db.ProbeResult, error) {
	return p.Recorder.Probe(ctx, pCtx, statement)
}
