package mysql

import (
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterGetDesignSchema(storepb.Engine_MYSQL, GetDesignSchema)
}

func GetDesignSchema(baselineSchema string, to *storepb.DatabaseSchemaMetadata) (string, error) {
	toState := convertToDatabaseState(to)
	list, err := mysqlparser.ParseMySQL(baselineSchema, tokenizer.KeepEmptyBlocks())
	if err != nil {
		return "", err
	}

	listener := &mysqlDesignSchemaGenerator{
		lastTokenIndex: 0,
		to:             toState,
		desired:        to,
	}

	for _, stmt := range list {
		listener.lastTokenIndex = 0
		antlr.ParseTreeWalkerDefault.Walk(listener, stmt.Tree)
		if listener.err != nil {
			break
		}

		if _, err := listener.result.WriteString(
			stmt.Tokens.GetTextFromInterval(antlr.Interval{
				Start: listener.lastTokenIndex,
				Stop:  stmt.Tokens.Size() - 1,
			}),
		); err != nil {
			return "", err
		}
	}
	if listener.err != nil {
		return "", listener.err
	}

	// Expectedly, EnterSetStatement is called when production setStatement is entered.
	// And we would like to generate the remaining tables before the set statement mentioned above.
	// But users can remove the set statement during the rebase process.
	if err := writeRemainingTables(&listener.result, to, toState); err != nil {
		return "", err
	}

	s := listener.result.String()
	if !strings.HasSuffix(s, "\n") {
		// The last statement of the result is SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;
		// We should append a 0xa to the end of the result to avoid the extra newline diff.
		// TODO(rebelice/zp): find a more elegant way to do this.
		if err := listener.result.WriteByte('\n'); err != nil {
			return "", err
		}
	}

	s = listener.result.String()
	result, err := mysqlparser.RestoreDelimiter(s)
	if err != nil {
		slog.Warn("Failed to restore delimiter", slog.String("result", s), slog.String("error", err.Error()))
		return s, nil
	}
	return result, nil
}

type mysqlDesignSchemaGenerator struct {
	*mysql.BaseMySQLParserListener

	to                  *databaseState
	result              strings.Builder
	currentTable        *tableState
	firstElementInTable bool
	columnDefine        strings.Builder
	tableConstraints    strings.Builder
	tableOptions        strings.Builder
	err                 error

	lastTokenIndex        int
	tableOptionTokenIndex int

	desired *storepb.DatabaseSchemaMetadata
}

// EnterCreateTable is called when production createTable is entered.
func (g *mysqlDesignSchemaGenerator) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if g.err != nil {
		return
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if databaseName != "" && g.to.name != "" && databaseName != g.to.name {
		g.err = errors.New("multiple database names found: " + g.to.name + ", " + databaseName)
		return
	}

	schema, ok := g.to.schemas[""]
	if !ok || schema == nil {
		return
	}

	table, ok := schema.tables[tableName]
	if !ok {
		g.lastTokenIndex = ctx.GetParser().GetTokenStream().Size() - 1
		return
	}

	if _, err := g.result.WriteString(
		ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: g.lastTokenIndex,
			Stop:  ctx.GetStart().GetTokenIndex() - 1,
		}),
	); err != nil {
		g.err = err
		return
	}

	g.currentTable = table
	g.firstElementInTable = true
	g.columnDefine.Reset()
	g.tableConstraints.Reset()
	g.tableOptions.Reset()

	delete(schema.tables, tableName)
	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: ctx.GetStart().GetTokenIndex(),
		Stop:  ctx.TableElementList().GetStart().GetTokenIndex() - 1,
	})); err != nil {
		g.err = err
		return
	}
}

// ExitCreateTable is called when production createTable is exited.
func (g *mysqlDesignSchemaGenerator) ExitCreateTable(ctx *mysql.CreateTableContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	var columnList []*columnState
	for _, column := range g.currentTable.columns {
		columnList = append(columnList, column)
	}
	sort.Slice(columnList, func(i, j int) bool {
		return columnList[i].id < columnList[j].id
	})
	for _, column := range columnList {
		if g.firstElementInTable {
			g.firstElementInTable = false
		} else {
			if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
				g.err = err
				return
			}
		}
		if err := column.toString(&g.columnDefine); err != nil {
			g.err = err
			return
		}
	}

	if g.currentTable.indexes["PRIMARY"] != nil {
		if g.firstElementInTable {
			g.firstElementInTable = false
		} else {
			if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
				g.err = err
				return
			}
		}
		if err := g.currentTable.indexes["PRIMARY"].toString(&g.tableConstraints); err != nil {
			return
		}
		delete(g.currentTable.indexes, "PRIMARY")
	}

	var indexes []*indexState
	for _, index := range g.currentTable.indexes {
		indexes = append(indexes, index)
	}
	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i].id < indexes[j].id
	})
	for _, index := range indexes {
		if g.firstElementInTable {
			g.firstElementInTable = false
		} else {
			if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
				g.err = err
				return
			}
		}
		if err := index.toString(&g.tableConstraints); err != nil {
			g.err = err
			return
		}
	}

	var fks []*foreignKeyState
	for _, fk := range g.currentTable.foreignKeys {
		fks = append(fks, fk)
	}
	sort.Slice(fks, func(i, j int) bool {
		return fks[i].id < fks[j].id
	})
	for _, fk := range fks {
		if g.firstElementInTable {
			g.firstElementInTable = false
		} else {
			if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
				g.err = err
				return
			}
		}
		if err := fk.toString(&g.tableConstraints); err != nil {
			g.err = err
			return
		}
	}

	if _, err := g.result.WriteString(g.columnDefine.String()); err != nil {
		g.err = err
		return
	}

	if _, err := g.result.WriteString(g.tableConstraints.String()); err != nil {
		g.err = err
		return
	}

	if ctx.CreateTableOptions() != nil {
		if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.TableElementList().GetStop().GetTokenIndex() + 1,
			Stop:  ctx.CreateTableOptions().GetStart().GetTokenIndex() - 1,
		})); err != nil {
			g.err = err
			return
		}

		if _, err := g.result.WriteString(g.tableOptions.String()); err != nil {
			g.err = err
			return
		}

		if g.currentTable.comment != "" {
			if _, err := g.result.WriteString(fmt.Sprintf(" COMMENT '%s'", strings.ReplaceAll(g.currentTable.comment, "'", "''"))); err != nil {
				g.err = err
				return
			}
		}
		g.lastTokenIndex = ctx.CreateTableOptions().GetStop().GetTokenIndex() + 1
	} else {
		if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.TableElementList().GetStop().GetTokenIndex() + 1,
			Stop:  ctx.CLOSE_PAR_SYMBOL().GetSymbol().GetTokenIndex(),
		})); err != nil {
			g.err = err
			return
		}
		if g.currentTable.comment != "" {
			if _, err := g.result.WriteString(fmt.Sprintf(" COMMENT '%s' ", strings.ReplaceAll(g.currentTable.comment, "'", "''"))); err != nil {
				g.err = err
				return
			}
		}
		g.lastTokenIndex = ctx.CLOSE_PAR_SYMBOL().GetSymbol().GetTokenIndex() + 1
	}

	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: g.lastTokenIndex,
		// Write all tokens until the end of the statement.
		// Because we listen one statement at a time, we can safely use the last token index.
		Stop: ctx.GetParser().GetTokenStream().Size() - 1,
	})); err != nil {
		g.err = err
		return
	}

	g.currentTable = nil
	g.firstElementInTable = false
	g.lastTokenIndex = ctx.GetParser().GetTokenStream().Size() - 1
}

func (g *mysqlDesignSchemaGenerator) EnterCreateTableOptions(ctx *mysql.CreateTableOptionsContext) {
	g.tableOptionTokenIndex = ctx.GetStart().GetTokenIndex()
}

func (g *mysqlDesignSchemaGenerator) ExitCreateTableOptions(ctx *mysql.CreateTableOptionsContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	if _, err := g.tableOptions.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: g.tableOptionTokenIndex,
		Stop:  ctx.GetStop().GetTokenIndex(),
	})); err != nil {
		g.err = err
		return
	}

	g.tableOptionTokenIndex = ctx.GetStop().GetTokenIndex() + 1
}

func (g *mysqlDesignSchemaGenerator) EnterCreateTableOption(ctx *mysql.CreateTableOptionContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	if ctx.COMMENT_SYMBOL() != nil {
		commentString := ctx.TextStringLiteral().GetText()
		if len(commentString) > 2 {
			quotes := commentString[0]
			escape := fmt.Sprintf("%c%c", quotes, quotes)
			commentString = strings.ReplaceAll(commentString[1:len(commentString)-1], escape, string(quotes))
		}
		if g.currentTable.comment == commentString {
			if _, err := g.tableOptions.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(
				antlr.Interval{
					Start: g.tableOptionTokenIndex,
					Stop:  ctx.GetStop().GetTokenIndex(),
				},
			)); err != nil {
				g.err = err
				return
			}
			g.tableOptionTokenIndex = ctx.GetStop().GetTokenIndex() + 1
		} else {
			if _, err := g.tableOptions.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(
				antlr.Interval{
					Start: g.tableOptionTokenIndex,
					Stop:  ctx.GetStart().GetTokenIndex() - 1,
				},
			)); err != nil {
				g.err = err
				return
			}
			g.tableOptionTokenIndex = ctx.GetStop().GetTokenIndex() + 1

			if len(g.currentTable.comment) == 0 {
				return
			}

			if _, err := g.tableOptions.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(
				antlr.Interval{
					Start: ctx.GetStart().GetTokenIndex(),
					Stop:  ctx.TextStringLiteral().GetStart().GetTokenIndex() - 1,
				},
			)); err != nil {
				g.err = err
				return
			}

			if _, err := g.tableOptions.WriteString(fmt.Sprintf("'%s'", strings.ReplaceAll(g.currentTable.comment, "'", "''"))); err != nil {
				g.err = err
				return
			}
		}
		// Reset the comment.
		g.currentTable.comment = ""
	}
}

// EnterTableConstraintDef is called when production tableConstraintDef is entered.
func (g *mysqlDesignSchemaGenerator) EnterTableConstraintDef(ctx *mysql.TableConstraintDefContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	if ctx.GetType_() == nil {
		if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)); err != nil {
			g.err = err
			return
		}
		return
	}

	upperTp := strings.ToUpper(ctx.GetType_().GetText())
	switch upperTp {
	case "PRIMARY":
		if g.currentTable.indexes["PRIMARY"] != nil {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
					g.err = err
					return
				}
			}

			keys, keyLengths := extractKeyListVariants(ctx.KeyListVariants())
			if equalKeys(keys, g.currentTable.indexes["PRIMARY"].keys) && equalKeyLengths(keyLengths, g.currentTable.indexes["PRIMARY"].lengths) {
				if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)); err != nil {
					g.err = err
					return
				}
			} else {
				if err := g.currentTable.indexes["PRIMARY"].toString(&g.tableConstraints); err != nil {
					g.err = err
					return
				}
			}
			delete(g.currentTable.indexes, "PRIMARY")
		}
	case "FOREIGN":
		var name string
		if ctx.ConstraintName() != nil && ctx.ConstraintName().Identifier() != nil {
			name = mysqlparser.NormalizeMySQLIdentifier(ctx.ConstraintName().Identifier())
		} else if ctx.IndexName() != nil {
			name = mysqlparser.NormalizeMySQLIdentifier(ctx.IndexName().Identifier())
		}
		if g.currentTable.foreignKeys[name] != nil {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
					g.err = err
					return
				}
			}

			fk := g.currentTable.foreignKeys[name]

			columns, _ := extractKeyList(ctx.KeyList())
			referencedTable, referencedColumns := extractReference(ctx.References())
			equal := equalKeys(columns, fk.columns) && referencedTable == fk.referencedTable && equalKeys(referencedColumns, fk.referencedColumns)
			if equal {
				if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)); err != nil {
					g.err = err
					return
				}
			} else {
				if err := fk.toString(&g.tableConstraints); err != nil {
					g.err = err
					return
				}
			}
			delete(g.currentTable.foreignKeys, name)
		}
	case "KEY", "INDEX":
		var name string
		if ctx.IndexNameAndType() != nil {
			if ctx.IndexNameAndType().IndexName() != nil {
				name = mysqlparser.NormalizeMySQLIdentifier(ctx.IndexNameAndType().IndexName().Identifier())
			}
		}
		if g.currentTable.indexes[name] != nil {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
					g.err = err
					return
				}
			}

			idx := g.currentTable.indexes[name]

			keys, keyLengths := extractKeyListVariants(ctx.KeyListVariants())
			equal := equalKeys(keys, idx.keys) && equalKeyLengths(keyLengths, idx.lengths)

			var comment string
			for _, v := range ctx.AllIndexOption() {
				if v.CommonIndexOption() != nil && v.CommonIndexOption().COMMENT_SYMBOL() != nil {
					comment = v.CommonIndexOption().TextLiteral().GetText()
					if len(comment) > 2 {
						quotes := comment[0]
						escape := fmt.Sprintf("%c%c", quotes, quotes)
						comment = strings.ReplaceAll(comment[1:len(comment)-1], escape, string(quotes))
					}
					break
				}
			}

			equal = equal && (comment == idx.comment)
			equal = equal && (!idx.primary) && (!idx.unique)

			if equal {
				if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)); err != nil {
					g.err = err
					return
				}
			} else {
				if err := idx.toString(&g.tableConstraints); err != nil {
					g.err = err
					return
				}
			}
			delete(g.currentTable.indexes, name)
		}
	case "UNIQUE":
		var name string
		if ctx.ConstraintName() != nil && ctx.ConstraintName().Identifier() != nil {
			name = mysqlparser.NormalizeMySQLIdentifier(ctx.ConstraintName().Identifier())
		}
		if ctx.IndexNameAndType() != nil {
			if ctx.IndexNameAndType().IndexName() != nil {
				name = mysqlparser.NormalizeMySQLIdentifier(ctx.IndexNameAndType().IndexName().Identifier())
			}
		}
		if g.currentTable.indexes[name] != nil {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
					g.err = err
					return
				}
			}

			var comment string
			for _, v := range ctx.AllFulltextIndexOption() {
				if v.CommonIndexOption() != nil {
					if v.CommonIndexOption().COMMENT_SYMBOL() != nil {
						comment = v.CommonIndexOption().TextLiteral().GetText()
						if len(comment) > 2 {
							quotes := comment[0]
							escape := fmt.Sprintf("%c%c", quotes, quotes)
							comment = strings.ReplaceAll(comment[1:len(comment)-1], escape, string(quotes))
						}
					}
				}
			}

			idx := g.currentTable.indexes[name]
			keys, keyLengths := extractKeyListVariants(ctx.KeyListVariants())
			equal := equalKeys(keys, idx.keys) && equalKeyLengths(keyLengths, idx.lengths)
			equal = equal && (!idx.primary) && (idx.unique) && (idx.comment == comment)

			if equal {
				if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)); err != nil {
					g.err = err
					return
				}
			} else {
				if err := idx.toString(&g.tableConstraints); err != nil {
					g.err = err
					return
				}
			}
			delete(g.currentTable.indexes, name)
		}
	case "FULLTEXT":
		var name string
		if ctx.IndexName() != nil {
			name = mysqlparser.NormalizeMySQLIdentifier(ctx.IndexName().Identifier())
		}
		if g.currentTable.indexes[name] != nil {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
					g.err = err
				}
			}

			var comment string
			for _, v := range ctx.AllFulltextIndexOption() {
				if v.CommonIndexOption() != nil {
					if v.CommonIndexOption().COMMENT_SYMBOL() != nil {
						comment = v.CommonIndexOption().TextLiteral().GetText()
						if len(comment) > 2 {
							quotes := comment[0]
							escape := fmt.Sprintf("%c%c", quotes, quotes)
							comment = strings.ReplaceAll(comment[1:len(comment)-1], escape, string(quotes))
						}
					}
				}
			}

			idx := g.currentTable.indexes[name]
			keys, keyLengths := extractKeyListVariants(ctx.KeyListVariants())
			equal := equalKeys(keys, idx.keys) && equalKeyLengths(keyLengths, idx.lengths)
			equal = equal && (!idx.primary) && (!idx.unique) && (idx.comment == comment)

			if equal {
				if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)); err != nil {
					g.err = err
					return
				}
			} else {
				if err := idx.toString(&g.tableConstraints); err != nil {
					g.err = err
					return
				}
			}
		}
	case "SPATIAL":
		var name string
		if ctx.IndexName() != nil {
			name = mysqlparser.NormalizeMySQLIdentifier(ctx.IndexName().Identifier())
		}
		if g.currentTable.indexes[name] != nil {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
					g.err = err
				}
			}

			var comment string
			for _, v := range ctx.AllSpatialIndexOption() {
				if v.CommonIndexOption() != nil {
					if v.CommonIndexOption().COMMENT_SYMBOL() != nil {
						comment = v.CommonIndexOption().TextLiteral().GetText()
						if len(comment) > 2 {
							quotes := comment[0]
							escape := fmt.Sprintf("%c%c", quotes, quotes)
							comment = strings.ReplaceAll(comment[1:len(comment)-1], escape, string(quotes))
						}
					}
				}
			}

			idx := g.currentTable.indexes[name]
			keys, keyLengths := extractKeyListVariants(ctx.KeyListVariants())
			equal := equalKeys(keys, idx.keys) && equalKeyLengths(keyLengths, idx.lengths)
			equal = equal && (!idx.primary) && (!idx.unique) && (idx.comment == comment)

			if equal {
				if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)); err != nil {
					g.err = err
					return
				}
			} else {
				if err := idx.toString(&g.tableConstraints); err != nil {
					g.err = err
					return
				}
			}
		}
	default:
		if g.firstElementInTable {
			g.firstElementInTable = false
		} else {
			if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
				g.err = err
				return
			}
		}
		if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)); err != nil {
			g.err = err
			return
		}
	}
}

// EnterColumnDefinition is called when production columnDefinition is entered.
func (g *mysqlDesignSchemaGenerator) EnterColumnDefinition(ctx *mysql.ColumnDefinitionContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	_, _, columnName := mysqlparser.NormalizeMySQLColumnName(ctx.ColumnName())
	column, ok := g.currentTable.columns[columnName]
	if !ok {
		return
	}

	delete(g.currentTable.columns, columnName)

	if g.firstElementInTable {
		g.firstElementInTable = false
	} else {
		if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
			g.err = err
			return
		}
	}

	// compare column type
	typeCtx := ctx.FieldDefinition().DataType()
	columnType := getDataTypePlainText(typeCtx)
	if !strings.EqualFold(columnType, column.tp) {
		if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.GetStart().GetTokenIndex(),
			Stop:  typeCtx.GetStart().GetTokenIndex() - 1,
		})); err != nil {
			g.err = err
			return
		}
		// write lower case column type for MySQL
		if _, err := g.columnDefine.WriteString(strings.ToLower(column.tp)); err != nil {
			g.err = err
			return
		}
	} else {
		if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.GetStart().GetTokenIndex(),
			Stop:  typeCtx.GetStop().GetTokenIndex(),
		})); err != nil {
			g.err = err
			return
		}
	}
	startPos := typeCtx.GetStop().GetTokenIndex() + 1

	// Column attributes.
	// TODO(zp): refactor column auto_increment.
	skipSchemaAutoIncrement := false
	for _, attr := range ctx.FieldDefinition().AllColumnAttribute() {
		if attr.AUTO_INCREMENT_SYMBOL() != nil || attr.DEFAULT_SYMBOL() != nil {
			// if schema string has default value or auto_increment.
			// and metdata has default value.
			// we skip the schema auto_increment and only compare default value.
			skipSchemaAutoIncrement = column.defaultValue != nil
			break
		}
	}
	newAttr := extractNewAttrs(column, ctx.FieldDefinition().AllColumnAttribute())

	for _, attribute := range ctx.FieldDefinition().AllColumnAttribute() {
		attrOrder := getAttrOrder(attribute)
		for ; len(newAttr) > 0 && newAttr[0].order < attrOrder; newAttr = newAttr[1:] {
			if _, err := g.columnDefine.WriteString(" " + newAttr[0].text); err != nil {
				g.err = err
				return
			}
		}
		switch {
		// nullable
		case attribute.NullLiteral() != nil:
			sameNullable := attribute.NOT_SYMBOL() == nil && column.nullable
			sameNullable = sameNullable || (attribute.NOT_SYMBOL() != nil && !column.nullable)
			if sameNullable {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: startPos,
					Stop:  attribute.GetStop().GetTokenIndex(),
				})); err != nil {
					g.err = err
					return
				}
			} else {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: startPos,
					Stop:  attribute.GetStart().GetTokenIndex() - 1,
				})); err != nil {
					g.err = err
					return
				}
				if !column.nullable {
					if _, err := g.columnDefine.WriteString(" NOT NULL"); err != nil {
						g.err = err
						return
					}
				}
			}
		// default value
		// https://dev.mysql.com/doc/refman/8.0/en/data-type-defaults.html
		case attribute.DEFAULT_SYMBOL() != nil && attribute.SERIAL_SYMBOL() == nil:
			defaultValueStart := nextDefaultChannelTokenIndex(attribute.GetParser().GetTokenStream(), attribute.DEFAULT_SYMBOL().GetSymbol().GetTokenIndex())
			defaultValueText := attribute.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: defaultValueStart,
				Stop:  attribute.GetStop().GetTokenIndex(),
			})
			var defaultValue defaultValue
			switch {
			case strings.EqualFold(defaultValueText, "NULL"):
				defaultValue = &defaultValueNull{}
			case strings.HasPrefix(defaultValueText, "'") && strings.HasSuffix(defaultValueText, "'"):
				defaultValue = &defaultValueString{value: strings.ReplaceAll(defaultValueText[1:len(defaultValueText)-1], "''", "'")}
			case strings.HasPrefix(defaultValueText, "\"") && strings.HasSuffix(defaultValueText, "\""):
				defaultValue = &defaultValueString{value: strings.ReplaceAll(defaultValueText[1:len(defaultValueText)-1], "\"\"", "\"")}
			default:
				defaultValue = &defaultValueExpression{value: defaultValueText}
			}
			if column.defaultValue != nil && column.defaultValue.toString() == defaultValue.toString() {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: startPos,
					Stop:  attribute.GetStop().GetTokenIndex(),
				})); err != nil {
					g.err = err
					return
				}
			} else if column.defaultValue != nil {
				// todo(zp): refactor column attribute.
				if strings.EqualFold(column.defaultValue.toString(), autoIncrementSymbol) {
					if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
						Start: startPos,
						Stop:  attribute.DEFAULT_SYMBOL().GetSymbol().GetTokenIndex() - 1,
					})); err != nil {
						g.err = err
						return
					}
				} else {
					if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
						Start: startPos,
						Stop:  defaultValueStart - 1,
					})); err != nil {
						g.err = err
						return
					}
				}
				_, isNull := column.defaultValue.(*defaultValueNull)
				dontWriteDefaultNull := isNull && column.nullable && expressionDefaultOnlyTypes[strings.ToUpper(column.tp)]
				if !dontWriteDefaultNull {
					if _, err := g.columnDefine.WriteString(column.defaultValue.toString()); err != nil {
						g.err = err
						return
					}
				}
			}
		case attribute.COMMENT_SYMBOL() != nil:
			commentStart := nextDefaultChannelTokenIndex(attribute.GetParser().GetTokenStream(), attribute.COMMENT_SYMBOL().GetSymbol().GetTokenIndex())
			commentValue := attribute.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: commentStart,
				Stop:  attribute.GetStop().GetTokenIndex(),
			})
			if commentValue != `''` && len(commentValue) > 2 && column.comment == commentValue[1:len(commentValue)-1] {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: startPos,
					Stop:  attribute.GetStop().GetTokenIndex(),
				})); err != nil {
					g.err = err
					return
				}
			} else if column.comment != "" {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: startPos,
					Stop:  commentStart - 1,
				})); err != nil {
					g.err = err
					return
				}
				if _, err := g.columnDefine.WriteString(fmt.Sprintf("'%s'", column.comment)); err != nil {
					g.err = err
					return
				}
			}

		case attribute.AUTO_INCREMENT_SYMBOL() != nil && skipSchemaAutoIncrement:
			// just skip this condition.
		default:
			if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: startPos,
				Stop:  attribute.GetStop().GetTokenIndex(),
			})); err != nil {
				g.err = err
				return
			}
		}
		startPos = attribute.GetStop().GetTokenIndex() + 1
	}

	for _, attr := range newAttr {
		if _, err := g.columnDefine.WriteString(" " + attr.text); err != nil {
			g.err = err
			return
		}
	}

	if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: startPos,
		Stop:  ctx.GetStop().GetTokenIndex(),
	})); err != nil {
		g.err = err
		return
	}
}

// DropViewStatement is called when production dropViewStatement is entered.
//
// mysqldump generate drop view if exists statement after all create table statement.
// To provide the better ux, we generate the new tables before the drop view statement.
func (g *mysqlDesignSchemaGenerator) EnterDropView(*mysql.DropViewContext) {
	if g.err != nil {
		return
	}

	if err := writeRemainingTables(&g.result, g.desired, g.to); err != nil {
		g.err = err
		return
	}
}

func (g *mysqlDesignSchemaGenerator) EnterCreateProcedure(*mysql.CreateProcedureContext) {
	if g.err != nil {
		return
	}

	if err := writeRemainingTables(&g.result, g.desired, g.to); err != nil {
		g.err = err
		return
	}
}

func (g *mysqlDesignSchemaGenerator) EnterCreateFunction(*mysql.CreateFunctionContext) {
	if g.err != nil {
		return
	}

	if err := writeRemainingTables(&g.result, g.desired, g.to); err != nil {
		g.err = err
		return
	}
}

func (g *mysqlDesignSchemaGenerator) EnterCreateEvent(*mysql.CreateEventContext) {
	if g.err != nil {
		return
	}

	if err := writeRemainingTables(&g.result, g.desired, g.to); err != nil {
		g.err = err
		return
	}
}

func (g *mysqlDesignSchemaGenerator) EnterCreateTriggers(*mysql.CreateTriggerContext) {
	if g.err != nil {
		return
	}

	if err := writeRemainingTables(&g.result, g.desired, g.to); err != nil {
		g.err = err
		return
	}
}

// EnterSetStatement is called when production setStatement is entered.
//
// mysqldump generates `SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;` statement at the end of the file, and
// generates `SET character_set_client` statement at the beginning of create function statement.
// to provide the better user experience, we generate the remaining tables before the set statement mentioned above.
func (g *mysqlDesignSchemaGenerator) EnterSetStatement(ctx *mysql.SetStatementContext) {
	if g.err != nil {
		return
	}

	curSet := strings.TrimSpace(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx))
	if curSet != `SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS` && !strings.HasPrefix(curSet, "SET character_set_client") {
		return
	}

	if err := writeRemainingTables(&g.result, g.desired, g.to); err != nil {
		g.err = err
		return
	}
}

func writeRemainingTables(w io.StringWriter, to *storepb.DatabaseSchemaMetadata, state *databaseState) error {
	firstTable := true
	// Follow the order of the input schemas.
	for _, schema := range to.Schemas {
		schemaState, ok := state.schemas[schema.Name]
		if !ok {
			continue
		}
		// Follow the order of the input tables.
		for idx, table := range schema.Tables {
			table, ok := schemaState.tables[table.Name]
			if !ok {
				continue
			}
			if firstTable {
				firstTable = false
				if _, err := w.WriteString("\n"); err != nil {
					return err
				}
			}
			if _, err := w.WriteString(getTableAnnouncement(table.name)); err != nil {
				return err
			}

			// Avoid new line.
			buf := &strings.Builder{}
			if err := table.toString(buf); err != nil {
				return err
			}
			if idx == len(schema.Tables)-1 && buf.String()[len(buf.String())-1] == '\n' {
				if _, err := w.WriteString(buf.String()[:len(buf.String())-1]); err != nil {
					return err
				}
			} else {
				if _, err := w.WriteString(buf.String()); err != nil {
					return err
				}
			}
			delete(schemaState.tables, table.name)
		}
	}
	return nil
}

func getTableAnnouncement(name string) string {
	return fmt.Sprintf("\n--\n-- Table structure for table `%s`\n--\n", name)
}

// getDataTypePlainText returns the plain text of the data type,
// which excludes the charset candidate.
// For example, for "varchar(10) CHARACTER SET utf8mb4",
// it returns "varchar(10)".
func getDataTypePlainText(typeCtx mysql.IDataTypeContext) string {
	begin := typeCtx.GetStart().GetTokenIndex()
	end := typeCtx.GetStop().GetTokenIndex()
	if typeCtx.CharsetWithOptBinary() != nil {
		end = typeCtx.CharsetWithOptBinary().GetStart().GetTokenIndex() - 1
	}
	// To skip the trailing spaces, we iterate the token stream reversely and find the first default channel token index.
	for i := end; i >= begin; i-- {
		if typeCtx.GetParser().GetTokenStream().Get(i).GetChannel() == antlr.TokenDefaultChannel {
			end = i
			break
		}
	}

	if end < begin {
		return ""
	}

	return typeCtx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: begin,
		Stop:  end,
	})
}
