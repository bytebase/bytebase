// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oracle/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

var (
	_ advisor.Advisor = (*NamingTableNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD, &NamingTableNoKeywordAdvisor{})
}

// NamingTableNoKeywordAdvisor is the advisor checking for table naming convention without keyword.
type NamingTableNoKeywordAdvisor struct {
}

// Check checks for table naming convention without keyword.
func (*NamingTableNoKeywordAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewNamingTableNoKeywordRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// NamingTableNoKeywordRule is the rule implementation for table naming convention without keyword.
type NamingTableNoKeywordRule struct {
	BaseRule

	currentDatabase string
}

// NewNamingTableNoKeywordRule creates a new NamingTableNoKeywordRule.
func NewNamingTableNoKeywordRule(level storepb.Advice_Status, title string, currentDatabase string) *NamingTableNoKeywordRule {
	return &NamingTableNoKeywordRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
	}
}

// Name returns the rule name.
func (*NamingTableNoKeywordRule) Name() string {
	return "naming.table-no-keyword"
}

// OnStatement checks table names in omni DDL statements.
func (r *NamingTableNoKeywordRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkTableName(omniLastObjectName(n.Name), n.Loc)
	case *ast.AlterTableStmt:
		for _, cmd := range omniAlterTableCmds(n) {
			if cmd.Action == ast.AT_RENAME && cmd.NewName != "" {
				r.checkTableName(cmd.NewName, cmd.Loc)
			}
		}
	default:
	}
}

func (r *NamingTableNoKeywordRule) checkTableName(tableName string, loc ast.Loc) {
	if tableName != "" && plsqlparser.IsOracleKeyword(tableName) {
		r.AddAdvice(
			r.level,
			code.NameIsKeywordIdentifier.Int32(),
			fmt.Sprintf("Table name %q is a keyword identifier and should be avoided.", tableName),
			common.ConvertANTLRLineToPosition(r.locLine(loc)),
		)
	}
}

// OnEnter is called when the parser enters a rule context.

// OnExit is called when the parser exits a rule context.
