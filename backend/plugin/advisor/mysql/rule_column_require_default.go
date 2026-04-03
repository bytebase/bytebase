package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnRequireDefaultAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, &ColumnRequireDefaultAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, &ColumnRequireDefaultAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT, &ColumnRequireDefaultAdvisor{})
}

// ColumnRequireDefaultAdvisor is the advisor checking for column default requirement.
type ColumnRequireDefaultAdvisor struct {
}

// Check checks for column default requirement.
func (*ColumnRequireDefaultAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnRequireDefaultOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnRequireDefaultOmniRule struct {
	OmniBaseRule
}

func (*columnRequireDefaultOmniRule) Name() string {
	return "ColumnRequireDefaultRule"
}

func (r *columnRequireDefaultOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnRequireDefaultOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}

	pkCols := omniPKColumnNames(n.Columns, n.Constraints)

	for _, col := range n.Columns {
		if pkCols[col.Name] {
			continue
		}
		r.checkColumn(tableName, col)
	}
}

func (r *columnRequireDefaultOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}

	for _, cmd := range n.Commands {
		switch cmd.Type {
		case ast.ATAddColumn:
			if cmd.Column != nil {
				r.checkColumn(tableName, cmd.Column)
			}
			for _, col := range cmd.Columns {
				r.checkColumn(tableName, col)
			}
		case ast.ATModifyColumn, ast.ATChangeColumn:
			if cmd.Column != nil {
				r.checkColumn(tableName, cmd.Column)
			}
		default:
		}
	}
}

func (r *columnRequireDefaultOmniRule) checkColumn(tableName string, col *ast.ColumnDef) {
	if !omniHasDefault(col) && !omniIsDefaultExemptType(col) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.NoDefault.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Column `%s`.`%s` doesn't have DEFAULT.", tableName, col.Name),
			StartPosition: &storepb.Position{
				Line:   r.LocToLine(col.Loc),
				Column: 0,
			},
		})
	}
}
