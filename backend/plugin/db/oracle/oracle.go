// Package oracle is the plugin for Oracle driver.
package oracle

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"strings"
	"time"

	// Import go-ora Oracle driver.

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"
	goora "github.com/sijms/go-ora/v2"
	"google.golang.org/protobuf/types/known/durationpb"

	plsql "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
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
	options := make(map[string]string)
	options["CONNECTION TIMEOUT"] = "0"
	if config.DataSource.GetSid() != "" {
		options["SID"] = config.DataSource.GetSid()
	}
	for key, value := range config.DataSource.GetExtraConnectionParameters() {
		options[key] = value
	}
	dsn := goora.BuildUrl(config.DataSource.Host, port, config.DataSource.GetServiceName(), config.DataSource.Username, config.Password, options)
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, err
	}
	if config.ConnectionContext.DatabaseName != "" {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER SESSION SET CURRENT_SCHEMA = \"%s\"", config.ConnectionContext.DatabaseName)); err != nil {
			return nil, errors.Wrapf(err, "failed to set current schema to %q", config.ConnectionContext.DatabaseName)
		}
	}
	d.db = db
	d.databaseName = config.ConnectionContext.DatabaseName
	d.serviceName = config.DataSource.GetServiceName()
	d.connectionCtx = config.ConnectionContext
	return d, nil
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
	transactionMode, cleanedStatement := base.ParseTransactionMode(statement)
	statement = cleanedStatement

	// Apply default when transaction mode is not specified
	if transactionMode == common.TransactionModeUnspecified {
		transactionMode = common.GetDefaultTransactionMode()
	}

	var commands []base.SingleSQL
	var originalIndex []int32
	if len(statement) <= common.MaxSheetCheckSize {
		// Use Oracle sql parser.
		singleSQLs, err := plsqlparser.SplitSQL(statement)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to split sql")
		}
		singleSQLs, originalIndex = base.FilterEmptySQLWithIndexes(singleSQLs)
		if len(singleSQLs) == 0 {
			return 0, nil
		}
		commands = singleSQLs
	} else {
		commands = []base.SingleSQL{
			{
				Text: statement,
			},
		}
		originalIndex = []int32{0}
	}

	conn, err := d.db.Conn(ctx)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get connection")
	}
	defer conn.Close()

	// Execute based on transaction mode
	if transactionMode == common.TransactionModeOff {
		return d.executeInAutoCommitMode(ctx, conn, commands, originalIndex, opts)
	}
	return d.executeInTransactionMode(ctx, conn, commands, originalIndex, opts)
}

// executeInTransactionMode executes statements within a single transaction
func (*Driver) executeInTransactionMode(ctx context.Context, conn *sql.Conn, commands []base.SingleSQL, originalIndex []int32, opts db.ExecuteOptions) (int64, error) {
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
	for i, command := range commands {
		indexes := []int32{originalIndex[i]}
		opts.LogCommandExecute(indexes, command.Text)

		sqlResult, err := tx.ExecContext(ctx, command.Text)
		if err != nil {
			opts.LogCommandResponse(0, nil, err.Error())
			return 0, &db.ErrorWithPosition{
				Err:   errors.Wrapf(err, "failed to execute context in a transaction"),
				Start: command.Start,
				End:   command.End,
			}
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			slog.Debug("rowsAffected returns error", log.BBError(err))
			rowsAffected = 0
		}
		opts.LogCommandResponse(int32(rowsAffected), []int32{int32(rowsAffected)}, "")
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
func (*Driver) executeInAutoCommitMode(ctx context.Context, conn *sql.Conn, commands []base.SingleSQL, originalIndex []int32, opts db.ExecuteOptions) (int64, error) {
	totalRowsAffected := int64(0)
	for i, command := range commands {
		indexes := []int32{originalIndex[i]}
		opts.LogCommandExecute(indexes, command.Text)

		sqlResult, err := conn.ExecContext(ctx, command.Text)
		if err != nil {
			opts.LogCommandResponse(0, nil, err.Error())
			// In auto-commit mode, we stop at the first error
			// The database is left in a partially migrated state
			return totalRowsAffected, &db.ErrorWithPosition{
				Err:   errors.Wrapf(err, "failed to execute statement %d in auto-commit mode", i+1),
				Start: command.Start,
				End:   command.End,
			}
		}
		rowsAffected, err := sqlResult.RowsAffected()
		if err != nil {
			// Since we cannot differentiate DDL and DML yet, we have to ignore the error.
			slog.Debug("rowsAffected returns error", log.BBError(err))
			rowsAffected = 0
		}
		opts.LogCommandResponse(int32(rowsAffected), []int32{int32(rowsAffected)}, "")
		totalRowsAffected += rowsAffected
	}
	return totalRowsAffected, nil
}

// QueryConn queries a SQL statement in a given connection.
func (d *Driver) QueryConn(ctx context.Context, conn *sql.Conn, statement string, queryContext db.QueryContext) ([]*v1pb.QueryResult, error) {
	singleSQLs, err := plsqlparser.SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	singleSQLs = base.FilterEmptySQL(singleSQLs)
	if len(singleSQLs) == 0 {
		return nil, nil
	}

	var results []*v1pb.QueryResult
	for _, singleSQL := range singleSQLs {
		statement := singleSQL.Text
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
			statement = fmt.Sprintf(`SELECT LPAD(' ', LEVEL-1) || OPERATION || ' (' || OPTIONS || ')' "Operation", OBJECT_NAME "Object", OPTIMIZER "Optimizer", COST "Cost", CARDINALITY "Cardinality", BYTES "Bytes", PARTITION_START "Partition Start", PARTITION_ID "Partition ID", ACCESS_PREDICATES "Access Predicates" FROM PLAN_TABLE START WITH ID = 0 AND statement_id = '%s' CONNECT BY PRIOR ID=PARENT_ID AND statement_id = '%s' ORDER BY id`, randomID, randomID)
		}

		if !queryContext.Explain && queryContext.Limit > 0 {
			stmt, err := d.getStatementWithResultLimit(statement, queryContext)
			if err != nil {
				slog.Error("fail to add limit clause", "statement", statement, log.BBError(err))
				stmt = getStatementWithResultLimitFor11g(stmt, queryContext.Limit)
			}
			statement = stmt
		}

		_, allQuery, err := base.ValidateSQLForEditor(storepb.Engine_ORACLE, statement)
		if err != nil {
			return nil, err
		}
		startTime := time.Now()
		queryResult, err := func() (*v1pb.QueryResult, error) {
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

func (d *Driver) getStatementWithResultLimit(stmt string, queryContext db.QueryContext) (string, error) {
	engineVersion := d.connectionCtx.EngineVersion
	versionIdx := strings.Index(engineVersion, ".")
	if versionIdx < 0 {
		return "", errors.New("instance version number is invalid")
	}
	versionNumber, err := strconv.Atoi(engineVersion[:versionIdx])
	if err != nil {
		return "", err
	}
	if ok, err := skipAddLimit(stmt); err == nil && ok {
		return stmt, nil
	}
	switch {
	case versionNumber < dbVersion12:
		return getStatementWithResultLimitFor11g(stmt, queryContext.Limit), nil
	default:
		return getStatementWithResultLimit(stmt, queryContext.Limit), nil
	}
}

// skipAddLimit checks if the statement needs a limit clause.
// For Oracle, we think the statement like "SELECT xxx FROM DUAL" does not need a limit clause.
// More details, xxx can not be a subquery.
func skipAddLimit(stmt string) (bool, error) {
	tree, _, err := plsqlparser.ParsePLSQL(stmt)
	if err != nil {
		return false, err
	}

	sqlScript, ok := tree.(*plsql.Sql_scriptContext)
	if !ok {
		return false, nil
	}

	if len(sqlScript.AllSql_plus_command()) > 0 {
		return false, nil
	}

	if len(sqlScript.AllUnit_statement()) > 1 {
		return false, nil
	}

	unitStatement := sqlScript.Unit_statement(0)
	if unitStatement == nil {
		return false, nil
	}

	dml := unitStatement.Data_manipulation_language_statements()
	if dml == nil {
		return false, nil
	}

	selectStatement := dml.Select_statement()
	if selectStatement == nil {
		return false, nil
	}

	switch {
	case len(selectStatement.AllFor_update_clause()) != 0:
		return false, nil
	case len(selectStatement.AllOrder_by_clause()) != 0:
		return false, nil
	case len(selectStatement.AllOffset_clause()) != 0:
		return false, nil
	case len(selectStatement.AllFetch_clause()) != 0:
		return false, nil
	default:
		// No additional clauses
	}

	selectOnly := selectStatement.Select_only_statement()
	if selectOnly == nil {
		return false, nil
	}

	subquery := selectOnly.Subquery()
	if subquery == nil {
		return false, nil
	}

	if len(subquery.AllSubquery_operation_part()) != 0 {
		return false, nil
	}

	subqueryBasicElements := subquery.Subquery_basic_elements()
	if subqueryBasicElements == nil {
		return false, nil
	}

	if subqueryBasicElements.Subquery() != nil {
		return false, nil
	}

	queryBlock := subqueryBasicElements.Query_block()
	if queryBlock == nil {
		return false, nil
	}

	switch {
	case queryBlock.Subquery_factoring_clause() != nil,
		queryBlock.DISTINCT() != nil,
		queryBlock.ALL() != nil,
		queryBlock.UNIQUE() != nil,
		queryBlock.Into_clause() != nil,
		queryBlock.Where_clause() != nil,
		queryBlock.Hierarchical_query_clause() != nil,
		queryBlock.Group_by_clause() != nil,
		queryBlock.Model_clause() != nil,
		queryBlock.Order_by_clause() != nil,
		queryBlock.Fetch_clause() != nil:
		return false, nil
	default:
		// Query block has no special clauses
	}

	from := queryBlock.From_clause()
	if !strings.EqualFold(from.GetText(), "FROMDUAL") {
		return false, nil
	}

	selectedList := queryBlock.Selected_list()
	if selectedList == nil {
		return false, nil
	}

	if selectedList.ASTERISK() != nil {
		return false, nil
	}

	for _, selectedElement := range selectedList.AllSelect_list_elements() {
		if selectedElement.Table_wild() != nil {
			return false, nil
		}

		l := subqueryListener{}
		antlr.ParseTreeWalkerDefault.Walk(&l, selectedElement)
		if l.hasSubquery {
			return false, nil
		}
	}

	return true, nil
}

type subqueryListener struct {
	*plsql.BasePlSqlParserListener
	hasSubquery bool
}

func (l *subqueryListener) EnterSubquery(*plsql.SubqueryContext) {
	l.hasSubquery = true
}
