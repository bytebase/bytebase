package catalog

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/mysql"
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	"github.com/bytebase/bytebase/backend/store/model"
)

// MySQLWalkThrough walks through MySQL AST and updates the database metadata.
func MySQLWalkThrough(d *model.DatabaseMetadata, ast any) error {
	// We define the Catalog as Database -> Schema -> Table. The Schema is only for PostgreSQL.
	// So we use a Schema whose name is empty for other engines, such as MySQL.
	// If there is no empty-string-name schema, create it to avoid corner cases.
	if d.GetSchema("") == nil {
		d.CreateSchema("")
	}

	nodeList, ok := ast.([]*mysqlparser.ParseResult)
	if !ok {
		return errors.Errorf("invalid ast type %T", ast)
	}
	for _, node := range nodeList {
		if err := mysqlChangeState(d, node); err != nil {
			return err
		}
	}

	return nil
}

type mysqlListener struct {
	*mysql.BaseMySQLParserListener

	baseLine         int
	lineNumber       int
	text             string
	databaseMetadata *model.DatabaseMetadata
	err              *WalkThroughError
}

func (l *mysqlListener) EnterQuery(ctx *mysql.QueryContext) {
	l.text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
	l.lineNumber = l.baseLine + ctx.GetStart().GetLine()
}

func mysqlChangeState(d *model.DatabaseMetadata, in *mysqlparser.ParseResult) (err *WalkThroughError) {
	defer func() {
		if err == nil {
			return
		}
		if err.Line == 0 {
			err.Line = in.BaseLine
		}
	}()

	listener := &mysqlListener{
		baseLine:         in.BaseLine,
		databaseMetadata: d,
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
	if databaseName != "" && !isCurrentDatabase(l.databaseMetadata, databaseName) {
		l.err = &WalkThroughError{
			Code:    code.NotCurrentDatabase,
			Content: fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, l.databaseMetadata.DatabaseName()),
		}
		return
	}

	schema := l.databaseMetadata.GetSchema("")
	if schema == nil {
		l.err = &WalkThroughError{
			Code:    code.SchemaNotExists,
			Content: "Schema does not exist",
		}
		return
	}
	if schema.GetTable(tableName) != nil {
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
		l.err = mysqlCopyTable(l.databaseMetadata, databaseName, tableName, referTable)
		return
	}

	table, err := schema.CreateTable(tableName)
	if err != nil {
		l.err = &WalkThroughError{Code: code.TableExists, Content: err.Error()}
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
						Content: fmt.Sprintf("There can be only one auto column for table `%s`", table.GetProto().Name),
					}
				}
				hasAutoIncrement = true
			}
			_, _, columnName := mysqlparser.NormalizeMySQLColumnName(tableElement.ColumnDefinition().ColumnName())
			if err := mysqlCreateColumn(table, columnName, tableElement.ColumnDefinition().FieldDefinition(), nil /* position */); err != nil {
				err.Line = l.baseLine + tableElement.GetStart().GetLine()
				l.err = err
				return
			}
		case tableElement.TableConstraintDef() != nil:
			if err := mysqlCreateConstraint(table, tableElement.TableConstraintDef()); err != nil {
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
		if databaseName != "" && !isCurrentDatabase(l.databaseMetadata, databaseName) {
			l.err = &WalkThroughError{
				Code:    code.NotCurrentDatabase,
				Content: fmt.Sprintf("Database `%s` is not the current database `%s`", databaseName, tableName),
			}
		}

		schema := l.databaseMetadata.GetSchema("")
		if schema == nil {
			l.err = &WalkThroughError{
				Code:    code.SchemaNotExists,
				Content: "Schema does not exist",
			}
			return
		}

		table := schema.GetTable(tableName)
		if table == nil {
			if ctx.IfExists() != nil {
				return
			}
			l.err = &WalkThroughError{
				Code:    code.TableNotExists,
				Content: fmt.Sprintf("Table `%s` does not exist", tableName),
			}
			return
		}

		// MySQL doesn't check view dependencies for DROP TABLE
		if err := schema.DropTable(tableName); err != nil {
			l.err = &WalkThroughError{Code: code.TableNotExists, Content: err.Error()}
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
	table, err := mysqlFindTableState(l.databaseMetadata, databaseName, tableName)
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
				table.GetProto().Engine = engine
			// table comment.
			case op.COMMENT_SYMBOL() != nil && op.TextStringLiteral() != nil:
				comment := mysqlparser.NormalizeMySQLTextStringLiteral(op.TextStringLiteral())
				table.GetProto().Comment = comment
			// table collation.
			case op.DefaultCollation() != nil && op.DefaultCollation().CollationName() != nil:
				collation := mysqlparser.NormalizeMySQLCollationName(op.DefaultCollation().CollationName())
				table.GetProto().Collation = collation
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
				if err := mysqlCreateColumn(table, columnName, item.FieldDefinition(), positionFromPlaceContext(item.Place())); err != nil {
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
					if err := mysqlCreateColumn(table, columnName, tableElement.ColumnDefinition().FieldDefinition(), nil); err != nil {
						l.err = err
						return
					}
				}
			// add constraint.
			case item.TableConstraintDef() != nil:
				if err := mysqlCreateConstraint(table, item.TableConstraintDef()); err != nil {
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
				// Validate column exists
				if table.GetColumn(columnName) == nil {
					l.err = NewColumnNotExistsError(table.GetProto().Name, columnName)
					return
				}
				if err := table.DropColumn(columnName); err != nil {
					l.err = &WalkThroughError{Code: code.Internal, Content: fmt.Sprintf("failed to drop column: %v", err)}
					return
				}
				// drop primary key.
			case item.PRIMARY_SYMBOL() != nil && item.KEY_SYMBOL() != nil:
				if err := table.DropIndex(PrimaryKeyName); err != nil {
					l.err = &WalkThroughError{Code: code.IndexNotExists, Content: err.Error()}
					return
				}
				// drop key/index.
			case item.KeyOrIndex() != nil && item.IndexRef() != nil:
				_, _, indexName := mysqlparser.NormalizeIndexRef(item.IndexRef())
				if err := table.DropIndex(indexName); err != nil {
					l.err = &WalkThroughError{Code: code.IndexNotExists, Content: err.Error()}
					return
				}
			default:
				// Ignore other DROP variations
			}
		// modify column.
		case item.MODIFY_SYMBOL() != nil && item.ColumnInternalRef() != nil:
			columnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			if err := mysqlChangeColumn(table, columnName, columnName, item.FieldDefinition(), positionFromPlaceContext(item.Place())); err != nil {
				l.err = err
				return
			}
		// change column
		case item.CHANGE_SYMBOL() != nil && item.ColumnInternalRef() != nil && item.Identifier() != nil:
			oldColumnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			newColumnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			if err := mysqlChangeColumn(table, oldColumnName, newColumnName, item.FieldDefinition(), positionFromPlaceContext(item.Place())); err != nil {
				l.err = err
				return
			}
		// rename column
		case item.RENAME_SYMBOL() != nil && item.COLUMN_SYMBOL() != nil:
			oldColumnName := mysqlparser.NormalizeMySQLColumnInternalRef(item.ColumnInternalRef())
			newColumnName := mysqlparser.NormalizeMySQLIdentifier(item.Identifier())
			// Validate old column exists
			if table.GetColumn(oldColumnName) == nil {
				l.err = NewColumnNotExistsError(table.GetProto().Name, oldColumnName)
				return
			}
			// Validate new column doesn't already exist
			if table.GetColumn(newColumnName) != nil {
				l.err = &WalkThroughError{
					Code:    code.ColumnExists,
					Content: fmt.Sprintf("Column `%s` already exists in table `%s`", newColumnName, table.GetProto().Name),
				}
				return
			}
			if err := table.RenameColumn(oldColumnName, newColumnName); err != nil {
				l.err = &WalkThroughError{Code: code.Internal, Content: fmt.Sprintf("failed to rename column: %v", err)}
				return
			}
		case item.ALTER_SYMBOL() != nil:
			switch {
			// alter column.
			case item.ColumnInternalRef() != nil:
				if err := mysqlAlterColumn(table, item); err != nil {
					l.err = err
					return
				}
			// alter index visibility.
			case item.INDEX_SYMBOL() != nil && item.IndexRef() != nil && item.Visibility() != nil:
				_, _, indexName := mysqlparser.NormalizeIndexRef(item.IndexRef())
				if err := mysqlChangeIndexVisibility(table, indexName, item.Visibility()); err != nil {
					l.err = err
					return
				}
			default:
			}
		// rename table.
		case item.RENAME_SYMBOL() != nil && item.TableName() != nil:
			_, newTableName := mysqlparser.NormalizeMySQLTableName(item.TableName())
			schema := l.databaseMetadata.GetSchema("")
			if schema == nil {
				l.err = &WalkThroughError{
					Code:    code.SchemaNotExists,
					Content: "Schema does not exist",
				}
				return
			}
			if err := schema.RenameTable(table.GetProto().Name, newTableName); err != nil {
				l.err = &WalkThroughError{Code: code.TableNotExists, Content: err.Error()}
				return
			}
		// rename index.
		case item.RENAME_SYMBOL() != nil && item.KeyOrIndex() != nil && item.IndexRef() != nil && item.IndexName() != nil:
			_, _, oldIndexName := mysqlparser.NormalizeIndexRef(item.IndexRef())
			newIndexName := mysqlparser.NormalizeIndexName(item.IndexName())
			if err := table.RenameIndex(oldIndexName, newIndexName); err != nil {
				l.err = &WalkThroughError{Code: code.IndexNotExists, Content: err.Error()}
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
	table, err := mysqlFindTableState(l.databaseMetadata, databaseName, tableName)
	if err != nil {
		l.err = err
		return
	}

	if ctx.IndexRef() == nil {
		return
	}

	_, _, indexName := mysqlparser.NormalizeIndexRef(ctx.IndexRef())
	if err := table.DropIndex(indexName); err != nil {
		l.err = &WalkThroughError{Code: code.IndexNotExists, Content: err.Error()}
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
	table, err := mysqlFindTableState(l.databaseMetadata, databaseName, tableName)
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
	if err := mysqlValidateKeyListVariants(table, ctx.CreateIndexTarget().KeyListVariants(), false /* primary */, isSpatial); err != nil {
		l.err = err
		return
	}

	columnList := mysqlparser.NormalizeKeyListVariants(ctx.CreateIndexTarget().KeyListVariants())
	if err := mysqlCreateIndex(table, indexName, columnList, unique, tp, mysql.NewEmptyTableConstraintDefContext(), ctx); err != nil {
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
		if !isCurrentDatabase(l.databaseMetadata, databaseName) {
			l.err = NewAccessOtherDatabaseError(l.databaseMetadata.DatabaseName(), databaseName)
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
			l.databaseMetadata.GetProto().CharacterSet = charset
		case option.CreateDatabaseOption().DefaultCollation() != nil && option.CreateDatabaseOption().DefaultCollation().CollationName() != nil:
			collation := mysqlparser.NormalizeMySQLCollationName(option.CreateDatabaseOption().DefaultCollation().CollationName())
			l.databaseMetadata.GetProto().Collation = collation
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
	if !isCurrentDatabase(l.databaseMetadata, databaseName) {
		l.err = NewAccessOtherDatabaseError(l.databaseMetadata.DatabaseName(), databaseName)
		return
	}

	// DROP DATABASE not supported - would mark database as deleted
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
	l.err = NewAccessOtherDatabaseError(l.databaseMetadata.DatabaseName(), databaseName)
}

// EnterRenameTableStatement is called when production renameTableStatement is entered.
func (l *mysqlListener) EnterRenameTableStatement(ctx *mysql.RenameTableStatementContext) {
	if !mysqlparser.IsTopMySQLRule(&ctx.BaseParserRuleContext) {
		return
	}
	for _, pair := range ctx.AllRenamePair() {
		schema := l.databaseMetadata.GetSchema("")
		if schema == nil {
			l.err = &WalkThroughError{
				Code:    code.SchemaNotExists,
				Content: "Schema does not exist",
			}
			return
		}

		_, oldTableName := mysqlparser.NormalizeMySQLTableRef(pair.TableRef())
		_, newTableName := mysqlparser.NormalizeMySQLTableName(pair.TableName())

		if mysqlTheCurrentDatabase(l.databaseMetadata, pair) {
			if compareIdentifier(oldTableName, newTableName, !l.databaseMetadata.GetIsObjectCaseSensitive()) {
				return
			}
			if schema.GetTable(oldTableName) == nil {
				l.err = NewTableNotExistsError(oldTableName)
				return
			}
			if schema.GetTable(newTableName) != nil {
				l.err = NewTableExistsError(newTableName)
				return
			}
			if err := schema.RenameTable(oldTableName, newTableName); err != nil {
				l.err = &WalkThroughError{Code: code.Internal, Content: err.Error()}
				return
			}
		} else if mysqlMoveToOtherDatabase(l.databaseMetadata, pair) {
			if schema.GetTable(oldTableName) == nil {
				l.err = NewTableNotExistsError(oldTableName)
				return
			}
			if err := schema.DropTable(oldTableName); err != nil {
				l.err = &WalkThroughError{Code: code.TableNotExists, Content: err.Error()}
				return
			}
		} else {
			l.err = NewAccessOtherDatabaseError(l.databaseMetadata.DatabaseName(), mysqlTargetDatabase(l.databaseMetadata, pair))
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
	_, err := mysqlFindTableState(l.databaseMetadata, databaseName, tableName)
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

func mysqlTargetDatabase(d *model.DatabaseMetadata, renamePair mysql.IRenamePairContext) string {
	oldDatabaseName, _ := mysqlparser.NormalizeMySQLTableRef(renamePair.TableRef())
	if oldDatabaseName != "" && !isCurrentDatabase(d, oldDatabaseName) {
		return oldDatabaseName
	}
	newDatabaseName, _ := mysqlparser.NormalizeMySQLTableName(renamePair.TableName())
	return newDatabaseName
}

func mysqlMoveToOtherDatabase(d *model.DatabaseMetadata, renamePair mysql.IRenamePairContext) bool {
	oldDatabaseName, _ := mysqlparser.NormalizeMySQLTableRef(renamePair.TableRef())
	if oldDatabaseName != "" && !isCurrentDatabase(d, oldDatabaseName) {
		return false
	}
	newDatabaseName, _ := mysqlparser.NormalizeMySQLTableName(renamePair.TableName())
	return oldDatabaseName != newDatabaseName
}

func mysqlTheCurrentDatabase(d *model.DatabaseMetadata, renamePair mysql.IRenamePairContext) bool {
	newDatabaseName, _ := mysqlparser.NormalizeMySQLTableName(renamePair.TableName())
	if newDatabaseName != "" && !isCurrentDatabase(d, newDatabaseName) {
		return false
	}
	oldDatabaseName, _ := mysqlparser.NormalizeMySQLTableRef(renamePair.TableRef())
	if oldDatabaseName != "" && !isCurrentDatabase(d, oldDatabaseName) {
		return false
	}
	return true
}

func mysqlChangeIndexVisibility(table *model.TableMetadata, indexName string, visibility mysql.IVisibilityContext) *WalkThroughError {
	index := table.GetIndex(indexName)
	if index == nil {
		return NewIndexNotExistsError(table.GetProto().Name, indexName)
	}
	indexProto := index.GetProto()
	switch {
	case visibility.VISIBLE_SYMBOL() != nil:
		indexProto.Visible = true
	case visibility.INVISIBLE_SYMBOL() != nil:
		indexProto.Visible = false
	default:
		// No visibility specified
	}
	return nil
}

func mysqlAlterColumn(table *model.TableMetadata, itemDef mysql.IAlterListItemContext) *WalkThroughError {
	if itemDef.ColumnInternalRef() == nil {
		// should not reach here.
		return nil
	}
	columnName := mysqlparser.NormalizeMySQLColumnInternalRef(itemDef.ColumnInternalRef())
	col := table.GetColumn(columnName)
	if col == nil {
		return NewColumnNotExistsError(table.GetProto().Name, columnName)
	}

	switch {
	case itemDef.SET_SYMBOL() != nil:
		switch {
		// SET DEFAULT.
		case itemDef.DEFAULT_SYMBOL() != nil:
			if itemDef.SignedLiteral() != nil && itemDef.SignedLiteral().Literal() != nil && itemDef.SignedLiteral().Literal().NullLiteral() == nil {
				if col.Type != "" {
					switch strings.ToLower(col.Type) {
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

				col.Default = defaultValue
			} else {
				if !col.Nullable {
					return &WalkThroughError{
						Code: code.SetNullDefaultForNotNullColumn,
						// Content comes from MySQL Error content.
						Content: fmt.Sprintf("Invalid default value for column `%s`", columnName),
					}
				}

				col.Default = ""
			}
		// SET VISIBLE/INVISIBLE.
		default:
		}
	case itemDef.DROP_SYMBOL() != nil && itemDef.DEFAULT_SYMBOL() != nil:
		// DROP DEFAULT.
		col.Default = ""
	default:
		// Other ALTER operations
	}
	return nil
}

// mysqlChangeColumn changes column definition.
// It works as:
// 1. rename column if name changed
// 2. update column properties from fieldDef
// 3. handle position changes by reordering columns in the table
func mysqlChangeColumn(table *model.TableMetadata, oldColumnName string, newColumnName string, fieldDef mysql.IFieldDefinitionContext, position *mysqlColumnPosition) *WalkThroughError {
	column := table.GetColumn(oldColumnName)
	if column == nil {
		return NewColumnNotExistsError(table.GetProto().Name, oldColumnName)
	}

	// If renaming, validate and use RenameColumn
	if oldColumnName != newColumnName {
		// Validate new column doesn't already exist
		if table.GetColumn(newColumnName) != nil {
			return &WalkThroughError{
				Code:    code.ColumnExists,
				Content: fmt.Sprintf("Column `%s` already exists in table `%s`", newColumnName, table.GetProto().Name),
			}
		}
		if err := table.RenameColumn(oldColumnName, newColumnName); err != nil {
			return &WalkThroughError{Code: code.Internal, Content: fmt.Sprintf("failed to rename column: %v", err)}
		}
		// Get the renamed column
		column = table.GetColumn(newColumnName)
	}

	// Update column properties from fieldDef
	if fieldDef.DataType() != nil {
		column.Type = mysqlparser.NormalizeMySQLDataType(fieldDef.DataType(), true /* compact */)
		column.CharacterSet = mysqlparser.GetCharSetName(fieldDef.DataType())
		column.Collation = mysqlparser.GetCollationName(fieldDef)
	}

	// Handle position changes by reordering columns in the proto
	if position != nil && position.tp != ColumnPositionNone {
		tableProto := table.GetProto()
		columns := tableProto.Columns

		// Find the current position of the column
		var currentIdx int
		for i, col := range columns {
			if col == column {
				currentIdx = i
				break
			}
		}

		// Remove from current position
		columns = append(columns[:currentIdx], columns[currentIdx+1:]...)

		// Insert at new position
		var newIdx int
		switch position.tp {
		case ColumnPositionFirst:
			newIdx = 0
		case ColumnPositionAfter:
			// Find the position after the relative column
			for i, col := range columns {
				if col.Name == position.relativeColumn {
					newIdx = i + 1
					break
				}
			}
		default:
			// ColumnPositionNone should not reach here, but keep at end if it does
			newIdx = len(columns)
		}

		// Insert at newIdx
		if newIdx >= len(columns) {
			columns = append(columns, column)
		} else {
			columns = append(columns[:newIdx], append([]*storepb.ColumnMetadata{column}, columns[newIdx:]...)...)
		}

		// Update the proto's column list
		tableProto.Columns = columns

		// Update position field for all columns
		for i, col := range columns {
			col.Position = int32(i + 1)
		}
	}

	return nil
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

func mysqlCopyTable(d *model.DatabaseMetadata, databaseName, tableName, referTable string) *WalkThroughError {
	targetTable, err := mysqlFindTableState(d, databaseName, referTable)
	if err != nil {
		return err
	}

	schema := d.GetSchema("")
	if schema == nil {
		return &WalkThroughError{
			Code:    code.SchemaNotExists,
			Content: "Schema does not exist",
		}
	}

	// Create the new table
	newTable, createErr := schema.CreateTable(tableName)
	if createErr != nil {
		return &WalkThroughError{Code: code.TableExists, Content: createErr.Error()}
	}

	// Copy table properties
	targetProto := targetTable.GetProto()
	newTableProto := newTable.GetProto()
	newTableProto.Engine = targetProto.Engine
	newTableProto.Collation = targetProto.Collation
	newTableProto.Comment = targetProto.Comment

	// Copy columns
	for _, col := range targetTable.GetColumns() {
		colCopy, ok := proto.Clone(col).(*storepb.ColumnMetadata)
		if !ok {
			return &WalkThroughError{Code: code.Internal, Content: "failed to clone column metadata"}
		}
		if err := newTable.CreateColumn(colCopy); err != nil {
			return &WalkThroughError{Code: code.ColumnExists, Content: err.Error()}
		}
	}

	// Copy indexes
	for _, idx := range targetProto.Indexes {
		idxCopy, ok := proto.Clone(idx).(*storepb.IndexMetadata)
		if !ok {
			return &WalkThroughError{Code: code.Internal, Content: "failed to clone index metadata"}
		}
		if err := newTable.CreateIndex(idxCopy); err != nil {
			return &WalkThroughError{Code: code.IndexExists, Content: err.Error()}
		}
	}

	return nil
}

func mysqlFindTableState(d *model.DatabaseMetadata, databaseName, tableName string) (*model.TableMetadata, *WalkThroughError) {
	if databaseName != "" && !isCurrentDatabase(d, databaseName) {
		return nil, NewAccessOtherDatabaseError(d.DatabaseName(), databaseName)
	}

	schema := d.GetSchema("")
	if schema == nil {
		return nil, &WalkThroughError{
			Code:    code.SchemaNotExists,
			Content: "Schema does not exist",
		}
	}

	table := schema.GetTable(tableName)
	if table == nil {
		return nil, NewTableNotExistsError(tableName)
	}

	return table, nil
}

func mysqlCreateConstraint(table *model.TableMetadata, constraintDef mysql.ITableConstraintDefContext) *WalkThroughError {
	if constraintDef.GetType_() != nil {
		switch constraintDef.GetType_().GetTokenType() {
		// PRIMARY KEY.
		case mysql.MySQLParserPRIMARY_SYMBOL:
			if constraintDef.KeyListVariants() == nil {
				// never reach here.
				return nil
			}
			if err := mysqlValidateKeyListVariants(table, constraintDef.KeyListVariants(), true /* primary */, false /* isSpatial*/); err != nil {
				return err
			}
			keyList := mysqlparser.NormalizeKeyListVariants(constraintDef.KeyListVariants())
			if err := mysqlCreatePrimaryKey(table, keyList, mysqlGetIndexType(constraintDef)); err != nil {
				return err
			}
		// normal KEY/INDEX.
		case mysql.MySQLParserKEY_SYMBOL, mysql.MySQLParserINDEX_SYMBOL:
			if constraintDef.KeyListVariants() == nil {
				// never reach here.
				return nil
			}
			if err := mysqlValidateKeyListVariants(table, constraintDef.KeyListVariants(), false /* primary */, false /* isSpatial */); err != nil {
				return err
			}

			indexName := ""
			if constraintDef.IndexNameAndType() != nil && constraintDef.IndexNameAndType().IndexName() != nil {
				indexName = mysqlparser.NormalizeIndexName(constraintDef.IndexNameAndType().IndexName())
			}
			keyList := mysqlparser.NormalizeKeyListVariants(constraintDef.KeyListVariants())
			if err := mysqlCreateIndex(table, indexName, keyList, false /* unique */, mysqlGetIndexType(constraintDef), constraintDef, mysql.NewEmptyCreateIndexContext()); err != nil {
				return err
			}
		// UNIQUE KEY.
		case mysql.MySQLParserUNIQUE_SYMBOL:
			if constraintDef.KeyListVariants() == nil {
				// never reach here.
				return nil
			}
			if err := mysqlValidateKeyListVariants(table, constraintDef.KeyListVariants(), false /* primary */, false /* isSpatial*/); err != nil {
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
			if err := mysqlCreateIndex(table, indexName, keyList, true /* unique */, mysqlGetIndexType(constraintDef), constraintDef, mysql.NewEmptyCreateIndexContext()); err != nil {
				return err
			}
		// FULLTEXT KEY.
		case mysql.MySQLParserFULLTEXT_SYMBOL:
			if constraintDef.KeyListVariants() == nil {
				// never reach here.
				return nil
			}
			if err := mysqlValidateKeyListVariants(table, constraintDef.KeyListVariants(), false /* primary */, false /* isSpatial*/); err != nil {
				return err
			}
			indexName := ""
			if constraintDef.IndexName() != nil {
				indexName = mysqlparser.NormalizeIndexName(constraintDef.IndexName())
			}
			keyList := mysqlparser.NormalizeKeyListVariants(constraintDef.KeyListVariants())
			if err := mysqlCreateIndex(table, indexName, keyList, false /* unique */, mysqlGetIndexType(constraintDef), constraintDef, mysql.NewEmptyCreateIndexContext()); err != nil {
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
func mysqlValidateKeyListVariants(table *model.TableMetadata, keyList mysql.IKeyListVariantsContext, primary bool, isSpatial bool) *WalkThroughError {
	if keyList.KeyList() != nil {
		columns := mysqlparser.NormalizeKeyList(keyList.KeyList())
		if err := mysqlValidateColumnList(table, columns, primary, isSpatial); err != nil {
			return err
		}
	}
	if keyList.KeyListWithExpression() != nil {
		expressions := mysqlparser.NormalizeKeyListWithExpression(keyList.KeyListWithExpression())
		if err := mysqlValidateExpressionList(table, expressions, primary, isSpatial); err != nil {
			return err
		}
	}
	return nil
}

func mysqlValidateColumnList(table *model.TableMetadata, columnList []string, primary bool, isSpatial bool) *WalkThroughError {
	for _, columnName := range columnList {
		column := table.GetColumn(columnName)
		if column == nil {
			return NewColumnNotExistsError(table.GetProto().Name, columnName)
		}
		if primary {
			column.Nullable = false
		}
		if isSpatial && column.Nullable {
			return &WalkThroughError{
				Code: code.SpatialIndexKeyNullable,
				// The error content comes from MySQL.
				Content: fmt.Sprintf("All parts of a SPATIAL index must be NOT NULL, but `%s` is nullable", column.Name),
			}
		}
	}
	return nil
}

// mysqlValidateExpressionList validates the expression list.
// TODO: update expression validation.
func mysqlValidateExpressionList(table *model.TableMetadata, expressionList []string, primary bool, isSpatial bool) *WalkThroughError {
	for _, expression := range expressionList {
		column := table.GetColumn(expression)
		// If expression is not a column, we do not need to validate it.
		if column == nil {
			continue
		}

		if primary {
			column.Nullable = false
		}
		if isSpatial && column.Nullable {
			return &WalkThroughError{
				Code: code.SpatialIndexKeyNullable,
				// The error content comes from MySQL.
				Content: fmt.Sprintf("All parts of a SPATIAL index must be NOT NULL, but `%s` is nullable", column.Name),
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

func mysqlCreateColumn(table *model.TableMetadata, columnName string, fieldDef mysql.IFieldDefinitionContext, position *mysqlColumnPosition) *WalkThroughError {
	if table.GetColumn(columnName) != nil {
		return &WalkThroughError{
			Code:    code.ColumnExists,
			Content: fmt.Sprintf("Column `%s` already exists in table `%s`", columnName, table.GetProto().Name),
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

	col := &storepb.ColumnMetadata{
		Name:         columnName,
		Position:     int32(len(table.GetColumns()) + 1),
		Default:      "",
		Nullable:     true,
		Type:         columnType,
		CharacterSet: characterSet,
		Collation:    collation,
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
			col.Nullable = false
		}
		if attribute.GetValue() != nil {
			switch attribute.GetValue().GetTokenType() {
			// default value.
			case mysql.MySQLParserDEFAULT_SYMBOL:
				if err := mysqlCheckDefault(table, columnName, fieldDef); err != nil {
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
				col.Default = defaultValue
			// comment.
			case mysql.MySQLParserCOMMENT_SYMBOL:
				if attribute.TextLiteral() == nil {
					continue
				}
				comment := mysqlparser.NormalizeMySQLTextLiteral(attribute.TextLiteral())
				col.Comment = comment
			// on update now().
			case mysql.MySQLParserON_SYMBOL:
				if attribute.UPDATE_SYMBOL() == nil || attribute.NOW_SYMBOL() == nil {
					continue
				}
				if !mysqlparser.IsTimeType(fieldDef.DataType()) {
					return &WalkThroughError{
						Code:    code.OnUpdateColumnNotDatetimeOrTimestamp,
						Content: fmt.Sprintf("Column `%s` use ON UPDATE but is not DATETIME or TIMESTAMP", col.Name),
					}
				}
			// primary key.
			case mysql.MySQLParserKEY_SYMBOL:
				// the key attribute for in a column meaning primary key.
				col.Nullable = false
				// we need to check the key type which generated by tidb parser.
				if err := mysqlCreatePrimaryKey(table, []string{strings.ToLower(col.Name)}, "BTREE"); err != nil {
					return err
				}
			// unique key.
			case mysql.MySQLParserUNIQUE_SYMBOL:
				// unique index.
				if err := mysqlCreateIndex(table, "", []string{strings.ToLower(col.Name)}, true /* unique */, "BTREE", mysql.NewEmptyTableConstraintDefContext(), mysql.NewEmptyCreateIndexContext()); err != nil {
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

	if !col.Nullable && setNullDefault {
		return &WalkThroughError{
			Code: code.SetNullDefaultForNotNullColumn,
			// Content comes from MySQL Error content.
			Content: fmt.Sprintf("Invalid default value for column `%s`", col.Name),
		}
	}

	if err := table.CreateColumn(col); err != nil {
		return &WalkThroughError{Code: code.ColumnExists, Content: err.Error()}
	}

	// Handle position by reordering columns in the proto
	if position != nil && position.tp != ColumnPositionNone {
		tableProto := table.GetProto()
		columns := tableProto.Columns

		// The new column was just appended, so it's at the end
		lastIdx := len(columns) - 1
		newColumn := columns[lastIdx]

		// Remove from the end
		columns = columns[:lastIdx]

		// Insert at the desired position
		var newIdx int
		switch position.tp {
		case ColumnPositionFirst:
			newIdx = 0
		case ColumnPositionAfter:
			// Find the position after the relative column
			for i, c := range columns {
				if c.Name == position.relativeColumn {
					newIdx = i + 1
					break
				}
			}
		default:
			// ColumnPositionNone should not reach here, but keep at end if it does
			newIdx = len(columns)
		}

		// Insert at newIdx
		if newIdx >= len(columns) {
			columns = append(columns, newColumn)
		} else {
			columns = append(columns[:newIdx], append([]*storepb.ColumnMetadata{newColumn}, columns[newIdx:]...)...)
		}

		// Update the proto's column list
		tableProto.Columns = columns

		// Update position field for all columns
		for i, c := range columns {
			c.Position = int32(i + 1)
		}
	}

	return nil
}

func mysqlCreateIndex(table *model.TableMetadata, name string, keyList []string, unique bool, tp string, tableConstraint mysql.ITableConstraintDefContext, createIndexDef mysql.ICreateIndexContext) *WalkThroughError {
	if len(keyList) == 0 {
		return &WalkThroughError{
			Code:    code.IndexEmptyKeys,
			Content: fmt.Sprintf("Index `%s` in table `%s` has empty key", name, table.GetProto().Name),
		}
	}
	// construct a index name if name is empty.
	if name != "" {
		if table.GetIndex(name) != nil {
			return NewIndexExistsError(table.GetProto().Name, name)
		}
	} else {
		suffix := 1
		for {
			name = keyList[0]
			if suffix > 1 {
				name = fmt.Sprintf("%s_%d", keyList[0], suffix)
			}
			if table.GetIndex(name) == nil {
				break
			}
			suffix++
		}
	}

	visible := true

	// Check visibility from table constraint
	for _, attribute := range tableConstraint.AllIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			visible = false
		}
	}

	// Check visibility from create index statement
	for _, attribute := range createIndexDef.AllIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			visible = false
		}
	}

	// Check FULLTEXT visibility
	for _, attribute := range tableConstraint.AllFulltextIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			visible = false
		}
	}

	for _, attribute := range createIndexDef.AllFulltextIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			visible = false
		}
	}

	// Check SPATIAL visibility
	for _, attribute := range tableConstraint.AllSpatialIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			visible = false
		}
	}

	for _, attribute := range createIndexDef.AllSpatialIndexOption() {
		if attribute == nil || attribute.CommonIndexOption() == nil {
			continue
		}
		if attribute.CommonIndexOption().Visibility() != nil && attribute.CommonIndexOption().Visibility().INVISIBLE_SYMBOL() != nil {
			visible = false
		}
	}

	index := &storepb.IndexMetadata{
		Name:        name,
		Expressions: keyList,
		Type:        tp,
		Unique:      unique,
		Primary:     false,
		Visible:     visible,
	}

	if err := table.CreateIndex(index); err != nil {
		return &WalkThroughError{Code: code.IndexExists, Content: err.Error()}
	}
	return nil
}

func mysqlCreatePrimaryKey(table *model.TableMetadata, keys []string, tp string) *WalkThroughError {
	if table.GetPrimaryKey() != nil {
		return &WalkThroughError{
			Code:    code.PrimaryKeyExists,
			Content: fmt.Sprintf("Primary key exists in table `%s`", table.GetProto().Name),
		}
	}

	pk := &storepb.IndexMetadata{
		Name:        PrimaryKeyName,
		Expressions: keys,
		Type:        tp,
		Unique:      true,
		Primary:     true,
		Visible:     true,
	}
	if err := table.CreateIndex(pk); err != nil {
		return &WalkThroughError{Code: code.PrimaryKeyExists, Content: err.Error()}
	}
	return nil
}

func mysqlCheckDefault(_ *model.TableMetadata, columnName string, fieldDefinition mysql.IFieldDefinitionContext) *WalkThroughError {
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
