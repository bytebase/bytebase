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
	_ advisor.Advisor = (*ColumnDisallowSetCharsetAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET, &ColumnDisallowSetCharsetAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET, &ColumnDisallowSetCharsetAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET, &ColumnDisallowSetCharsetAdvisor{})
}

// ColumnDisallowSetCharsetAdvisor is the advisor checking for disallow set column charset.
type ColumnDisallowSetCharsetAdvisor struct {
}

// Check checks for disallow set column charset.
func (*ColumnDisallowSetCharsetAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnDisallowSetCharsetOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnDisallowSetCharsetOmniRule struct {
	OmniBaseRule
}

func (*columnDisallowSetCharsetOmniRule) Name() string {
	return "ColumnDisallowSetCharsetRule"
}

func (r *columnDisallowSetCharsetOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnDisallowSetCharsetOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	for _, col := range n.Columns {
		if col == nil || col.TypeName == nil {
			continue
		}
		if !checkCharsetAllowed(col.TypeName.Charset) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.SetColumnCharset.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Disallow set column charset but \"%s\" does", r.QueryText()),
				StartPosition: common.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
			})
		}
	}
}

func (r *columnDisallowSetCharsetOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	if n.Table == nil {
		return
	}

	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}

		var charsetList []string
		switch cmd.Type {
		case ast.ATAddColumn:
			if cmd.Column != nil && cmd.Column.TypeName != nil {
				charsetList = append(charsetList, cmd.Column.TypeName.Charset)
			}
			for _, col := range cmd.Columns {
				if col != nil && col.TypeName != nil {
					charsetList = append(charsetList, col.TypeName.Charset)
				}
			}
		case ast.ATChangeColumn:
			if cmd.Column != nil && cmd.Column.TypeName != nil {
				charsetList = append(charsetList, cmd.Column.TypeName.Charset)
			}
		case ast.ATModifyColumn:
			if cmd.Column != nil && cmd.Column.TypeName != nil {
				charsetList = append(charsetList, cmd.Column.TypeName.Charset)
			}
		default:
			continue
		}

		for _, charset := range charsetList {
			if !checkCharsetAllowed(charset) {
				r.AddAdvice(&storepb.Advice{
					Status:        r.Level,
					Code:          code.SetColumnCharset.Int32(),
					Title:         r.Title,
					Content:       fmt.Sprintf("Disallow set column charset but \"%s\" does", r.QueryText()),
					StartPosition: common.ConvertANTLRLineToPosition(int(r.ContentStartLine())),
				})
			}
		}
	}
}

func checkCharsetAllowed(charset string) bool {
	switch charset {
	case "", "binary":
		return true
	default:
		return false
	}
}
