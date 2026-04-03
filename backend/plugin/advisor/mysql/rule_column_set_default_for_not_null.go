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
	_ advisor.Advisor = (*ColumnSetDefaultForNotNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL, &ColumnSetDefaultForNotNullAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL, &ColumnSetDefaultForNotNullAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL, &ColumnSetDefaultForNotNullAdvisor{})
}

// ColumnSetDefaultForNotNullAdvisor is the advisor checking for set default value for not null column.
type ColumnSetDefaultForNotNullAdvisor struct {
}

// Check checks for set default value for not null column.
func (*ColumnSetDefaultForNotNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnSetDefaultForNotNullOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnSetDefaultForNotNullOmniRule struct {
	OmniBaseRule
}

func (*columnSetDefaultForNotNullOmniRule) Name() string {
	return "ColumnSetDefaultForNotNullRule"
}

func (r *columnSetDefaultForNotNullOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnSetDefaultForNotNullOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
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

func (r *columnSetDefaultForNotNullOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
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

func (r *columnSetDefaultForNotNullOmniRule) checkColumn(tableName string, col *ast.ColumnDef) {
	if !omniIsNullable(col) && !omniHasDefault(col) && omniColumnNeedDefaultNotNull(col) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.NotNullColumnWithNoDefault.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Column `%s`.`%s` is NOT NULL but doesn't have DEFAULT", tableName, col.Name),
			StartPosition: &storepb.Position{
				Line:   r.LocToLine(col.Loc),
				Column: 0,
			},
		})
	}
}
