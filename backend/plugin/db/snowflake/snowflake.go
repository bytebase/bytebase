// Package snowflake is the plugin for Snowflake driver.
package snowflake

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	snow "github.com/snowflakedb/gosnowflake"
	"go.uber.org/zap"
)

var (
	bytebaseDatabase = "BYTEBASE"

	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(db.Snowflake, newDriver)
}

// Driver is the Snowflake driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	dbType        db.Type
	db            *sql.DB
	databaseName  string
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a Snowflake driver.
func (driver *Driver) Open(_ context.Context, dbType db.Type, config db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	dsn, loggedDSN, err := buildSnowflakeDSN(config)
	if err != nil {
		return nil, err
	}

	log.Debug("Opening Snowflake driver",
		zap.String("dsn", loggedDSN),
		zap.String("environment", connCtx.EnvironmentID),
		zap.String("database", connCtx.InstanceID),
	)
	db, err := sql.Open("snowflake", dsn)
	if err != nil {
		panic(err)
	}
	driver.dbType = dbType
	driver.db = db
	driver.connectionCtx = connCtx
	driver.databaseName = config.Database

	return driver, nil
}

// buildSnowflakeDSN returns the Snowflake Golang DSN and a redacted version of the DSN.
func buildSnowflakeDSN(config db.ConnectionConfig) (string, string, error) {
	snowConfig := &snow.Config{
		Database: fmt.Sprintf(`"%s"`, config.Database),
		User:     config.Username,
		Password: config.Password,
	}
	// Host can also be account e.g. xma12345, or xma12345@host_ip where host_ip is the proxy server IP.
	if strings.Contains(config.Host, "@") {
		parts := strings.Split(config.Host, "@")
		if len(parts) != 2 {
			return "", "", errors.Errorf("expected one @ in the host at most, got %q", config.Host)
		}
		snowConfig.Account = parts[0]
		snowConfig.Host = parts[1]
	} else {
		snowConfig.Account = config.Host
	}
	dsn, err := snow.DSN(snowConfig)
	if err != nil {
		return "", "", errors.Wrapf(err, "failed to build Snowflake DSN")
	}
	snowConfig.Password = "xxxxxx"
	redactedDSN, err := snow.DSN(snowConfig)
	if err != nil {
		// nolint
		log.Warn("failed to build redacted Snowflake DSN", zap.Error(err))
		return dsn, "", nil
	}
	return dsn, redactedDSN, nil
}

// Close closes the driver.
func (driver *Driver) Close(context.Context) error {
	return driver.db.Close()
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetType returns the database type.
func (*Driver) GetType() db.Type {
	return db.Snowflake
}

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
}

// getVersion gets the version.
func (driver *Driver) getVersion(ctx context.Context) (string, error) {
	query := "SELECT CURRENT_VERSION()"
	var version string
	if err := driver.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", util.FormatErrorWithQuery(err, query)
	}
	return version, nil
}

func (driver *Driver) getDatabases(ctx context.Context) ([]string, error) {
	txn, err := driver.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()

	databases, err := getDatabasesTxn(ctx, txn)
	if err != nil {
		return nil, err
	}

	if err := txn.Commit(); err != nil {
		return nil, err
	}

	return databases, nil
}

func getDatabasesTxn(ctx context.Context, tx *sql.Tx) ([]string, error) {
	// Filter inbound shared databases because they are immutable and we cannot get their DDLs.
	inboundDatabases := make(map[string]bool)
	shareQuery := "SHOW SHARES"
	shareRows, err := tx.Query(shareQuery)
	if err != nil {
		return nil, err
	}
	defer shareRows.Close()

	cols, err := shareRows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	// created_on, kind, name, database_name, to, owner, comment, listing_global_name.
	if len(cols) < 8 {
		return nil, nil
	}
	values := make([]*sql.NullString, len(cols))
	refs := make([]any, len(cols))
	for i := 0; i < len(cols); i++ {
		refs[i] = &values[i]
	}
	for shareRows.Next() {
		if err := shareRows.Scan(refs...); err != nil {
			return nil, err
		}
		if values[1].String == "INBOUND" {
			inboundDatabases[values[3].String] = true
		}
	}
	if err := shareRows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, shareQuery)
	}

	query := `
		SELECT
			DATABASE_NAME
		FROM SNOWFLAKE.INFORMATION_SCHEMA.DATABASES`
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(
			&name,
		); err != nil {
			return nil, err
		}

		if _, ok := inboundDatabases[name]; !ok {
			databases = append(databases, name)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, util.FormatErrorWithQuery(err, query)
	}
	return databases, nil
}

// Execute executes a SQL statement and returns the affected rows.
func (driver *Driver) Execute(ctx context.Context, statement string, _ bool, _ db.ExecuteOptions) (int64, error) {
	count := 0
	f := func(stmt string) error {
		count++
		return nil
	}

	if err := util.ApplyMultiStatements(strings.NewReader(statement), f); err != nil {
		return 0, err
	}

	if count <= 0 {
		return 0, nil
	}

	tx, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	mctx, err := snow.WithMultiStatement(ctx, count)
	if err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(mctx, statement)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
	if err != nil {
		log.Debug("rowsAffected returns error", zap.Error(err))
		return 0, nil
	}
	return rowsAffected, nil
}

// QueryConn querys a SQL statement in a given connection.
func (*Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]any, error) {
	return util.Query(ctx, db.Snowflake, conn, statement, queryContext)
}

// QueryConn2 queries a SQL statement in a given connection.
func (driver *Driver) QueryConn2(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	// TODO(rebelice): support multiple queries in a single statement.
	var results []*v1pb.QueryResult

	result, err := driver.querySingleSQL(ctx, conn, parser.SingleSQL{Text: statement}, queryContext)
	if err != nil {
		results = append(results, &v1pb.QueryResult{
			Error: err.Error(),
		})
	} else {
		results = append(results, result)
	}

	return results, nil
}

func getStatementWithResultLimit(stmt string, limit int) string {
	// return fmt.Sprintf("WITH result AS (%s) SELECT * FROM result LIMIT %d;", stmt, limit)
	return fmt.Sprintf("SELECT * FROM (%s) LIMIT %d", stmt, limit)
}

func (*Driver) querySingleSQL(ctx context.Context, conn *sql.Conn, singleSQL parser.SingleSQL, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	statement := strings.TrimRight(singleSQL.Text, " \n\t;")

	stmt := statement
	if !strings.HasPrefix(stmt, "EXPLAIN") && queryContext.Limit > 0 {
		stmt = getStatementWithResultLimit(stmt, queryContext.Limit)
	}

	// Snowflake doesn't support READ ONLY transactions.
	// https://github.com/snowflakedb/gosnowflake/blob/0450f0b16a4679b216baecd3fd6cdce739dbb683/connection.go#L166
	if queryContext.ReadOnly {
		queryContext.ReadOnly = false
	}

	startTime := time.Now()
	result, err := util.Query2(ctx, db.Snowflake, conn, stmt, queryContext)
	if err != nil {
		return nil, err
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = statement
	return result, nil
}

// RunStatement runs a SQL statement in a given connection.
func (*Driver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return util.RunStatement(ctx, parser.Snowflake, conn, statement)
}
