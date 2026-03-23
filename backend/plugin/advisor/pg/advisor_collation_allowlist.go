package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*CollationAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, &CollationAllowlistAdvisor{})
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

	allowlist := make(map[string]bool)
	for _, collation := range stringArrayPayload.List {
		allowlist[collation] = true
	}

	rule := &collationAllowlistRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		allowlist: allowlist,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type collationAllowlistRule struct {
	OmniBaseRule

	allowlist map[string]bool
}

func (*collationAllowlistRule) Name() string {
	return "collation-allowlist"
}

func (r *collationAllowlistRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	default:
	}
}

func (r *collationAllowlistRule) handleCreateStmt(n *ast.CreateStmt) {
	cols, _ := omniTableElements(n)
	for _, col := range cols {
		r.checkColumnCollation(col)
	}
}

func (r *collationAllowlistRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	for _, cmd := range omniAlterTableCmds(n) {
		switch ast.AlterTableType(cmd.Subtype) {
		case ast.AT_AddColumn:
			if colDef, ok := cmd.Def.(*ast.ColumnDef); ok {
				r.checkColumnCollation(colDef)
			}
		case ast.AT_AlterColumnType:
			if colDef, ok := cmd.Def.(*ast.ColumnDef); ok {
				r.checkColumnCollation(colDef)
			}
		default:
		}
	}
}

func (r *collationAllowlistRule) checkColumnCollation(col *ast.ColumnDef) {
	if col == nil || col.CollClause == nil {
		return
	}
	collName := r.extractCollationName(col.CollClause)
	if collName != "" && !r.allowlist[collName] {
		r.addCollationAdvice(collName)
	}
}

func (*collationAllowlistRule) extractCollationName(cc *ast.CollateClause) string {
	if cc == nil || cc.Collname == nil {
		return ""
	}
	var parts []string
	for _, item := range cc.Collname.Items {
		if s, ok := item.(*ast.String); ok {
			parts = append(parts, s.Str)
		}
	}
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func (r *collationAllowlistRule) addCollationAdvice(collation string) {
	stmtText := strings.TrimSpace(r.StmtText)
	r.AddAdvice(&storepb.Advice{
		Status:  r.Level,
		Code:    code.DisabledCollation.Int32(),
		Title:   r.Title,
		Content: fmt.Sprintf("Use disabled collation \"%s\", related statement \"%s\"", collation, stmtText),
		StartPosition: &storepb.Position{
			Line:   r.ContentStartLine(),
			Column: 0,
		},
	})
}
