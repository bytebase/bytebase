package oracle

import (
	"fmt"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	plsqlparser "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterGetDesignSchema(storepb.Engine_ORACLE, GetDesignSchema)
}

func GetDesignSchema(defaultSchema, baselineSchema string, to *storepb.DatabaseSchemaMetadata) (string, error) {
	toState := convertToDatabaseState(to)
	tree, tokens, err := plsql.ParsePLSQL(baselineSchema)
	if err != nil {
		return "", err
	}

	generator := &designSchemaGenerator{
		to:            toState,
		defaultSchema: defaultSchema,
	}
	antlr.ParseTreeWalkerDefault.Walk(generator, tree)
	if generator.err != nil {
		return "", generator.err
	}

	for _, schema := range to.Schemas {
		schemaState, ok := toState.schemas[schema.Name]
		if !ok {
			continue
		}
		for _, table := range schema.Tables {
			tableState, ok := schemaState.tables[table.Name]
			if !ok {
				continue
			}
			if tableState.deleted {
				// Add indexes.
				for _, index := range table.Indexes {
					if index.Primary || index.Unique {
						continue
					}
					if indexState := tableState.indexes[index.Name]; indexState != nil {
						var buf strings.Builder
						if err := indexState.toOutlineString(schemaState.name, tableState.name, &buf); err != nil {
							return "", err
						}
						generator.actions = append(generator.actions, plsql.NewAddIndexAction(schema.Name, table.Name, buf.String()))
					}
				}
				continue
			}
			buf := &strings.Builder{}
			if err := tableState.toString(schema.Name, buf); err != nil {
				return "", err
			}
			generator.actions = append(generator.actions, plsql.NewAddTableAction(schema.Name, table.Name, buf.String()))
			for _, index := range table.Indexes {
				indexState := tableState.indexes[index.Name]
				if indexState == nil {
					continue
				}
				if index.Primary || index.Unique {
					continue
				}
				buf := &strings.Builder{}
				if err := indexState.toOutlineString(schemaState.name, tableState.name, buf); err != nil {
					return "", err
				}
				generator.actions = append(generator.actions, plsql.NewAddIndexAction(schema.Name, table.Name, buf.String()))
			}
		}
	}
	manipulator := plsql.NewStringsManipulator(tree, tokens)
	return manipulator.Manipulate(generator.actions...)
}

type designSchemaGenerator struct {
	*plsqlparser.BasePlSqlParserListener

	to            *databaseState
	currentTable  *tableState
	defaultSchema string
	err           error

	actions []base.StringsManipulatorAction
}

func (g *designSchemaGenerator) EnterCreate_index(ctx *plsqlparser.Create_indexContext) {
	if g.err != nil {
		return
	}

	indexDefinition := ctx.Table_index_clause()
	if indexDefinition == nil {
		return
	}

	_, tableName := plsql.NormalizeTableViewName("", indexDefinition.Tableview_name())
	schemaName, indexName := plsql.NormalizeIndexName(ctx.Index_name())

	if schemaName == "" {
		schemaName = g.defaultSchema
	}

	schema, ok := g.to.schemas[schemaName]
	if !ok {
		g.actions = append(g.actions, plsql.NewDropIndexAction(schemaName, tableName, indexName))
		return
	}

	table, ok := schema.tables[tableName]
	if !ok || table.deleted {
		g.actions = append(g.actions, plsql.NewDropIndexAction(schemaName, tableName, indexName))
		return
	}

	index, ok := table.indexes[indexName]
	if !ok {
		g.actions = append(g.actions, plsql.NewDropIndexAction(schemaName, tableName, indexName))
		return
	}

	delete(table.indexes, indexName)
	var buf strings.Builder
	if err := index.toOutlineString(schemaName, tableName, &buf); err != nil {
		g.err = err
		return
	}
	if index.primary {
		g.actions = append(g.actions, plsql.NewDropIndexAction(schemaName, tableName, indexName))
		g.actions = append(g.actions, plsql.NewAddTableConstraintAction(schemaName, tableName, base.TableConstraintTypePrimaryKey, buf.String()))
		return
	}

	isUnique := ctx.UNIQUE() != nil
	if index.unique != isUnique {
		g.actions = append(g.actions, plsql.NewModifyIndexAction(schemaName, tableName, indexName, buf.String()))
		return
	}

	var keys []string
	for _, expr := range indexDefinition.AllIndex_expr_option() {
		if expr.Index_expr().Column_name() != nil {
			_, _, columnName := plsql.NormalizeColumnName(expr.Index_expr().Column_name())
			keys = append(keys, fmt.Sprintf("\"%s\"", columnName))
		} else if expr.Index_expr().Expression() != nil {
			keys = append(keys, ctx.GetParser().GetTokenStream().GetTextFromRuleContext(expr.Index_expr().Expression()))
		}
	}

	if !equalIndexKeys(keys, index.keys) {
		g.actions = append(g.actions, plsql.NewModifyIndexAction(schemaName, tableName, indexName, buf.String()))
		return
	}
}

func equalIndexKeys(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, key := range a {
		if key != b[i] {
			return false
		}
	}
	return true
}

func (g *designSchemaGenerator) EnterCreate_table(ctx *plsqlparser.Create_tableContext) {
	if g.err != nil {
		return
	}

	schemaName := plsql.NormalizeSchemaName(ctx.Schema_name())
	tableName := plsql.NormalizeTableName(ctx.Table_name())

	if schemaName == "" {
		schemaName = g.defaultSchema
	}

	if g.defaultSchema == "" {
		g.defaultSchema = schemaName
	}

	tableDefine := ctx.Relational_table()
	if tableDefine == nil {
		return
	}

	schema, ok := g.to.schemas[schemaName]
	if !ok {
		g.actions = append(g.actions, plsql.NewDropTableAction(schemaName, tableName))
		return
	}

	table, ok := schema.tables[tableName]
	if !ok {
		g.actions = append(g.actions, plsql.NewDropTableAction(schemaName, tableName))
		return
	}
	g.currentTable = table

	table.deleted = true
}

func (g *designSchemaGenerator) ExitCreate_table(_ *plsqlparser.Create_tableContext) {
	if g.err != nil {
		return
	}

	if g.currentTable == nil {
		return
	}

	defer func() {
		g.currentTable = nil
	}()

	var columnList []*columnState
	for _, column := range g.currentTable.columns {
		columnList = append(columnList, column)
	}
	sort.Slice(columnList, func(i, j int) bool {
		return columnList[i].id < columnList[j].id
	})
	for _, column := range columnList {
		var buf strings.Builder
		if err := column.toString(&buf); err != nil {
			g.err = err
			return
		}
		g.actions = append(g.actions, plsql.NewAddColumnAction(g.defaultSchema, g.currentTable.name, buf.String()))
	}

	var indexes []*indexState
	for _, index := range g.currentTable.indexes {
		indexes = append(indexes, index)
	}
	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i].id < indexes[j].id
	})
	for _, index := range indexes {
		if !index.primary && !index.unique {
			continue
		}
		var buf strings.Builder
		if err := index.toInlineString(&buf); err != nil {
			g.err = err
			return
		}
		if index.primary {
			g.actions = append(g.actions, plsql.NewAddTableConstraintAction(g.defaultSchema, g.currentTable.name, base.TableConstraintTypePrimaryKey, buf.String()))
		} else if index.unique {
			g.actions = append(g.actions, plsql.NewAddTableConstraintAction(g.defaultSchema, g.currentTable.name, base.TableConstraintTypeUnique, buf.String()))
		}
	}
}

func (g *designSchemaGenerator) EnterOut_of_line_constraint(ctx *plsqlparser.Out_of_line_constraintContext) {
	if g.err != nil {
		return
	}

	if g.currentTable == nil {
		return
	}

	_, constraintName := plsql.NormalizeConstraintName(ctx.Constraint_name())
	if constraintName == "" {
		return
	}

	indexState, ok := g.currentTable.indexes[constraintName]
	if !ok {
		g.actions = append(g.actions, plsql.NewDropTableConstraintAction(g.defaultSchema, g.currentTable.name, constraintName))
		return
	}

	delete(g.currentTable.indexes, constraintName)
	var buf strings.Builder
	if err := indexState.toInlineString(&buf); err != nil {
		g.err = err
		return
	}
	if ctx.PRIMARY() != nil {
		if !indexState.primary || !equalKeys(ctx, indexState) {
			g.actions = append(g.actions, plsql.NewModifyTableConstraintAction(g.defaultSchema, g.currentTable.name, base.TableConstraintTypePrimaryKey, constraintName, buf.String()))
		}
	} else if ctx.UNIQUE() != nil {
		if !indexState.unique || indexState.primary || !equalKeys(ctx, indexState) {
			g.actions = append(g.actions, plsql.NewModifyTableConstraintAction(g.defaultSchema, g.currentTable.name, base.TableConstraintTypeUnique, constraintName, buf.String()))
		}
	}
}

func equalKeys(ctx *plsqlparser.Out_of_line_constraintContext, state *indexState) bool {
	if len(ctx.AllColumn_name()) != len(state.keys) {
		return false
	}

	for i, column := range ctx.AllColumn_name() {
		_, _, columnName := plsql.NormalizeColumnName(column)
		if columnName != state.keys[i] {
			return false
		}
	}

	return true
}

func (g *designSchemaGenerator) EnterColumn_definition(ctx *plsqlparser.Column_definitionContext) {
	if g.err != nil {
		return
	}

	if g.currentTable == nil {
		return
	}

	_, _, columnName := plsql.NormalizeColumnName(ctx.Column_name())
	stateColumn, ok := g.currentTable.columns[columnName]
	if !ok {
		g.actions = append(g.actions, plsql.NewDropColumnAction(g.defaultSchema, g.currentTable.name, columnName))
		return
	}

	delete(g.currentTable.columns, columnName)

	// Compare column types.
	if !isEqualColumnType(ctx, stateColumn) {
		g.actions = append(g.actions, plsql.NewModifyColumnTypeAction(g.defaultSchema, g.currentTable.name, columnName, stateColumn.tp))
	}

	// Compare attributes.
	nullable := true
	for _, item := range ctx.AllInline_constraint() {
		if item.NOT() != nil {
			nullable = false
		}
	}

	if nullable != stateColumn.nullable {
		if stateColumn.nullable {
			g.actions = append(g.actions, plsql.NewModifyColumnOptionAction(g.defaultSchema, g.currentTable.name, columnName, base.ColumnOptionTypeNotNull, "NOT NULL"))
		} else {
			g.actions = append(g.actions, plsql.NewDropColumnOptionAction(g.defaultSchema, g.currentTable.name, columnName, base.ColumnOptionTypeNotNull))
		}
	}

	defaultValue := (defaultValue)(nil)
	if ctx.DEFAULT() != nil && ctx.Expression() != nil {
		defaultValue = &defaultValueExpression{
			value: ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Expression()),
		}
	}
	if !isEqualDefaultValue(defaultValue, stateColumn.defaultValue) {
		switch {
		case defaultValue == nil && stateColumn.defaultValue != nil:
			g.actions = append(g.actions, plsql.NewAddColumnOptionAction(g.defaultSchema, g.currentTable.name, columnName, base.ColumnOptionTypeDefault, stateColumn.defaultValue.toString()))
		case defaultValue != nil && stateColumn.defaultValue == nil:
			g.actions = append(g.actions, plsql.NewDropColumnOptionAction(g.defaultSchema, g.currentTable.name, columnName, base.ColumnOptionTypeDefault))
		case defaultValue != nil && stateColumn.defaultValue != nil:
			g.actions = append(g.actions, plsql.NewModifyColumnOptionAction(g.defaultSchema, g.currentTable.name, columnName, base.ColumnOptionTypeDefault, defaultValue.toString()))
		}
	}
}

func isEqualDefaultValue(a, b defaultValue) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.toString() == b.toString()
}

func isEqualColumnType(ctx *plsqlparser.Column_definitionContext, stateColumn *columnState) bool {
	if ctx.Datatype() != nil {
		return getDataTypePlainText(ctx.Datatype()) == stateColumn.tp
	} else if ctx.Regular_id() != nil {
		return ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Regular_id()) == stateColumn.tp
	}

	return false
}
