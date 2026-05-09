package tidb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*DatabaseAllowDropIfEmptyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE, &DatabaseAllowDropIfEmptyAdvisor{})
}

// DatabaseAllowDropIfEmptyAdvisor is the advisor checking the MySQLDatabaseAllowDropIfEmpty rule.
type DatabaseAllowDropIfEmptyAdvisor struct {
}

// Check checks for drop table naming convention.
func (*DatabaseAllowDropIfEmptyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
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

	for _, ostmt := range stmts {
		checker.checkStmt(ostmt)
	}

	return checker.adviceList, nil
}

type allowDropEmptyDBChecker struct {
	adviceList       []*storepb.Advice
	level            storepb.Advice_Status
	title            string
	originalMetadata *model.DatabaseMetadata
}

func (c *allowDropEmptyDBChecker) checkStmt(ostmt OmniStmt) {
	node, ok := ostmt.Node.(*ast.DropDatabaseStmt)
	if !ok {
		return
	}
	line := ostmt.AbsoluteLine(node.Loc.Start)
	if c.originalMetadata.DatabaseName() != node.Name {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        c.level,
			Code:          code.NotCurrentDatabase.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("Database `%s` that is trying to be deleted is not the current database `%s`", node.Name, c.originalMetadata.DatabaseName()),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
	} else if !c.originalMetadata.HasNoTable() {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        c.level,
			Code:          code.DatabaseNotEmpty.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("Database `%s` is not allowed to drop if not empty", node.Name),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
	}
}
