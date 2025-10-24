package pgantlr

import (
	"context"
	"fmt"
	"slices"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
)

var (
	_ advisor.Advisor = (*ColumnRequireDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleColumnRequireDefault, &ColumnRequireDefaultAdvisor{})
}

// ColumnRequireDefaultAdvisor is the advisor checking for column default requirement.
type ColumnRequireDefaultAdvisor struct {
}

// Check checks for column default requirement.
func (*ColumnRequireDefaultAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &columnRequireDefaultChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		catalog:                      checkCtx.Catalog,
		columnSet:                    make(map[string]columnRequireDefaultData),
	}

	if checker.catalog != nil && checker.catalog.Final.Usable() {
		antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)
	}

	return checker.generateAdvice(), nil
}

type columnRequireDefaultChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	columnSet  map[string]columnRequireDefaultData
	catalog    *catalog.Finder
}

type columnRequireDefaultData struct {
	schema string
	table  string
	name   string
	line   int
}

func (c *columnRequireDefaultChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	qualifiedNames := ctx.AllQualified_name()
	if len(qualifiedNames) == 0 {
		return
	}

	tableName := c.extractTableName(qualifiedNames[0])
	if tableName == "" {
		return
	}

	// Track all columns
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			if elem.ColumnDef() != nil && elem.ColumnDef().Colid() != nil {
				columnName := elem.ColumnDef().Colid().GetText()
				c.addColumn("public", tableName, columnName, elem.GetStart().GetLine())
			}
		}
	}
}

func (c *columnRequireDefaultChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	tableName := c.extractTableName(ctx.Relation_expr().Qualified_name())
	if tableName == "" {
		return
	}

	// Check ALTER TABLE ADD COLUMN
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// ADD COLUMN
			if cmd.ADD_P() != nil && cmd.ColumnDef() != nil && cmd.ColumnDef().Colid() != nil {
				columnName := cmd.ColumnDef().Colid().GetText()
				c.addColumn("public", tableName, columnName, ctx.GetStart().GetLine())
			}
		}
	}
}

func (c *columnRequireDefaultChecker) extractTableName(qualifiedNameCtx parser.IQualified_nameContext) string {
	if qualifiedNameCtx == nil {
		return ""
	}

	text := qualifiedNameCtx.GetText()
	parts := splitIdentifier(text)
	if len(parts) == 0 {
		return ""
	}

	// Return the last part (table name)
	return parts[len(parts)-1]
}

func (c *columnRequireDefaultChecker) addColumn(schema string, table string, column string, line int) {
	if schema == "" {
		schema = "public"
	}

	c.columnSet[fmt.Sprintf("%s.%s.%s", schema, table, column)] = columnRequireDefaultData{
		schema: schema,
		table:  table,
		name:   column,
		line:   line,
	}
}

func (c *columnRequireDefaultChecker) generateAdvice() []*storepb.Advice {
	var columnList []columnRequireDefaultData
	for _, column := range c.columnSet {
		columnList = append(columnList, column)
	}
	slices.SortFunc(columnList, func(i, j columnRequireDefaultData) int {
		if i.line < j.line {
			return -1
		}
		if i.line > j.line {
			return 1
		}
		return 0
	})

	for _, column := range columnList {
		columnInfo := c.catalog.Final.FindColumn(&catalog.ColumnFind{
			SchemaName: column.schema,
			TableName:  column.table,
			ColumnName: column.name,
		})
		if columnInfo != nil && !columnInfo.HasDefault() {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  c.level,
				Code:    advisor.NoDefault.Int32(),
				Title:   c.title,
				Content: fmt.Sprintf("Column %q.%q in schema %q doesn't have DEFAULT", column.table, column.name, column.schema),
				StartPosition: &storepb.Position{
					Line:   int32(column.line),
					Column: 0,
				},
			})
		}
	}

	return c.adviceList
}
