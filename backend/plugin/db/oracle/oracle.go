// Package oracle is the plugin for Oracle driver.
package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	// Import go-ora Oracle driver.
	"github.com/pkg/errors"
	goora "github.com/sijms/go-ora/v2"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	_ db.Driver = (*Driver)(nil)
)

const dbVersion12 = 12

func init() {
	db.Register(storepb.Engine_ORACLE, newDriver)
}

// Driver is the Oracle driver.
type Driver struct {
	db               *sql.DB
	databaseName     string
	serviceName      string
	schemaTenantMode bool
}

func newDriver(db.DriverConfig) db.Driver {
	return &Driver{}
}

// Open opens a Oracle driver.
func (driver *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig, _ db.ConnectionContext) (db.Driver, error) {
	port, err := strconv.Atoi(config.Port)
	if err != nil {
		return nil, errors.Errorf("invalid port %q", config.Port)
	}
	options := make(map[string]string)
	if config.SID != "" {
		options["SID"] = config.SID
	}
	dsn := goora.BuildUrl(config.Host, port, config.ServiceName, config.Username, config.Password, options)
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, err
	}
	if config.SchemaTenantMode && config.Database != "" {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER SESSION SET CURRENT_SCHEMA = \"%s\"", config.Database)); err != nil {
			return nil, errors.Wrapf(err, "failed to set current schema to %q", config.Database)
		}
	}
	driver.db = db
	driver.databaseName = config.Database
	driver.serviceName = config.ServiceName
	driver.schemaTenantMode = config.SchemaTenantMode
	return driver, nil
}

// Close closes the driver.
func (driver *Driver) Close(_ context.Context) error {
	return driver.db.Close()
}

// Ping pings the database.
func (driver *Driver) Ping(ctx context.Context) error {
	return driver.db.PingContext(ctx)
}

// GetType returns the database type.
func (*Driver) GetType() storepb.Engine {
	return storepb.Engine_ORACLE
}

// GetDB gets the database.
func (driver *Driver) GetDB() *sql.DB {
	return driver.db
}

// Execute executes the migration, `beforeCommitTxFunc` will be called before transaction commit and after executing `statement`.
//
// Callers can use `beforeCommitTx` to do some extra work before transaction commit, like get the transaction id.
// Any error returned by `beforeCommitTx` will rollback the transaction, so it is the callers' responsibility to return nil if the error occurs in `beforeCommitTx` is not fatal.
func (driver *Driver) Execute(ctx context.Context, statement string, _ bool, opts db.ExecuteOptions) (int64, error) {
	conn, err := driver.db.Conn(ctx)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get connection")
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	totalRowsAffected := int64(0)
	f := func(stmt string) error {
		// The underlying oracle golang driver go-ora does not support semicolon, so we should trim the suffix semicolon.
		stmt = strings.TrimSuffix(stmt, ";")
		sqlResult, err := tx.ExecContext(ctx, stmt)
		if err != nil {
			return err
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			slog.Debug("rowsAffected returns error", log.BBError(err))
		} else {
			totalRowsAffected += rowsAffected
		}
		return nil
	}

	if _, err := plsqlparser.SplitMultiSQLStream(strings.NewReader(statement), f); err != nil {
		return 0, err
	}

	if opts.EndTransactionFunc != nil {
		if err := opts.EndTransactionFunc(tx); err != nil {
			return 0, errors.Wrapf(err, "failed to execute beforeCommitTx")
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "failed to commit transaction")
	}
	return totalRowsAffected, nil
}

// QueryConn queries a SQL statement in a given connection.
func (driver *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext *db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := plsqlparser.SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		result, err := driver.querySingleSQL(ctx, conn, singleSQL, queryContext)
		if err != nil {
			results = append(results, &v1pb.QueryResult{
				Error: err.Error(),
			})
		} else {
			results = append(results, result)
		}
	}

	return results, nil
}

func (*Driver) getOracleStatementWithResultLimit(stmt string, queryContext *db.QueryContext) (string, error) {
	engineVersion := queryContext.EngineVersion
	versionIdx := strings.Index(engineVersion, ".")
	if versionIdx < 0 {
		return "", errors.New("instance version number is invalid")
	}
	versionNumber, err := strconv.Atoi(engineVersion[:versionIdx])
	if err != nil {
		return "", err
	}
	switch {
	case versionNumber < dbVersion12:
		return getStatementWithResultLimitFor11g(stmt, queryContext.Limit), nil
	default:
		res, err := getStatementWithResultLimitFor12c(stmt, queryContext.Limit)
		if err != nil {
			return "", err
		}
		return res, nil
	}
}

func (driver *Driver) querySingleSQL(ctx context.Context, conn *sql.Conn, singleSQL base.SingleSQL, queryContext *db.QueryContext) (*v1pb.QueryResult, error) {
	statement := strings.TrimRight(singleSQL.Text, " \n\t;")

	stmt := statement
	if !strings.HasPrefix(strings.ToUpper(stmt), "EXPLAIN") && queryContext.Limit > 0 {
		var err error
		stmt, err = driver.getOracleStatementWithResultLimit(stmt, queryContext)
		if err != nil {
			slog.Error("fail to add limit clause", "statement", statement, log.BBError(err))
			stmt = getStatementWithResultLimitFor11g(stmt, queryContext.Limit)
		}
	}

	if queryContext.ReadOnly {
		// Oracle does not support transaction isolation level for read-only queries.
		queryContext.ReadOnly = false
	}

	if queryContext.SensitiveSchemaInfo != nil {
		for _, database := range queryContext.SensitiveSchemaInfo.DatabaseList {
			if len(database.SchemaList) == 0 {
				continue
			}
			if len(database.SchemaList) > 1 {
				return nil, errors.Errorf("Oracle schema info should only have one schema per database, but got %d, %v", len(database.SchemaList), database.SchemaList)
			}
			if database.SchemaList[0].Name != database.Name {
				return nil, errors.Errorf("Oracle schema info should have the same database name and schema name, but got %s and %s", database.Name, database.SchemaList[0].Name)
			}
		}
	}

	startTime := time.Now()
	result, err := util.Query(ctx, storepb.Engine_ORACLE, conn, stmt, queryContext)
	if err != nil {
		return nil, err
	}
	result.Latency = durationpb.New(time.Since(startTime))
	result.Statement = statement
	return result, nil
}

// RunStatement runs a SQL statement in a given connection.
func (*Driver) RunStatement(ctx context.Context, conn *sql.Conn, statement string) ([]*v1pb.QueryResult, error) {
	return util.RunStatement(ctx, storepb.Engine_ORACLE, conn, statement)
}

func (driver *Driver) getVersion(ctx context.Context) (int, int, error) {
	// https://docs.oracle.com/en/database/oracle/oracle-database/19/upgrd/oracle-database-release-numbers.html#GUID-1E2F3945-C0EE-4EB2-A933-8D1862D8ECE2
	var banner string
	if err := driver.db.QueryRowContext(ctx, "SELECT BANNER FROM v$version").Scan(&banner); err != nil {
		return 0, 0, err
	}

	return parseVersion(banner)
}

func parseVersion(banner string) (int, int, error) {
	re := regexp.MustCompile(`(\d+)\.(\d+)`)
	match := re.FindStringSubmatch(banner)
	if len(match) >= 3 {
		firstVersion, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, 0, errors.Errorf("failed to parse first version from banner: %s", banner)
		}
		secondVersion, err := strconv.Atoi(match[2])
		if err != nil {
			return 0, 0, errors.Errorf("failed to parse second version from banner: %s", banner)
		}
		return firstVersion, secondVersion, nil
	}
	return 0, 0, errors.Errorf("failed to parse version from banner: %s", banner)
}
