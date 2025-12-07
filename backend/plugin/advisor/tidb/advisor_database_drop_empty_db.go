package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/pingcap/tidb/pkg/parser/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*DatabaseAllowDropIfEmptyAdvisor)(nil)
	_ ast.Visitor     = (*allowDropEmptyDBChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, &DatabaseAllowDropIfEmptyAdvisor{})
}

// DatabaseAllowDropIfEmptyAdvisor is the advisor checking the MySQLDatabaseAllowDropIfEmpty rule.
type DatabaseAllowDropIfEmptyAdvisor struct {
}

// Check checks for drop table naming convention.
func (*DatabaseAllowDropIfEmptyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, err := getTiDBNodes(checkCtx)

	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &allowDropEmptyDBChecker{
		level:            level,
		title:            checkCtx.Rule.Type.String(),
		originalMetadata: checkCtx.OriginalMetadata,
	}
	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	return checker.adviceList, nil
}

type allowDropEmptyDBChecker struct {
	adviceList       []*storepb.Advice
	level            storepb.Advice_Status
	title            string
	originalMetadata *model.DatabaseMetadata
}

// Enter implements the ast.Visitor interface.
func (v *allowDropEmptyDBChecker) Enter(in ast.Node) (ast.Node, bool) {
	if node, ok := in.(*ast.DropDatabaseStmt); ok {
		if v.originalMetadata.DatabaseName() != node.Name.O {
			v.adviceList = append(v.adviceList, &storepb.Advice{
				Status:        v.level,
				Code:          code.NotCurrentDatabase.Int32(),
				Title:         v.title,
				Content:       fmt.Sprintf("Database `%s` that is trying to be deleted is not the current database `%s`", node.Name, v.originalMetadata.DatabaseName()),
				StartPosition: common.ConvertANTLRLineToPosition(node.OriginTextPosition()),
			})
		} else if !v.originalMetadata.HasNoTable() {
			v.adviceList = append(v.adviceList, &storepb.Advice{
				Status:        v.level,
				Code:          code.DatabaseNotEmpty.Int32(),
				Title:         v.title,
				Content:       fmt.Sprintf("Database `%s` is not allowed to drop if not empty", node.Name),
				StartPosition: common.ConvertANTLRLineToPosition(node.OriginTextPosition()),
			})
		}
	}
	return in, false
}

// Leave implements the ast.Visitor interface.
func (*allowDropEmptyDBChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
