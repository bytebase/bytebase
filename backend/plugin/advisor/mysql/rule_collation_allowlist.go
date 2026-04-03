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
	_ advisor.Advisor = (*CollationAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, &CollationAllowlistAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, &CollationAllowlistAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, &CollationAllowlistAdvisor{})
}

// CollationAllowlistAdvisor is the advisor checking for collation allowlist.
type CollationAllowlistAdvisor struct {
}

// Check checks for collation allowlist.
func (*CollationAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()

	allowList := make(map[string]bool)
	for _, collation := range stringArrayPayload.List {
		allowList[strings.ToLower(collation)] = true
	}

	rule := &collationAllowlistOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		allowList: allowList,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type collationAllowlistOmniRule struct {
	OmniBaseRule
	allowList map[string]bool
}

func (*collationAllowlistOmniRule) Name() string {
	return "CollationAllowlistRule"
}

func (r *collationAllowlistOmniRule) OnStatement(node ast.Node) {
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

func (r *collationAllowlistOmniRule) checkCollation(collation string, line int) {
	collation = strings.ToLower(collation)
	if collation != "" && !r.allowList[collation] {
		r.AddAdvice(&storepb.Advice{
			Status:        r.Level,
			Code:          code.DisabledCollation.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("\"%s\" used disabled collation '%s'", r.QueryText(), collation),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
	}
}

func (r *collationAllowlistOmniRule) checkCreateDatabase(n *ast.CreateDatabaseStmt) {
	for _, opt := range n.Options {
		if strings.EqualFold(opt.Name, "COLLATE") {
			r.checkCollation(opt.Value, r.BaseLine+int(r.LocToLine(n.Loc)))
		}
	}
}

func (r *collationAllowlistOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	// Check table-level collation option.
	for _, opt := range n.Options {
		if strings.EqualFold(opt.Name, "COLLATE") {
			r.checkCollation(opt.Value, r.BaseLine+int(r.LocToLine(opt.Loc)))
		}
	}
	// Check column-level collation.
	for _, col := range n.Columns {
		if col == nil || col.TypeName == nil {
			continue
		}
		if col.TypeName.Collate != "" {
			r.checkCollation(col.TypeName.Collate, r.BaseLine+int(r.LocToLine(col.Loc)))
		}
		// Also check column constraints for COLLATE.
		for _, c := range col.Constraints {
			if c.Type == ast.ColConstrCollate && c.Name != "" {
				r.checkCollation(c.Name, r.BaseLine+int(r.LocToLine(col.Loc)))
			}
		}
	}
}

func (r *collationAllowlistOmniRule) checkAlterDatabase(n *ast.AlterDatabaseStmt) {
	for _, opt := range n.Options {
		if strings.EqualFold(opt.Name, "COLLATE") {
			r.checkCollation(opt.Value, r.BaseLine+int(r.LocToLine(n.Loc)))
		}
	}
}

func (r *collationAllowlistOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
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
				if col.TypeName.Collate != "" {
					r.checkCollation(col.TypeName.Collate, r.BaseLine+int(r.LocToLine(cmd.Loc)))
				}
				for _, c := range col.Constraints {
					if c.Type == ast.ColConstrCollate && c.Name != "" {
						r.checkCollation(c.Name, r.BaseLine+int(r.LocToLine(cmd.Loc)))
					}
				}
			}
		case ast.ATTableOption:
			if cmd.Option != nil && strings.EqualFold(cmd.Option.Name, "COLLATE") {
				r.checkCollation(cmd.Option.Value, r.BaseLine+int(r.LocToLine(cmd.Option.Loc)))
			}
		default:
		}
	}
}
