package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnRequireCollationAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_REQUIRE_COLLATION, &ColumnRequireCollationAdvisor{})
}

// ColumnRequireCollationAdvisor is the advisor checking for require collation.
type ColumnRequireCollationAdvisor struct {
}

func (*ColumnRequireCollationAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnRequireCollationOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnRequireCollationOmniRule struct {
	OmniBaseRule
}

func (*columnRequireCollationOmniRule) Name() string {
	return "ColumnRequireCollationRule"
}

func (r *columnRequireCollationOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnRequireCollationOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	for _, col := range n.Columns {
		if col == nil || col.TypeName == nil {
			continue
		}
		if omniIsCharsetDataType(col.TypeName) && col.TypeName.Collate == "" {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.NoCollation.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Column %s does not have a collation specified", col.Name),
				StartPosition: common.ConvertANTLRLineToPosition(int(r.LocToLine(col.Loc))),
			})
		}
	}
}

func (r *columnRequireCollationOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	if n.Table == nil {
		return
	}
	for _, cmd := range n.Commands {
		if cmd == nil || cmd.Type != ast.ATAddColumn {
			continue
		}
		if cmd.Column != nil && cmd.Column.TypeName != nil {
			if omniIsCharsetDataType(cmd.Column.TypeName) && cmd.Column.TypeName.Collate == "" {
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          code.NoCollation.Int32(),
					Title:         r.Title,
					Content:       fmt.Sprintf("Column %s does not have a collation specified", cmd.Column.Name),
					StartPosition: common.ConvertANTLRLineToPosition(int(r.LocToLine(cmd.Loc))),
				})
			}
		}
		for _, col := range cmd.Columns {
			if col == nil || col.TypeName == nil {
				continue
			}
			if omniIsCharsetDataType(col.TypeName) && col.TypeName.Collate == "" {
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          code.NoCollation.Int32(),
					Title:         r.Title,
					Content:       fmt.Sprintf("Column %s does not have a collation specified", col.Name),
					StartPosition: common.ConvertANTLRLineToPosition(int(r.LocToLine(cmd.Loc))),
				})
			}
		}
	}
}
