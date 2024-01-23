package mysql

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterParseToMetadatas(storepb.Engine_MYSQL, ParseToMetadata)
}

// ParseToMetadata converts a schema string to database metadata.
func ParseToMetadata(schema string) (*storepb.DatabaseSchemaMetadata, error) {
	list, err := mysqlparser.ParseMySQL(schema)
	if err != nil {
		return nil, err
	}

	listener := &mysqlTransformer{
		state: newDatabaseState(),
	}
	listener.state.schemas[""] = newSchemaState()

	for _, stmt := range list {
		antlr.ParseTreeWalkerDefault.Walk(listener, stmt.Tree)
	}

	return listener.state.convertToDatabaseMetadata(), listener.err
}

type mysqlTransformer struct {
	*mysql.BaseMySQLParserListener

	state        *databaseState
	currentTable string
	err          error
}

// EnterCreateTable is called when production createTable is entered.
func (t *mysqlTransformer) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if t.err != nil {
		return
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if databaseName != "" {
		if t.state.name == "" {
			t.state.name = databaseName
		} else if t.state.name != databaseName {
			t.err = errors.New("multiple database names found: " + t.state.name + ", " + databaseName)
			return
		}
	}

	schema := t.state.schemas[""]
	if _, ok := schema.tables[tableName]; ok {
		t.err = errors.New("multiple table names found: " + tableName)
		return
	}

	schema.tables[tableName] = newTableState(len(schema.tables), tableName)
	t.currentTable = tableName
}

// ExitCreateTable is called when production createTable is exited.
func (t *mysqlTransformer) ExitCreateTable(_ *mysql.CreateTableContext) {
	t.currentTable = ""
}

// EnterCreateTableOption is called when production createTableOption is entered.
func (t *mysqlTransformer) EnterCreateTableOption(ctx *mysql.CreateTableOptionContext) {
	if t.err != nil || t.currentTable == "" {
		return
	}

	if ctx.ENGINE_SYMBOL() != nil {
		engineString := ctx.EngineRef().TextOrIdentifier().GetParser().GetTokenStream().GetTextFromRuleContext(ctx.EngineRef().TextOrIdentifier())
		schema := t.state.schemas[""]
		table, ok := schema.tables[t.currentTable]
		if !ok {
			// This should never happen.
			return
		}
		table.engine = engineString
	}

	if defaultCollation := ctx.DefaultCollation(); defaultCollation != nil {
		collationString := defaultCollation.CollationName().GetParser().GetTokenStream().GetTextFromRuleContext(defaultCollation.CollationName())
		schema := t.state.schemas[""]
		table, ok := schema.tables[t.currentTable]
		if !ok {
			// This should never happen.
			return
		}
		table.collation = collationString
	}

	if ctx.COMMENT_SYMBOL() != nil {
		commentString := ctx.TextStringLiteral().GetText()
		if len(commentString) > 2 {
			quotes := commentString[0]
			escape := fmt.Sprintf("%c%c", quotes, quotes)
			commentString = strings.ReplaceAll(commentString[1:len(commentString)-1], escape, string(quotes))
		}

		schema := t.state.schemas[""]
		table, ok := schema.tables[t.currentTable]
		if !ok {
			// This should never happen.
			return
		}
		table.comment = commentString
	}
}

// EnterColumnDefinition is called when production columnDefinition is entered.
func (t *mysqlTransformer) EnterColumnDefinition(ctx *mysql.ColumnDefinitionContext) {
	if t.err != nil || t.currentTable == "" {
		return
	}

	_, _, columnName := mysqlparser.NormalizeMySQLColumnName(ctx.ColumnName())
	dataType := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.FieldDefinition().DataType())
	table := t.state.schemas[""].tables[t.currentTable]
	if _, ok := table.columns[columnName]; ok {
		t.err = errors.New("multiple column names found: " + columnName + " in table " + t.currentTable)
		return
	}
	columnState := &columnState{
		id:           len(table.columns),
		name:         columnName,
		tp:           dataType,
		hasDefault:   false,
		defaultValue: nil,
		comment:      "",
		nullable:     true,
	}

	for _, attribute := range ctx.FieldDefinition().AllColumnAttribute() {
		switch {
		case attribute.NullLiteral() != nil && attribute.NOT_SYMBOL() != nil:
			columnState.nullable = false
		case attribute.DEFAULT_SYMBOL() != nil && attribute.SERIAL_SYMBOL() == nil:
			defaultValueStart := nextDefaultChannelTokenIndex(ctx.GetParser().GetTokenStream(), attribute.DEFAULT_SYMBOL().GetSymbol().GetTokenIndex())
			defaultValue := attribute.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: defaultValueStart,
				Stop:  attribute.GetStop().GetTokenIndex(),
			})
			columnState.hasDefault = true
			switch {
			case strings.EqualFold(defaultValue, "NULL"):
				columnState.defaultValue = &defaultValueNull{}
			case strings.HasPrefix(defaultValue, "'") && strings.HasSuffix(defaultValue, "'"):
				columnState.defaultValue = &defaultValueString{value: strings.ReplaceAll(defaultValue[1:len(defaultValue)-1], "''", "'")}
			case strings.HasPrefix(defaultValue, "\"") && strings.HasSuffix(defaultValue, "\""):
				columnState.defaultValue = &defaultValueString{value: strings.ReplaceAll(defaultValue[1:len(defaultValue)-1], "\"\"", "\"")}
			default:
				columnState.defaultValue = &defaultValueExpression{value: defaultValue}
			}
		case attribute.COMMENT_SYMBOL() != nil:
			commentStart := nextDefaultChannelTokenIndex(ctx.GetParser().GetTokenStream(), attribute.COMMENT_SYMBOL().GetSymbol().GetTokenIndex())
			comment := attribute.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: commentStart,
				Stop:  attribute.GetStop().GetTokenIndex(),
			})
			if comment != `''` && len(comment) > 2 {
				columnState.comment = comment[1 : len(comment)-1]
			}
		// todo(zp): refactor column attribute.
		case attribute.AUTO_INCREMENT_SYMBOL() != nil:
			defaultValue := autoIncrementSymbol
			columnState.hasDefault = true
			columnState.defaultValue = &defaultValueExpression{value: defaultValue}
		}
	}

	table.columns[columnName] = columnState
}

// EnterTableConstraintDef is called when production tableConstraintDef is entered.
func (t *mysqlTransformer) EnterTableConstraintDef(ctx *mysql.TableConstraintDefContext) {
	if t.err != nil || t.currentTable == "" {
		return
	}

	if ctx.GetType_() != nil {
		symbol := strings.ToUpper(ctx.GetType_().GetText())
		switch symbol {
		case "PRIMARY":
			keys, keyLengths := extractKeyListVariants(ctx.KeyListVariants())
			table := t.state.schemas[""].tables[t.currentTable]
			table.indexes["PRIMARY"] = &indexState{
				id:      len(table.indexes),
				name:    "PRIMARY",
				keys:    keys,
				lengths: keyLengths,
				primary: true,
				unique:  true,
			}
		case "FOREIGN":
			var name string
			if ctx.ConstraintName() != nil && ctx.ConstraintName().Identifier() != nil {
				name = mysqlparser.NormalizeMySQLIdentifier(ctx.ConstraintName().Identifier())
			} else if ctx.IndexName() != nil {
				name = mysqlparser.NormalizeMySQLIdentifier(ctx.IndexName().Identifier())
			}
			keys, _ := extractKeyList(ctx.KeyList())
			table := t.state.schemas[""].tables[t.currentTable]
			if table.foreignKeys[name] != nil {
				t.err = errors.New("multiple foreign keys found: " + name)
				return
			}
			referencedTable, referencedColumns := extractReference(ctx.References())
			fk := &foreignKeyState{
				id:                len(table.foreignKeys),
				name:              name,
				columns:           keys,
				referencedTable:   referencedTable,
				referencedColumns: referencedColumns,
			}
			table.foreignKeys[name] = fk
		case "FULLTEXT":
			var name string
			if ctx.IndexName() != nil {
				name = mysqlparser.NormalizeMySQLIdentifier(ctx.IndexName().Identifier())
			}
			keys, keyLengths := extractKeyListVariants(ctx.KeyListVariants())
			table := t.state.schemas[""].tables[t.currentTable]
			if table.indexes[name] != nil {
				t.err = errors.New("multiple indexes found: " + name)
				return
			}
			idx := &indexState{
				id:      len(table.indexes),
				name:    name,
				keys:    keys,
				lengths: keyLengths,
				primary: false,
				unique:  false,
				tp:      symbol,
			}
			table.indexes[name] = idx
		case "SPATIAL":
			var name string
			if ctx.IndexName() != nil {
				name = mysqlparser.NormalizeMySQLIdentifier(ctx.IndexName().Identifier())
			}
			keys, keyLengths := extractKeyListVariants(ctx.KeyListVariants())
			table := t.state.schemas[""].tables[t.currentTable]
			if table.indexes[name] != nil {
				t.err = errors.New("multiple indexes found: " + name)
				return
			}
			idx := &indexState{
				id:      len(table.indexes),
				name:    name,
				keys:    keys,
				lengths: keyLengths,
				primary: false,
				unique:  false,
				tp:      symbol,
			}
			table.indexes[name] = idx
		case "KEY", "INDEX", "UNIQUE":
			var name string
			if v := ctx.IndexNameAndType(); v != nil {
				name = mysqlparser.NormalizeMySQLIdentifier(v.IndexName().Identifier())
			} else {
				t.err = errors.New("index name not found")
			}
			keys, keyLengths := extractKeyListVariants(ctx.KeyListVariants())
			table := t.state.schemas[""].tables[t.currentTable]
			if table.indexes[name] != nil {
				t.err = errors.New("multiple indexes found: " + name)
				return
			}
			tp := "BTREE"
			if v := ctx.IndexNameAndType(); v != nil && v.IndexType() != nil {
				tp = strings.ToUpper(v.IndexType().GetText())
			}
			idx := &indexState{
				id:      len(table.indexes),
				name:    name,
				keys:    keys,
				lengths: keyLengths,
				primary: false,
				unique:  symbol == "UNIQUE",
				tp:      tp,
			}
			table.indexes[name] = idx
		}
	}
}
