// Package snowflake is the plugin for Snowflake driver.
package snowflake

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/youmark/pkcs8"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/standard"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"

	snow "github.com/snowflakedb/gosnowflake"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_SNOWFLAKE, newDriver)
}

// Driver is the Snowflake driver.
type Driver struct {
	connectionCtx db.ConnectionContext
	dbType        storepb.Engine
	db            *sql.DB
	databaseName  string
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a Snowflake driver.
func (driver *Driver) Open(_ context.Context, dbType storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	dsn, loggedDSN, err := buildSnowflakeDSN(config)
	if err != nil {
		return nil, err
	}

	slog.Debug("Opening Snowflake driver",
		slog.String("dsn", loggedDSN),
		slog.String("environment", config.ConnectionContext.EnvironmentID),
		slog.String("database", config.ConnectionContext.InstanceID),
	)
	db, err := sql.Open("snowflake", dsn)
	if err != nil {
		panic(err)
	}
	driver.dbType = dbType
	driver.db = db
	driver.connectionCtx = config.ConnectionContext
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
	if config.AuthenticationPrivateKey != "" {
		rsaPrivKey, err := decodeRSAPrivateKey(config.AuthenticationPrivateKey)
		if err != nil {
			return "", "", errors.Wrapf(err, "failed to decode rsa private key")
		}
		snowConfig.PrivateKey = rsaPrivKey
		snowConfig.Authenticator = snow.AuthTypeJwt
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
	if snowConfig.PrivateKey != nil {
		snowConfig.PrivateKey = nil
	}
	redactedDSN, err := snow.DSN(snowConfig)
	if err != nil {
		// nolint
		slog.Warn("failed to build redacted Snowflake DSN", log.BBError(err))
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
func (driver *Driver) Execute(ctx context.Context, statement string, _ db.ExecuteOptions) (int64, error) {
	singleSQLs, err := standard.SplitSQL(statement)
	if err != nil {
		return 0, err
	}
	singleSQLs = base.FilterEmptySQL(singleSQLs)
	if len(singleSQLs) == 0 {
		return 0, nil
	}

	count := len(singleSQLs)
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
		slog.Debug("rowsAffected returns error", log.BBError(err))
		return 0, nil
	}
	return rowsAffected, nil
}

// QueryConn queries a SQL statement in a given connection.
func (driver *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	// TODO(rebelice): support multiple queries in a single statement.
	var results []*v1pb.QueryResult

	result, err := driver.querySingleSQL(ctx, conn, base.SingleSQL{Text: statement}, queryContext)
	if err != nil {
		results = append(results, &v1pb.QueryResult{
			Error: err.Error(),
		})
	} else {
		results = append(results, result)
	}

	return results, nil
}

func (*Driver) querySingleSQL(ctx context.Context, conn *sql.Conn, singleSQL base.SingleSQL, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	statement := singleSQL.Text
	if queryContext != nil && queryContext.Explain {
		statement = fmt.Sprintf("EXPLAIN %s", statement)
	} else if queryContext != nil && queryContext.Limit > 0 {
		stmt, err := getStatementWithResultLimit(statement, queryContext.Limit)
		if err != nil {
			slog.Error("fail to add limit clause", "statement", statement, log.BBError(err))
			stmt = fmt.Sprintf("SELECT * FROM (%s) LIMIT %d", util.TrimStatement(stmt), queryContext.Limit)
		}
		statement = stmt
	}

	startTime := time.Now()
	rows, err := conn.QueryContext(ctx, statement)
	if err != nil {
		return nil, util.FormatErrorWithQuery(err, statement)
	}
	defer rows.Close()

	result, err := util.RowsToQueryResult(storepb.Engine_SNOWFLAKE, rows)
	if err != nil {
		// nolint
		return &v1pb.QueryResult{
			Error: err.Error(),
		}, nil
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = statement
	return result, nil
}

// RunStatement runs a SQL statement in a given connection.
func (*Driver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return util.RunStatement(ctx, storepb.Engine_SNOWFLAKE, conn, statement)
}

func decodeRSAPrivateKey(key string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return nil, errors.Errorf("failed to get private key PEM block from key")
	}
	switch block.Type {
	case "PRIVATE KEY":
		privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse pkcs8 private key")
		}
		rsaKey, ok := privateKey.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.Errorf("expected RSA private key, got %T", privateKey)
		}
		return rsaKey, nil
	case "ENCRYPTED PRIVATE KEY":
		// NOTE: As of Jun 2024, Golang's official library does not support passcode-encrypted PKCS8 private key. And
		// Snowflake do not introduce an external library to achieve this due to the security purposes.
		// We introduce https://pkg.go.dev/github.com/youmark/pkcs8 to help us to achieve this goal.
		// TODO(zp): Assume the passphrase is empty because we do not have input area in frontend.
		pk, err := pkcs8.ParsePKCS8PrivateKeyRSA([]byte(block.Bytes), []byte(""))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse pkcs8 private key to rsa private key with passphrase")
		}
		return pk, nil
	default:
		return nil, errors.Errorf("unsupported pem block type: %s", block.Type)
	}
}
