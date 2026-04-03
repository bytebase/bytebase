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
)

var (
	_ advisor.Advisor = (*ColumnRequireCharsetAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_REQUIRE_CHARSET, &ColumnRequireCharsetAdvisor{})
}

// ColumnRequireCharsetAdvisor is the advisor checking for require charset.
type ColumnRequireCharsetAdvisor struct {
}

func (*ColumnRequireCharsetAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnRequireCharsetOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnRequireCharsetOmniRule struct {
	OmniBaseRule
}

func (*columnRequireCharsetOmniRule) Name() string {
	return "ColumnRequireCharsetRule"
}

func (r *columnRequireCharsetOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnRequireCharsetOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	for _, col := range n.Columns {
		if col == nil || col.TypeName == nil {
			continue
		}
		if omniIsCharsetDataType(col.TypeName) && col.TypeName.Charset == "" {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.NoCharset.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Column %s does not have a character set specified", col.Name),
				StartPosition: common.ConvertANTLRLineToPosition(int(r.LocToLine(col.Loc))),
			})
		}
	}
}

func (r *columnRequireCharsetOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	if n.Table == nil {
		return
	}
	for _, cmd := range n.Commands {
		if cmd == nil || cmd.Type != ast.ATAddColumn {
			continue
		}
		// Single column ADD.
		if cmd.Column != nil && cmd.Column.TypeName != nil {
			if omniIsCharsetDataType(cmd.Column.TypeName) && cmd.Column.TypeName.Charset == "" {
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          code.NoCharset.Int32(),
					Title:         r.Title,
					Content:       fmt.Sprintf("Column %s does not have a character set specified", cmd.Column.Name),
					StartPosition: common.ConvertANTLRLineToPosition(int(r.LocToLine(cmd.Loc))),
				})
			}
		}
		// Multi-column ADD.
		for _, col := range cmd.Columns {
			if col == nil || col.TypeName == nil {
				continue
			}
			if omniIsCharsetDataType(col.TypeName) && col.TypeName.Charset == "" {
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          code.NoCharset.Int32(),
					Title:         r.Title,
					Content:       fmt.Sprintf("Column %s does not have a character set specified", col.Name),
					StartPosition: common.ConvertANTLRLineToPosition(int(r.LocToLine(cmd.Loc))),
				})
			}
		}
	}
}

// omniIsCharsetDataType checks if a data type supports character sets.
func omniIsCharsetDataType(dt *ast.DataType) bool {
	if dt == nil {
		return false
	}
	switch strings.ToUpper(dt.Name) {
	case "CHAR", "VARCHAR", "TINYTEXT", "TEXT", "MEDIUMTEXT", "LONGTEXT":
		return true
	default:
		return false
	}
}
