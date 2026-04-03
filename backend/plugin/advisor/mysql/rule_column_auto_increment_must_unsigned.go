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
	_ advisor.Advisor = (*ColumnAutoIncrementMustUnsignedAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED, &ColumnAutoIncrementMustUnsignedAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED, &ColumnAutoIncrementMustUnsignedAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED, &ColumnAutoIncrementMustUnsignedAdvisor{})
}

// ColumnAutoIncrementMustUnsignedAdvisor is the advisor checking for unsigned auto-increment column.
type ColumnAutoIncrementMustUnsignedAdvisor struct {
}

// Check checks for unsigned auto-increment column.
func (*ColumnAutoIncrementMustUnsignedAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnAutoIncrementMustUnsignedOmniRule{
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

type columnAutoIncrementMustUnsignedOmniRule struct {
	OmniBaseRule
}

func (*columnAutoIncrementMustUnsignedOmniRule) Name() string {
	return "ColumnAutoIncrementMustUnsignedRule"
}

func (r *columnAutoIncrementMustUnsignedOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnAutoIncrementMustUnsignedOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}
	for _, col := range n.Columns {
		r.checkColumn(tableName, col)
	}
}

func (r *columnAutoIncrementMustUnsignedOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
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

func (r *columnAutoIncrementMustUnsignedOmniRule) checkColumn(tableName string, col *ast.ColumnDef) {
	if !omniIsAutoIncrement(col) {
		return
	}
	if col.TypeName == nil || (!col.TypeName.Unsigned && !col.TypeName.Zerofill) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.AutoIncrementColumnSigned.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Auto-increment column `%s`.`%s` is not UNSIGNED type", tableName, col.Name),
			StartPosition: &storepb.Position{
				Line: r.LocToLine(col.Loc),
			},
		})
	}
}
