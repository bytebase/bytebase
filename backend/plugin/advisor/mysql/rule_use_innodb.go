package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/bytebase/bytebase/backend/common"
)

const (
	innoDB              string = "innodb"
	defaultStorageEngin string = "default_storage_engine"
)

var _ advisor.Advisor = (*UseInnoDBAdvisor)(nil)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_ENGINE_MYSQL_USE_INNODB, &UseInnoDBAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_ENGINE_MYSQL_USE_INNODB, &UseInnoDBAdvisor{})
}

// UseInnoDBAdvisor is the advisor checking for using InnoDB engine.
type UseInnoDBAdvisor struct {
}

// Check checks for using InnoDB engine.
func (*UseInnoDBAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &useInnoDBOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type useInnoDBOmniRule struct {
	OmniBaseRule
}

func (*useInnoDBOmniRule) Name() string {
	return "UseInnoDBRule"
}

func (r *useInnoDBOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	case *ast.SetStmt:
		r.checkSetStatement(n)
	default:
	}
}

func (r *useInnoDBOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	engine := omniTableOptionValue(n.Options, "ENGINE")
	if engine == "" {
		return
	}
	if strings.ToLower(engine) != innoDB {
		content := r.TrimmedStmtText()
		r.addAdvice(content, r.FindLineByName(engine))
	}
}

func (r *useInnoDBOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	for _, cmd := range n.Commands {
		if cmd.Type == ast.ATTableOption && cmd.Option != nil {
			if strings.EqualFold(cmd.Option.Name, "ENGINE") && strings.ToLower(cmd.Option.Value) != innoDB {
				content := r.TrimmedStmtText()
				r.addAdvice(content, r.ContentStartLine())
				return
			}
		}
	}
}

func (r *useInnoDBOmniRule) checkSetStatement(n *ast.SetStmt) {
	for _, assign := range n.Assignments {
		if assign.Column == nil {
			continue
		}
		name := assign.Column.Column
		if strings.ToLower(name) != defaultStorageEngin {
			continue
		}
		if assign.Value != nil {
			if lit, ok := assign.Value.(*ast.StringLit); ok {
				if strings.ToLower(lit.Value) != innoDB {
					content := r.TrimmedStmtText()
					r.addAdvice(content, r.ContentStartLine())
					return
				}
			}
			// For identifiers like CSV (without quotes), they may be parsed as ColumnRef
			if ref, ok := assign.Value.(*ast.ColumnRef); ok {
				if strings.ToLower(ref.Column) != innoDB {
					content := r.TrimmedStmtText()
					r.addAdvice(content, r.ContentStartLine())
					return
				}
			}
		}
	}
}

func (r *useInnoDBOmniRule) addAdvice(content string, lineNumber int32) {
	absoluteLine := r.BaseLine + int(lineNumber)
	r.AddAdviceAbsolute(&storepb.Advice{
		Status:        r.Level,
		Code:          advisorcode.NotInnoDBEngine.Int32(),
		Title:         r.Title,
		Content:       fmt.Sprintf("\"%s;\" doesn't use InnoDB engine", content),
		StartPosition: common.ConvertANTLRLineToPosition(absoluteLine),
	})
}
