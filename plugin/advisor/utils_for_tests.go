package advisor

import (
	"context"
	"database/sql"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/plugin/advisor/db"
	database "github.com/bytebase/bytebase/plugin/db"
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
	MockMySQLDatabase = &catalog.Database{
		Name:   "test",
		DbType: db.MySQL,
		SchemaList: []*catalog.Schema{
			{
				TableList: []*catalog.Table{
					{
						Name: MockTableName,
						ColumnList: []*catalog.Column{
							{
								Name: "id",
								Type: "int",
							},
							{
								Name: "name",
								Type: "varchar(255)",
							},
						},
						IndexList: []*catalog.Index{
							{
								Name:           MockOldMySQLPKName,
								ExpressionList: []string{"id", "name"},
								Unique:         true,
								Primary:        true,
							},
							{
								Name:           MockOldUKName,
								ExpressionList: []string{"id", "name"},
								Unique:         true,
							},
							{
								Name:           MockOldIndexName,
								ExpressionList: []string{"id", "name"},
							},
						},
					},
				},
			},
		},
	}
	// MockPostgreSQLDatabase is the mock PostgreSQL database for test.
	MockPostgreSQLDatabase = &catalog.Database{
		Name:   "test",
		DbType: db.Postgres,
		SchemaList: []*catalog.Schema{
			{
				Name: "public",
				TableList: []*catalog.Table{
					{
						Name: MockTableName,
						ColumnList: []*catalog.Column{
							{Name: "id"},
							{Name: "name"},
						},
						IndexList: []*catalog.Index{
							{
								Name:           MockOldPostgreSQLPKName,
								ExpressionList: []string{"id", "name"},
								Unique:         true,
								Primary:        true,
							},
							{
								Name:           MockOldUKName,
								ExpressionList: []string{"id", "name"},
								Unique:         true,
							},
							{
								Name:           MockOldIndexName,
								ExpressionList: []string{"id", "name"},
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
	Statement string
	Want      []Advice
}

// RunSQLReviewRuleTests helps to test the SQL review rule.
func RunSQLReviewRuleTests(
	t *testing.T,
	tests []TestCase,
	adv Advisor,
	rule *SQLReviewRule,
	database *catalog.Database,
) {
	ctx := Context{
		Charset:   "",
		Collation: "",
		Rule:      rule,
		Driver:    nil,
		Context:   context.Background(),
	}
	for _, tc := range tests {
		finder := catalog.NewFinder(database, &catalog.FinderContext{CheckIntegrity: true})
		if database.DbType == db.MySQL || database.DbType == db.TiDB {
			err := finder.WalkThrough(tc.Statement)
			require.NoError(t, err, tc.Statement)
		}
		ctx.Catalog = finder
		adviceList, err := adv.Check(ctx, tc.Statement)
		require.NoError(t, err)
		assert.Equal(t, tc.Want, adviceList, tc.Statement)
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
func (*MockDriver) SyncDBSchema(_ context.Context, _ string) (*database.Schema, error) {
	return nil, nil
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
