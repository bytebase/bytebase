package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// GetStatementType return the type of statement.
func GetStatementType(stmt *ParseResult) string {
	result := "UNKNOWN"
	for _, child := range stmt.Tree.GetChildren() {
		switch ctx := child.(type) {
		case *mysql.QueryContext:
			for _, child := range ctx.GetChildren() {
				switch ctx := child.(type) {
				case *mysql.SimpleStatementContext:
					for _, child := range ctx.GetChildren() {
						switch ctx := child.(type) {
						case *mysql.CreateStatementContext:
							for _, child := range ctx.GetChildren() {
								switch child.(type) {
								case *mysql.CreateDatabaseContext:
									result = "CREATE_DATABASE"
								case *mysql.CreateIndexContext:
									result = "CREATE_INDEX"
								case *mysql.CreateTableContext:
									result = "CREATE_TABLE"
								case *mysql.CreateViewContext:
									result = "CREATE_VIEW"
								case *mysql.CreateEventContext:
									result = "CREATE_EVENT"
								case *mysql.CreateTriggerContext:
									result = "CREATE_TRIGGER"
								case *mysql.CreateFunctionContext:
									result = "CREATE_FUNCTION"
								case *mysql.CreateProcedureContext:
									result = "CREATE_PROCEDURE"
								}
							}
						case *mysql.DropStatementContext:
							for _, child := range ctx.GetChildren() {
								switch child.(type) {
								case *mysql.DropIndexContext:
									result = "DROP_INDEX"
								case *mysql.DropTableContext:
									result = "DROP_TABLE"
								case *mysql.DropDatabaseContext:
									result = "DROP_DATABASE"
								case *mysql.DropViewContext:
									result = "DROP_VIEW"
								case *mysql.DropTriggerContext:
									result = "DROP_TRIGGER"
								case *mysql.DropEventContext:
									result = "DROP_EVENT"
								case *mysql.DropFunctionContext:
									result = "DROP_FUNCTION"
								case *mysql.DropProcedureContext:
									result = "DROP_PROCEDURE"
								}
							}
						case *mysql.AlterStatementContext:
							for _, child := range ctx.GetChildren() {
								switch child.(type) {
								case *mysql.AlterTableContext:
									result = "ALTER_TABLE"
								case *mysql.AlterDatabaseContext:
									result = "ALTER_DATABASE"
								case *mysql.AlterViewContext:
									result = "ALTER_VIEW"
								case *mysql.AlterEventContext:
									result = "ALTER_EVENT"
								}
							}
						case *mysql.TruncateTableStatementContext:
							result = "TRUNCATE"
						case *mysql.RenameTableStatementContext:
							result = "RENAME"

						// dml.
						case *mysql.DeleteStatementContext:
							result = "DELETE"
						case *mysql.InsertStatementContext:
							result = "INSERT"
						case *mysql.UpdateStatementContext:
							result = "UPDATE"
						}
					}
				default:
				}
			}
		default:
		}
	}
	return result
}

type AffectedRowsListener struct {
	*mysql.BaseMySQLParserListener

	ctx          context.Context
	sqlDB        *sql.DB
	text         string
	metadata     *storepb.DatabaseSchemaMetadata
	affectedRows int64
	err          error
}

// GetAffectedRows return the rows count affected by the sql.
func GetAffectedRows(ctx context.Context, sqlDB *sql.DB, metadata *storepb.DatabaseSchemaMetadata, stmt *ParseResult) (int64, error) {
	listener := &AffectedRowsListener{
		ctx:      ctx,
		sqlDB:    sqlDB,
		metadata: metadata,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, stmt.Tree)
	if listener.err != nil {
		return 0, listener.err
	}

	return listener.affectedRows, nil
}

func (l *AffectedRowsListener) EnterQuery(ctx *mysql.QueryContext) {
	l.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterInsertStatement is called when production insertStatement is entered.
func (l *AffectedRowsListener) EnterInsertStatement(ctx *mysql.InsertStatementContext) {
	if ctx.GetParent() == nil {
		return
	}
	simpleCtx, ok := ctx.GetParent().(*mysql.SimpleStatementContext)
	if !ok || simpleCtx.GetParent() == nil {
		return
	}
	if _, ok := simpleCtx.GetParent().(*mysql.QueryContext); !ok {
		return
	}

	if ctx.InsertQueryExpression() != nil {
		affectedRows, err := getAffectedRowsByQuery(l.ctx, l.sqlDB, l.text)
		if err != nil {
			l.err = err
			return
		}
		l.affectedRows = affectedRows
		return
	}

	if ctx.InsertFromConstructor() == nil || ctx.InsertFromConstructor().InsertValues() == nil || ctx.InsertFromConstructor().InsertValues().ValueList() == nil {
		return
	}

	l.affectedRows = int64(len(ctx.InsertFromConstructor().InsertValues().ValueList().AllValues()))
}

// EnterUpdateStatement is called when production updateStatement is entered.
func (l *AffectedRowsListener) EnterUpdateStatement(ctx *mysql.UpdateStatementContext) {
	if ctx.GetParent() == nil {
		return
	}
	simpleCtx, ok := ctx.GetParent().(*mysql.SimpleStatementContext)
	if !ok || simpleCtx.GetParent() == nil {
		return
	}
	if _, ok := simpleCtx.GetParent().(*mysql.QueryContext); !ok {
		return
	}
	affectedRows, err := getAffectedRowsByQuery(l.ctx, l.sqlDB, l.text)
	if err != nil {
		l.err = err
	}
	l.affectedRows = affectedRows
}

// EnterDeleteStatement is called when production deleteStatement is entered.
func (l *AffectedRowsListener) EnterDeleteStatement(ctx *mysql.DeleteStatementContext) {
	if ctx.GetParent() == nil {
		return
	}
	simpleCtx, ok := ctx.GetParent().(*mysql.SimpleStatementContext)
	if !ok || simpleCtx.GetParent() == nil {
		return
	}
	if _, ok := simpleCtx.GetParent().(*mysql.QueryContext); !ok {
		return
	}
	affectedRows, err := getAffectedRowsByQuery(l.ctx, l.sqlDB, l.text)
	if err != nil {
		l.err = err
	}
	l.affectedRows = affectedRows
}

// EnterAlterTable is called when production alterTable is entered.
func (l *AffectedRowsListener) EnterAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.GetParent() == nil {
		return
	}
	alertCtx, ok := ctx.GetParent().(*mysql.AlterStatementContext)
	if !ok || alertCtx.GetParent() == nil {
		return
	}
	simpleCtx, ok := alertCtx.GetParent().(*mysql.SimpleStatementContext)
	if !ok || simpleCtx.GetParent() == nil {
		return
	}
	if _, ok := simpleCtx.GetParent().(*mysql.QueryContext); !ok {
		return
	}

	if ctx.TableRef() == nil {
		return
	}
	databaseName, tableName := NormalizeMySQLTableRef(ctx.TableRef())
	l.affectedRows = l.getTableDataSize(databaseName, tableName)
}

// EnterDropTable is called when production dropTable is entered.
func (l *AffectedRowsListener) EnterDropTable(ctx *mysql.DropTableContext) {
	if ctx.GetParent() == nil {
		return
	}

	dropCtx, ok := ctx.GetParent().(*mysql.DropStatementContext)
	if !ok || dropCtx.GetParent() == nil {
		return
	}
	simpleCtx, ok := dropCtx.GetParent().(*mysql.SimpleStatementContext)
	if !ok || simpleCtx.GetParent() == nil {
		return
	}
	if _, ok := simpleCtx.GetParent().(*mysql.QueryContext); !ok {
		return
	}

	if ctx.TableRefList() == nil {
		return
	}
	for _, tableRef := range ctx.TableRefList().AllTableRef() {
		if tableRef == nil {
			continue
		}
		databaseName, tableName := NormalizeMySQLTableRef(tableRef)
		l.affectedRows += l.getTableDataSize(databaseName, tableName)
	}
}

func (l *AffectedRowsListener) getTableDataSize(schemaName, tableName string) int64 {
	if l.metadata == nil {
		return 0
	}
	for _, schema := range l.metadata.Schemas {
		if schema.Name != schemaName {
			continue
		}
		for _, table := range schema.Tables {
			if table.Name != tableName {
				continue
			}
			return table.RowCount
		}
	}
	return 0
}

func getAffectedRowsByQuery(ctx context.Context, sqlDB *sql.DB, statement string) (int64, error) {
	res, err := query(ctx, sqlDB, fmt.Sprintf("EXPLAIN %s", statement))
	if err != nil {
		return 0, err
	}
	rowCount, err := getAffectedRowsCount(res)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get affected rows count, res %+v", res)
	}
	return rowCount, nil
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

func getAffectedRowsCount(res []any) (int64, error) {
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

	// MySQL EXPLAIN statement result has 12 columns.
	// the column 9 is the data 'rows'.
	// the first not-NULL value of column 9 is the affected rows count.
	//
	// mysql> explain delete from td;
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------+
	// | id | select_type | table | partitions | type | possible_keys | key  | key_len | ref  | rows | filtered | Extra |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------+
	// |  1 | DELETE      | td    | NULL       | ALL  | NULL          | NULL | NULL    | NULL |    1 |   100.00 | NULL  |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-------+
	//
	// mysql> explain insert into td select * from td;
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-----------------+
	// | id | select_type | table | partitions | type | possible_keys | key  | key_len | ref  | rows | filtered | Extra           |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-----------------+
	// |  1 | INSERT      | td    | NULL       | ALL  | NULL          | NULL | NULL    | NULL | NULL |     NULL | NULL            |
	// |  1 | SIMPLE      | td    | NULL       | ALL  | NULL          | NULL | NULL    | NULL |    1 |   100.00 | Using temporary |
	// +----+-------------+-------+------------+------+---------------+------+---------+------+------+----------+-----------------+

	for _, rowAny := range rowList {
		row, ok := rowAny.([]any)
		if !ok {
			return 0, errors.Errorf("expected []any but got %t", row)
		}
		if len(row) != 12 {
			return 0, errors.Errorf("expected 12 but got %d", len(row))
		}
		switch col := row[9].(type) {
		case int:
			return int64(col), nil
		case int32:
			return int64(col), nil
		case int64:
			return col, nil
		case string:
			v, err := strconv.ParseInt(col, 10, 64)
			if err != nil {
				return 0, errors.Errorf("expected int or int64 but got string(%s)", col)
			}
			return v, nil
		default:
			continue
		}
	}

	return 0, errors.Errorf("failed to extract rows from query plan")
}
