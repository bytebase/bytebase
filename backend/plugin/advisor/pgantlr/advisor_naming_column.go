package pgantlr

import (
	"context"
	"fmt"
	"regexp"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*NamingColumnConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleColumnNaming, &NamingColumnConventionAdvisor{})
}

// NamingColumnConventionAdvisor is the advisor checking for column naming convention.
type NamingColumnConventionAdvisor struct {
}

// Check checks for column naming convention.
func (*NamingColumnConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, maxLength, err := advisor.UnmarshalNamingRulePayloadAsRegexp(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &namingColumnConventionChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		format:                       format,
		maxLength:                    maxLength,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type namingColumnConventionChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	format     *regexp.Regexp
	maxLength  int
}

// EnterCreatestmt handles CREATE TABLE statements
func (c *namingColumnConventionChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Get table name
	qualifiedNames := ctx.AllQualified_name()
	if len(qualifiedNames) == 0 {
		return
	}
	tableName := extractTableName(qualifiedNames[0])

	// Get OptTableElementList which contains column definitions
	if ctx.Opttableelementlist() == nil || ctx.Opttableelementlist().Tableelementlist() == nil {
		return
	}

	// Iterate through all table elements
	allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
	for _, elem := range allElements {
		// Check if this is a column definition
		if elem.ColumnDef() != nil {
			colDef := elem.ColumnDef()
			if colDef.Colid() != nil {
				columnName := pg.NormalizePostgreSQLColid(colDef.Colid())
				c.checkColumnName(tableName, columnName, elem.GetStart().GetLine())
			}
		}
	}
}

// EnterAltertablestmt handles ALTER TABLE ADD COLUMN statements
func (c *namingColumnConventionChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Get table name
	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}
	tableName := extractTableName(ctx.Relation_expr().Qualified_name())

	// Get ALTER TABLE commands
	if ctx.Alter_table_cmds() == nil {
		return
	}

	allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
	for _, cmd := range allCmds {
		// Check for ADD COLUMN
		if cmd.ADD_P() != nil && cmd.ColumnDef() != nil {
			colDef := cmd.ColumnDef()
			if colDef.Colid() != nil {
				columnName := pg.NormalizePostgreSQLColid(colDef.Colid())
				c.checkColumnName(tableName, columnName, cmd.GetStart().GetLine())
			}
		}
	}
}

// EnterRenamestmt handles RENAME COLUMN statements
func (c *namingColumnConventionChecker) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is RENAME COLUMN
	if ctx.Opt_column() == nil || ctx.Opt_column().COLUMN() == nil {
		return
	}

	// Get table name
	var tableName string
	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		tableName = extractTableName(ctx.Relation_expr().Qualified_name())
	}
	if tableName == "" {
		return
	}

	// Get new column name
	allNames := ctx.AllName()
	if len(allNames) < 2 {
		return
	}

	newColumnName := pg.NormalizePostgreSQLName(allNames[1])
	c.checkColumnName(tableName, newColumnName, ctx.GetStart().GetLine())
}

// checkColumnName validates a column name against the format and max length rules
func (c *namingColumnConventionChecker) checkColumnName(tableName, columnName string, line int) {
	if !c.format.MatchString(columnName) {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.NamingColumnConventionMismatch.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("\"%s\".\"%s\" mismatches column naming convention, naming format should be %q", tableName, columnName, c.format),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}

	if c.maxLength > 0 && len(columnName) > c.maxLength {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.NamingColumnConventionMismatch.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("\"%s\".\"%s\" mismatches column naming convention, its length should be within %d characters", tableName, columnName, c.maxLength),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}
