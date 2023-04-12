package taskcheck

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	tidbparser "github.com/pingcap/tidb/parser"
	tidbast "github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser"
	"github.com/bytebase/bytebase/backend/plugin/parser/ast"
	"github.com/bytebase/bytebase/backend/store"
)

// NewStatementAffectedRowsReportExecutor creates a task check statement affected rows report executor.
func NewStatementAffectedRowsReportExecutor(store *store.Store, dbFactory *dbfactory.DBFactory) Executor {
	return &StatementAffectedRowsReportExecutor{
		store:     store,
		dbFactory: dbFactory,
	}
}

// StatementAffectedRowsReportExecutor is the task check statement affected rows report executor. It reports the affected rows of each statement.
type StatementAffectedRowsReportExecutor struct {
	store     *store.Store
	dbFactory *dbfactory.DBFactory
}

// Run will run the task check statement affected rows report executor once.
func (s *StatementAffectedRowsReportExecutor) Run(ctx context.Context, _ *store.TaskCheckRunMessage, task *store.TaskMessage) ([]api.TaskCheckResult, error) {
	if !api.IsTaskCheckReportNeededForTaskType(task.Type) {
		return nil, nil
	}
	payload := &TaskPayload{}
	if err := json.Unmarshal([]byte(task.Payload), payload); err != nil {
		return nil, err
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &task.InstanceID})
	if err != nil {
		return nil, err
	}
	if !api.IsTaskCheckReportSupported(instance.Engine) {
		return nil, nil
	}
	if payload.SheetID > 0 {
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusSuccess,
				Namespace: api.BBNamespace,
				Code:      common.Ok.Int(),
				Title:     "Large SQL affected rows report is disabled",
				Content:   "",
			},
		}, nil
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: task.DatabaseID})
	if err != nil {
		return nil, err
	}
	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, err
	}
	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database.DatabaseName)
	if err != nil {
		return nil, err
	}
	defer driver.Close(ctx)

	sqlDB := driver.GetDB()
	switch instance.Engine {
	case db.Postgres:
		return reportStatementAffectedRowsForPostgres(ctx, sqlDB, payload.Statement)
	case db.MySQL:
		return reportStatementAffectedRowsForMySQL(ctx, sqlDB, payload.Statement, dbSchema.Metadata.CharacterSet, dbSchema.Metadata.Collation)
	default:
		return nil, errors.New("unsupported db type")
	}
}

// Postgres

func reportStatementAffectedRowsForPostgres(ctx context.Context, sqlDB *sql.DB, statement string) ([]api.TaskCheckResult, error) {
	stmts, err := parser.Parse(parser.Postgres, parser.ParseContext{}, statement)
	if err != nil {
		// nolint:nilerr
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusError,
				Namespace: api.AdvisorNamespace,
				Code:      advisor.StatementSyntaxError.Int(),
				Title:     "Syntax error",
				Content:   err.Error(),
			},
		}, nil
	}

	var result []api.TaskCheckResult

	for _, stmt := range stmts {
		rowCount, err := getAffectedRowsForPostgres(ctx, sqlDB, stmt)
		if err != nil {
			// nolint:nilerr
			return []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusError,
					Namespace: api.BBNamespace,
					Code:      common.Internal.Int(),
					Title:     "Failed to report statement affected rows",
					Content:   err.Error(),
				},
			}, nil
		}
		result = append(result, api.TaskCheckResult{
			Status:    api.TaskCheckStatusSuccess,
			Namespace: api.BBNamespace,
			Code:      common.Ok.Int(),
			Title:     "OK",
			Content:   fmt.Sprintf("%v", rowCount),
		})
	}

	return result, nil
}

func getAffectedRowsForPostgres(ctx context.Context, sqlDB *sql.DB, node ast.Node) (int64, error) {
	switch node := node.(type) {
	case *ast.InsertStmt:
		return getInsertAffectedRowsForPostgres(ctx, sqlDB, node)
	case *ast.UpdateStmt, *ast.DeleteStmt:
		return getUpdateOrDeleteAffectedRowsForPostgres(ctx, sqlDB, node)
	default:
		return 0, nil
	}
}

func getInsertAffectedRowsForPostgres(ctx context.Context, sqlDB *sql.DB, node *ast.InsertStmt) (int64, error) {
	if len(node.ValueList) > 0 {
		return int64(len(node.ValueList)), nil
	}
	res, err := query(ctx, sqlDB, fmt.Sprintf("EXPLAIN %s", node.Text()))
	if err != nil {
		return 0, err
	}
	rowCount, err := getAffectedRowsCountForPostgres(res)
	if err != nil {
		return 0, err
	}
	return rowCount, nil
}

func getUpdateOrDeleteAffectedRowsForPostgres(ctx context.Context, sqlDB *sql.DB, node ast.Node) (int64, error) {
	res, err := query(ctx, sqlDB, fmt.Sprintf("EXPLAIN %s", node.Text()))
	if err != nil {
		return 0, err
	}
	rowCount, err := getAffectedRowsCountForPostgres(res)
	if err != nil {
		return 0, err
	}
	return rowCount, nil
}

func getAffectedRowsCountForPostgres(res []any) (int64, error) {
	// the res struct is []any{columnName, columnTable, rowDataList}
	if len(res) != 3 {
		return 0, errors.Errorf("expected 3 but got %d", len(res))
	}
	rowList, ok := res[2].([]any)
	if !ok {
		return 0, errors.Errorf("expected []any but got %t", res[2])
	}
	// test-bb=# EXPLAIN INSERT INTO t SELECT * FROM t;
	// QUERY PLAN
	// -------------------------------------------------------------
	//  Insert on t  (cost=0.00..1.03 rows=0 width=0)
	//    ->  Seq Scan on t t_1  (cost=0.00..1.03 rows=3 width=520)
	// (2 rows)
	if len(rowList) < 2 {
		return 0, errors.Errorf("not found any data")
	}
	// We need the row 2.
	rowTwo, ok := rowList[1].([]any)
	if !ok {
		return 0, errors.Errorf("expected []any but got %t", rowList[0])
	}
	// PostgreSQL EXPLAIN statement result has one column.
	if len(rowTwo) != 1 {
		return 0, errors.Errorf("expected one but got %d", len(rowTwo))
	}
	// Get the string value.
	text, ok := rowTwo[0].(string)
	if !ok {
		return 0, errors.Errorf("expected string but got %t", rowTwo[0])
	}

	rowsRegexp := regexp.MustCompile("rows=([0-9]+)")
	matches := rowsRegexp.FindStringSubmatch(text)
	if len(matches) != 2 {
		return 0, errors.Errorf("failed to find rows in %q", text)
	}
	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, errors.Errorf("failed to get integer from %q", matches[1])
	}
	return value, nil
}

// MySQL

func reportStatementAffectedRowsForMySQL(ctx context.Context, sqlDB *sql.DB, statement, charset, collation string) ([]api.TaskCheckResult, error) {
	singleSQLs, err := parser.SplitMultiSQL(parser.MySQL, statement)
	if err != nil {
		// nolint:nilerr
		return []api.TaskCheckResult{
			{
				Status:    api.TaskCheckStatusError,
				Namespace: api.AdvisorNamespace,
				Code:      advisor.StatementSyntaxError.Int(),
				Title:     "Syntax error",
				Content:   err.Error(),
			},
		}, nil
	}

	var result []api.TaskCheckResult

	p := tidbparser.New()
	p.EnableWindowFunc(true)

	for _, stmt := range singleSQLs {
		if stmt.Empty {
			continue
		}
		if parser.IsTiDBUnsupportDDLStmt(stmt.Text) {
			result = append(result, api.TaskCheckResult{
				Status:    api.TaskCheckStatusSuccess,
				Namespace: api.BBNamespace,
				Code:      common.Ok.Int(),
				Title:     "OK",
				Content:   "0",
			})
			continue
		}
		root, _, err := p.Parse(stmt.Text, charset, collation)
		if err != nil {
			// nolint:nilerr
			return []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusError,
					Namespace: api.AdvisorNamespace,
					Code:      advisor.StatementSyntaxError.Int(),
					Title:     "Syntax error",
					Content:   err.Error(),
				},
			}, nil
		}
		if len(root) != 1 {
			return []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusError,
					Namespace: api.BBNamespace,
					Code:      common.Internal.Int(),
					Title:     "Failed to report statement affected rows",
					Content:   "Expect to get one node from parser",
				},
			}, nil
		}
		affectedRows, err := getAffectedRowsForMysql(ctx, sqlDB, root[0])
		if err != nil {
			// nolint:nilerr
			return []api.TaskCheckResult{
				{
					Status:    api.TaskCheckStatusError,
					Namespace: api.BBNamespace,
					Code:      common.Internal.Int(),
					Title:     "Failed to report statement affected rows",
					Content:   err.Error(),
				},
			}, nil
		}
		result = append(result, api.TaskCheckResult{
			Status:    api.TaskCheckStatusSuccess,
			Namespace: api.BBNamespace,
			Code:      common.Ok.Int(),
			Title:     "OK",
			Content:   fmt.Sprintf("%v", affectedRows),
		})
	}

	return result, nil
}

func getAffectedRowsForMysql(ctx context.Context, sqlDB *sql.DB, node tidbast.StmtNode) (int64, error) {
	switch node := node.(type) {
	case *tidbast.InsertStmt:
		return getInsertAffectedRowsForMysql(ctx, sqlDB, node)
	case *tidbast.UpdateStmt, *tidbast.DeleteStmt:
		return getUpdateOrDeleteAffectedRowsForMysql(ctx, sqlDB, node)
	default:
		return 0, nil
	}
}

func getInsertAffectedRowsForMysql(ctx context.Context, sqlDB *sql.DB, node *tidbast.InsertStmt) (int64, error) {
	if node.Select == nil {
		return int64(len(node.Lists)), nil
	}
	res, err := query(ctx, sqlDB, fmt.Sprintf("EXPLAIN %s", node.Text()))
	if err != nil {
		return 0, err
	}
	rowCount, err := getInsertAffectedRowsCountForMysql(res)
	if err != nil {
		return 0, err
	}
	return rowCount, nil
}

func getUpdateOrDeleteAffectedRowsForMysql(ctx context.Context, sqlDB *sql.DB, node tidbast.StmtNode) (int64, error) {
	res, err := query(ctx, sqlDB, fmt.Sprintf("EXPLAIN %s", node.Text()))
	if err != nil {
		return 0, err
	}
	rowCount, err := getUpdateOrDeleteAffectedRowsCountForMysql(res)
	if err != nil {
		return 0, err
	}
	return rowCount, nil
}

func getUpdateOrDeleteAffectedRowsCountForMysql(res []any) (int64, error) {
	// the res struct is []any{columnName, columnTable, rowDataList}
	if len(res) != 3 {
		return 0, errors.Errorf("expected 3 but got %d", len(res))
	}
	rowList, ok := res[2].([]any)
	if !ok {
		return 0, errors.Errorf("expected []any but got %t", res[2])
	}
	if len(rowList) < 1 {
		return 0, errors.Errorf("not found any data")
	}
	rowOne, ok := rowList[0].([]any)
	if !ok {
		return 0, errors.Errorf("expected []any but got %t", rowList[0])
	}
	// MySQL EXPLAIN statement result has 12 columns.
	//
	// mysql> explain delete from td;
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------+
	// | id | select_type | table | partitions | type | possible_keys | key  | key_len | ref  | rows | filtered | Extra |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------+
	// |  1 | DELETE      | td    | NULL       | ALL  | NULL          | NULL | NULL    | NULL |    1 |   100.00 | NULL  |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------+
	if len(rowOne) != 12 {
		return 0, errors.Errorf("expected 12 but got %d", len(rowOne))
	}
	// the column 9 is the data 'rows'.
	switch rows := rowOne[9].(type) {
	case int:
		return int64(rows), nil
	case int64:
		return rows, nil
	case string:
		v, err := strconv.ParseInt(rows, 10, 64)
		if err != nil {
			return 0, errors.Errorf("expected int or int64 but got string(%s)", rows)
		}
		return v, nil
	default:
		return 0, errors.Errorf("expected int or int64 but got %t", rowOne[9])
	}
}

func getInsertAffectedRowsCountForMysql(res []any) (int64, error) {
	// the res struct is []any{columnName, columnTable, rowDataList}
	if len(res) != 3 {
		return 0, errors.Errorf("expected 3 but got %d", len(res))
	}
	rowList, ok := res[2].([]any)
	if !ok {
		return 0, errors.Errorf("expected []any but got %t", res[2])
	}
	// mysql> explain insert into td select * from td;
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-----------------+
	// | id | select_type | table | partitions | type | possible_keys | key  | key_len | ref  | rows | filtered | Extra           |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-----------------+
	// |  1 | INSERT      | td    | NULL       | ALL  | NULL          | NULL | NULL    | NULL | NULL |     NULL | NULL            |
	// |  1 | SIMPLE      | td    | NULL       | ALL  | NULL          | NULL | NULL    | NULL |    1 |   100.00 | Using temporary |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-----------------+
	if len(rowList) < 2 {
		return 0, errors.Errorf("not found any data")
	}
	// We need the row 2.
	rowTwo, ok := rowList[1].([]any)
	if !ok {
		return 0, errors.Errorf("expected []any but got %t", rowList[0])
	}
	// MySQL EXPLAIN statement result has 12 columns.
	if len(rowTwo) != 12 {
		return 0, errors.Errorf("expected 12 but got %d", len(rowTwo))
	}
	// the column 9 is the data 'rows'.
	switch rows := rowTwo[9].(type) {
	case int:
		return int64(rows), nil
	case int64:
		return rows, nil
	case string:
		v, err := strconv.ParseInt(rows, 10, 64)
		if err != nil {
			return 0, errors.Errorf("expected int or int64 but got string(%s)", rows)
		}
		return v, nil
	default:
		return 0, errors.Errorf("expected int or in64 but got %t", rowTwo[9])
	}
}

// Query runs the EXPLAIN or SELECT statements for advisors.
func query(ctx context.Context, connection *sql.DB, statement string) ([]any, error) {
	tx, err := connection.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	colCount := len(columnTypes)

	var columnTypeNames []string
	for _, v := range columnTypes {
		// DatabaseTypeName returns the database system name of the column type.
		// refer: https://pkg.go.dev/database/sql#ColumnType.DatabaseTypeName
		columnTypeNames = append(columnTypeNames, strings.ToUpper(v.DatabaseTypeName()))
	}

	data := []any{}
	for rows.Next() {
		scanArgs := make([]any, colCount)
		for i, v := range columnTypeNames {
			// TODO(steven need help): Consult a common list of data types from database driver documentation. e.g. MySQL,PostgreSQL.
			switch v {
			case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
				scanArgs[i] = new(sql.NullString)
			case "BOOL":
				scanArgs[i] = new(sql.NullBool)
			case "INT", "INTEGER":
				scanArgs[i] = new(sql.NullInt64)
			case "FLOAT":
				scanArgs[i] = new(sql.NullFloat64)
			default:
				scanArgs[i] = new(sql.NullString)
			}
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}

		rowData := []any{}
		for i := range columnTypes {
			if v, ok := (scanArgs[i]).(*sql.NullBool); ok && v.Valid {
				rowData = append(rowData, v.Bool)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullString); ok && v.Valid {
				rowData = append(rowData, v.String)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullInt64); ok && v.Valid {
				rowData = append(rowData, v.Int64)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullInt32); ok && v.Valid {
				rowData = append(rowData, v.Int32)
				continue
			}
			if v, ok := (scanArgs[i]).(*sql.NullFloat64); ok && v.Valid {
				rowData = append(rowData, v.Float64)
				continue
			}
			// If none of them match, set nil to its value.
			rowData = append(rowData, nil)
		}

		data = append(data, rowData)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return []any{columnNames, columnTypeNames, data}, nil
}
