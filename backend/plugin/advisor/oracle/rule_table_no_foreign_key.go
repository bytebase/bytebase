// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/omni/oracle/ast"
	parser "github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableNoForeignKeyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, &TableNoForeignKeyAdvisor{})
}

// TableNoForeignKeyAdvisor is the advisor checking for table disallow foreign key.
type TableNoForeignKeyAdvisor struct {
}

// Check checks for table disallow foreign key.
func (*TableNoForeignKeyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewTableNoForeignKeyRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// TableNoForeignKeyRule is the rule implementation for table disallow foreign key.
type TableNoForeignKeyRule struct {
	BaseRule

	currentDatabase string
	tableName       string
	tableWithFK     map[string]bool
	tableLine       map[string]int
}

// NewTableNoForeignKeyRule creates a new TableNoForeignKeyRule.
func NewTableNoForeignKeyRule(level storepb.Advice_Status, title string, currentDatabase string) *TableNoForeignKeyRule {
	return &TableNoForeignKeyRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		tableWithFK:     make(map[string]bool),
		tableLine:       make(map[string]int),
	}
}

// Name returns the rule name.
func (*TableNoForeignKeyRule) Name() string {
	return "table.no-foreign-key"
}

// OnStatement checks foreign keys from omni CREATE/ALTER TABLE nodes.
func (r *TableNoForeignKeyRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		tableName := omniObjectName(n.Name, r.currentDatabase)
		if r.createTableHasFK(n) {
			r.tableWithFK[tableName] = true
			r.tableLine[tableName] = r.locLine(n.Loc)
		}
	case *ast.AlterTableStmt:
		tableName := omniObjectName(n.Name, r.currentDatabase)
		for _, cmd := range omniAlterTableCmds(n) {
			if cmd.Constraint != nil && cmd.Constraint.Type == ast.CONSTRAINT_FOREIGN {
				r.tableWithFK[tableName] = true
				r.tableLine[tableName] = r.locLine(cmd.Constraint.Loc)
			}
		}
	default:
	}
}

func (*TableNoForeignKeyRule) createTableHasFK(stmt *ast.CreateTableStmt) bool {
	for _, col := range omniColumnDefs(stmt.Columns) {
		if omniColumnHasConstraint(col, ast.CONSTRAINT_FOREIGN) {
			return true
		}
	}
	for _, c := range omniTableConstraints(stmt.Constraints) {
		if c.Type == ast.CONSTRAINT_FOREIGN {
			return true
		}
	}
	return false
}

// OnEnter is called when the parser enters a rule context.
func (r *TableNoForeignKeyRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Create_table":
		r.handleCreateTable(ctx.(*parser.Create_tableContext))
	case "References_clause":
		r.handleReferencesClause(ctx.(*parser.References_clauseContext))
	case "Alter_table":
		r.handleAlterTable(ctx.(*parser.Alter_tableContext))
	default:
	}
	return nil
}

// OnExit is called when the parser exits a rule context.
func (r *TableNoForeignKeyRule) OnExit(_ antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Create_table":
		r.tableName = ""
	case "Alter_table":
		r.tableName = ""
	default:
	}
	return nil
}

// GetAdviceList returns the advice list.
func (r *TableNoForeignKeyRule) GetAdviceList() ([]*storepb.Advice, error) {
	for tableName, hasFK := range r.tableWithFK {
		if hasFK {
			r.AddAdvice(
				r.level,
				code.TableHasFK.Int32(),
				fmt.Sprintf("Foreign key is not allowed in the table %s.", normalizeIdentifierName(tableName)),
				common.ConvertANTLRLineToPosition(r.tableLine[tableName]),
			)
		}
	}
	return r.BaseRule.GetAdviceList()
}

func (r *TableNoForeignKeyRule) handleCreateTable(ctx *parser.Create_tableContext) {
	schemaName := r.currentDatabase
	if ctx.Schema_name() != nil {
		schemaName = normalizeIdentifier(ctx.Schema_name(), r.currentDatabase)
	}

	r.tableName = fmt.Sprintf("%s.%s", schemaName, normalizeIdentifier(ctx.Table_name(), r.currentDatabase))
}

func (r *TableNoForeignKeyRule) handleReferencesClause(ctx *parser.References_clauseContext) {
	r.tableWithFK[r.tableName] = true
	r.tableLine[r.tableName] = r.baseLine + ctx.GetStop().GetLine()
}

func (r *TableNoForeignKeyRule) handleAlterTable(ctx *parser.Alter_tableContext) {
	r.tableName = normalizeIdentifier(ctx.Tableview_name(), r.currentDatabase)
}
