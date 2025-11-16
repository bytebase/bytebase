package pg

import (
	"context"
	"fmt"
	"regexp"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
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

	rule := &namingColumnConventionRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		format:    format,
		maxLength: maxLength,
	}

	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type namingColumnConventionRule struct {
	BaseRule

	format    *regexp.Regexp
	maxLength int
}

func (*namingColumnConventionRule) Name() string {
	return "naming_column_convention"
}

func (r *namingColumnConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		r.handleCreatestmt(ctx)
	case "Altertablestmt":
		r.handleAltertablestmt(ctx)
	case "Renamestmt":
		r.handleRenamestmt(ctx)
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*namingColumnConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *namingColumnConventionRule) handleCreatestmt(ctx antlr.ParserRuleContext) {
	createCtx, ok := ctx.(*parser.CreatestmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Get table name
	qualifiedNames := createCtx.AllQualified_name()
	if len(qualifiedNames) == 0 {
		return
	}
	tableName := extractTableName(qualifiedNames[0])

	// Get OptTableElementList which contains column definitions
	if createCtx.Opttableelementlist() == nil || createCtx.Opttableelementlist().Tableelementlist() == nil {
		return
	}

	// Iterate through all table elements
	allElements := createCtx.Opttableelementlist().Tableelementlist().AllTableelement()
	for _, elem := range allElements {
		// Check if this is a column definition
		if elem.ColumnDef() != nil {
			colDef := elem.ColumnDef()
			if colDef.Colid() != nil {
				columnName := pg.NormalizePostgreSQLColid(colDef.Colid())
				r.checkColumnName(tableName, columnName, colDef.GetStart().GetLine())
			}
		}
	}
}

func (r *namingColumnConventionRule) handleAltertablestmt(ctx antlr.ParserRuleContext) {
	alterCtx, ok := ctx.(*parser.AltertablestmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Get table name
	if alterCtx.Relation_expr() == nil || alterCtx.Relation_expr().Qualified_name() == nil {
		return
	}
	tableName := extractTableName(alterCtx.Relation_expr().Qualified_name())

	// Get ALTER TABLE commands
	if alterCtx.Alter_table_cmds() == nil {
		return
	}

	allCmds := alterCtx.Alter_table_cmds().AllAlter_table_cmd()
	for _, cmd := range allCmds {
		// Check for ADD COLUMN
		if cmd.ADD_P() != nil && cmd.ColumnDef() != nil {
			colDef := cmd.ColumnDef()
			if colDef.Colid() != nil {
				columnName := pg.NormalizePostgreSQLColid(colDef.Colid())
				r.checkColumnName(tableName, columnName, colDef.GetStart().GetLine())
			}
		}
	}
}

func (r *namingColumnConventionRule) handleRenamestmt(ctx antlr.ParserRuleContext) {
	renameCtx, ok := ctx.(*parser.RenamestmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is RENAME COLUMN
	if renameCtx.Opt_column() == nil || renameCtx.Opt_column().COLUMN() == nil {
		return
	}

	// Get table name
	var tableName string
	if renameCtx.Relation_expr() != nil && renameCtx.Relation_expr().Qualified_name() != nil {
		tableName = extractTableName(renameCtx.Relation_expr().Qualified_name())
	}
	if tableName == "" {
		return
	}

	// Get new column name
	allNames := renameCtx.AllName()
	if len(allNames) < 2 {
		return
	}

	newColumnName := pg.NormalizePostgreSQLName(allNames[1])
	r.checkColumnName(tableName, newColumnName, renameCtx.GetStart().GetLine())
}

func (r *namingColumnConventionRule) checkColumnName(tableName, columnName string, line int) {
	if !r.format.MatchString(columnName) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NamingColumnConventionMismatch.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("\"%s\".\"%s\" mismatches column naming convention, naming format should be %q", tableName, columnName, r.format),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}

	if r.maxLength > 0 && len(columnName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NamingColumnConventionMismatch.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("\"%s\".\"%s\" mismatches column naming convention, its length should be within %d characters", tableName, columnName, r.maxLength),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}
