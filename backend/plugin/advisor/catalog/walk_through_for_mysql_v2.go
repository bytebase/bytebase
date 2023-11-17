package catalog

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

func (d *DatabaseState) mysqlV2WalkThrough(stmt string) error {
	// We define the Catalog as Database -> Schema -> Table. The Schema is only for PostgreSQL.
	// So we use a Schema whose name is empty for other engines, such as MySQL.
	// If there is no empty-string-name schema, create it to avoid corner cases.
	if _, exists := d.schemaSet[""]; !exists {
		d.createSchema("")
	}

	nodeList, err := mysqlparser.ParseMySQL(stmt + ";")
	if err != nil {
		return NewParseError(err.Error())
	}
	for _, node := range nodeList {
		if err := d.mysqlV2ChangeState(node); err != nil {
			return err
		}
	}

	return nil
}

type mysqlV2Listener struct {
	*mysql.BaseMySQLParserListener

	baseLine      int
	databaseState *DatabaseState
	err           *WalkThroughError
}

func (d *DatabaseState) mysqlV2ChangeState(in *mysqlparser.ParseResult) (err *WalkThroughError) {
	defer func() {
		if err == nil {
			return
		}
		if err.Line == 0 {
			err.Line = in.BaseLine
		}
	}()

	if d.deleted {
		return &WalkThroughError{
			Type:    ErrorTypeDatabaseIsDeleted,
			Content: fmt.Sprintf("Database `%s` is deleted", d.name),
		}
	}

	listener := &mysqlV2Listener{
		baseLine:      in.BaseLine,
		databaseState: d,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, in.Tree)
	if listener.err != nil {
		return listener.err
	}
	return nil
}

// EnterCreateTable is called when production createTable is entered.
func (l *mysqlV2Listener) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if ctx.TableName() == nil {
		return
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if databaseName != "" && !l.databaseState.isCurrentDatabase(databaseName) {
		l.err = &WalkThroughError{
			Type:    ErrorTypeAccessOtherDatabase,
			Content: fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, l.databaseState.name),
		}
		return
	}

	schema, exists := l.databaseState.schemaSet[""]
	if !exists {
		schema = l.databaseState.createSchema("")
	}
	if _, exists = schema.getTable(tableName); exists {
		if ctx.IfNotExists() != nil {
			return
		}
		l.err = &WalkThroughError{
			Type:    ErrorTypeTableExists,
			Content: fmt.Sprintf("Table `%s` already exists", tableName),
		}
		return
	}

	table := &TableState{
		name:      tableName,
		engine:    newEmptyStringPointer(),
		collation: newEmptyStringPointer(),
		comment:   newEmptyStringPointer(),
		columnSet: make(columnStateMap),
		indexSet:  make(IndexStateMap),
	}
	schema.tableSet[table.name] = table

	if ctx.TableElementList() == nil {
		return
	}

	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		switch {
		// handle column
		case tableElement.ColumnDefinition() != nil:
			if tableElement.ColumnDefinition().FieldDefinition() == nil {
				continue
			}
			if err := table.mysqlV2CreateColumn(l.databaseState.ctx, tableElement.ColumnDefinition()); err != nil {
				err.Line = l.baseLine + tableElement.GetStart().GetLine()
				l.err = err
				return
			}
		default:
		}
	}
}

func (t *TableState) mysqlV2CreateColumn(_ *FinderContext, columnDef mysql.IColumnDefinitionContext) *WalkThroughError {
	if columnDef.ColumnName() == nil || columnDef.FieldDefinition() == nil {
		// todo: add more error info
		return nil
	}
	_, _, columnName := mysqlparser.NormalizeMySQLColumnName(columnDef.ColumnName())
	if _, exists := t.columnSet[columnName]; exists {
		return &WalkThroughError{
			Type:    ErrorTypeColumnExists,
			Content: fmt.Sprintf("Column `%s` already exists in table `%s`", columnName, t.name),
		}
	}

	// todo: handle position.
	pos := len(t.columnSet) + 1
	columnType := ""
	characterSet := ""
	collation := ""
	// todo: use fail-open pattern.
	if columnDef.FieldDefinition() != nil && columnDef.FieldDefinition().DataType() != nil {
		columnType = mysqlparser.NormalizeMySQLDataType(columnDef.FieldDefinition().DataType(), true /* compact */)
		characterSet = mysqlparser.GetCharSetName(columnDef.FieldDefinition().DataType())
		collation = mysqlparser.GetCollationName(columnDef.FieldDefinition())
	}

	vTrue := true
	col := &ColumnState{
		name:         columnName,
		position:     &pos,
		defaultValue: nil,
		nullable:     &vTrue,
		columnType:   newStringPointer(columnType),
		characterSet: newStringPointer(characterSet),
		collation:    newStringPointer(collation),
		comment:      newEmptyStringPointer(),
	}
	setNullDefault := false

	if col.nullable != nil && !*col.nullable && setNullDefault {
		return &WalkThroughError{
			Type: ErrorTypeSetNullDefaultForNotNullColumn,
			// Content comes from MySQL Error content.
			Content: fmt.Sprintf("Invalid default value for column `%s`", col.name),
		}
	}

	for _, attribute := range columnDef.FieldDefinition().AllColumnAttribute() {
		if attribute == nil {
			continue
		}
	}

	t.columnSet[col.name] = col
	return nil
}
