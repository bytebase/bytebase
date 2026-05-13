package tidb

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnTypeDisallowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, &ColumnTypeDisallowListAdvisor{})
}

// ColumnTypeDisallowListAdvisor is the advisor checking for column type restriction.
type ColumnTypeDisallowListAdvisor struct {
}

// Check checks for column type restriction.
func (*ColumnTypeDisallowListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := checkCtx.Rule.Type.String()
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	typeRestriction := make(map[string]bool)
	for _, tp := range stringArrayPayload.List {
		typeRestriction[strings.ToUpper(tp)] = true
	}

	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			if n.Table == nil {
				continue
			}
			tableName := n.Table.Name
			for _, column := range n.Columns {
				if column == nil {
					continue
				}
				if advice := checkColumnTypeDisallow(typeRestriction, level, title, tableName, column, ostmt.AbsoluteLine(column.Loc.Start)); advice != nil {
					adviceList = append(adviceList, advice)
				}
			}
		case *ast.AlterTableStmt:
			if n.Table == nil {
				continue
			}
			tableName := n.Table.Name
			stmtLine := ostmt.AbsoluteLine(n.Loc.Start)
			for _, cmd := range n.Commands {
				if cmd == nil {
					continue
				}
				switch cmd.Type {
				case ast.ATAddColumn:
					for _, column := range addColumnTargets(cmd) {
						if column == nil {
							continue
						}
						if advice := checkColumnTypeDisallow(typeRestriction, level, title, tableName, column, stmtLine); advice != nil {
							adviceList = append(adviceList, advice)
						}
					}
				case ast.ATChangeColumn, ast.ATModifyColumn:
					if cmd.Column == nil {
						continue
					}
					if advice := checkColumnTypeDisallow(typeRestriction, level, title, tableName, cmd.Column, stmtLine); advice != nil {
						adviceList = append(adviceList, advice)
					}
				default:
				}
			}
		default:
		}
	}

	return adviceList, nil
}

// checkColumnTypeDisallow returns an advice if the column's rendered
// type is in the user-supplied blocklist, or nil if not.
//
// Uses omniBuildCompactTypeString — the CompactStr-equivalent — to
// match the pre-migration pingcap behavior. Pingcap's
// `column.Tp.CompactStr()` rendered length, scale, and ENUM/SET
// literals (e.g. "varchar(255)", "enum('x','y')", "tinyint(1)"),
// applied canonical defaults to bare integer forms ("int" →
// "int(11)"), and canonicalized aliases ("BOOLEAN" → "tinyint(1)").
// A user blocklist of ["VARCHAR(255)"] matched VARCHAR(255) columns
// exactly; a blocklist of ["JSON"] matched JSON columns. The earlier
// commit on this PR used omniDataTypeNameCompact (bare lowercase
// Name only), which silently broke length/literal blocklist entries
// — Codex P1 catch.
func checkColumnTypeDisallow(typeRestriction map[string]bool, level storepb.Advice_Status, title, tableName string, col *ast.ColumnDef, line int) *storepb.Advice {
	if col.TypeName == nil {
		return nil
	}
	columnType := strings.ToUpper(omniBuildCompactTypeString(col.TypeName, strings.ToLower(col.TypeName.Name)))
	if !typeRestriction[columnType] {
		return nil
	}
	return &storepb.Advice{
		Status:        level,
		Code:          code.DisabledColumnType.Int32(),
		Title:         title,
		Content:       fmt.Sprintf("Disallow column type %s but column `%s`.`%s` is", columnType, tableName, col.Name),
		StartPosition: common.ConvertANTLRLineToPosition(line),
	}
}
