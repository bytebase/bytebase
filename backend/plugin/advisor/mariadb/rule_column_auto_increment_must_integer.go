package mariadb

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mariadb/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mariadbparser "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
)

var (
	_ advisor.Advisor = (*ColumnAutoIncrementMustIntegerAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER, &ColumnAutoIncrementMustIntegerAdvisor{})
}

// ColumnAutoIncrementMustIntegerAdvisor is the advisor checking for auto-increment column type.
type ColumnAutoIncrementMustIntegerAdvisor struct {
}

// Check checks for auto-increment column type.
func (*ColumnAutoIncrementMustIntegerAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnAutoIncrementMustIntegerRule{
		BaseRule: BaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mariadbparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		rule.SetStatement(stmt.BaseLine(), stmt.Text)
		rule.OnStatement(node)
	}

	return rule.GetAdviceList(), nil
}

type columnAutoIncrementMustIntegerRule struct {
	BaseRule
}

func (r *columnAutoIncrementMustIntegerRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnAutoIncrementMustIntegerRule) checkCreateTable(n *ast.CreateTableStmt) {
	tableName := tableName(n.Table)
	if tableName == "" {
		return
	}
	for _, col := range n.Columns {
		r.checkColumn(tableName, col)
	}
}

func (r *columnAutoIncrementMustIntegerRule) checkAlterTable(n *ast.AlterTableStmt) {
	tableName := tableName(n.Table)
	if tableName == "" {
		return
	}
	for _, cmd := range n.Commands {
		for _, col := range getColumnsFromCmd(cmd) {
			r.checkColumn(tableName, col)
		}
	}
}

func (r *columnAutoIncrementMustIntegerRule) checkColumn(tableName string, col *ast.ColumnDef) {
	if !isAutoIncrement(col) {
		return
	}
	if !isIntegerType(col.TypeName) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.AutoIncrementColumnNotInteger.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Auto-increment column `%s`.`%s` requires integer type", tableName, col.Name),
			StartPosition: &storepb.Position{
				Line: r.LocToLine(col.Loc),
			},
		})
	}
}
