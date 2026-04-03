package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*ColumnDisallowChangingTypeAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE, &ColumnDisallowChangingTypeAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE, &ColumnDisallowChangingTypeAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE, &ColumnDisallowChangingTypeAdvisor{})
}

// ColumnDisallowChangingTypeAdvisor is the advisor checking for disallow changing column type.
type ColumnDisallowChangingTypeAdvisor struct {
}

// Check checks for disallow changing column type.
func (*ColumnDisallowChangingTypeAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnDisallowChangingTypeOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		originalMetadata: checkCtx.OriginalMetadata,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnDisallowChangingTypeOmniRule struct {
	OmniBaseRule
	originalMetadata *model.DatabaseMetadata
}

func (*columnDisallowChangingTypeOmniRule) Name() string {
	return "ColumnDisallowChangingTypeRule"
}

func (r *columnDisallowChangingTypeOmniRule) OnStatement(node ast.Node) {
	n, ok := node.(*ast.AlterTableStmt)
	if !ok || n.Table == nil {
		return
	}

	tableName := n.Table.Name
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}

		var columnName string
		switch cmd.Type {
		case ast.ATChangeColumn:
			// For CHANGE, Name is the old column name.
			columnName = cmd.Name
		case ast.ATModifyColumn:
			// For MODIFY, the column name comes from the column def or Name.
			if cmd.Column != nil {
				columnName = cmd.Column.Name
			} else {
				columnName = cmd.Name
			}
		default:
			continue
		}

		if cmd.Column == nil || cmd.Column.TypeName == nil {
			continue
		}
		r.checkColumnType(tableName, columnName, cmd.Column.TypeName, cmd.Loc)
	}
}

func normalizeColumnType(tp string) string {
	switch strings.ToLower(tp) {
	case "tinyint":
		return "tinyint(4)"
	case "tinyint unsigned":
		return "tinyint(4) unsigned"
	case "smallint":
		return "smallint(6)"
	case "smallint unsigned":
		return "smallint(6) unsigned"
	case "mediumint":
		return "mediumint(9)"
	case "mediumint unsigned":
		return "mediumint(9) unsigned"
	case "int":
		return "int(11)"
	case "int unsigned":
		return "int(11) unsigned"
	case "bigint":
		return "bigint(20)"
	case "bigint unsigned":
		return "bigint(20) unsigned"
	default:
		return strings.ToLower(tp)
	}
}

func (r *columnDisallowChangingTypeOmniRule) checkColumnType(tableName, columnName string, dt *ast.DataType, loc ast.Loc) {
	column := r.originalMetadata.GetSchemaMetadata("").GetTable(tableName).GetColumn(columnName)
	if column == nil {
		return
	}

	// Build the type string from the omni DataType.
	tp := strings.ToLower(dt.Name)
	if dt.Length > 0 && dt.Scale > 0 {
		tp = fmt.Sprintf("%s(%d,%d)", tp, dt.Length, dt.Scale)
	} else if dt.Length > 0 {
		tp = fmt.Sprintf("%s(%d)", tp, dt.Length)
	}
	if dt.Unsigned {
		tp += " unsigned"
	}

	if normalizeColumnType(column.GetProto().Type) != normalizeColumnType(tp) {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.ChangeColumnType.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("\"%s\" changes column type", r.QueryText()),
			StartPosition: common.ConvertANTLRLineToPosition(int(r.LocToLine(loc))),
		})
	}
}
