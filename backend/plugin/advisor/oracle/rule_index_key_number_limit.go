// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/oracle/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*IndexKeyNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, &IndexKeyNumberLimitAdvisor{})
}

// IndexKeyNumberLimitAdvisor is the advisor checking for index key number limit.
type IndexKeyNumberLimitAdvisor struct {
}

// Check checks for index key number limit.
func (*IndexKeyNumberLimitAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	if int(numberPayload.Number) <= 0 {
		return nil, nil
	}

	rule := NewIndexKeyNumberLimitRule(level, checkCtx.Rule.Type.String(), checkCtx.CurrentDatabase, int(numberPayload.Number))

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})
}

// IndexKeyNumberLimitRule is the rule implementation for index key number limit.
type IndexKeyNumberLimitRule struct {
	BaseRule

	currentDatabase string
	max             int
}

// NewIndexKeyNumberLimitRule creates a new IndexKeyNumberLimitRule.
func NewIndexKeyNumberLimitRule(level storepb.Advice_Status, title string, currentDatabase string, maxKeys int) *IndexKeyNumberLimitRule {
	return &IndexKeyNumberLimitRule{
		BaseRule:        NewBaseRule(level, title, 0),
		currentDatabase: currentDatabase,
		max:             maxKeys,
	}
}

// Name returns the rule name.
func (*IndexKeyNumberLimitRule) Name() string {
	return "index.key-number-limit"
}

// OnStatement checks CREATE INDEX and constraint column counts in the omni AST.
func (r *IndexKeyNumberLimitRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateIndexStmt:
		if n.Columns != nil && len(n.Columns.Items) > r.max {
			r.AddAdvice(
				r.level,
				code.IndexKeyNumberExceedsLimit.Int32(),
				fmt.Sprintf("Index key number should be less than or equal to %d", r.max),
				common.ConvertANTLRLineToPosition(r.locLine(n.Loc)),
			)
		}
	case *ast.CreateTableStmt:
		for _, c := range omniTableConstraints(n.Constraints) {
			r.checkConstraint(c)
		}
	case *ast.AlterTableStmt:
		for _, cmd := range omniAlterTableCmds(n) {
			if cmd.Constraint != nil {
				r.checkConstraint(cmd.Constraint)
			}
		}
	default:
	}
}

func (r *IndexKeyNumberLimitRule) checkConstraint(c *ast.TableConstraint) {
	if c == nil || (c.Type != ast.CONSTRAINT_PRIMARY && c.Type != ast.CONSTRAINT_UNIQUE) {
		return
	}
	if c.Columns != nil && len(c.Columns.Items) > r.max {
		r.AddAdvice(
			r.level,
			code.IndexKeyNumberExceedsLimit.Int32(),
			fmt.Sprintf("Index key number should be less than or equal to %d", r.max),
			common.ConvertANTLRLineToPosition(r.locLine(c.Loc)),
		)
	}
}

// OnEnter is called when the parser enters a rule context.

// OnExit is called when the parser exits a rule context.
