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
	_ advisor.Advisor = (*NamingIdentifierNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD, &NamingIdentifierNoKeywordAdvisor{})
}

// NamingIdentifierNoKeywordAdvisor is the advisor checking for identifier naming convention without keyword.
type NamingIdentifierNoKeywordAdvisor struct {
}

// Check checks for identifier naming convention without keyword.
func (*NamingIdentifierNoKeywordAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := NewNamingIdentifierNoKeywordRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// NamingIdentifierNoKeywordRule is the rule implementation for identifier naming convention without keyword.
type NamingIdentifierNoKeywordRule struct {
	BaseRule

	currentDatabase string
}

// NewNamingIdentifierNoKeywordRule creates a new NamingIdentifierNoKeywordRule.
func NewNamingIdentifierNoKeywordRule(level storepb.Advice_Status, title string, currentDatabase string) *NamingIdentifierNoKeywordRule {
	return &NamingIdentifierNoKeywordRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
	}
}

// Name returns the rule name.
func (*NamingIdentifierNoKeywordRule) Name() string {
	return "naming.identifier-no-keyword"
}

// OnStatement checks identifiers exposed by the omni AST.
func (r *NamingIdentifierNoKeywordRule) OnStatement(node ast.Node) {
	for _, ident := range omniIdentifiers(node) {
		if plsqlparser.IsOracleKeyword(ident.name) {
			r.AddAdvice(
				r.level,
				code.NameIsKeywordIdentifier.Int32(),
				fmt.Sprintf("Identifier %q is a keyword and should be avoided", ident.name),
				common.ConvertANTLRLineToPosition(r.locLine(ident.loc)),
			)
		}
	}
}

// OnEnter is called when the parser enters a rule context.

// OnExit is called when the parser exits a rule context.
