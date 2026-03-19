package pg

import (
	"context"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*StatementCreateSpecifySchema)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_CREATE_SPECIFY_SCHEMA, &StatementCreateSpecifySchema{})
}

type StatementCreateSpecifySchema struct {
}

func (*StatementCreateSpecifySchema) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &statementCreateSpecifySchemaRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementCreateSpecifySchemaRule struct {
	OmniBaseRule
}

func (*statementCreateSpecifySchemaRule) Name() string {
	return "statement_create_specify_schema"
}

func (r *statementCreateSpecifySchemaRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		if n.Relation != nil && n.Relation.Schemaname == "" {
			r.addMissingSchemaAdvice("Table")
		}
	default:
	}
}

func (r *statementCreateSpecifySchemaRule) addMissingSchemaAdvice(objectType string) {
	r.AddAdvice(&storepb.Advice{
		Status:  r.Level,
		Code:    code.StatementCreateWithoutSchemaName.Int32(),
		Title:   r.Title,
		Content: objectType + " schema should be specified.",
		StartPosition: &storepb.Position{
			Line:   r.ContentStartLine(),
			Column: 0,
		},
	})
}
