// Package oracle is the advisor for oracle database.
package oracle

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnTypeDisallowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleColumnTypeDisallowList, &ColumnTypeDisallowListAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleColumnTypeDisallowList, &ColumnTypeDisallowListAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleColumnTypeDisallowList, &ColumnTypeDisallowListAdvisor{})
}

// ColumnTypeDisallowListAdvisor is the advisor checking for column type disallow list.
type ColumnTypeDisallowListAdvisor struct {
}

// Check checks for column type disallow list.
func (*ColumnTypeDisallowListAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	payload, err := advisor.UnmarshalStringArrayTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	listener := &columnTypeDisallowListListener{
		level:           level,
		title:           string(ctx.Rule.Type),
		currentDatabase: ctx.CurrentDatabase,
		disallowList:    payload.List,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// columnTypeDisallowListListener is the listener for column type disallow list.
type columnTypeDisallowListListener struct {
	*parser.BasePlSqlParserListener

	level           storepb.Advice_Status
	title           string
	currentDatabase string
	disallowList    []string
	adviceList      []*storepb.Advice
}

func (l *columnTypeDisallowListListener) generateAdvice() ([]*storepb.Advice, error) {
	return l.adviceList, nil
}

func (l *columnTypeDisallowListListener) isDisallowType(tp parser.IDatatypeContext) bool {
	if tp == nil {
		return false
	}
	for _, disallowType := range l.disallowList {
		if equivalent, err := plsqlparser.EquivalentType(tp, disallowType); err == nil && equivalent {
			return true
		}
	}
	return false
}

// EnterColumn_definition is called when production column_definition is entered.
func (l *columnTypeDisallowListListener) EnterColumn_definition(ctx *parser.Column_definitionContext) {
	if l.isDisallowType(ctx.Datatype()) {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:  l.level,
			Code:    advisor.DisabledColumnType.Int32(),
			Title:   l.title,
			Content: fmt.Sprintf("Disallow column type %s but column \"%s\" is", ctx.Datatype().GetText(), normalizeIdentifier(ctx.Column_name(), l.currentDatabase)),
			StartPosition: &storepb.Position{
				Line: int32(ctx.Datatype().GetStart().GetLine()),
			},
		})
	}
	if ctx.Regular_id() != nil {
		for _, tp := range l.disallowList {
			if ctx.Regular_id().GetText() == tp {
				l.adviceList = append(l.adviceList, &storepb.Advice{
					Status:  l.level,
					Code:    advisor.DisabledColumnType.Int32(),
					Title:   l.title,
					Content: fmt.Sprintf("Disallow column type %s but column \"%s\" is", ctx.Regular_id().GetText(), normalizeIdentifier(ctx.Column_name(), l.currentDatabase)),
					StartPosition: &storepb.Position{
						Line: int32(ctx.Regular_id().GetStart().GetLine()),
					},
				})
				break
			}
		}
	}
}

// EnterModify_col_properties is called when production modify_col_properties is entered.
func (l *columnTypeDisallowListListener) EnterModify_col_properties(ctx *parser.Modify_col_propertiesContext) {
	if l.isDisallowType(ctx.Datatype()) {
		l.adviceList = append(l.adviceList, &storepb.Advice{
			Status:  l.level,
			Code:    advisor.DisabledColumnType.Int32(),
			Title:   l.title,
			Content: fmt.Sprintf("Disallow column type %s but column \"%s\" is", ctx.Datatype().GetText(), normalizeIdentifier(ctx.Column_name(), l.currentDatabase)),
			StartPosition: &storepb.Position{
				Line: int32(ctx.Datatype().GetStart().GetLine()),
			},
		})
	}
}
