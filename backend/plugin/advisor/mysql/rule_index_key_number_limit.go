package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"
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
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, &IndexKeyNumberLimitAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, &IndexKeyNumberLimitAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT, &IndexKeyNumberLimitAdvisor{})
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

	rule := &indexKeyNumberLimitOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		max: int(numberPayload.Number),
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type indexKeyNumberLimitOmniRule struct {
	OmniBaseRule
	max int
}

func (*indexKeyNumberLimitOmniRule) Name() string {
	return "IndexKeyNumberLimitRule"
}

func (r *indexKeyNumberLimitOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	case *ast.CreateIndexStmt:
		r.checkCreateIndex(n)
	default:
	}
}

func (r *indexKeyNumberLimitOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	for _, constraint := range n.Constraints {
		if constraint == nil {
			continue
		}
		r.handleConstraint(tableName, constraint, r.LocToLine(constraint.Loc))
	}
}

func (r *indexKeyNumberLimitOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		switch cmd.Type {
		case ast.ATAddConstraint, ast.ATAddIndex:
			if cmd.Constraint != nil {
				r.handleConstraint(tableName, cmd.Constraint, r.LocToLine(n.Loc))
			}
		default:
		}
	}
}

func (r *indexKeyNumberLimitOmniRule) handleConstraint(tableName string, constraint *ast.Constraint, line int32) {
	var columnList []string
	switch constraint.Type {
	case ast.ConstrPrimaryKey, ast.ConstrUnique, ast.ConstrIndex:
		columnList = constraint.Columns
	case ast.ConstrForeignKey:
		columnList = constraint.Columns
	default:
		return
	}

	indexName := constraint.Name
	if indexName == "" && constraint.Type == ast.ConstrPrimaryKey {
		indexName = extractPKNameFromText(r.StmtText)
	}
	if r.max > 0 && len(columnList) > r.max {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.IndexKeyNumberExceedsLimit.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("The number of index `%s` in table `%s` should be not greater than %d", indexName, tableName, r.max),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(line)),
		})
	}
}

func (r *indexKeyNumberLimitOmniRule) checkCreateIndex(n *ast.CreateIndexStmt) {
	if n.Table == nil {
		return
	}
	if n.Fulltext || n.Spatial {
		return
	}

	tableName := n.Table.Name
	indexName := n.IndexName
	columnList := omniIndexColumns(n.Columns)
	if r.max > 0 && len(columnList) > r.max {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.IndexKeyNumberExceedsLimit.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("The number of index `%s` in table `%s` should be not greater than %d", indexName, tableName, r.max),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(n.Loc))),
		})
	}
}
