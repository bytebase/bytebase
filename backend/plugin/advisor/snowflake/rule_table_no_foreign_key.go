// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*TableNoForeignKeyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, advisor.SchemaRuleTableNoFK, &TableNoForeignKeyAdvisor{})
}

// TableNoForeignKeyAdvisor is the advisor checking for table disallow foreign key.
type TableNoForeignKeyAdvisor struct {
}

// Check checks for table disallow foreign key.
func (*TableNoForeignKeyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewTableNoForeignKeyRule(level, string(checkCtx.Rule.Type))
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// TableNoForeignKeyRule checks for table disallow foreign key.
type TableNoForeignKeyRule struct {
	BaseRule
	currentConstraintAction currentConstraintAction
	// currentNormalizedTableName is the current table name, and it is normalized.
	// It should be set then entering create_table, alter_table and so on,
	// and should be reset then exiting them.
	currentNormalizedTableName string

	// tableForeignKeyTimes is a map of normalized table name to the times of FOREIGN KEY.
	tableForeignKeyTimes map[string]int
	// tableOriginalName is a map of normalized table name to original table name.
	// The key of the tableOriginalName is the superset of the key of the tableHasForeignKey.
	tableOriginalName map[string]string
	// tableLine is a map of normalized table name to the line number of the table.
	// The key of the tableLine is the superset of the key of the tableHasForeignKey.
	tableLine map[string]int
}

// NewTableNoForeignKeyRule creates a new TableNoForeignKeyRule.
func NewTableNoForeignKeyRule(level storepb.Advice_Status, title string) *TableNoForeignKeyRule {
	return &TableNoForeignKeyRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		currentConstraintAction:    currentConstraintActionNone,
		currentNormalizedTableName: "",
		tableForeignKeyTimes:       make(map[string]int),
		tableOriginalName:          make(map[string]string),
		tableLine:                  make(map[string]int),
	}
}

// Name returns the rule name.
func (*TableNoForeignKeyRule) Name() string {
	return "TableNoForeignKeyRule"
}

// OnEnter is called when entering a parse tree node.
func (r *TableNoForeignKeyRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.enterCreateTable(ctx.(*parser.Create_tableContext))
	case NodeTypeInlineConstraint:
		r.enterInlineConstraint(ctx.(*parser.Inline_constraintContext))
	case NodeTypeOutOfLineConstraint:
		r.enterOutOfLineConstraint(ctx.(*parser.Out_of_line_constraintContext))
	case "Constraint_action":
		r.enterConstraintAction(ctx.(*parser.Constraint_actionContext))
	case NodeTypeAlterTable:
		r.enterAlterTable(ctx.(*parser.Alter_tableContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *TableNoForeignKeyRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.exitCreateTable()
	case NodeTypeAlterTable:
		r.exitAlterTable()
	default:
		// Ignore other node types
	}
	return nil
}

// GetAdviceList returns the accumulated advice list, generating final advice for tables with FK.
func (r *TableNoForeignKeyRule) GetAdviceList() []*storepb.Advice {
	for tableName, times := range r.tableForeignKeyTimes {
		if times > 0 {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.TableHasFK.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("FOREIGN KEY is not allowed in the table %s.", r.tableOriginalName[tableName]),
				StartPosition: common.ConvertANTLRLineToPosition(r.tableLine[tableName]),
			})
		}
	}
	return r.adviceList
}

func (r *TableNoForeignKeyRule) enterCreateTable(ctx *parser.Create_tableContext) {
	originalTableName := ctx.Object_name()
	normalizedTableName := snowsqlparser.NormalizeSnowSQLObjectName(originalTableName, "", "PUBLIC")

	r.tableForeignKeyTimes[normalizedTableName] = 0
	r.tableOriginalName[normalizedTableName] = originalTableName.GetText()
	r.tableLine[normalizedTableName] = ctx.GetStart().GetLine()
	r.currentNormalizedTableName = normalizedTableName
	r.currentConstraintAction = currentConstraintActionAdd
}

func (r *TableNoForeignKeyRule) exitCreateTable() {
	r.currentNormalizedTableName = ""
	r.currentConstraintAction = currentConstraintActionNone
}

func (r *TableNoForeignKeyRule) enterInlineConstraint(ctx *parser.Inline_constraintContext) {
	if ctx.REFERENCES() == nil || r.currentNormalizedTableName == "" {
		return
	}
	r.tableForeignKeyTimes[r.currentNormalizedTableName]++
}

func (r *TableNoForeignKeyRule) enterOutOfLineConstraint(ctx *parser.Out_of_line_constraintContext) {
	if ctx.REFERENCES() == nil || r.currentNormalizedTableName == "" || r.currentConstraintAction == currentConstraintActionNone {
		return
	}
	switch r.currentConstraintAction {
	case currentConstraintActionAdd:
		r.tableForeignKeyTimes[r.currentNormalizedTableName]++
		r.tableLine[r.currentNormalizedTableName] = ctx.GetStart().GetLine()
	case currentConstraintActionDrop:
		if times, ok := r.tableForeignKeyTimes[r.currentNormalizedTableName]; ok && times > 0 {
			r.tableForeignKeyTimes[r.currentNormalizedTableName]--
		}
	default:
		// Other constraint actions
	}
}

func (r *TableNoForeignKeyRule) enterConstraintAction(ctx *parser.Constraint_actionContext) {
	if r.currentNormalizedTableName == "" {
		return
	}
	if ctx.DROP() != nil && ctx.FOREIGN() != nil {
		if times, ok := r.tableForeignKeyTimes[r.currentNormalizedTableName]; ok && times > 0 {
			r.tableForeignKeyTimes[r.currentNormalizedTableName]--
		}
		return
	}
	if ctx.ADD() != nil {
		r.currentConstraintAction = currentConstraintActionAdd
		return
	}
}

func (r *TableNoForeignKeyRule) enterAlterTable(ctx *parser.Alter_tableContext) {
	if ctx.Constraint_action() == nil {
		return
	}
	originalTableName := ctx.Object_name(0)
	normalizedTableName := snowsqlparser.NormalizeSnowSQLObjectName(originalTableName, "", "PUBLIC")

	r.currentNormalizedTableName = normalizedTableName
	r.tableOriginalName[normalizedTableName] = originalTableName.GetText()
}

func (r *TableNoForeignKeyRule) exitAlterTable() {
	r.currentNormalizedTableName = ""
	r.currentConstraintAction = currentConstraintActionNone
}
