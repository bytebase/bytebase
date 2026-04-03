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
	_ advisor.Advisor = (*NamingIdentifierNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD, &NamingIdentifierNoKeywordAdvisor{})
}

// NamingIdentifierNoKeywordAdvisor is the advisor checking for identifier naming convention without keyword.
type NamingIdentifierNoKeywordAdvisor struct {
}

// Check checks for identifier naming convention without keyword.
func (*NamingIdentifierNoKeywordAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &namingIdentifierNoKeywordOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type namingIdentifierNoKeywordOmniRule struct {
	OmniBaseRule
}

func (*namingIdentifierNoKeywordOmniRule) Name() string {
	return "NamingIdentifierNoKeywordRule"
}

func (r *namingIdentifierNoKeywordOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	case *ast.CreateIndexStmt:
		r.checkCreateIndex(n)
	default:
	}
}

func (r *namingIdentifierNoKeywordOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table != nil {
		r.checkIdentifierAtLine(n.Table.Name, r.LocToLine(n.Loc))
	}
	for _, col := range n.Columns {
		if col == nil {
			continue
		}
		r.checkIdentifierAtLine(col.Name, r.findColumnLine(col.Name))
	}
	for _, constraint := range n.Constraints {
		if constraint == nil {
			continue
		}
		if constraint.Name != "" {
			r.checkIdentifierAtLine(constraint.Name, r.LocToLine(constraint.Loc))
		}
	}
}

func (r *namingIdentifierNoKeywordOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		switch cmd.Type {
		case ast.ATAddColumn:
			for _, col := range omniGetColumnsFromCmd(cmd) {
				r.checkIdentifierAtLine(col.Name, r.LocToLine(n.Loc))
			}
		case ast.ATRenameTable:
			if cmd.NewName != "" {
				r.checkIdentifierAtLine(cmd.NewName, r.LocToLine(n.Loc))
			}
		case ast.ATRenameColumn:
			if cmd.NewName != "" {
				r.checkIdentifierAtLine(cmd.NewName, r.LocToLine(n.Loc))
			}
		case ast.ATChangeColumn:
			if cmd.Column != nil {
				r.checkIdentifierAtLine(cmd.Column.Name, r.LocToLine(n.Loc))
			}
		case ast.ATAddConstraint, ast.ATAddIndex:
			if cmd.Constraint != nil && cmd.Constraint.Name != "" {
				r.checkIdentifierAtLine(cmd.Constraint.Name, r.LocToLine(n.Loc))
			}
		case ast.ATRenameIndex:
			if cmd.NewName != "" {
				r.checkIdentifierAtLine(cmd.NewName, r.LocToLine(n.Loc))
			}
		default:
		}
	}
}

func (r *namingIdentifierNoKeywordOmniRule) checkCreateIndex(n *ast.CreateIndexStmt) {
	if n.IndexName != "" {
		r.checkIdentifierAtLine(n.IndexName, r.LocToLine(n.Loc))
	}
}

func (r *namingIdentifierNoKeywordOmniRule) checkIdentifierAtLine(identifier string, lineNumber int32) {
	// Strip backticks if present.
	identifier = strings.Trim(identifier, "`")
	if !isKeyword(identifier) {
		return
	}
	absoluteLine := r.BaseLine + int(lineNumber)
	r.AddAdviceAbsolute(&storepb.Advice{
		Status:        r.Level,
		Code:          code.NameIsKeywordIdentifier.Int32(),
		Title:         r.Title,
		Content:       fmt.Sprintf("Identifier %q is a keyword and should be avoided", identifier),
		StartPosition: common.ConvertANTLRLineToPosition(absoluteLine),
	})
}

func (r *namingIdentifierNoKeywordOmniRule) findColumnLine(name string) int32 {
	return r.FindLineByName(name)
}
