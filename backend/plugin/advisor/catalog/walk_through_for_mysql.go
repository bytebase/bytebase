package catalog

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/parser/tidb"
)

// MySQLWalkThrough walks through MySQL AST and updates the database state.
func MySQLWalkThrough(d *DatabaseState, ast any) error {
	// We define the Catalog as Database -> Schema -> Table. The Schema is only for PostgreSQL.
	// So we use a Schema whose name is empty for other engines, such as MySQL.
	// If there is no empty-string-name schema, create it to avoid corner cases.
	d.GetOrCreateSchema("")

	nodeList, ok := ast.([]*mysqlparser.ParseResult)
	if !ok {
		return errors.Errorf("invalid ast type %T", ast)
	}
	for _, node := range nodeList {
		if err := d.mysqlChangeState(node); err != nil {
			return err
		}
	}

	return nil
}

type mysqlListener struct {
	*mysql.BaseMySQLParserListener

	baseLine      int
	lineNumber    int
	text          string
	databaseState *DatabaseState
	err           *WalkThroughError
}

func (l *mysqlListener) EnterQuery(ctx *mysql.QueryContext) {
	l.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
	l.lineNumber = l.baseLine + ctx.GetStart().GetLine()
}

func (d *DatabaseState) mysqlChangeState(in *mysqlparser.ParseResult) (err *WalkThroughError) {
	defer func() {
		if err == nil {
			return
		}
		if err.Line == 0 {
			err.Line = in.BaseLine
		}
	}()

	if d.IsDeleted() {
		return &WalkThroughError{
			Code:    code.DatabaseIsDeleted,
			Content: fmt.Sprintf("Database `%s` is deleted", d.name),
		}
	}

	listener := &mysqlListener{
		baseLine:      in.BaseLine,
		databaseState: d,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, in.Tree)
	if listener.err != nil {
		if listener.err.Line == 0 {
			listener.err.Line = listener.lineNumber
		}
		return listener.err
	}
	return nil
}

// EnterCreateTable is called when production createTable is entered.
func (l *mysqlListener) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableName() == nil {
		return
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if databaseName != "" && !l.databaseState.isCurrentDatabase(databaseName) {
		l.err = &WalkThroughError{
			Code:    code.NotCurrentDatabase,
			Content: fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, l.databaseState.name),
		}
		return
	}

	schema := l.databaseState.GetOrCreateSchema("")
	if _, exists := schema.getTable(tableName); exists {
		if ctx.IfNotExists() != nil {
			return
		}
		l.err = &WalkThroughError{
			Code:    code.TableExists,
			Content: fmt.Sprintf("Table `%s` already exists", tableName),
		}
		return
	}

	if ctx.DuplicateAsQueryExpression() != nil {
		l.err = &WalkThroughError{
			Code:    code.StatementCreateTableAs,
			Content: fmt.Sprintf("Disallow the CREATE TABLE AS statement but \"%s\" uses", l.text),
		}
		return
	}

	if ctx.LIKE_SYMBOL() != nil {
		_, referTable := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
		l.err = l.databaseState.mysqlCopyTable(databaseName, tableName, referTable)
		return
	}

	table, err := schema.CreateTable(tableName)
	if err != nil {
		l.err = err
		return
	}

	if ctx.TableElementList() == nil {
		return
	}

	hasAutoIncrement := false
	for _, tableElement := range ctx.TableElementList().AllTableElement() {
		switch {
		// handle column
		case tableElement.ColumnDefinition() != nil:
			if tableElement.ColumnDefinition().FieldDefinition() == nil || tableElement.ColumnDefinition().ColumnName() == nil {
				continue
			}
			if mysqlparser.IsAutoIncrement(tableElement.ColumnDefinition().FieldDefinition()) {
				if hasAutoIncrement {
					l.err = &WalkThroughError{
						Code: code.AutoIncrementExists,
						// The content comes from MySQL error content.
						Content: fmt.Sprintf("There can be only one auto column for table `%s`", table.name),
					}
				}
				hasAutoIncrement = true
			}
			_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
			if err := table.mysqlCreateColumn(columnName, tableElement.ColumnDefinition().FieldDefinition(), nil /* position */); err != nil {
				err.Line = l.baseLine + tableElement.GetStart().GetLine()
				l.err = err
				return
			}
		case tableElement.TableConstraintDef() != nil:
			if err := table.mysqlCreateConstraint(tableElement.TableConstraintDef()); err != nil {
				err.Line = tableElement.GetStart().GetLine()
				l.err = err
				return
			}
		default:
			// Ignore other table element types
		}
	}
}

// EnterDropTable is called when production dropTable is entered.
func (l *mysqlListener) EnterDropTable(ctx *mysql.DropTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableRefList() == nil {
		return
	}

	for _, tableRef := range ctx.TableRefList().AllTableRef() {
		databaseName, tableName := mysqlparser.NormalizeMySQLTableRef(tableRef)
		if databaseName != "" && !l.databaseState.isCurrentDatabase(databaseName) {
			l.err = &WalkThroughError{
				Code:    code.NotCurrentDatabase,
				Content: fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, tableName),
			}
		}

		schema := l.databaseState.GetOrCreateSchema("")

		table, exists := schema.getTable(tableName)
		if !exists {
			if ctx.IfExists() != nil {
				return
			}
			l.err = &WalkThroughError{
				Code:    code.TableNotExists,
				Content: fmt.Sprintf("Table `%s` does not exist", tableName),
			}
			return
		}

		// MySQL doesn't check view dependencies for DROP TABLE, so pass nil
		if err := schema.DropTable(table.name, nil); err != nil {
			l.err = err
			return
		}
	}
}

// EnterAlterTable is called when production alterTable is entered.
func (l *mysqlListener) EnterAlterTable(ctx *mysql.AlterTableContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableRef() == nil {
		// todo: maybe need to do error handle.
		return
	}

	databaseName, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	table, err := l.databaseState.mysqlFindTableState(databaseName, tableName)
	if err != nil {
		l.err = err
		return
	}

	if ctx.AlterTableActions() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}

	for _, option := range ctx.AlterTableActions().AlterCommandList().AlterList().AllCreateTableOptionsSpaceSeparated() {
		for _, op := range option.AllCreateTableOption() {
			switch {
			// engine.
			case op.ENGINE_SYMBOL() != nil:
				if op.EngineRef() == nil {
					continue
				}
				engine := op.EngineRef().GetText()
				table.SetEngine(engine)
			// table comment.
			case op.COMMENT_SYMBOL() != nil && op.TextStringLiteral() != nil:
				comment := mysqlparser.NormalizeMySQLTextStringLiteral(op.TextStringLiteral())
				table.SetComment(comment)
			// table collation.
			case op.DefaultCollation() != nil && op.DefaultCollation().CollationName() != nil:
				collation := mysqlparser.NormalizeMySQLCollationName(op.DefaultCollation().CollationName())
				table.SetCollation(collation)
			default:
			}
		}
	}

	// alter table add column, change column, modify column.
	for _, item := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if item == nil {
			continue
		}

		switch {
		case item.ADD_SYMBOL() != nil:
			switch {
			// add single column.
			case item.Identifier() != nil && item.FieldDefinition() != nil:
				columnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
				if err := table.mysqlCreateColumn(columnName, item.FieldDefinition(), positionFromPlaceContext(item.Place())); err != nil {
					l.err = err
					return
				}
			// add multi columns.
			case item.OPEN_PAR_SYMBOL() != nil && item.TableElementList() != nil:
				for _, tableElement := range item.TableElementList().AllTableElement() {
					if tableElement.ColumnDefinition() == nil || tableElement.ColumnDefinition().ColumnName() == nil || tableElement.ColumnDefinition().FieldDefinition() == nil {
						continue
					}
					_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
					if err := table.mysqlCreateColumn(columnName, tableElement.ColumnDefinition().FieldDefinition(), nil); err != nil {
						l.err = err
						return
					}
				}
			// add constraint.
			case item.TableConstraintDef() != nil:
				if err := table.mysqlCreateConstraint(item.TableConstraintDef()); err != nil {
					l.err = err
					return
				}
			default:
				// Ignore other ADD variations
			}
		// drop column or key.
		case item.DROP_SYMBOL() != nil && item.ALTER_SYMBOL() == nil:
			switch {
			// drop foreign key.
			// we do not deal with DROP FOREIGN KEY statements.
			case item.FOREIGN_SYMBOL() != nil && item.KEY_SYMBOL() != nil:
			// drop column.
			case item.ColumnInternalRef() != nil:
				columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
				if err := table.DropColumn(columnName, nil); err != nil {
					l.err = err
					return
				}
				// drop primary key.
			case item.PRIMARY_SYMBOL() != nil && item.KEY_SYMBOL() != nil:
				if err := table.DropIndex(PrimaryKeyName, nil); err != nil {
					l.err = err
					return
				}
				// drop key/index.
			case item.KeyOrIndex() != nil && item.IndexRef() != nil:
				_, _, indexName := mysqlparser.NormalizeIndexRef(item.IndexRef())
				if err := table.DropIndex(indexName, nil); err != nil {
					l.err = err
					return
				}
			default:
				// Ignore other DROP variations
			}
		// modify column.
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil:
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			if err := table.mysqlChangeColumn(columnName, columnName, item.FieldDefinition(), positionFromPlaceContext(item.Place())); err != nil {
				l.err = err
				return
			}
		// change column
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil:
			oldColumnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			newColumnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			if err := table.mysqlChangeColumn(oldColumnName, newColumnName, item.FieldDefinition(), positionFromPlaceContext(item.Place())); err != nil {
				l.err = err
				return
			}
		// rename column
		case item.RENAME_SYMBOL() != nil && item.COLUMN_SYMBOL() != nil:
			oldColumnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			newColumnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			if err := table.RenameColumn(oldColumnName, newColumnName); err != nil {
				l.err = err
				return
			}
		case item.ALTER_SYMBOL() != nil:
			switch {
			// alter column.
			case item.ColumnInternalRef() != nil:
				if err := table.mysqlAlterColumn(item); err != nil {
					l.err = err
					return
				}
			// alter index visibility.
			case item.INDEX_SYMBOL() != nil && item.IndexRef() != nil && item.Visibility() != nil:
				_, _, indexName := mysqlparser.NormalizeIndexRef(item.IndexRef())
				if err := table.mysqlChangeIndexVisibility(indexName, item.Visibility()); err != nil {
					l.err = err
					return
				}
			default:
			}
		// rename table.
		case item.RENAME_SYMBOL() != nil && item.TableName() != nil:
			_, newTableName := mysqlparser.NormalizeMySQLTableName(item.TableName())
			schema := l.databaseState.GetOrCreateSchema("")
			if err := schema.RenameTable(table.name, newTableName); err != nil {
				l.err = err
				return
			}
		// rename index.
		case item.RENAME_SYMBOL() != nil && item.KeyOrIndex() != nil && item.IndexRef() != nil && item.IndexName() != nil:
			_, _, oldIndexName := mysqlparser.NormalizeIndexRef(item.IndexRef())
			newIndexName := mysqlparser.NormalizeIndexName(item.IndexName())
			if err := table.RenameIndex(oldIndexName, newIndexName, nil); err != nil {
				l.err = err
				return
			}
		default:
			// Ignore other alter table actions
		}
	}
}

// EnterDropIndex is called when production dropIndex is entered.
func (l *mysqlListener) EnterDropIndex(ctx *mysql.DropIndexContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TableRef() == nil {
		return
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	table, err := l.databaseState.mysqlFindTableState(databaseName, tableName)
	if err != nil {
		l.err = err
		return
	}

	if ctx.IndexRef() == nil {
		return
	}

	_, _, indexName := mysqlparser.NormalizeIndexRef(ctx.IndexRef())
	if err := table.DropIndex(indexName, nil); err != nil {
		l.err = err
	}
}

func (l *mysqlListener) EnterCreateIndex(ctx *mysql.CreateIndexContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.CreateIndexTarget() == nil || ctx.CreateIndexTarget().TableRef() == nil {
		return
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.CreateIndexTarget().TableRef())
	table, err := l.databaseState.mysqlFindTableState(databaseName, tableName)
	if err != nil {
		l.err = err
		return
	}

	unique := false
	isSpatial := false
	tp := "BTREE"

	if ctx.GetType_() == nil {
		return
	}
	switch ctx.GetType_().GetTokenType() {
	case mysql.MySQLParserFULLTEXT_SYMBOL:
		tp = FullTextName
	case mysql.MySQLParserSPATIAL_SYMBOL:
		isSpatial = true
		tp = SpatialName
	case mysql.MySQLParserINDEX_SYMBOL:
	default:
		// Other index types
	}
	if ctx.UNIQUE_SYMBOL() != nil {
		unique = true
	}

	indexName := ""
	if ctx.IndexName() != nil {
		indexName = mysqlparser.NormalizeIndexName(ctx.IndexName())
	}
	if ctx.IndexNameAndType() != nil && ctx.IndexNameAndType().IndexName() != nil {
		indexName = mysqlparser.NormalizeIndexName(ctx.IndexNameAndType().IndexName())
	}

	if ctx.CreateIndexTarget() == nil || ctx.CreateIndexTarget().KeyListVariants() == nil {
		return
	}
	if err := table.mysqlValidateKeyListVariants(ctx.CreateIndexTarget().KeyListVariants(), false /* primary */, isSpatial); err != nil {
		l.err = err
		return
	}

	columnList := mysqlparser.NormalizeKeyListVariants(ctx.CreateIndexTarget().KeyListVariants())
	if err := table.mysqlCreateIndex(indexName, columnList, unique, tp, mysql.NewEmptyTableConstraintDefContext(), ctx); err != nil {
		l.err = err
		return
	}
}

// EnterAlterDatabase is called when production alterDatabase is entered.
func (l *mysqlListener) EnterAlterDatabase(ctx *mysql.AlterDatabaseContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.SchemaRef() != nil {
		databaseName := mysqlparser.NormalizeMySQLSchemaRef(ctx.SchemaRef())
		if !l.databaseState.isCurrentDatabase(databaseName) {
			l.err = NewAccessOtherDatabaseError(l.databaseState.name, databaseName)
			return
		}
	}

	for _, option := range ctx.AllAlterDatabaseOption() {
		if option.CreateDatabaseOption() == nil {
			continue
		}

		switch {
		case option.CreateDatabaseOption().DefaultCharset() != nil && option.CreateDatabaseOption().DefaultCharset().CharsetName() != nil:
			charset := mysqlparser.NormalizeMySQLCharsetName(option.CreateDatabaseOption().DefaultCharset().CharsetName())
			l.databaseState.SetCharacterSet(charset)
		case option.CreateDatabaseOption().DefaultCollation() != nil && option.CreateDatabaseOption().DefaultCollation().CollationName() != nil:
			collation := mysqlparser.NormalizeMySQLCollationName(option.CreateDatabaseOption().DefaultCollation().CollationName())
			l.databaseState.SetCollation(collation)
		default:
			// Other options
		}
	}
}

// EnterDropDatabase is called when production dropDatabase is entered.
func (l *mysqlListener) EnterDropDatabase(ctx *mysql.DropDatabaseContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.SchemaRef() == nil {
		return
	}

	databaseName := mysqlparser.NormalizeMySQLSchemaRef(ctx.SchemaRef())
	if !l.databaseState.isCurrentDatabase(databaseName) {
		l.err = NewAccessOtherDatabaseError(l.databaseState.name, databaseName)
		return
	}

	l.databaseState.MarkDeleted()
}

// EnterCreateDatabase is called when production createDatabase is entered.
func (l *mysqlListener) EnterCreateDatabase(ctx *mysql.CreateDatabaseContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.SchemaName() == nil {
		return
	}
	databaseName := mysqlparser.NormalizeMySQLSchemaName(ctx.SchemaName())
	l.err = NewAccessOtherDatabaseError(l.databaseState.name, databaseName)
}

// EnterRenameTableStatement is called when production renameTableStatement is entered.
func (l *mysqlListener) EnterRenameTableStatement(ctx *mysql.RenameTableStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	for _, pair := range ctx.AllRenamePair() {
		schema := l.databaseState.GetOrCreateSchema("")

		_, oldTableName := mysqlparser.NormalizeMySQLTableRef(pair.TableRef())
		_, newTableName := mysqlparser.NormalizeMySQLTableName(pair.TableName())

		if l.databaseState.mysqlTheCurrentDatabase(pair) {
			if compareIdentifier(oldTableName, newTableName, l.databaseState.ignoreCaseSensitive) {
				return
			}
			table, exists := schema.getTable(oldTableName)
			if !exists {
				l.err = NewTableNotExistsError(oldTableName)
				return
			}
			if _, exists := schema.getTable(newTableName); exists {
				l.err = NewTableExistsError(newTableName)
				return
			}
			delete(schema.tableSet, table.name)
			table.name = newTableName
			schema.tableSet[table.name] = table
		} else if l.databaseState.mysqlMoveToOtherDatabase(pair) {
			_, exists := schema.getTable(oldTableName)
			if !exists {
				l.err = NewTableNotExistsError(oldTableName)
				return
			}
			delete(schema.tableSet, oldTableName)
		} else {
			l.err = NewAccessOtherDatabaseError(l.databaseState.name, l.databaseState.mysqlTargetDatabase(pair))
			return
		}
	}
}

func (l *mysqlListener) EnterCreateTrigger(ctx *mysql.CreateTriggerContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.TriggerName() == nil {
		return
	}

	// Check if related table exists.
	if ctx.TableRef() == nil {
		return
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableRef(ctx.TableRef())
	_, err := l.databaseState.mysqlFindTableState(databaseName, tableName)
	if err != nil {
		l.err = err
		return
	}
}

func (*mysqlListener) EnterCreateProcedure(ctx *mysql.CreateProcedureContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.ProcedureName() == nil {
		return
	}
	// Skip other checks for now.
}

func (*mysqlListener) EnterCreateEvent(ctx *mysql.CreateEventContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	if ctx.EventName() == nil {
		return
	}
	// Skip other checks for now.
}

func (d *DatabaseState) mysqlTargetDatabase(renamePair mysql.IRenamePairContext) string {
	oldDatabaseName, _ := mysqlparser.NormalizeMySQLTableRef(renamePair.TableRef())
	if oldDatabaseName != "" && !d.isCurrentDatabase(oldDatabaseName) {
		return oldDatabaseName
	}
	newDatabaseName, _ := mysqlparser.NormalizeMySQLTableName(renamePair.TableName())
	return newDatabaseName
}

func (d *DatabaseState) mysqlMoveToOtherDatabase(renamePair mysql.IRenamePairContext) bool {
	oldDatabaseName, _ := mysqlparser.NormalizeMySQLTableRef(renamePair.TableRef())
	if oldDatabaseName != "" && !d.isCurrentDatabase(oldDatabaseName) {
		return false
	}
	newDatabaseName, _ := mysqlparser.NormalizeMySQLTableName(renamePair.TableName())
	return oldDatabaseName != newDatabaseName
}

func (d *DatabaseState) mysqlTheCurrentDatabase(renamePair mysql.IRenamePairContext) bool {
	newDatabaseName, _ := mysqlparser.NormalizeMySQLTableName(renamePair.TableName())
	if newDatabaseName != "" && !d.isCurrentDatabase(newDatabaseName) {
		return false
	}
	oldDatabaseName, _ := mysqlparser.NormalizeMySQLTableRef(renamePair.TableRef())
	if oldDatabaseName != "" && !d.isCurrentDatabase(oldDatabaseName) {
		return false
	}
	return true
}

func (t *TableState) mysqlChangeIndexVisibility(indexName string, visibility mysql.IVisibilityContext) *WalkThroughError {
	index, exists := t.indexSet[strings.ToLower(indexName)]
	if !exists {
		return NewIndexNotExistsError(t.name, indexName)
	}
	switch {
	case visibility.VISIBLE_SYMBOL() != nil:
		index.visible = newTruePointer()
	case visibility.INVISIBLE_SYMBOL() != nil:
		index.visible = newFalsePointer()
	default:
		// No visibility specified
	}
	return nil
}

func (t *TableState) mysqlAlterColumn(itemDef mysql.IAlterListItemContext) *WalkThroughError {
	if itemDef.ColumnInternalRef() == nil {
		// should not reach here.
		return nil
	}
	columnName := mysqlparser.NormalizeMySQLColumnInternalRef(itemDef.ColumnInternalRef())
	colState, exists := t.columnSet[strings.ToLower(columnName)]
	if !exists {
		return NewColumnNotExistsError(t.name, columnName)
	}

	switch {
	case itemDef.SET_SYMBOL() != nil:
		switch {
		// SET DEFAULT.
		case itemDef.DEFAULT_SYMBOL() != nil:
			if itemDef.SignedLiteral() != nil && itemDef.SignedLiteral().Literal() != nil && itemDef.SignedLiteral().Literal().NullLiteral() == nil {
				if colState.columnType != nil {
					switch strings.ToLower(*colState.columnType) {
					case "blob", "tinyblob", "mediumblob", "longblob",
						"text", "tinytext", "mediumtext", "longtext",
						"json",
						"geometry":
						return &WalkThroughError{
							Code: code.InvalidColumnDefault,
							// Content comes from MySQL Error content.
							Content: fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", columnName),
						}
					default:
						// Other column types allow default values
					}
				}

				var defaultValue string
				switch {
				case itemDef.ExprWithParentheses() != nil:
					defaultValue = itemDef.ExprWithParentheses().GetText()
				case itemDef.SignedLiteral() != nil:
					defaultValue = itemDef.SignedLiteral().GetText()
				default:
					// No default value expression
				}

				colState.defaultValue = &defaultValue
			} else {
				if colState.nullable != nil && !*colState.nullable {
					return &WalkThroughError{
						Code: code.SetNullDefaultForNotNullColumn,
						// Content comes from MySQL Error content.
						Content: fmt.Sprintf("Invalid default value for column `%s`", columnName),
					}
				}

				colState.defaultValue = nil
			}
		// SET VISIBLE/INVISIBLE.
		default:
		}
	case itemDef.DROP_SYMBOL() != nil && itemDef.DEFAULT_SYMBOL() != nil:
		// DROP DEFAULT.
		colState.defaultValue = nil
	default:
		// Other ALTER operations
	}
	return nil
}

func (t *TableState) mysqlChangeColumn(oldColumnName string, newColumnName string, fieldDef mysql.IFieldDefinitionContext, position *mysqlColumnPosition) *WalkThroughError {
	return t.mysqlCompleteTableChangeColumn(oldColumnName, newColumnName, fieldDef, position)
}

// mysqlCompleteTableChangeColumn changes column definition.
// It works as:
// 1. drop column from tableState.columnSet, but do not drop column from indexSet.
// 2. rename column from indexSet.
// 3. create a new column in columnSet.
func (t *TableState) mysqlCompleteTableChangeColumn(oldColumnName string, newColumnName string, fieldDef mysql.IFieldDefinitionContext, position *mysqlColumnPosition) *WalkThroughError {
	column, exists := t.columnSet[strings.ToLower(oldColumnName)]
	if !exists {
		return NewColumnNotExistsError(t.name, oldColumnName)
	}

	pos := *column.position

	if position == nil {
		position = &mysqlColumnPosition{
			tp: ColumnPositionNone,
		}
	}
	if position.tp == ColumnPositionNone {
		if pos == 1 {
			position.tp = ColumnPositionFirst
		} else {
			for _, col := range t.columnSet {
				if *col.position == pos-1 {
					position.tp = ColumnPositionAfter
					position.relativeColumn = col.name
					break
				}
			}
		}
	}

	// drop column from columnSet.
	for _, col := range t.columnSet {
		if *col.position > pos {
			*col.position--
		}
	}
	delete(t.columnSet, strings.ToLower(column.name))

	// rename column from indexSet
	t.renameColumnInIndexKey(oldColumnName, newColumnName)

	// create a new column in columnSet
	return t.mysqlCreateColumn(newColumnName, fieldDef, position)
}

type columnPositionType int

const (
	ColumnPositionNone columnPositionType = iota
	ColumnPositionFirst
	ColumnPositionAfter
)

type mysqlColumnPosition struct {
	tp             columnPositionType
	relativeColumn string
}

func positionFromPlaceContext(place mysql.IPlaceContext) *mysqlColumnPosition {
	columnPosition := &mysqlColumnPosition{
		tp: ColumnPositionNone,
	}
	if place, ok := place.(*mysql.PlaceContext); ok {
		if place != nil {
			switch {
			case place.FIRST_SYMBOL() != nil:
				columnPosition.tp = ColumnPositionFirst
			case place.AFTER_SYMBOL() != nil:
				columnPosition.tp = ColumnPositionAfter
				columnName := mysqlparser.NormalizeMySQLIdentifier(place.Identifier())
				columnPosition.relativeColumn = columnName
			default:
				// No position specified
			}
		}
	}
	return columnPosition
}

func (d *DatabaseState) mysqlCopyTable(databaseName, tableName, referTable string) *WalkThroughError {
	targetTable, err := d.mysqlFindTableState(databaseName, referTable)
	if err != nil {
		return err
	}

	schema := d.GetOrCreateSchema("")
	table := targetTable.copy()
	table.name = tableName
	schema.tableSet[table.name] = table
	return nil
}

func (d *DatabaseState) mysqlFindTableState(databaseName, tableName string) (*TableState, *WalkThroughError) {
	if databaseName != "" && !d.isCurrentDatabase(databaseName) {
		return nil, NewAccessOtherDatabaseError(d.name, databaseName)
	}

	schema := d.GetOrCreateSchema("")

	table, exists := schema.getTable(tableName)
	if !exists {
		return nil, NewTableNotExistsError(tableName)
	}

	return table, nil
}

func (t *TableState) mysqlCreateConstraint(constraintDef mysql.ITableConstraintDefContext) *WalkThroughError {
	if constraintDef.GetType_() != nil {
		switch constraintDef.GetType_().GetTokenType() {
		// PRIMARY KEY.
		case mysql.MySQLParserPRIMARY_SYMBOL:
			if constraintDef.KeyListVariants() == nil {
				// never reach here.
				return nil
			}
			if err := t.mysqlValidateKeyListVariants(constraintDef.KeyListVariants(), true /* primary */, false /* isSpatial*/); err != nil {
				return err
			}
			keyList := mysqlparser.NormalizeKeyListVariants(constraintDef.KeyListVariants())
			if err := t.mysqlCreatePrimaryKey(keyList, mysqlGetIndexType(constraintDef)); err != nil {
				return err
			}
		// normal KEY/INDEX.
		case mysql.MySQLParserKEY_SYMBOL, mysql.MySQLParserINDEX_SYMBOL:
			if constraintDef.KeyListVariants() == nil {
				// never reach here.
				return nil
			}
			if err := t.mysqlValidateKeyListVariants(constraintDef.KeyListVariants(), false /* primary */, false /* isSpatial */); err != nil {
				return err
			}

			indexName := ""
			if constraintDef.IndexNameAndType() != nil && constraintDef.IndexNameAndType().IndexName() != nil {
				indexName = mysqlparser.NormalizeIndexName(constraintDef.IndexNameAndType().IndexName())
			}
			keyList := mysqlparser.NormalizeKeyListVariants(constraintDef.KeyListVariants())
			if err := t.mysqlCreateIndex(indexName, keyList, false /* unique */, mysqlGetIndexType(constraintDef), constraintDef, mysql.NewEmptyCreateIndexContext()); err != nil {
				return err
			}
		// UNIQUE KEY.
		case mysql.MySQLParserUNIQUE_SYMBOL:
			if constraintDef.KeyListVariants() == nil {
				// never reach here.
				return nil
			}
			if err := t.mysqlValidateKeyListVariants(constraintDef.KeyListVariants(), false /* primary */, false /* isSpatial*/); err != nil {
				return err
			}

			indexName := ""
			if constraintDef.ConstraintName() != nil {
				indexName = mysqlparser.NormalizeConstraintName(constraintDef.ConstraintName())
			}
			if constraintDef.IndexNameAndType() != nil && constraintDef.IndexNameAndType().IndexName() != nil {
				indexName = mysqlparser.NormalizeIndexName(constraintDef.IndexNameAndType().IndexName())
			}
			keyList := mysqlparser.NormalizeKeyListVariants(constraintDef.KeyListVariants())
			if err := t.mysqlCreateIndex(indexName, keyList, true /* unique */, mysqlGetIndexType(constraintDef), constraintDef, mysql.NewEmptyCreateIndexContext()); err != nil {
				return err
			}
		// FULLTEXT KEY.
		case mysql.MySQLParserFULLTEXT_SYMBOL:
			if constraintDef.KeyListVariants() == nil {
				// never reach here.
				return nil
			}
			if err := t.mysqlValidateKeyListVariants(constraintDef.KeyListVariants(), false /* primary */, false /* isSpatial*/); err != nil {
				return err
			}
			indexName := ""
			if constraintDef.IndexName() != nil {
				indexName = mysqlparser.NormalizeIndexName(constraintDef.IndexName())
			}
			keyList := mysqlparser.NormalizeKeyListVariants(constraintDef.KeyListVariants())
			if err := t.mysqlCreateIndex(indexName, keyList, false /* unique */, mysqlGetIndexType(constraintDef), constraintDef, mysql.NewEmptyCreateIndexContext()); err != nil {
				return err
			}
		case mysql.MySQLParserFOREIGN_SYMBOL:
			// we do not deal with FOREIGN KEY constraints.
		default:
			// Other constraint types
		}
	}

	// we do not deal with check constraints.
	// if constraintDef.CheckConstraint() != nil {}
	return nil
}

// mysqlValidateKeyListVariants validates the key list variants.
func (t *TableState) mysqlValidateKeyListVariants(keyList mysql.IKeyListVariantsContext, primary bool, isSpatial bool) *WalkThroughError {
	if keyList.KeyList() != nil {
		columns := mysqlparser.NormalizeKeyList(keyList.KeyList())
		if err := t.mysqlValidateColumnList(columns, primary, isSpatial); err != nil {
			return err
		}
	}
	if keyList.KeyListWithExpression() != nil {
		expressions := mysqlparser.NormalizeKeyListWithExpression(keyList.KeyListWithExpression())
		if err := t.mysqlValidateExpressionList(expressions, primary, isSpatial); err != nil {
			return err
		}
	}
	return nil
}

func (t *TableState) mysqlValidateColumnList(columnList []string, primary bool, isSpatial bool) *WalkThroughError {
	for _, columnName := range columnList {
		column, exists := t.columnSet[strings.ToLower(columnName)]
		if !exists {
			return NewColumnNotExistsError(t.name, columnName)
		}
		if primary {
			column.nullable = newFalsePointer()
		}
		if isSpatial && column.nullable != nil && *column.nullable {
			return &WalkThroughError{
				Code: code.SpatialIndexKeyNullable,
				// The error content comes from MySQL.
				Content: fmt.Sprintf("All parts of a SPATIAL index must be NOT NULL, but `%s` is nullable", column.name),
			}
		}
	}
	return nil
}

// mysqlValidateExpressionList validates the expression list.
// TODO: update expression validation.
func (t *TableState) mysqlValidateExpressionList(expressionList []string, primary bool, isSpatial bool) *WalkThroughError {
	for _, expression := range expressionList {
		column, exists := t.columnSet[strings.ToLower(expression)]
		// If expression is not a column, we do not need to validate it.
		if !exists {
			continue
		}

		if primary {
			column.nullable = newFalsePointer()
		}
		if isSpatial && column.nullable != nil && *column.nullable {
			return &WalkThroughError{
				Code: code.SpatialIndexKeyNullable,
				// The error content comes from MySQL.
				Content: fmt.Sprintf("All parts of a SPATIAL index must be NOT NULL, but `%s` is nullable", column.name),
			}
		}
	}
	return nil
}

func mysqlGetIndexType(tableConstraint mysql.ITableConstraintDefContext) string {
	if tableConstraint.GetType_() == nil {
		return "BTREE"
	}

	// I still need to handle IndexNameAndType to get index type(algorithm).
	switch tableConstraint.GetType_().GetTokenType() {
	case mysql.MySQLParserPRIMARY_SYMBOL,
		mysql.MySQLParserKEY_SYMBOL,
		mysql.MySQLParserINDEX_SYMBOL,
		mysql.MySQLParserUNIQUE_SYMBOL:

		if tableConstraint.IndexNameAndType() != nil {
			if tableConstraint.IndexNameAndType().IndexType() != nil {
				indexType := tableConstraint.IndexNameAndType().IndexType().GetText()
				return strings.ToUpper(indexType)
			}
		}

		for _, option := range tableConstraint.AllIndexOption() {
			if option == nil || option.IndexTypeClause() == nil {
				continue
			}

			indexType := option.IndexTypeClause().IndexType().GetText()
			return strings.ToUpper(indexType)
		}
	case mysql.MySQLParserFULLTEXT_SYMBOL:
		return "FULLTEXT"
	case mysql.MySQLParserFOREIGN_SYMBOL:
		// Foreign key - no specific index type
	default:
		// Other constraint types
	}
	// for mysql, we use BTREE as default index type.
	return "BTREE"
}

func (t *TableState) mysqlCreateColumn(columnName string, fieldDef mysql.IFieldDefinitionContext, position *mysqlColumnPosition) *WalkThroughError {
	if _, exists := t.columnSet[strings.ToLower(columnName)]; exists {
		return &WalkThroughError{
			Code:    code.ColumnExists,
			Content: fmt.Sprintf("Column `%s` already exists in table `%s`", columnName, t.name),
		}
	}

	// todo: handle position.
	pos := len(t.columnSet) + 1
	if position != nil {
		var err *WalkThroughError
		pos, err = t.mysqlReorderColumn(position)
		if err != nil {
			return err
		}
	}
	columnType := ""
	characterSet := ""
	collation := ""
	if fieldDef.DataType() == nil {
		// todo: add more error detail.
		return nil
	}
	columnType = mysqlparser.NormalizeMySQLDataType(fieldDef.DataType(), true /* compact */)
	characterSet = mysqlparser.GetCharSetName(fieldDef.DataType())
	collation = mysqlparser.GetCollationName(fieldDef)

	col := &ColumnState{
		name:         columnName,
		position:     &pos,
		defaultValue: nil,
		nullable:     newTruePointer(),
		columnType:   newStringPointer(columnType),
		characterSet: newStringPointer(characterSet),
		collation:    newStringPointer(collation),
		comment:      newEmptyStringPointer(),
	}
	setNullDefault := false

	for _, attribute := range fieldDef.AllColumnAttribute() {
		if attribute == nil {
			continue
		}
		if attribute.CheckConstraint() != nil {
			// we do not deal with CHECK constraint.
			continue
		}
		// not null.
		if attribute.NullLiteral() != nil && attribute.NOT_SYMBOL() != nil {
			col.nullable = newFalsePointer()
		}
		if attribute.GetValue() != nil {
			switch attribute.GetValue().GetTokenType() {
			// default value.
			case mysql.MySQLParserDEFAULT_SYMBOL:
				if err := mysqlCheckDefault(columnName, fieldDef); err != nil {
					return err
				}
				if attribute.SignedLiteral() == nil {
					continue
				}
				// handle default null.
				if attribute.SignedLiteral().Literal() != nil && attribute.SignedLiteral().Literal().NullLiteral() != nil {
					setNullDefault = true
					continue
				}
				// handle default 'null' etc.
				defaultValue := mysqlparser.NormalizeMySQLSignedLiteral(attribute.SignedLiteral())
				col.defaultValue = &defaultValue
			// comment.
			case mysql.MySQLParserCOMMENT_SYMBOL:
				if attribute.TextLiteral() == nil {
					continue
				}
				comment := mysqlparser.NormalizeMySQLTextLiteral(attribute.TextLiteral())
				col.comment = &comment
			// on update now().
			case mysql.MySQLParserON_SYMBOL:
				if attribute.UPDATE_SYMBOL() == nil || attribute.NOW_SYMBOL() == nil {
					continue
				}
				if !mysqlparser.IsTimeType(fieldDef.DataType()) {
					return &WalkThroughError{
						Code:    code.OnUpdateColumnNotDatetimeOrTimestamp,
						Content: fmt.Sprintf("Column `%s` use ON UPDATE but is not DATETIME or TIMESTAMP", col.name),
					}
				}
			// primary key.
			case mysql.MySQLParserKEY_SYMBOL:
				// the key attribute for in a column meaning primary key.
				col.nullable = newFalsePointer()
				// we need to check the key type which generated by tidb parser.
				if err := t.mysqlCreatePrimaryKey([]string{strings.ToLower(col.name)}, "BTREE"); err != nil {
					return err
				}
			// unique key.
			case mysql.MySQLParserUNIQUE_SYMBOL:
				// unique index.
				if err := t.mysqlCreateIndex("", []string{strings.ToLower(col.name)}, true /* unique */, "BTREE", mysql.NewEmptyTableConstraintDefContext(), mysql.NewEmptyCreateIndexContext()); err != nil {
					return err
				}
			// auto_increment.
			case mysql.MySQLParserAUTO_INCREMENT_SYMBOL:
				// we do not deal with AUTO_INCREMENT.
			// column_format.
			case mysql.MySQLParserCOLUMN_FORMAT_SYMBOL:
				// we do not deal with COLUMN_FORMAT.
			// storage.
			case mysql.MySQLParserSTORAGE_SYMBOL:
				// we do not deal with STORAGE.
			default:
				// Other column attributes
			}
		}
	}

	if col.nullable != nil && !*col.nullable && setNullDefault {
		return &WalkThroughError{
			Code: code.SetNullDefaultForNotNullColumn,
			// Content comes from MySQL Error content.
			Content: fmt.Sprintf("Invalid default value for column `%s`", col.name),
		}
	}

	t.columnSet[strings.ToLower(col.name)] = col
	return nil
}

// reorderColumn reorders the columns for new column and returns the new column position.
func (t *TableState) mysqlReorderColumn(position *mysqlColumnPosition) (int, *WalkThroughError) {
	switch position.tp {
	case ColumnPositionNone:
		return len(t.columnSet) + 1, nil
	case ColumnPositionFirst:
		for _, column := range t.columnSet {
			*column.position++
		}
		return 1, nil
	case ColumnPositionAfter:
		columnName := strings.ToLower(position.relativeColumn)
		column, exist := t.columnSet[columnName]
		if !exist {
			return 0, NewColumnNotExistsError(t.name, columnName)
		}
		for _, col := range t.columnSet {
			if *col.position > *column.position {
				*col.position++
			}
		}
		return *column.position + 1, nil
	default:
		return 0, &WalkThroughError{
			Code:    code.Unsupported,
			Content: fmt.Sprintf("Unsupported column position type: %d", position.tp),
		}
	}
}

func (t *TableState) mysqlCreateIndex(name string, keyList []string, unique bool, tp string, tableConstraint mysql.ITableConstraintDefContext, createIndexDef mysql.ICreateIndexContext) *WalkThroughError {
	if len(keyList) == 0 {
		return &WalkThroughError{
			Code:    code.IndexEmptyKeys,
			Content: fmt.Sprintf("Index `%s` in table `%s` has empty key", name, t.name),
		}
	}
	// construct a index name if name is empty.
	if name != "" {
		if _, exists := t.indexSet[strings.ToLower(name)]; exists {
			return NewIndexExistsError(t.name, name)
		}
	} else {
		suffix := 1
		for {
			name = keyList[0]
			if suffix > 1 {
				name = fmt.Sprintf("%s_%d", keyList[0], suffix)
			}
			if _, exists := t.indexSet[strings.ToLower(name)]; !exists {
				break
			}
			suffix++
		}
	}

	index := &IndexState{
		name:           name,
		expressionList: keyList,
		indexType:      &tp,
		unique:         &unique,
		primary:        newFalsePointer(),
		visible:        newTruePointer(),
		comment:        newEmptyStringPointer(),
	}

	// need to check the visibility of index.
	// we need a for-loop to determined the visibility of index.

	// NORMAL KEY/INDEX.
	// PRIMARY KEY.
	// UNIQUE KEY.

	// for create table statement.
	for _, attribute := range tableConstraint.AllIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			index.visible = newFalsePointer()
		}
	}

	// for create index statement.
	for _, attribute := range createIndexDef.AllIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			index.visible = newFalsePointer()
		}
	}

	// FULLTEXT INDEX.
	// for create table statement.
	for _, attribute := range tableConstraint.AllFulltextIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			index.visible = newFalsePointer()
		}
	}

	// for create index statement.
	for _, attribute := range createIndexDef.AllFulltextIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			index.visible = newFalsePointer()
		}
	}

	// SPATIAL INDEX.
	// for create table statement.
	for _, attribute := range tableConstraint.AllSpatialIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			index.visible = newFalsePointer()
		}
	}

	// for create index statement.
	for _, attribute := range createIndexDef.AllSpatialIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			index.visible = newFalsePointer()
		}
	}

	t.indexSet[strings.ToLower(name)] = index
	return nil
}

func (t *TableState) mysqlCreatePrimaryKey(keys []string, tp string) *WalkThroughError {
	if _, exists := t.indexSet[strings.ToLower(PrimaryKeyName)]; exists {
		return &WalkThroughError{
			Code:    code.PrimaryKeyExists,
			Content: fmt.Sprintf("Primary key exists in table `%s`", t.name),
		}
	}

	pk := &IndexState{
		name:           PrimaryKeyName,
		expressionList: keys,
		indexType:      &tp,
		unique:         newTruePointer(),
		primary:        newTruePointer(),
		visible:        newTruePointer(),
		comment:        newEmptyStringPointer(),
	}
	t.indexSet[strings.ToLower(pk.name)] = pk
	return nil
}

func mysqlCheckDefault(columnName string, fieldDefinition mysql.IFieldDefinitionContext) *WalkThroughError {
	if fieldDefinition.DataType() == nil || fieldDefinition.DataType().GetType_() == nil {
		return nil
	}

	switch fieldDefinition.DataType().GetType_().GetTokenType() {
	case mysql.MySQLParserTEXT_SYMBOL,
		mysql.MySQLParserTINYTEXT_SYMBOL,
		mysql.MySQLParserMEDIUMTEXT_SYMBOL,
		mysql.MySQLParserLONGTEXT_SYMBOL,
		mysql.MySQLParserBLOB_SYMBOL,
		mysql.MySQLParserTINYBLOB_SYMBOL,
		mysql.MySQLParserMEDIUMBLOB_SYMBOL,
		mysql.MySQLParserLONGBLOB_SYMBOL,
		mysql.MySQLParserLONG_SYMBOL,
		mysql.MySQLParserSERIAL_SYMBOL,
		mysql.MySQLParserJSON_SYMBOL,
		mysql.MySQLParserGEOMETRY_SYMBOL,
		mysql.MySQLParserGEOMETRYCOLLECTION_SYMBOL,
		mysql.MySQLParserPOINT_SYMBOL,
		mysql.MySQLParserMULTIPOINT_SYMBOL,
		mysql.MySQLParserLINESTRING_SYMBOL,
		mysql.MySQLParserMULTILINESTRING_SYMBOL,
		mysql.MySQLParserPOLYGON_SYMBOL,
		mysql.MySQLParserMULTIPOLYGON_SYMBOL:
		return &WalkThroughError{
			Code: code.InvalidColumnDefault,
			// Content comes from MySQL Error content.
			Content: fmt.Sprintf("BLOB, TEXT, GEOMETRY or JSON column `%s` can't have a default value", columnName),
		}
	default:
		// Other data types are allowed to have default values
	}

	return checkDefaultConvert(columnName, fieldDefinition)
}

func checkDefaultConvert(columnName string, fieldDefinition mysql.IFieldDefinitionContext) *WalkThroughError {
	if fieldDefinition == nil {
		return nil
	}
	list, err := tidb.ParseTiDB(fmt.Sprintf("CREATE TABLE t(%s %s)", columnName, fieldDefinition.GetParser().GetTokenStream().GetTextFromRuleContext(fieldDefinition)), "", "")
	if err != nil {
		// For now, we do not handle this case.
		// nolint:nilerr
		return nil
	}
	if len(list) != 1 {
		return nil
	}
	createTable, ok := list[0].(*tidbast.CreateTableStmt)
	if !ok {
		return nil
	}
	if len(createTable.Cols) != 1 {
		return nil
	}
	col := createTable.Cols[0]
	for _, option := range col.Options {
		if option.Tp == tidbast.ColumnOptionDefaultValue {
			return checkDefault(columnName, col.Tp, option.Expr)
		}
	}

	return nil
}
