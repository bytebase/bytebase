package mssql

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*TableRequirePkAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, advisor.SchemaRuleTableRequirePK, &TableRequirePkAdvisor{})
}

// TableRequirePkAdvisor is the advisor checking for table require primary key..
type TableRequirePkAdvisor struct {
}

// Check checks for table require primary key..
func (*TableRequirePkAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewTableRequirePkRule(level, string(checkCtx.Rule.Type))

	// Create the generic checker with the rule
	checker := NewGenericChecker([]Rule{rule})

	antlr.ParseTreeWalkerDefault.Walk(checker, tree)

	// Process the final advice after walking
	rule.generateFinalAdvice()

	return checker.GetAdviceList(), nil
}

// TableRequirePkRule is the rule for table require primary key.
type TableRequirePkRule struct {
	BaseRule

	currentNormalizedTableName string
	currentConstraintAction    currentConstraintAction

	// tableHasPrimaryKey is a map from normalized table name to whether the table has primary key.
	tableHasPrimaryKey map[string]bool
	// tableOriginalName is a map from normalized table name to the original table name.
	tableOriginalName map[string]string
	// tableLine is a map from normalized table name to the line number of the table.
	tableLine map[string]int
}

// NewTableRequirePkRule creates a new TableRequirePkRule.
func NewTableRequirePkRule(level storepb.Advice_Status, title string) *TableRequirePkRule {
	return &TableRequirePkRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		currentConstraintAction: currentConstraintActionNone,
		tableHasPrimaryKey:      make(map[string]bool),
		tableOriginalName:       make(map[string]string),
		tableLine:               make(map[string]int),
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
	case "Column_def_table_constraints":
		r.enterColumnDefTableConstraints(ctx.(*parser.Column_def_table_constraintsContext))
	case NodeTypeAlterTable:
		r.enterAlterTable(ctx.(*parser.Alter_tableContext))
	default:
		// Ignore other node types
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *TableRequirePkRule) OnExit(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.exitCreateTable(ctx.(*parser.Create_tableContext))
	case NodeTypeAlterTable:
		r.exitAlterTable(ctx.(*parser.Alter_tableContext))
	default:
	}
	return nil
}

func (r *TableRequirePkRule) enterCreateTable(ctx *parser.Create_tableContext) {
	tableName := ctx.Table_name()
	if tableName == nil {
		return
	}
	normalizedTableName := tsqlparser.NormalizeTSQLTableName(tableName, "" /* fallbackDatabase */, "dbo" /* fallbackSchema */, false /* caseSensitive */)

	r.tableHasPrimaryKey[normalizedTableName] = false
	r.tableOriginalName[normalizedTableName] = tableName.GetText()
	r.tableLine[normalizedTableName] = tableName.GetStart().GetLine()

	r.currentNormalizedTableName = normalizedTableName
	r.currentConstraintAction = currentConstraintActionAdd
}

func (r *TableRequirePkRule) exitCreateTable(*parser.Create_tableContext) {
	r.currentNormalizedTableName = ""
	r.currentConstraintAction = currentConstraintActionNone
}

func (r *TableRequirePkRule) enterColumnDefTableConstraints(ctx *parser.Column_def_table_constraintsContext) {
	if r.currentNormalizedTableName == "" {
		return
	}

	allColumnDefTableConstraints := ctx.AllColumn_def_table_constraint()
	for _, columnDefTableConstraint := range allColumnDefTableConstraints {
		if v := columnDefTableConstraint.Column_definition(); v != nil {
			allColumnDefinitionElements := v.AllColumn_definition_element()
			for _, columnDefinitionElement := range allColumnDefinitionElements {
				if v := columnDefinitionElement.Column_constraint(); v != nil {
					if v.PRIMARY() != nil {
						if r.currentConstraintAction == currentConstraintActionAdd {
							r.tableHasPrimaryKey[r.currentNormalizedTableName] = true
						}
						return
					}
				}
			}
		} else if v := columnDefTableConstraint.Table_constraint(); v != nil {
			if v.PRIMARY() != nil {
				if r.currentConstraintAction == currentConstraintActionAdd {
					r.tableHasPrimaryKey[r.currentNormalizedTableName] = true
				}
				return
			}
		}
	}
}

func (r *TableRequirePkRule) enterAlterTable(ctx *parser.Alter_tableContext) {
	tableName := ctx.Table_name(0)
	if tableName == nil {
		return
	}
	normalizedTableName := tsqlparser.NormalizeTSQLTableName(tableName, "" /* fallbackDatabase */, "dbo" /* fallbackSchema */, false /* caseSensitive */)
	if ctx.ADD() != nil && ctx.Column_def_table_constraints() != nil {
		r.currentNormalizedTableName = normalizedTableName
		r.currentConstraintAction = currentConstraintActionAdd
	} else if ctx.DROP() != nil && ctx.CONSTRAINT() != nil && ctx.GetConstraint() != nil {
		r.currentNormalizedTableName = normalizedTableName
		r.currentConstraintAction = currentConstraintActionDrop
	}
}

func (r *TableRequirePkRule) exitAlterTable(*parser.Alter_tableContext) {
	r.currentNormalizedTableName = ""
	r.currentConstraintAction = currentConstraintActionNone
}

func (r *TableRequirePkRule) generateFinalAdvice() {
	for tableName, hasPK := range r.tableHasPrimaryKey {
		if !hasPK {
			r.AddAdvice(&storepb.Advice{
				Status:        r.level,
				Code:          code.TableNoPK.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("Table %s requires PRIMARY KEY.", r.tableOriginalName[tableName]),
				StartPosition: common.ConvertANTLRLineToPosition(r.tableLine[tableName]),
			})
		}
	}
}
