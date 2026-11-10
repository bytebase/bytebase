package mysql

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/mysql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

var (
	_ advisor.Advisor = (*IndexTypeAllowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_TYPE_ALLOW_LIST, &IndexTypeAllowListAdvisor{})
}

// IndexTypeAllowListAdvisor is the advisor checking for index types.
type IndexTypeAllowListAdvisor struct {
}

func (*IndexTypeAllowListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()

	rule := &indexTypeAllowListRule{
		BaseRule: BaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		allowList: stringArrayPayload.List,
	}

	return RunRules(checkCtx.ParsedStatements, []Rule{rule}), nil
}

type indexTypeAllowListRule struct {
	BaseRule
	allowList []string
}

func (*indexTypeAllowListRule) Name() string {
	return "IndexTypeAllowListRule"
}

func (r *indexTypeAllowListRule) OnStatement(node ast.Node) {
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

func (r *indexTypeAllowListRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	for _, constraint := range n.Constraints {
		if constraint == nil {
			continue
		}
		r.handleConstraint(constraint, r.LocToLine(constraint.Loc))
	}
}

func (r *indexTypeAllowListRule) checkAlterTable(n *ast.AlterTableStmt) {
	if n.Table == nil {
		return
	}
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		switch cmd.Type {
		case ast.ATAddConstraint, ast.ATAddIndex:
			if cmd.Constraint != nil {
				r.handleConstraint(cmd.Constraint, r.LocToLine(n.Loc))
			}
		default:
		}
	}
}

func (r *indexTypeAllowListRule) handleConstraint(constraint *ast.Constraint, line int32) {
	switch constraint.Type {
	case ast.ConstrPrimaryKey, ast.ConstrUnique, ast.ConstrIndex, ast.ConstrFulltextIndex, ast.ConstrSpatialIndex:
	default:
		return
	}

	indexType := "BTREE"
	if constraint.IndexType != "" {
		indexType = constraint.IndexType
	} else {
		switch constraint.Type {
		case ast.ConstrFulltextIndex:
			indexType = "FULLTEXT"
		case ast.ConstrSpatialIndex:
			indexType = "SPATIAL"
		default:
		}
	}
	r.validateIndexType(indexType, line)
}

func (r *indexTypeAllowListRule) checkCreateIndex(n *ast.CreateIndexStmt) {
	if n.Table == nil {
		return
	}

	indexType := "BTREE"
	if n.IndexType != "" {
		indexType = n.IndexType
	} else if n.Fulltext {
		indexType = "FULLTEXT"
	} else if n.Spatial {
		indexType = "SPATIAL"
	}
	r.validateIndexType(indexType, r.LocToLine(n.Loc))
}

func (r *indexTypeAllowListRule) validateIndexType(indexType string, line int32) {
	if slices.Contains(r.allowList, indexType) {
		return
	}

	r.AddAdviceAbsolute(&storepb.Advice{
		Status:        r.Level,
		Code:          code.IndexTypeNotAllowed.Int32(),
		Title:         r.Title,
		Content:       fmt.Sprintf("Index type `%s` is not allowed", indexType),
		StartPosition: base.ConvertANTLRLineToPosition(r.BaseLine + int(line)),
	})
}
