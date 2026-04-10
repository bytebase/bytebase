package mssql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mssql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

var (
	_ advisor.Advisor = (*NamingIdentifierNoKeywordAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD, &NamingIdentifierNoKeywordAdvisor{})
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

	rule := &namingIdentifierNoKeywordRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type namingIdentifierNoKeywordRule struct {
	OmniBaseRule
}

func (*namingIdentifierNoKeywordRule) Name() string {
	return "NamingIdentifierNoKeywordRule"
}

func (r *namingIdentifierNoKeywordRule) OnStatement(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		var name string
		switch v := n.(type) {
		case *ast.ColumnDef:
			name = v.Name
		case *ast.ConstraintDef:
			name = v.Name
		case *ast.CreateSchemaStmt:
			name = v.Name
		case *ast.CreateDatabaseStmt:
			name = v.Name
		case *ast.CreateIndexStmt:
			name = v.Name
		case *ast.CreateTableStmt:
			if v.Name != nil {
				name = v.Name.Object
			}
		default:
			return true
		}
		if name == "" {
			return true
		}
		lower := strings.ToLower(name)
		if tsqlparser.IsTSQLReservedKeyword(lower, false) {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.NameIsKeywordIdentifier.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Identifier [%s] is a keyword identifier and should be avoided.", lower),
				StartPosition: &storepb.Position{Line: r.FindLineByName(name)},
			})
		}
		return true
	})
}
