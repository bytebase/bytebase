package tidb

import (
	"context"
	"fmt"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*DatabaseAllowDropIfEmptyAdvisor)(nil)
	_ ast.Visitor     = (*allowDropEmptyDBChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLDatabaseAllowDropIfEmpty, &DatabaseAllowDropIfEmptyAdvisor{})
}

// DatabaseAllowDropIfEmptyAdvisor is the advisor checking the MySQLDatabaseAllowDropIfEmpty rule.
type DatabaseAllowDropIfEmptyAdvisor struct {
}

// Check checks for drop table naming convention.
func (*DatabaseAllowDropIfEmptyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &allowDropEmptyDBChecker{
		level:   level,
		title:   string(checkCtx.Rule.Type),
		catalog: checkCtx.Catalog,
	}
	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	return checker.adviceList, nil
}

type allowDropEmptyDBChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	catalog    *catalog.Finder
}

// Enter implements the ast.Visitor interface.
func (v *allowDropEmptyDBChecker) Enter(in ast.Node) (ast.Node, bool) {
	if node, ok := in.(*ast.DropDatabaseStmt); ok {
		if v.catalog.Origin.DatabaseName() != node.Name.O {
			v.adviceList = append(v.adviceList, &storepb.Advice{
				Status:        v.level,
				Code:          advisor.NotCurrentDatabase.Int32(),
				Title:         v.title,
				Content:       fmt.Sprintf("Database `%s` that is trying to be deleted is not the current database `%s`", node.Name, v.catalog.Origin.DatabaseName()),
				StartPosition: advisor.ConvertANTLRLineToPosition(node.OriginTextPosition()),
			})
		} else if !v.catalog.Origin.HasNoTable() {
			v.adviceList = append(v.adviceList, &storepb.Advice{
				Status:        v.level,
				Code:          advisor.DatabaseNotEmpty.Int32(),
				Title:         v.title,
				Content:       fmt.Sprintf("Database `%s` is not allowed to drop if not empty", node.Name),
				StartPosition: advisor.ConvertANTLRLineToPosition(node.OriginTextPosition()),
			})
		}
	}
	return in, false
}

// Leave implements the ast.Visitor interface.
func (*allowDropEmptyDBChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
