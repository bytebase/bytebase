// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/snowflake"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*TableRequirePkAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, advisor.SchemaRuleTableRequirePK, &TableRequirePkAdvisor{})
}

// TableRequirePkAdvisor is the advisor checking for table require primary key.
type TableRequirePkAdvisor struct {
}

// Check checks for table require primary key.
func (*TableRequirePkAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewTableRequirePkRule(level, string(checkCtx.Rule.Type))
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	return checker.GetAdviceList(), nil
}

// TableRequirePkRule checks for table require primary key.
type TableRequirePkRule struct {
	BaseRule
	currentConstraintAction currentConstraintAction
	// currentNormalizedTableName is the current table name, and it is normalized.
	// It should be set then entering create_table, alter_table and so on,
	// and should be reset then exiting them.
	currentNormalizedTableName string

	// tableHasPrimaryKey is a map of normalized table name to whether the table has primary key.
	tableHasPrimaryKey map[string]bool
	// tableOriginalName is a map of normalized table name to original table name.
	// The key of the tableOriginalName is the superset of the key of the tableHasPrimaryKey.
	tableOriginalName map[string]string
	// tableLine is a map of normalized table name to the line number of the table.
	// The key of the tableLine is the superset of the key of the tableHasPrimaryKey.
	tableLine map[string]int
}

// NewTableRequirePkRule creates a new TableRequirePkRule.
func NewTableRequirePkRule(level storepb.Advice_Status, title string) *TableRequirePkRule {
	return &TableRequirePkRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		currentConstraintAction:    currentConstraintActionNone,
		currentNormalizedTableName: "",
		tableHasPrimaryKey:         make(map[string]bool),
		tableOriginalName:          make(map[string]string),
		tableLine:                  make(map[string]int),
	}
}

// Name returns the rule name.
func (*TableRequirePkRule) Name() string {
	return "TableRequirePkRule"
}

// OnEnter is called when entering a parse tree node.
func (r *TableRequirePkRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.enterCreateTable(ctx.(*parser.Create_tableContext))
	case NodeTypeDropTable:
		r.enterDropTable(ctx.(*parser.Drop_tableContext))
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
func (r *TableRequirePkRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.exitCreateTable()
	case NodeTypeAlterTable:
		r.exitAlterTable()
	default:
		// Other node types
	}
	return nil
}

// GetAdviceList returns the accumulated advice list, generating final advice for tables without PK.
func (r *TableRequirePkRule) GetAdviceList() []*storepb.Advice {
	for tableName, has := range r.tableHasPrimaryKey {
		if !has {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.TableNoPK.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Table %s requires PRIMARY KEY.", r.tableOriginalName[tableName]),
				StartPosition: common.ConvertANTLRLineToPosition(r.tableLine[tableName]),
			})
		}
	}
	return r.adviceList
}

func (r *TableRequirePkRule) enterCreateTable(ctx *parser.Create_tableContext) {
	originalTableName := ctx.Object_name()
	normalizedTableName := snowsqlparser.NormalizeSnowSQLObjectName(originalTableName, "", "PUBLIC")

	r.tableHasPrimaryKey[normalizedTableName] = false
	r.tableOriginalName[normalizedTableName] = originalTableName.GetText()
	r.tableLine[normalizedTableName] = ctx.GetStart().GetLine()
	r.currentNormalizedTableName = normalizedTableName
	r.currentConstraintAction = currentConstraintActionAdd
}

func (r *TableRequirePkRule) enterDropTable(ctx *parser.Drop_tableContext) {
	originalTableName := ctx.Object_name()
	normalizedTableName := snowsqlparser.NormalizeSnowSQLObjectName(originalTableName, "", "PUBLIC")

	delete(r.tableHasPrimaryKey, normalizedTableName)
	delete(r.tableOriginalName, normalizedTableName)
	delete(r.tableLine, normalizedTableName)
}

func (r *TableRequirePkRule) exitCreateTable() {
	r.currentNormalizedTableName = ""
	r.currentConstraintAction = currentConstraintActionNone
}

func (r *TableRequirePkRule) enterInlineConstraint(ctx *parser.Inline_constraintContext) {
	if ctx.PRIMARY() == nil || r.currentNormalizedTableName == "" {
		return
	}
	r.tableHasPrimaryKey[r.currentNormalizedTableName] = true
}

func (r *TableRequirePkRule) enterOutOfLineConstraint(ctx *parser.Out_of_line_constraintContext) {
	if ctx.PRIMARY() == nil || r.currentNormalizedTableName == "" || r.currentConstraintAction == currentConstraintActionNone {
		return
	}
	switch r.currentConstraintAction {
	case currentConstraintActionAdd:
		r.tableHasPrimaryKey[r.currentNormalizedTableName] = true
	case currentConstraintActionDrop:
		r.tableHasPrimaryKey[r.currentNormalizedTableName] = false
		r.tableLine[r.currentNormalizedTableName] = ctx.GetStart().GetLine()
	default:
		// No action for other constraint actions
	}
}

func (r *TableRequirePkRule) enterConstraintAction(ctx *parser.Constraint_actionContext) {
	if r.currentNormalizedTableName == "" {
		return
	}
	if ctx.DROP() != nil && ctx.PRIMARY() != nil {
		if _, ok := r.tableHasPrimaryKey[r.currentNormalizedTableName]; ok {
			r.tableHasPrimaryKey[r.currentNormalizedTableName] = false
			r.tableLine[r.currentNormalizedTableName] = ctx.GetStart().GetLine()
		}
		return
	}
	if ctx.ADD() != nil {
		r.currentConstraintAction = currentConstraintActionAdd
		return
	}
}

func (r *TableRequirePkRule) enterAlterTable(ctx *parser.Alter_tableContext) {
	if ctx.Constraint_action() == nil {
		return
	}
	originalTableName := ctx.Object_name(0)
	normalizedTableName := snowsqlparser.NormalizeSnowSQLObjectName(originalTableName, "", "PUBLIC")

	r.currentNormalizedTableName = normalizedTableName
	r.tableOriginalName[normalizedTableName] = originalTableName.GetText()
}

func (r *TableRequirePkRule) exitAlterTable() {
	r.currentNormalizedTableName = ""
	r.currentConstraintAction = currentConstraintActionNone
}
