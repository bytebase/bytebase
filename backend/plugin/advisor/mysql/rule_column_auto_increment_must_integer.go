package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnAutoIncrementMustIntegerAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER, &ColumnAutoIncrementMustIntegerAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER, &ColumnAutoIncrementMustIntegerAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER, &ColumnAutoIncrementMustIntegerAdvisor{})
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

	rule := &columnAutoIncrementMustIntegerOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		rule.SetStatement(stmt.BaseLine(), stmt.Text)
		rule.OnStatement(node)
	}

	return rule.GetAdviceList(), nil
}

type columnAutoIncrementMustIntegerOmniRule struct {
	OmniBaseRule
}

func (*columnAutoIncrementMustIntegerOmniRule) Name() string {
	return "ColumnAutoIncrementMustIntegerRule"
}

func (r *columnAutoIncrementMustIntegerOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnAutoIncrementMustIntegerOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}
	for _, col := range n.Columns {
		r.checkColumn(tableName, col)
	}
}

func (r *columnAutoIncrementMustIntegerOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}
	for _, cmd := range n.Commands {
		for _, col := range omniGetColumnsFromCmd(cmd) {
			r.checkColumn(tableName, col)
		}
	}
}

func (r *columnAutoIncrementMustIntegerOmniRule) checkColumn(tableName string, col *ast.ColumnDef) {
	if !omniIsAutoIncrement(col) {
		return
	}
	if !omniIsIntegerType(col.TypeName) {
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
