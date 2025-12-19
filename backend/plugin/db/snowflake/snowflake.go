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
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

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

func newDriver() db.Driver {
	return &Driver{}
}

// Open opens a Snowflake driver.
func (d *Driver) Open(_ context.Context, dbType storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
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
	d.dbType = dbType
	d.db = db
	d.connectionCtx = config.ConnectionContext
	d.databaseName = config.ConnectionContext.DatabaseName
	return d, nil
}

// buildSnowflakeDSN returns the Snowflake Golang DSN and a redacted version of the DSN.
func buildSnowflakeDSN(config db.ConnectionConfig) (string, string, error) {
	snowConfig := &snow.Config{
		User: config.DataSource.Username,
	}
	if config.ConnectionContext.DatabaseName != "" {
		snowConfig.Database = fmt.Sprintf(`"%s"`, config.ConnectionContext.DatabaseName)
	}
	if config.DataSource.GetAuthenticationPrivateKey() != "" {
		// Use key-pair authentication (JWT) - password is not required
		rsaPrivKey, err := decodeRSAPrivateKey(config.DataSource.GetAuthenticationPrivateKey(), config.DataSource.GetAuthenticationPrivateKeyPassphrase())
		if err != nil {
			return "", "", errors.Wrapf(err, "failed to decode rsa private key")
		}
		snowConfig.PrivateKey = rsaPrivKey
		snowConfig.Authenticator = snow.AuthTypeJwt
	} else {
		// Use password authentication
		snowConfig.Password = config.Password
	}

	// Host can also be account e.g. xma12345, or xma12345@host_ip where host_ip is the proxy server IP.
	if strings.Contains(config.DataSource.Host, "@") {
		parts := strings.Split(config.DataSource.Host, "@")
		if len(parts) != 2 {
			return "", "", errors.Errorf("expected one @ in the host at most, got %q", config.DataSource.Host)
		}
		snowConfig.Account = parts[0]
		snowConfig.Host = parts[1]
	} else {
		snowConfig.Account = config.DataSource.Host
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
func (d *Driver) Close(context.Context) error {
	return d.db.Close()
}

// Ping pings the database.
func (d *Driver) Ping(ctx context.Context) error {
	return d.db.PingContext(ctx)
}

// GetDB gets the database.
func (d *Driver) GetDB() *sql.DB {
	return d.db
}

// getVersion gets the version.
func (d *Driver) getVersion(ctx context.Context) (string, error) {
	query := "SELECT CURRENT_VERSION()"
	var version string
	if err := d.db.QueryRowContext(ctx, query).Scan(&version); err != nil {
		if err == sql.ErrNoRows {
			return "", common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return "", util.FormatErrorWithQuery(err, query)
	}
	return version, nil
}

func (d *Driver) getDatabases(ctx context.Context) ([]string, error) {
	txn, err := d.db.BeginTx(ctx, &sql.TxOptions{})
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

func getDatabasesTxn(ctx context.Context, txn *sql.Tx) ([]string, error) {
	// Filter inbound shared databases because they are immutable and we cannot get their DDLs.
	inboundDatabases := make(map[string]bool)
	shareQuery := "SHOW SHARES"
	shareRows, err := txn.Query(shareQuery)
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
	rows, err := txn.QueryContext(ctx, query)
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
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	// Parse transaction mode from the script
	config, cleanedStatement := base.ParseTransactionConfig(statement)
	statement = cleanedStatement
	transactionMode := config.Mode

	// Apply default when transaction mode is not specified
	if transactionMode == common.TransactionModeUnspecified {
		transactionMode = common.GetDefaultTransactionMode()
	}

	// Execute based on transaction mode
	if transactionMode == common.TransactionModeOff {
		return d.executeInAutoCommitMode(ctx, statement, opts)
	}
	return d.executeInTransactionMode(ctx, statement, opts)
}

// executeInTransactionMode executes statements within a single transaction
func (d *Driver) executeInTransactionMode(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, err.Error())
		return 0, err
	}
	opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, "")

	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(); err != nil {
				opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_ROLLBACK, err.Error())
			} else {
				opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_ROLLBACK, "")
			}
		}
	}()

	// To submit a variable number of SQL statements in the statement field, set MULTI_STATEMENT_COUNT to 0."
	// https://docs.snowflake.com/en/developer-guide/sql-api/submitting-multiple-statements
	mctx, err := snow.WithMultiStatement(ctx, 0 /* MULTI_STATEMENT_COUNT */)
	if err != nil {
		return 0, err
	}

	// Log the entire multi-statement execution
	opts.LogCommandExecute(&storepb.Range{Start: 0, End: int32(len(statement))}, statement)

	result, err := tx.ExecContext(mctx, statement)
	if err != nil {
		opts.LogCommandResponse(0, nil, err.Error())
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, err.Error())
		return 0, err
	}
	opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, "")
	committed = true

	rowsAffected, err := result.RowsAffected()
	// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
	if err != nil {
		slog.Debug("rowsAffected returns error", log.BBError(err))
		opts.LogCommandResponse(0, nil, "")
		return 0, nil
	}
	opts.LogCommandResponse(rowsAffected, nil, "")
	return rowsAffected, nil
}

// executeInAutoCommitMode executes statements with autocommit enabled (no explicit transaction)
func (d *Driver) executeInAutoCommitMode(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	// To submit a variable number of SQL statements in the statement field, set MULTI_STATEMENT_COUNT to 0."
	// https://docs.snowflake.com/en/developer-guide/sql-api/submitting-multiple-statements
	mctx, err := snow.WithMultiStatement(ctx, 0 /* MULTI_STATEMENT_COUNT */)
	if err != nil {
		return 0, err
	}

	// Log the entire multi-statement execution
	opts.LogCommandExecute(&storepb.Range{Start: 0, End: int32(len(statement))}, statement)

	result, err := d.db.ExecContext(mctx, statement)
	if err != nil {
		opts.LogCommandResponse(0, nil, err.Error())
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
	if err != nil {
		slog.Debug("rowsAffected returns error", log.BBError(err))
		opts.LogCommandResponse(0, nil, "")
		return 0, nil
	}
	opts.LogCommandResponse(rowsAffected, nil, "")
	return rowsAffected, nil
}

// QueryConn queries a SQL statement in a given connection.
func (*Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_SNOWFLAKE, statement)
	if err != nil {
		return nil, err
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		statement := singleSQL.Text
		if queryContext.Explain {
			statement = fmt.Sprintf("EXPLAIN %s", statement)
		} else if queryContext.Limit > 0 {
			statement = getStatementWithResultLimit(statement, queryContext.Limit)
		}

		_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_SNOWFLAKE, statement)
		if err != nil {
			slog.Error("failed to validate sql", slog.String("statement", statement), log.BBError(err))
			allQuery = true
		}

		// If the queryContext.Schema is not empty, set the current schema to the given schema.
		// Reference: https://docs.snowflake.com/en/sql-reference/sql/use-schema
		if queryContext.Schema != "" {
			if _, err := conn.ExecContext(ctx, fmt.Sprintf("USE SCHEMA %s;", queryContext.Schema)); err != nil {
				return nil, err
			}
		} else {
			// If the queryContext.Schema is empty, we try to set the current schema to "PUBLIC" and ignore the error because
			// the schema may not exist.
			if _, err := conn.ExecContext(ctx, "USE SCHEMA PUBLIC;"); err != nil {
				slog.Debug("failed to set schema to PUBLIC", log.BBError(err))
			}
		}

		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			if allQuery {
				rows, err := conn.QueryContext(ctx, statement)
				if err != nil {
					return nil, err
				}
				defer rows.Close()
				r, err := util.RowsToQueryResult(rows, util.MakeCommonValueByTypeName, util.ConvertCommonValue, queryContext.MaximumSQLResultSize)
				if err != nil {
					return nil, err
				}
				if err := rows.Err(); err != nil {
					return nil, err
				}
				return r, nil
			}

			sqlResult, err := conn.ExecContext(ctx, statement)
			if err != nil {
				return nil, err
			}
			affectedRows, err := sqlResult.RowsAffected()
			if err != nil {
				slog.Info("rowsAffected returns error", log.BBError(err))
			}
			return util.BuildAffectedRowsResult(affectedRows, nil), nil
		}()
		stop := false
		if err != nil {
			queryResult = &v1pb.QueryResult{
				Error: err.Error(),
			}
			stop = true
		}
		queryResult.Statement = statement
		queryResult.Latency = durationpb.New(time.Since(startTime))
		queryResult.RowsCount = int64(len(queryResult.Rows))
		results = append(results, queryResult)
		if stop {
			break
		}
	}

	return results, nil
}

func decodeRSAPrivateKey(key, passphrase string) (*rsa.PrivateKey, error) {
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
		pk, err := pkcs8.ParsePKCS8PrivateKeyRSA(block.Bytes, []byte(passphrase))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse pkcs8 private key to rsa private key with passphrase")
		}
		return pk, nil
	default:
		return nil, errors.Errorf("unsupported pem block type: %s", block.Type)
	}
}
