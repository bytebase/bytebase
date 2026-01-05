package pg

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"

	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*ColumnRequirementAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_REQUIRED, &ColumnRequirementAdvisor{})
}

type columnSet map[string]bool

// ColumnRequirementAdvisor is the advisor checking for column requirement.
type ColumnRequirementAdvisor struct {
}

// Check checks for the column requirement.
func (*ColumnRequirementAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for column required rule")
	}

	// Convert to map for O(1) lookup
	requiredColumnsMap := make(columnSet)
	for _, col := range stringArrayPayload.List {
		requiredColumnsMap[col] = true
	}

	rule := &columnRequirementRule{
		BaseRule: BaseRule{
			level: level,
			title: checkCtx.Rule.Type.String(),
		},
		requiredColumnsMap: requiredColumnsMap,
	}

	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

type columnRequirementRule struct {
	BaseRule

	requiredColumnsMap columnSet // Map of all required columns (from config)
	requiredColumns    columnSet // Temp map for checking CREATE TABLE
}

func (*columnRequirementRule) Name() string {
	return "column_requirement"
}

func (r *columnRequirementRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
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

func (*columnRequirementRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

func (r *columnRequirementRule) handleCreatestmt(ctx antlr.ParserRuleContext) {
	createCtx, ok := ctx.(*parser.CreatestmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Initialize required columns for this statement (copy from config map)
	r.requiredColumns = make(columnSet)
	for column := range r.requiredColumnsMap {
		r.requiredColumns[column] = true
	}

	qualifiedNames := createCtx.AllQualified_name()
	if len(qualifiedNames) == 0 {
		return
	}

	tableName := extractTableName(qualifiedNames[0])
	if tableName == "" {
		return
	}

	// Mark columns as present
	if createCtx.Opttableelementlist() != nil && createCtx.Opttableelementlist().Tableelementlist() != nil {
		allElements := createCtx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			if elem.ColumnDef() != nil && elem.ColumnDef().Colid() != nil {
				columnName := pg.NormalizePostgreSQLColid(elem.ColumnDef().Colid())
				delete(r.requiredColumns, columnName)
			}
		}
	}

	// Check if any required columns are missing
	if len(r.requiredColumns) > 0 {
		var missingColumns []string
		for column := range r.requiredColumns {
			missingColumns = append(missingColumns, column)
		}
		slices.Sort(missingColumns)

		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NoRequiredColumn.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("Table %q requires columns: %s", tableName, strings.Join(missingColumns, ", ")),
			StartPosition: &storepb.Position{
				Line:   int32(createCtx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

func (r *columnRequirementRule) handleAltertablestmt(ctx antlr.ParserRuleContext) {
	alterCtx, ok := ctx.(*parser.AltertablestmtContext)
	if !ok {
		return
	}
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if alterCtx.Relation_expr() == nil || alterCtx.Relation_expr().Qualified_name() == nil {
		return
	}

	tableName := extractTableName(alterCtx.Relation_expr().Qualified_name())
	if tableName == "" {
		return
	}

	// Check ALTER TABLE commands
	if alterCtx.Alter_table_cmds() != nil {
		allCmds := alterCtx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// DROP COLUMN (note: COLUMN keyword is optional in PostgreSQL)
			if cmd.DROP() != nil {
				allColids := cmd.AllColid()
				if len(allColids) > 0 {
					columnName := pg.NormalizePostgreSQLColid(allColids[0])
					// Check if this is a required column (O(1) lookup)
					if r.requiredColumnsMap[columnName] {
						r.AddAdvice(&storepb.Advice{
							Status:  r.level,
							Code:    code.NoRequiredColumn.Int32(),
							Title:   r.title,
							Content: fmt.Sprintf("Table %q requires columns: %s", tableName, columnName),
							StartPosition: &storepb.Position{
								Line:   int32(alterCtx.GetStart().GetLine()),
								Column: 0,
							},
						})
					}
				}
			}
		}
	}
}

func (r *columnRequirementRule) handleRenamestmt(ctx antlr.ParserRuleContext) {
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

	// Get old and new column names
	allNames := renameCtx.AllName()
	if len(allNames) < 2 {
		return
	}

	oldName := pg.NormalizePostgreSQLName(allNames[0])
	newName := pg.NormalizePostgreSQLName(allNames[1])

	// Check if renaming away from a required column name (O(1) lookup)
	if r.requiredColumnsMap[oldName] && oldName != newName {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NoRequiredColumn.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf("Table %q requires columns: %s", tableName, oldName),
			StartPosition: &storepb.Position{
				Line:   int32(renameCtx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}
