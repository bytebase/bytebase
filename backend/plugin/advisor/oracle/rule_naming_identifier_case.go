// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/oracle/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NamingIdentifierCaseAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE, &NamingIdentifierCaseAdvisor{})
}

// NamingIdentifierCaseAdvisor is the advisor checking for identifier case.
type NamingIdentifierCaseAdvisor struct {
}

// Check checks for identifier case.
func (*NamingIdentifierCaseAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	namingCasePayload := checkCtx.Rule.GetNamingCasePayload()

	rule := NewNamingIdentifierCaseRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase, namingCasePayload.Upper)

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// NamingIdentifierCaseRule is the rule implementation for identifier case.
type NamingIdentifierCaseRule struct {
	BaseRule

	currentDatabase string
	upper           bool
}

// NewNamingIdentifierCaseRule creates a new NamingIdentifierCaseRule.
func NewNamingIdentifierCaseRule(level storepb.Advice_Status, title string, currentDatabase string, upper bool) *NamingIdentifierCaseRule {
	return &NamingIdentifierCaseRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		upper:           upper,
	}
}

// Name returns the rule name.
func (*NamingIdentifierCaseRule) Name() string {
	return "naming.identifier-case"
}

// OnStatement checks identifier case from the omni AST.
func (r *NamingIdentifierCaseRule) OnStatement(node ast.Node) {
	for _, ident := range omniIdentifiers(node) {
		if r.upper {
			if ident.name != strings.ToUpper(ident.name) {
				r.AddAdvice(
					r.level,
					code.NamingCaseMismatch.Int32(),
					fmt.Sprintf("Identifier %q should be upper case", ident.name),
					common.ConvertANTLRLineToPosition(r.locLine(ident.loc)),
				)
			}
		} else if ident.name != strings.ToLower(ident.name) {
			r.AddAdvice(
				r.level,
				code.NamingCaseMismatch.Int32(),
				fmt.Sprintf("Identifier %q should be lower case", ident.name),
				common.ConvertANTLRLineToPosition(r.locLine(ident.loc)),
			)
		}
	}
}

// OnEnter is called when the parser enters a rule context.

// OnExit is called when the parser exits a rule context.
