package pg

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*ColumnRequirementAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleRequiredColumn, &ColumnRequirementAdvisor{})
}

type columnSet map[string]bool

// ColumnRequirementAdvisor is the advisor checking for column requirement.
type ColumnRequirementAdvisor struct {
}

// Check checks for the column requirement.
func (*ColumnRequirementAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	columnList, err := advisor.UnmarshalRequiredColumnList(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	// Convert to map for O(1) lookup
	requiredColumnsMap := make(columnSet)
	for _, col := range columnList {
		requiredColumnsMap[col] = true
	}

	checker := &columnRequirementChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		requiredColumnsMap:           requiredColumnsMap,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type columnRequirementChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList         []*storepb.Advice
	level              storepb.Advice_Status
	title              string
	requiredColumnsMap columnSet // Map of all required columns (from config)
	requiredColumns    columnSet // Temp map for checking CREATE TABLE
}

func (c *columnRequirementChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Initialize required columns for this statement (copy from config map)
	c.requiredColumns = make(columnSet)
	for column := range c.requiredColumnsMap {
		c.requiredColumns[column] = true
	}

	qualifiedNames := ctx.AllQualified_name()
	if len(qualifiedNames) == 0 {
		return
	}

	tableName := extractTableName(qualifiedNames[0])
	if tableName == "" {
		return
	}

	// Mark columns as present
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			if elem.ColumnDef() != nil && elem.ColumnDef().Colid() != nil {
				columnName := pg.NormalizePostgreSQLColid(elem.ColumnDef().Colid())
				delete(c.requiredColumns, columnName)
			}
		}
	}

	// Check if any required columns are missing
	if len(c.requiredColumns) > 0 {
		var missingColumns []string
		for column := range c.requiredColumns {
			missingColumns = append(missingColumns, column)
		}
		slices.Sort(missingColumns)

		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.NoRequiredColumn.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("Table %q requires columns: %s", tableName, strings.Join(missingColumns, ", ")),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

func (c *columnRequirementChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	tableName := extractTableName(ctx.Relation_expr().Qualified_name())
	if tableName == "" {
		return
	}

	// Check ALTER TABLE commands
	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// DROP COLUMN (note: COLUMN keyword is optional in PostgreSQL)
			if cmd.DROP() != nil {
				allColids := cmd.AllColid()
				if len(allColids) > 0 {
					columnName := pg.NormalizePostgreSQLColid(allColids[0])
					// Check if this is a required column (O(1) lookup)
					if c.requiredColumnsMap[columnName] {
						c.adviceList = append(c.adviceList, &storepb.Advice{
							Status:  c.level,
							Code:    advisor.NoRequiredColumn.Int32(),
							Title:   c.title,
							Content: fmt.Sprintf("Table %q requires columns: %s", tableName, columnName),
							StartPosition: &storepb.Position{
								Line:   int32(ctx.GetStart().GetLine()),
								Column: 0,
							},
						})
					}
				}
			}
		}
	}
}

func (c *columnRequirementChecker) EnterRenamestmt(ctx *parser.RenamestmtContext) {
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

	// Get old and new column names
	allNames := ctx.AllName()
	if len(allNames) < 2 {
		return
	}

	oldName := pg.NormalizePostgreSQLName(allNames[0])
	newName := pg.NormalizePostgreSQLName(allNames[1])

	// Check if renaming away from a required column name (O(1) lookup)
	if c.requiredColumnsMap[oldName] && oldName != newName {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.NoRequiredColumn.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf("Table %q requires columns: %s", tableName, oldName),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}
