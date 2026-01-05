package mssql

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/tsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*TableNoForeignKeyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, &TableNoForeignKeyAdvisor{})
}

// TableNoForeignKeyAdvisor is the advisor checking for table disallow foreign key..
type TableNoForeignKeyAdvisor struct {
}

// Check checks for table disallow foreign key..
func (*TableNoForeignKeyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	// Create the rule
	rule := NewTableNoForeignKeyRule(level, checkCtx.Rule.Type.String())

	// Create the generic checker with the rule
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
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	// Process the final advice after walking
	rule.generateFinalAdvice()

	return checker.GetAdviceList(), nil
}

// TableNoForeignKeyRule is the rule for table disallow foreign key.
type TableNoForeignKeyRule struct {
	BaseRule

	// currentNormalizedTableName is the normalized table name of the current table.
	currentNormalizedTableName string
	// currentConstraintAction is the current constraint action.
	currentConstraintAction currentConstraintAction
	// tableHasForeignKey is true if the current table has foreign key.
	tableHasForeignKey map[string]bool
	// tableOriginalName is the original table name of the current table.
	tableOriginalName map[string]string
	// tableLine is the line number of the current table.
	tableLine map[string]int
}

// NewTableNoForeignKeyRule creates a new TableNoForeignKeyRule.
func NewTableNoForeignKeyRule(level storepb.Advice_Status, title string) *TableNoForeignKeyRule {
	return &TableNoForeignKeyRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		currentConstraintAction: currentConstraintActionNone,
		tableHasForeignKey:      make(map[string]bool),
		tableOriginalName:       make(map[string]string),
		tableLine:               make(map[string]int),
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
	case "Column_def_table_constraints":
		r.enterColumnDefTableConstraints(ctx.(*parser.Column_def_table_constraintsContext))
	case NodeTypeAlterTable:
		r.enterAlterTable(ctx.(*parser.Alter_tableContext))
	default:
	}
	return nil
}

// OnExit is called when exiting a parse tree node.
func (r *TableNoForeignKeyRule) OnExit(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case NodeTypeCreateTable:
		r.exitCreateTable(ctx.(*parser.Create_tableContext))
	case NodeTypeAlterTable:
		r.exitAlterTable(ctx.(*parser.Alter_tableContext))
	default:
	}
	return nil
}

func (r *TableNoForeignKeyRule) enterCreateTable(ctx *parser.Create_tableContext) {
	tableName := ctx.Table_name()
	if tableName == nil {
		return
	}
	normalizedTableName := tsqlparser.NormalizeTSQLTableName(tableName, "" /* fallbackDatabase */, "dbo" /* fallbackSchema */, false /* caseSensitive */)

	r.tableHasForeignKey[normalizedTableName] = false
	r.tableOriginalName[normalizedTableName] = tableName.GetText()
	// Store absolute line number (with baseLine offset) so generateFinalAdvice can use it directly
	r.tableLine[normalizedTableName] = tableName.GetStart().GetLine() + r.baseLine

	r.currentNormalizedTableName = normalizedTableName
	r.currentConstraintAction = currentConstraintActionAdd
}

func (r *TableNoForeignKeyRule) exitCreateTable(*parser.Create_tableContext) {
	r.currentNormalizedTableName = ""
	r.currentConstraintAction = currentConstraintActionNone
}

func (r *TableNoForeignKeyRule) enterColumnDefTableConstraints(ctx *parser.Column_def_table_constraintsContext) {
	if r.currentNormalizedTableName == "" {
		return
	}

	allColumnDefTableConstraints := ctx.AllColumn_def_table_constraint()
	for _, columnDefTableConstraint := range allColumnDefTableConstraints {
		if v := columnDefTableConstraint.Column_definition(); v != nil {
			allColumnDefinitionElements := v.AllColumn_definition_element()
			for _, columnDefinitionElement := range allColumnDefinitionElements {
				if v := columnDefinitionElement.Column_constraint(); v != nil {
					if v.Foreign_key_options() != nil {
						if r.currentConstraintAction == currentConstraintActionAdd {
							r.tableHasForeignKey[r.currentNormalizedTableName] = true
							// Store absolute line number (with baseLine offset)
							r.tableLine[r.currentNormalizedTableName] = v.Foreign_key_options().GetStart().GetLine() + r.baseLine
						}
						return
					}
				}
			}
		} else if v := columnDefTableConstraint.Table_constraint(); v != nil {
			if v.Foreign_key_options() != nil {
				if r.currentConstraintAction == currentConstraintActionAdd {
					r.tableHasForeignKey[r.currentNormalizedTableName] = true
					// Store absolute line number (with baseLine offset)
					r.tableLine[r.currentNormalizedTableName] = v.Foreign_key_options().GetStart().GetLine() + r.baseLine
				}
				return
			}
		}
	}
}

func (r *TableNoForeignKeyRule) enterAlterTable(ctx *parser.Alter_tableContext) {
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

func (r *TableNoForeignKeyRule) exitAlterTable(*parser.Alter_tableContext) {
	r.currentNormalizedTableName = ""
	r.currentConstraintAction = currentConstraintActionNone
}

func (r *TableNoForeignKeyRule) generateFinalAdvice() {
	for tableName, hasForeignKey := range r.tableHasForeignKey {
		if hasForeignKey {
			// Directly append to adviceList instead of using AddAdvice,
			// because tableLine already contains the absolute line number
			r.adviceList = append(r.adviceList, &storepb.Advice{
				Status:        r.level,
				Code:          code.TableHasFK.Int32(),
				Title:         r.title,
				Content:       fmt.Sprintf("FOREIGN KEY is not allowed in the table %s.", r.tableOriginalName[tableName]),
				StartPosition: common.ConvertANTLRLineToPosition(r.tableLine[tableName]),
			})
		}
	}
}
