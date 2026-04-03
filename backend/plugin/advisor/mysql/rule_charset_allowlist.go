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
	_ advisor.Advisor = (*CharsetAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST, &CharsetAllowlistAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST, &CharsetAllowlistAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST, &CharsetAllowlistAdvisor{})
}

// CharsetAllowlistAdvisor is the advisor checking for charset allowlist.
type CharsetAllowlistAdvisor struct {
}

// Check checks for charset allowlist.
func (*CharsetAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()

	allowList := make(map[string]bool)
	for _, charset := range stringArrayPayload.List {
		allowList[strings.ToLower(charset)] = true
	}

	rule := &charsetAllowlistOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		allowList: allowList,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type charsetAllowlistOmniRule struct {
	OmniBaseRule
	allowList map[string]bool
}

func (*charsetAllowlistOmniRule) Name() string {
	return "CharsetAllowlistRule"
}

func (r *charsetAllowlistOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateDatabaseStmt:
		r.checkCreateDatabase(n)
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterDatabaseStmt:
		r.checkAlterDatabase(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *charsetAllowlistOmniRule) checkCharset(charset string, line int) {
	charset = strings.ToLower(charset)
	if charset != "" && !r.allowList[charset] {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.DisabledCharset.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("\"%s\" used disabled charset '%s'", r.QueryText(), charset),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
	}
}

func (r *charsetAllowlistOmniRule) checkCreateDatabase(n *ast.CreateDatabaseStmt) {
	for _, opt := range n.Options {
		if strings.EqualFold(opt.Name, "CHARACTER SET") || strings.EqualFold(opt.Name, "CHARSET") {
			r.checkCharset(opt.Value, r.BaseLine+int(r.LocToLine(n.Loc)))
			break
		}
	}
}

func (r *charsetAllowlistOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	// Check table-level charset option.
	for _, opt := range n.Options {
		if strings.EqualFold(opt.Name, "CHARACTER SET") || strings.EqualFold(opt.Name, "CHARSET") || strings.EqualFold(opt.Name, "DEFAULT CHARSET") {
			r.checkCharset(opt.Value, r.BaseLine+int(r.LocToLine(n.Loc)))
			break
		}
	}
	// Check column-level charset.
	for _, col := range n.Columns {
		if col == nil || col.TypeName == nil {
			continue
		}
		if col.TypeName.Charset != "" {
			r.checkCharset(col.TypeName.Charset, r.BaseLine+int(r.LocToLine(n.Loc)))
		}
	}
}

func (r *charsetAllowlistOmniRule) checkAlterDatabase(n *ast.AlterDatabaseStmt) {
	for _, opt := range n.Options {
		if strings.EqualFold(opt.Name, "CHARACTER SET") || strings.EqualFold(opt.Name, "CHARSET") {
			r.checkCharset(opt.Value, r.BaseLine+int(r.LocToLine(n.Loc)))
		}
	}
}

func (r *charsetAllowlistOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		switch cmd.Type {
		case ast.ATAddColumn, ast.ATModifyColumn, ast.ATChangeColumn:
			for _, col := range omniGetColumnsFromCmd(cmd) {
				if col == nil || col.TypeName == nil {
					continue
				}
				if col.TypeName.Charset != "" {
					r.checkCharset(col.TypeName.Charset, r.BaseLine+int(r.LocToLine(n.Loc)))
				}
			}
		case ast.ATTableOption:
			if cmd.Option != nil {
				if strings.EqualFold(cmd.Option.Name, "CHARACTER SET") || strings.EqualFold(cmd.Option.Name, "CHARSET") || strings.EqualFold(cmd.Option.Name, "DEFAULT CHARSET") {
					r.checkCharset(cmd.Option.Value, r.BaseLine+int(r.LocToLine(n.Loc)))
				}
			}
		case ast.ATConvertCharset:
			if cmd.Option != nil {
				if strings.EqualFold(cmd.Option.Name, "CHARACTER SET") || strings.EqualFold(cmd.Option.Name, "CHARSET") {
					r.checkCharset(cmd.Option.Value, r.BaseLine+int(r.LocToLine(n.Loc)))
				}
			}
		default:
		}
	}
}
