package pg

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*IndexKeyNumberLimitAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, &IndexKeyNumberLimitAdvisor{})
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

	rule := &indexKeyNumberLimitRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		max: int(numberPayload.Number),
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type indexKeyNumberLimitRule struct {
	OmniBaseRule

	max int
}

func (*indexKeyNumberLimitRule) Name() string {
	return "index_key_number_limit"
}

func (r *indexKeyNumberLimitRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.IndexStmt:
		r.handleIndexStmt(n)
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	default:
	}
}

func (r *indexKeyNumberLimitRule) handleIndexStmt(n *ast.IndexStmt) {
	keyCount := len(omniIndexColumns(n))
	if r.max > 0 && keyCount > r.max {
		indexName := n.Idxname
		tableName := omniTableName(n.Relation)
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.IndexKeyNumberExceedsLimit.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("The number of keys of index %q in table %q should be not greater than %d", indexName, tableName, r.max),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}

func (r *indexKeyNumberLimitRule) handleCreateStmt(n *ast.CreateStmt) {
	tableName := omniTableName(n.Relation)
	_, constraints := omniTableElements(n)
	for _, c := range constraints {
		r.checkConstraint(c, tableName)
	}
}

func (r *indexKeyNumberLimitRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Relation)
	for _, cmd := range omniAlterTableCmds(n) {
		if cmd.Subtype == int(ast.AT_AddConstraint) {
			if c, ok := cmd.Def.(*ast.Constraint); ok {
				r.checkConstraint(c, tableName)
			}
		}
	}
}

func (r *indexKeyNumberLimitRule) checkConstraint(c *ast.Constraint, tableName string) {
	var keyCount int
	switch c.Contype {
	case ast.CONSTR_PRIMARY, ast.CONSTR_UNIQUE:
		keyCount = len(omniConstraintColumns(c))
	case ast.CONSTR_FOREIGN:
		keyCount = len(omniListStrings(c.FkAttrs))
	default:
		return
	}
	if r.max > 0 && keyCount > r.max {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.IndexKeyNumberExceedsLimit.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("The number of keys of index %q in table %q should be not greater than %d", c.Conname, tableName, r.max),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}
