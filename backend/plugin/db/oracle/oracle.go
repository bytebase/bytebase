// Package oracle is the plugin for Oracle driver.
package oracle

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"strings"
	"time"

	// Import go-ora Oracle driver.

	"github.com/pkg/errors"
	goora "github.com/sijms/go-ora/v2"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/transaction"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ db.Driver = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_ORACLE, newDriver)
}

// Driver is the Oracle driver.
type Driver struct {
	db            *sql.DB
	databaseName  string
	serviceName   string
	connectionCtx db.ConnectionContext
}

func newDriver() db.Driver {
	return &Driver{}
}

// GetVersion gets the Oracle version.
func (d *Driver) GetVersion() (*plsqlparser.Version, error) {
	return plsqlparser.ParseVersion(d.connectionCtx.EngineVersion)
}

// Open opens a Oracle driver.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	port, err := strconv.Atoi(config.DataSource.Port)
	if err != nil {
		return nil, errors.Errorf("invalid port %q", config.DataSource.Port)
	}
	tlsConfig, err := getTLSConfig(config.DataSource)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build TLS configuration")
	}
	options := connectionOptions(config.DataSource.GetSid(), config.DataSource.GetExtraConnectionParameters(), tlsConfig != nil)
	dsn := goora.BuildUrl(config.DataSource.Host, port, config.DataSource.GetServiceName(), config.DataSource.Username, config.Password, options)
	db := sql.OpenDB(&connector{dsn: dsn, tlsConfig: tlsConfig, currentSchema: config.ConnectionContext.DatabaseName})
	if config.ConnectionContext.DatabaseName != "" {
		// Connect now so that a schema the session can't switch to fails Open, not the first statement.
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	d.db = db
	d.databaseName = config.ConnectionContext.DatabaseName
	d.serviceName = config.DataSource.GetServiceName()
	d.connectionCtx = config.ConnectionContext
	return d, nil
}

// getTLSConfig returns the TLS configuration for the data source's TLS section, or nil when TLS
// is off. It leaves the name check to crypto/tls instead of util.GetTLSConfig's verifier, which
// checks Hostname: go-ora sets ServerName to each server it dials, so a SERVER address or a
// listener redirect is checked against its own name.
func getTLSConfig(dataSource *storepb.DataSource) (*tls.Config, error) {
	config, err := util.GetTLSConfig(dataSource)
	if err != nil || config == nil {
		return config, err
	}
	config.InsecureSkipVerify = !dataSource.GetVerifyTlsCertificate()
	config.VerifyPeerCertificate = nil
	return config, nil
}

// connector opens each connection through go-ora's default driver, as sql.Open("oracle") does,
// with its own copy of the TLS configuration: go-ora sets ServerName on the configuration it is
// given, so connections opened in parallel can't share one.
//
// It also sets currentSchema on every session it opens, because the pool opens a new session
// whenever none is idle or one goes bad. go-ora's AddSessionParam can't do this: it stores the
// parameter on the default driver, which every Bytebase Oracle connection shares.
type connector struct {
	dsn           string
	tlsConfig     *tls.Config
	currentSchema string
}

func (c *connector) Connect(ctx context.Context) (driver.Conn, error) {
	oracleConnector, err := goora.GetDefaultDriver().OpenConnector(c.dsn)
	if err != nil {
		return nil, err
	}
	oracleConnector.(*goora.OracleConnector).WithTLSConfig(c.tlsConfig.Clone())
	conn, err := oracleConnector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	if c.currentSchema == "" {
		return conn, nil
	}
	if _, err := conn.(driver.ExecerContext).ExecContext(ctx, fmt.Sprintf("ALTER SESSION SET CURRENT_SCHEMA = \"%s\"", c.currentSchema), nil); err != nil {
		_ = conn.Close()
		return nil, errors.Wrapf(err, "failed to set current schema to %q", c.currentSchema)
	}
	return conn, nil
}

func (*connector) Driver() driver.Driver {
	return goora.GetDefaultDriver()
}

const (
	sidOption              = "SID"
	sslOption              = "SSL"
	dialTimeoutOption      = "CONNECTION TIMEOUT"
	dialTimeoutOptionAlias = "CONNECT TIMEOUT"
	// defaultDialTimeoutSeconds bounds the wait for one address to accept a connection, DNS
	// lookup included, so an unreachable address fails over in seconds. It does not limit
	// how long a statement runs.
	defaultDialTimeoutSeconds = "10"
)

// connectionOptions returns the go-ora URL options for a data source: the extra connection
// parameters as typed, the SID field and the default dial timeout unless an extra parameter
// sets the same option, and SSL=TRUE when TLS is on.
//
// go-ora upper-cases option keys and reads CONNECT TIMEOUT as CONNECTION TIMEOUT. A Bytebase
// key sent next to a user key for the same option would leave the winner to go-ora's map
// iteration order, so only one is sent: the user's key for SID and the dial timeout, and
// Bytebase's SSL, which the TLS section decides.
func connectionOptions(sid string, extraParameters map[string]string, useTLS bool) map[string]string {
	options := make(map[string]string, len(extraParameters)+3)
	userOptions := make(map[string]bool, len(extraParameters))
	for key, value := range extraParameters {
		option := strings.ToUpper(key)
		if useTLS && option == sslOption {
			continue
		}
		options[key] = value
		userOptions[option] = true
	}
	if sid != "" && !userOptions[sidOption] {
		options[sidOption] = sid
	}
	if !userOptions[dialTimeoutOption] && !userOptions[dialTimeoutOptionAlias] {
		options[dialTimeoutOption] = defaultDialTimeoutSeconds
	}
	if useTLS {
		options[sslOption] = "TRUE"
	}
	return options
}

// Close closes the driver.
func (d *Driver) Close(_ context.Context) error {
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

// Execute executes the migration, `beforeCommitTxFunc` will be called before transaction commit and after executing `statement`.
//
// Callers can use `beforeCommitTx` to do some extra work before transaction commit, like get the transaction id.
// Any error returned by `beforeCommitTx` will rollback the transaction, so it is the callers' responsibility to return nil if the error occurs in `beforeCommitTx` is not fatal.
func (d *Driver) Execute(ctx context.Context, statement string, opts db.ExecuteOptions) (int64, error) {
	if opts.CreateDatabase {
		return 0, errors.New("create database is not supported for Oracle")
	}

	// Parse transaction mode from the script
	config, cleanedStatement := base.ParseTransactionConfig(statement)
	statement = cleanedStatement
	transactionMode := config.Mode

	// Apply default when transaction mode is not specified
	if transactionMode == transaction.ModeUnspecified {
		transactionMode = transaction.DefaultMode()
	}

	commands, err := buildExecuteCommands(statement)
	if err != nil {
		return 0, err
	}
	if len(commands) == 0 {
		return 0, nil
	}

	conn, err := d.db.Conn(ctx)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get connection")
	}
	defer conn.Close()

	// Execute based on transaction mode
	if transactionMode == transaction.ModeOff {
		return d.executeInAutoCommitMode(ctx, conn, commands, opts)
	}
	return d.executeInTransactionMode(ctx, conn, commands, opts)
}

func buildExecuteCommands(statement string) ([]base.Statement, error) {
	commands, err := plsqlparser.SplitSQL(statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split sql")
	}
	return base.FilterEmptyStatements(commands), nil
}

// executeInTransactionMode executes statements within a single transaction
func (*Driver) executeInTransactionMode(ctx context.Context, conn *sql.Conn, commands []base.Statement, opts db.ExecuteOptions) (int64, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, err.Error())
		return 0, errors.Wrapf(err, "failed to begin transaction")
	}
	opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_BEGIN, "")

	committed := false
	defer func() {
		err := tx.Rollback()
		if committed {
			return
		}
		var rerr string
		if err != nil {
			rerr = err.Error()
		}
		opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_ROLLBACK, rerr)
	}()

	totalRowsAffected := int64(0)
	for _, command := range commands {
		opts.LogCommandExecute(command.Range, command.Text)

		sqlResult, err := tx.ExecContext(ctx, command.Text)
		if err != nil {
			opts.LogCommandResponse(0, nil, err.Error())
			return 0, err
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			slog.Debug("rowsAffected returns error", log.BBError(err))
			rowsAffected = 0
		}
		opts.LogCommandResponse(rowsAffected, []int64{rowsAffected}, "")
		totalRowsAffected += rowsAffected
	}

	if err := tx.Commit(); err != nil {
		opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, err.Error())
		return 0, errors.Wrapf(err, "failed to commit transaction")
	}
	opts.LogTransactionControl(storepb.TaskRunLog_TransactionControl_COMMIT, "")
	committed = true
	return totalRowsAffected, nil
}

// executeInAutoCommitMode executes statements sequentially in auto-commit mode
func (*Driver) executeInAutoCommitMode(ctx context.Context, conn *sql.Conn, commands []base.Statement, opts db.ExecuteOptions) (int64, error) {
	totalRowsAffected := int64(0)
	for _, command := range commands {
		opts.LogCommandExecute(command.Range, command.Text)

		sqlResult, err := conn.ExecContext(ctx, command.Text)
		if err != nil {
			opts.LogCommandResponse(0, nil, err.Error())
			// In auto-commit mode, we stop at the first error
			// The database is left in a partially migrated state
			return totalRowsAffected, err
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			slog.Debug("rowsAffected returns error", log.BBError(err))
			rowsAffected = 0
		}
		opts.LogCommandResponse(rowsAffected, []int64{rowsAffected}, "")
		totalRowsAffected += rowsAffected
	}
	return totalRowsAffected, nil
}

// OwnsPrivateDatabaseLink reports whether the session's account owns a private database link.
// SYS.USER_DB_LINKS lists the links owned by the current user; the SYS qualifier keeps a
// same-named local object from standing in for the dictionary view.
func (*Driver) OwnsPrivateDatabaseLink(ctx context.Context, conn *sql.Conn) (bool, error) {
	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM SYS.USER_DB_LINKS").Scan(&count); err != nil {
		return false, errors.Wrap(err, "failed to count private database links")
	}
	return count > 0, nil
}

// QueryConn queries a SQL statement in a given connection.
func (d *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := plsqlparser.SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	singleSQLs = base.FilterEmptyStatements(singleSQLs)
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		statement := singleSQL.Text
		planAsJSON := false
		if queryContext.Explain {
			startTime := time.Now()
			randNum, err := rand.Int(rand.Reader, big.NewInt(999))
			if err != nil {
				return nil, errors.Wrapf(err, "failed to generate random statement ID")
			}
			randomID := fmt.Sprintf("%d%d", startTime.UnixMilli(), randNum.Int64())

			if _, err := conn.ExecContext(ctx, fmt.Sprintf("EXPLAIN PLAN SET STATEMENT_ID = '%s' FOR %s", randomID, statement)); err != nil {
				return nil, err
			}
			planAsJSON = db.ExplainFormat(queryContext.Option.GetExplainFormat()) == base.ExplainFormatJSON
			if planAsJSON {
				statement = fmt.Sprintf(planStepsQuery, randomID)
			} else {
				statement = fmt.Sprintf(`SELECT LPAD(' ', LEVEL-1) || OPERATION || ' (' || OPTIONS || ')' "Operation", OBJECT_NAME "Object", OPTIMIZER "Optimizer", COST "Cost", CARDINALITY "Cardinality", BYTES "Bytes", PARTITION_START "Partition Start", PARTITION_ID "Partition ID", ACCESS_PREDICATES "Access Predicates" FROM PLAN_TABLE START WITH ID = 0 AND statement_id = '%s' CONNECT BY PRIOR ID=PARENT_ID AND statement_id = '%s' ORDER BY id`, randomID, randomID)
			}
		}

		if !queryContext.Explain && queryContext.Limit > 0 {
			statement = base.StatementWithResultLimit(storepb.Engine_ORACLE, statement, queryContext.Limit, d.connectionCtx.EngineVersion)
		}

		_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_ORACLE, statement)
		if err != nil {
			return nil, err
		}
		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
			if planAsJSON {
				return queryPlanSteps(ctx, conn, statement)
			}
			if allQuery {
				rows, err := conn.QueryContext(ctx, statement)
				if err != nil {
					return nil, err
				}
				defer rows.Close()
				r, err := util.RowsToQueryResult(rows, makeValueByTypeName, convertValue, queryContext.MaximumSQLResultSize)
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

// planStepsQuery reads the plan Oracle recorded for one EXPLAIN PLAN: a row per
// operator, each naming its parent, which is how Oracle states a plan. The
// columns are what the query-plan viewer reads — the tree's shape, what each
// step does, the object it reads, and what the optimizer expects it to cost.
const planStepsQuery = `SELECT id, parent_id, position, operation, options, object_owner, object_name, object_type, optimizer, cost, cardinality, bytes, partition_start, partition_stop, access_predicates, filter_predicates FROM plan_table WHERE statement_id = '%s' ORDER BY id`

// queryPlanSteps returns those rows as one JSON array, which is the form the
// query-plan viewer parses; the viewer rebuilds the tree from the parent links.
//
// Oracle renders a plan as rows rather than as a document, and no EXPLAIN option
// renders it as one, so this is where the rendering happens. It is built here
// rather than with Oracle's own JSON_ARRAYAGG so that it works on releases older
// than 12.2, which do not have those functions. A tab-separated grid would not
// do: a predicate can hold a tab or a newline, and nothing downstream could tell
// that apart from the next column or the next row.
func queryPlanSteps(ctx context.Context, conn *sql.Conn, query string) (*v1pb.QueryResult, error) {
	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var steps [][]any
	for rows.Next() {
		values := make([]any, len(columns))
		scan := make([]any, len(columns))
		for i := range values {
			scan[i] = &values[i]
		}
		if err := rows.Scan(scan...); err != nil {
			return nil, err
		}
		steps = append(steps, values)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	payload, err := renderPlanSteps(columns, steps)
	if err != nil {
		return nil, err
	}
	return &v1pb.QueryResult{
		ColumnNames:     []string{"PLAN"},
		ColumnTypeNames: []string{"CLOB"},
		Rows: []*v1pb.QueryRow{{Values: []*v1pb.RowValue{
			{Kind: &v1pb.RowValue_StringValue{StringValue: payload}},
		}}},
	}, nil
}

// renderPlanSteps writes the scanned plan rows as a JSON array, one object per
// operator keyed by the lower-cased column names.
//
// A column the optimizer had nothing to say about is left out rather than
// carried as a null, so a reader of the plan sees only what Oracle stated.
func renderPlanSteps(columns []string, rows [][]any) (string, error) {
	steps := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		step := make(map[string]any, len(columns))
		for i, column := range columns {
			if i >= len(row) {
				break
			}
			if value := jsonValue(row[i]); value != nil {
				step[strings.ToLower(column)] = value
			}
		}
		steps = append(steps, step)
	}
	payload, err := json.Marshal(steps)
	if err != nil {
		return "", errors.Wrap(err, "failed to render the query plan")
	}
	return string(payload), nil
}

// jsonValue renders one scanned column, or nil where the row has no value for
// it. The driver hands back text as bytes, which JSON would otherwise encode as
// base64.
func jsonValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case []byte:
		return string(typed)
	default:
		return typed
	}
}
