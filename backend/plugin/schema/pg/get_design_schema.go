package pg

import (
	"fmt"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	postgres "github.com/bytebase/postgresql-parser"

	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/plugin/schema"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterGetDesignSchema(storepb.Engine_POSTGRES, GetDesignSchema)
}

type designSchemaGenerator struct {
	*postgres.BasePostgreSQLParserListener

	to                  *databaseState
	result              strings.Builder
	currentTable        *tableState
	firstElementInTable bool
	columnDefine        strings.Builder
	tableConstraints    strings.Builder
	err                 error

	lastTokenIndex int
}

// GetDesignSchema returns the schema string for the design schema.
func GetDesignSchema(baselineSchema string, to *storepb.DatabaseSchemaMetadata) (string, error) {
	toState := convertToDatabaseState(to)
	parseResult, err := pgparser.ParsePostgreSQL(baselineSchema)
	if err != nil {
		return "", err
	}
	if parseResult == nil {
		return "", nil
	}
	if parseResult.Tree == nil {
		return "", nil
	}

	listener := &designSchemaGenerator{
		lastTokenIndex: 0,
		to:             toState,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, parseResult.Tree)
	if listener.err != nil {
		return "", listener.err
	}
	root, ok := parseResult.Tree.(*postgres.RootContext)
	if !ok {
		return "", errors.Errorf("failed to convert to RootContext")
	}
	if root.GetStop() != nil {
		if _, err := listener.result.WriteString(root.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: listener.lastTokenIndex,
			Stop:  root.GetStop().GetTokenIndex(),
		})); err != nil {
			return "", err
		}
	}

	// Follow the order of the input schema.
	for _, schema := range to.Schemas {
		schemaState, exists := listener.to.schemas[schema.Name]
		if !exists {
			continue
		}
		if err := schemaState.printCreateSchema(&listener.result); err != nil {
			return "", err
		}
		// Follow the order of the input table.
		for _, table := range schema.Tables {
			tableState, exists := schemaState.tables[table.Name]
			if !exists {
				continue
			}
			if err := tableState.toString(&listener.result, schema.Name); err != nil {
				return "", err
			}

			if _, err := listener.result.WriteString("\n"); err != nil {
				return "", err
			}

			for _, column := range table.Columns {
				columnState, exists := tableState.columns[column.Name]
				if !exists {
					continue
				}

				if column.Comment != "" && !columnState.ignoreComment {
					if err := columnState.commentToString(&listener.result, schema.Name, table.Name); err != nil {
						return "", err
					}
					if _, err := listener.result.WriteString("\n"); err != nil {
						return "", err
					}
				}
			}

			if table.Comment != "" && !tableState.ignoreComment {
				if err := tableState.commentToString(&listener.result, schema.Name); err != nil {
					return "", err
				}
				if _, err := listener.result.WriteString("\n"); err != nil {
					return "", err
				}
			}
		}
	}

	return listener.result.String(), nil
}

// EnterCreatestmt is called when production createstmt is entered.
func (g *designSchemaGenerator) EnterCreatestmt(ctx *postgres.CreatestmtContext) {
	if g.err != nil {
		return
	}
	if ctx.Opttableelementlist() == nil {
		// Skip other create statement for now.
		return
	}
	schemaName, tableName, err := pgparser.NormalizePostgreSQLQualifiedNameAsTableName(ctx.Qualified_name(0))
	if err != nil {
		g.err = err
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
	g.lastTokenIndex = ctx.GetStart().GetTokenIndex()

	schema, exists := g.to.schemas[schemaName]
	if !exists {
		// Skip not found schema.
		g.lastTokenIndex = skipFollowingSemiIndex(ctx.GetParser().GetTokenStream(), ctx.GetStop().GetTokenIndex()+1)
		return
	}

	table, exists := schema.tables[tableName]
	if !exists {
		// Skip not found table.
		g.lastTokenIndex = skipFollowingSemiIndex(ctx.GetParser().GetTokenStream(), ctx.GetStop().GetTokenIndex()+1)
		return
	}

	g.currentTable = table
	g.firstElementInTable = true
	g.columnDefine.Reset()
	g.tableConstraints.Reset()

	table.ignoreTable = true
	// Write the text before the table element list.
	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: ctx.GetStart().GetTokenIndex(),
		Stop:  ctx.Opttableelementlist().GetStart().GetTokenIndex() - 1,
	})); err != nil {
		g.err = err
		return
	}
}

func (g *designSchemaGenerator) ExitCreatestmt(ctx *postgres.CreatestmtContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	var columnList []*columnState
	for _, column := range g.currentTable.columns {
		if column.ignore {
			continue
		}
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

	if _, err := g.result.WriteString(g.columnDefine.String()); err != nil {
		g.err = err
		return
	}
	if _, err := g.result.WriteString(g.tableConstraints.String()); err != nil {
		g.err = err
		return
	}

	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: ctx.Opttableelementlist().GetStop().GetTokenIndex() + 1,
		Stop:  ctx.GetStop().GetTokenIndex(),
	})); err != nil {
		g.err = err
		return
	}
	g.lastTokenIndex = ctx.GetStop().GetTokenIndex() + 1
	g.currentTable = nil
	g.firstElementInTable = false
}

func skipFollowingSemiIndex(stream antlr.TokenStream, index int) int {
	for i := index; i < stream.Size(); i++ {
		token := stream.Get(i)
		if token.GetTokenType() == postgres.PostgreSQLParserSEMI {
			return i + 1
		}
		if token.GetTokenType() == postgres.PostgreSQLParserEOF {
			return i
		}
	}
	return index
}

func (g *designSchemaGenerator) EnterAltertablestmt(ctx *postgres.AltertablestmtContext) {
	if g.err != nil {
		return
	}

	if ctx.TABLE() == nil || ctx.Alter_table_cmds() == nil || len(ctx.Alter_table_cmds().AllAlter_table_cmd()) != 1 {
		// Skip other alter table statement for now.
		return
	}

	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: g.lastTokenIndex,
		Stop:  ctx.GetStart().GetTokenIndex() - 1,
	})); err != nil {
		g.err = err
		return
	}
	g.lastTokenIndex = skipFollowingSemiIndex(ctx.GetParser().GetTokenStream(), ctx.GetStop().GetTokenIndex()+1)

	schemaName, tableName, err := pgparser.NormalizePostgreSQLQualifiedNameAsTableName(ctx.Relation_expr().Qualified_name())
	if err != nil {
		g.err = err
		return
	}

	schema, exists := g.to.schemas[schemaName]
	if !exists {
		// Skip not found schema.
		return
	}

	table, exists := schema.tables[tableName]
	if !exists {
		// Skip not found table.
		return
	}

	cmd := ctx.Alter_table_cmds().Alter_table_cmd(0)
	switch {
	case cmd.ADD_P() != nil && cmd.Tableconstraint() != nil:
		constraint := cmd.Tableconstraint().Constraintelem()
		switch {
		case constraint.PRIMARY() != nil && constraint.KEY() != nil:
			name := cmd.Tableconstraint().Name()
			if name == nil {
				g.err = errors.Errorf("primary key constraint must have a name")
				return
			}
			nameText := pgparser.NormalizePostgreSQLColid(name.Colid())
			index, exists := table.indexes[nameText]
			if !exists {
				// Skip not found primary key.
				return
			}
			delete(table.indexes, nameText)
			keys := extractColumnList(constraint.Columnlist())
			if equalKeys(keys, index.keys) {
				if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: ctx.GetStart().GetTokenIndex(),
					Stop:  g.lastTokenIndex - 1,
				})); err != nil {
					g.err = err
					return
				}
			} else {
				if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: ctx.GetStart().GetTokenIndex(),
					Stop:  constraint.Columnlist().GetStart().GetTokenIndex() - 1,
				})); err != nil {
					g.err = err
					return
				}
				newKeys := []string{}
				for _, key := range index.keys {
					newKeys = append(newKeys, fmt.Sprintf(`"%s"`, key))
				}
				if _, err := g.result.WriteString(strings.Join(newKeys, ", ")); err != nil {
					g.err = err
					return
				}
				if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: constraint.Columnlist().GetStop().GetTokenIndex() + 1,
					Stop:  g.lastTokenIndex - 1,
				})); err != nil {
					g.err = err
					return
				}
			}
		case constraint.FOREIGN() != nil && constraint.KEY() != nil:
			name := cmd.Tableconstraint().Name()
			if name == nil {
				g.err = errors.Errorf("foreign key constraint must have a name")
				return
			}
			nameText := pgparser.NormalizePostgreSQLColid(name.Colid())
			fk, exists := table.foreignKeys[nameText]
			if !exists {
				// Skip not found foreign key.
				return
			}
			delete(table.foreignKeys, nameText)
			columns := extractColumnList(constraint.Columnlist())
			referencedSchemaName, referencedTableName, err := pgparser.NormalizePostgreSQLQualifiedNameAsTableName(constraint.Qualified_name())
			if err != nil {
				g.err = err
				return
			}
			referencedColumns := extractColumnList(constraint.Opt_column_list().Columnlist())
			equal := equalKeys(columns, fk.columns) && equalKeys(referencedColumns, fk.referencedColumns) && referencedSchemaName == fk.referencedSchema && referencedTableName == fk.referencedTable
			if equal {
				if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: ctx.GetStart().GetTokenIndex(),
					Stop:  g.lastTokenIndex - 1,
				})); err != nil {
					g.err = err
					return
				}
			} else {
				if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: ctx.GetStart().GetTokenIndex(),
					Stop:  constraint.Columnlist().GetStart().GetTokenIndex() - 1,
				})); err != nil {
					g.err = err
					return
				}
				newColumns := []string{}
				for _, column := range fk.columns {
					newColumns = append(newColumns, fmt.Sprintf(`"%s"`, column))
				}
				if _, err := g.result.WriteString(strings.Join(newColumns, ", ")); err != nil {
					g.err = err
					return
				}
				if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: constraint.Columnlist().GetStop().GetTokenIndex() + 1,
					Stop:  constraint.Qualified_name().GetStart().GetTokenIndex() - 1,
				})); err != nil {
					g.err = err
					return
				}
				if _, err := g.result.WriteString(fmt.Sprintf(`"%s"."%s"(`, fk.referencedSchema, fk.referencedTable)); err != nil {
					g.err = err
					return
				}
				newReferencedColumns := []string{}
				for _, column := range fk.referencedColumns {
					newReferencedColumns = append(newReferencedColumns, fmt.Sprintf(`"%s"`, column))
				}
				if _, err := g.result.WriteString(strings.Join(newReferencedColumns, ", ")); err != nil {
					g.err = err
					return
				}
				if _, err := g.result.WriteString(")"); err != nil {
					g.err = err
					return
				}
				if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: constraint.Opt_column_list().GetStop().GetTokenIndex() + 1,
					Stop:  g.lastTokenIndex - 1,
				})); err != nil {
					g.err = err
					return
				}
			}
		default:
			if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.GetStart().GetTokenIndex(),
				Stop:  g.lastTokenIndex - 1,
			})); err != nil {
				g.err = err
				return
			}
		}
	default:
		if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.GetStart().GetTokenIndex(),
			Stop:  g.lastTokenIndex - 1,
		})); err != nil {
			g.err = err
			return
		}
	}
}

func equalKeys(keys1, keys2 []string) bool {
	if len(keys1) != len(keys2) {
		return false
	}
	for i, key := range keys1 {
		if key != keys2[i] {
			return false
		}
	}
	return true
}

func extractColumnList(columnList postgres.IColumnlistContext) []string {
	result := []string{}
	for _, item := range columnList.AllColumnElem() {
		result = append(result, pgparser.NormalizePostgreSQLColid(item.Colid()))
	}
	return result
}

func (g *designSchemaGenerator) EnterTableconstraint(ctx *postgres.TableconstraintContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	if g.firstElementInTable {
		g.firstElementInTable = false
	} else {
		if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
			g.err = err
			return
		}
	}
	if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: ctx.GetStart().GetTokenIndex(),
		Stop:  ctx.GetStop().GetTokenIndex(),
	})); err != nil {
		g.err = err
		return
	}
}

func (g *designSchemaGenerator) EnterTablelikeclause(ctx *postgres.TablelikeclauseContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	if g.firstElementInTable {
		g.firstElementInTable = false
	} else {
		if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
			g.err = err
			return
		}
	}
	if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: ctx.GetStart().GetTokenIndex(),
		Stop:  ctx.GetStop().GetTokenIndex(),
	})); err != nil {
		g.err = err
		return
	}
}

func (g *designSchemaGenerator) EnterCreateschemastmt(ctx *postgres.CreateschemastmtContext) {
	if g.err != nil {
		return
	}
	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: g.lastTokenIndex,
		Stop:  ctx.GetStart().GetTokenIndex() - 1,
	})); err != nil {
		g.err = err
		return
	}

	g.lastTokenIndex = skipFollowingSemiIndex(ctx.GetParser().GetTokenStream(), ctx.GetStop().GetTokenIndex()+1)
	endTokenIndex := g.lastTokenIndex - 1

	var schemaName string
	if ctx.Colid() != nil {
		schemaName = pgparser.NormalizePostgreSQLColid(ctx.Colid())
	} else if ctx.Optschemaname() != nil && ctx.Optschemaname().Colid() != nil {
		schemaName = pgparser.NormalizePostgreSQLColid(ctx.Optschemaname().Colid())
	}

	schema, exists := g.to.schemas[schemaName]
	if !exists {
		// Skip not found schema.
		return
	}

	schema.ignore = true
	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: ctx.GetStart().GetTokenIndex(),
		Stop:  endTokenIndex,
	})); err != nil {
		g.err = err
		return
	}
}

func (g *designSchemaGenerator) EnterCommentstmt(ctx *postgres.CommentstmtContext) {
	if g.err != nil {
		return
	}
	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: g.lastTokenIndex,
		Stop:  ctx.GetStart().GetTokenIndex() - 1,
	})); err != nil {
		g.err = err
		return
	}

	g.lastTokenIndex = skipFollowingSemiIndex(ctx.GetParser().GetTokenStream(), ctx.GetStop().GetTokenIndex()+1)
	endTokenIndex := g.lastTokenIndex - 1

	if ctx.Object_type_any_name() != nil {
		if ctx.Object_type_any_name().TABLE() != nil && ctx.Object_type_any_name().FOREIGN() == nil {
			schemaName, tableName, err := pgparser.NormalizePostgreSQLAnyNameAsTableName(ctx.Any_name())
			if err != nil {
				g.err = err
				return
			}
			schema, exists := g.to.schemas[schemaName]
			if !exists {
				// Skip not found schema.
				return
			}
			_, exists = schema.tables[tableName]
			if !exists {
				// Skip not found table.
				return
			}
		}
	}

	switch {
	case ctx.COLUMN() != nil:
		schemaName, tableName, columnName, err := pgparser.NormalizePostgreSQLAnyNameAsColumnName(ctx.Any_name())
		if err != nil {
			g.err = err
			return
		}
		schema, exists := g.to.schemas[schemaName]
		if !exists {
			// Skip not found schema.
			return
		}
		table, exists := schema.tables[tableName]
		if !exists {
			// Skip not found table.
			return
		}
		column, exists := table.columns[columnName]
		if !exists {
			// Skip not found column.
			return
		}
		equal := false
		column.ignoreComment = true
		if ctx.Comment_text().NULL_P() != nil {
			equal = len(column.comment) == 0
		} else {
			if len(column.comment) == 0 {
				// Skip for empty comment string.
				return
			}
			commentText := ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.Comment_text().GetStart().GetTokenIndex(),
				Stop:  ctx.Comment_text().GetStop().GetTokenIndex(),
			})
			if len(commentText) > 2 && commentText[0] == '\'' && commentText[len(commentText)-1] == '\'' {
				commentText = unescapePostgreSQLString(commentText[1 : len(commentText)-1])
			}

			equal = commentText == column.comment
		}

		if equal {
			if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.GetStart().GetTokenIndex(),
				Stop:  endTokenIndex,
			})); err != nil {
				g.err = err
				return
			}
		} else {
			if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.GetStart().GetTokenIndex(),
				Stop:  ctx.Comment_text().GetStart().GetTokenIndex() - 1,
			})); err != nil {
				g.err = err
				return
			}
			if _, err := g.result.WriteString(fmt.Sprintf("'%s'", escapePostgreSQLString(column.comment))); err != nil {
				g.err = err
				return
			}
			if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.Comment_text().GetStop().GetTokenIndex() + 1,
				Stop:  endTokenIndex,
			})); err != nil {
				g.err = err
				return
			}
		}
	case ctx.Object_type_any_name() != nil && ctx.Object_type_any_name().TABLE() != nil:
		schemaName, tableName, err := pgparser.NormalizePostgreSQLAnyNameAsTableName(ctx.Any_name())
		if err != nil {
			g.err = err
			return
		}
		schema, exists := g.to.schemas[schemaName]
		if !exists {
			// Skip not found schema.
			return
		}
		table, exists := schema.tables[tableName]
		if !exists {
			// Skip not found table.
			return
		}
		equal := false
		table.ignoreComment = true
		if ctx.Comment_text().NULL_P() != nil {
			equal = len(table.comment) == 0
		} else {
			if len(table.comment) == 0 {
				// Skip for empty comment string.
				return
			}
			commentText := ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.Comment_text().GetStart().GetTokenIndex(),
				Stop:  ctx.Comment_text().GetStop().GetTokenIndex(),
			})
			if len(commentText) > 2 && commentText[0] == '\'' && commentText[len(commentText)-1] == '\'' {
				commentText = unescapePostgreSQLString(commentText[1 : len(commentText)-1])
			}

			equal = commentText == table.comment
		}

		if equal {
			if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.GetStart().GetTokenIndex(),
				Stop:  endTokenIndex,
			})); err != nil {
				g.err = err
				return
			}
		} else {
			if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.GetStart().GetTokenIndex(),
				Stop:  ctx.Comment_text().GetStart().GetTokenIndex() - 1,
			})); err != nil {
				g.err = err
				return
			}
			if _, err := g.result.WriteString(fmt.Sprintf("'%s'", escapePostgreSQLString(table.comment))); err != nil {
				g.err = err
				return
			}
			if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.Comment_text().GetStop().GetTokenIndex() + 1,
				Stop:  endTokenIndex,
			})); err != nil {
				g.err = err
				return
			}
		}
	default:
		// Keep other comment statements.
		if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.GetStart().GetTokenIndex(),
			Stop:  endTokenIndex,
		})); err != nil {
			g.err = err
			return
		}
		return
	}
}

func (g *designSchemaGenerator) EnterColumnDef(ctx *postgres.ColumnDefContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	columnName := pgparser.NormalizePostgreSQLColid(ctx.Colid())
	column, exists := g.currentTable.columns[columnName]
	if !exists {
		return
	}
	column.ignore = true

	if g.firstElementInTable {
		g.firstElementInTable = false
	} else {
		if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
			g.err = err
			return
		}
	}

	// compare column type
	columnType := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(
		ctx.Typename(),
	)
	equal, err := equalType(column.tp, columnType)
	if err != nil {
		g.err = err
		return
	}
	if !equal {
		if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.GetStart().GetTokenIndex(),
			Stop:  ctx.Typename().GetStart().GetTokenIndex() - 1,
		})); err != nil {
			g.err = err
			return
		}
		if _, err := g.columnDefine.WriteString(column.tp); err != nil {
			g.err = err
			return
		}
	} else {
		if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.GetStart().GetTokenIndex(),
			Stop:  ctx.Typename().GetStop().GetTokenIndex(),
		})); err != nil {
			g.err = err
			return
		}
	}
	needOneSpace := false

	// if there are other tokens between column type and column constraint, write them.
	if ctx.Colquallist().GetStop().GetTokenIndex() > ctx.Colquallist().GetStart().GetTokenIndex() {
		if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.Typename().GetStop().GetTokenIndex() + 1,
			Stop:  ctx.Colquallist().GetStart().GetTokenIndex() - 1,
		})); err != nil {
			g.err = err
			return
		}
	} else {
		needOneSpace = true
	}
	startPos := ctx.Colquallist().GetStart().GetTokenIndex()

	if !column.nullable && !nullableExists(ctx.Colquallist()) {
		if needOneSpace {
			if _, err := g.columnDefine.WriteString(" "); err != nil {
				g.err = err
				return
			}
		}
		if _, err := g.columnDefine.WriteString("NOT NULL"); err != nil {
			g.err = err
			return
		}
		needOneSpace = true
	}

	if column.hasDefault && !defaultExists(ctx.Colquallist()) {
		if needOneSpace {
			if _, err := g.columnDefine.WriteString(" "); err != nil {
				g.err = err
				return
			}
		}
		if _, err := g.columnDefine.WriteString(fmt.Sprintf("DEFAULT %s", column.defaultValue.toString())); err != nil {
			g.err = err
			return
		}
		needOneSpace = true
	}

	for i, item := range ctx.Colquallist().AllColconstraint() {
		if i == 0 && needOneSpace {
			if _, err := g.columnDefine.WriteString(" "); err != nil {
				g.err = err
				return
			}
		}
		if item.Colconstraintelem() == nil {
			if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(
				antlr.Interval{
					Start: startPos,
					Stop:  item.GetStop().GetTokenIndex(),
				},
			)); err != nil {
				g.err = err
				return
			}
			startPos = item.GetStop().GetTokenIndex() + 1
			continue
		}

		constraint := item.Colconstraintelem()

		switch {
		case constraint.NULL_P() != nil:
			sameNullable := (constraint.NOT() == nil && column.nullable) || (constraint.NOT() != nil && !column.nullable)
			if sameNullable {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(
					antlr.Interval{
						Start: startPos,
						Stop:  item.GetStop().GetTokenIndex(),
					},
				)); err != nil {
					g.err = err
					return
				}
			}
		case constraint.DEFAULT() != nil:
			defaultValue := ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: constraint.B_expr().GetStart().GetTokenIndex(),
				Stop:  constraint.B_expr().GetStop().GetTokenIndex(),
			})
			if column.hasDefault && column.defaultValue.toString() == defaultValue {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(
					antlr.Interval{
						Start: startPos,
						Stop:  item.GetStop().GetTokenIndex(),
					},
				)); err != nil {
					g.err = err
					return
				}
			} else if column.hasDefault {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(
					antlr.Interval{
						Start: startPos,
						Stop:  constraint.B_expr().GetStart().GetTokenIndex() - 1,
					},
				)); err != nil {
					g.err = err
					return
				}
				if _, err := g.columnDefine.WriteString(column.defaultValue.toString()); err != nil {
					g.err = err
					return
				}
			}
		default:
			if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(
				antlr.Interval{
					Start: startPos,
					Stop:  item.GetStop().GetTokenIndex(),
				},
			)); err != nil {
				g.err = err
				return
			}
		}
		startPos = item.GetStop().GetTokenIndex() + 1
	}
}

func defaultExists(colquallist postgres.IColquallistContext) bool {
	if colquallist == nil {
		return false
	}

	for _, item := range colquallist.AllColconstraint() {
		if item.Colconstraintelem() == nil {
			continue
		}

		if item.Colconstraintelem().DEFAULT() != nil {
			return true
		}
	}

	return false
}

func nullableExists(colquallist postgres.IColquallistContext) bool {
	if colquallist == nil {
		return false
	}

	for _, item := range colquallist.AllColconstraint() {
		if item.Colconstraintelem() == nil {
			continue
		}

		if item.Colconstraintelem().NULL_P() != nil {
			return true
		}
	}

	return false
}

func equalType(typeA, typeB string) (bool, error) {
	list, err := pgrawparser.Parse(pgrawparser.ParseContext{}, fmt.Sprintf("CREATE TABLE t (a %s)", typeA))
	if err != nil {
		return false, err
	}
	if len(list) != 1 {
		return false, errors.Errorf("failed to compare type %q and %q: more than one statement", typeA, typeB)
	}
	node, ok := list[0].(*ast.CreateTableStmt)
	if !ok {
		return false, errors.Errorf("failed to compare type %q and %q: not CreateTableStmt", typeA, typeB)
	}
	if len(node.ColumnList) != 1 {
		return false, errors.Errorf("failed to compare type %q and %q: more than one column", typeA, typeB)
	}
	column := node.ColumnList[0]
	return column.Type.EquivalentType(typeB), nil
}
