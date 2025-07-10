// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*NamingIdentifierCaseAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleIdentifierCase, &NamingIdentifierCaseAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleIdentifierCase, &NamingIdentifierCaseAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleIdentifierCase, &NamingIdentifierCaseAdvisor{})
}

// NamingIdentifierCaseAdvisor is the advisor checking for identifier case.
type NamingIdentifierCaseAdvisor struct {
}

// Check checks for identifier case.
func (*NamingIdentifierCaseAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNamingCaseRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	listener := &namingIdentifierCaseListener{
		level:           level,
		title:           string(checkCtx.Rule.Type),
		currentDatabase: checkCtx.CurrentDatabase,
		upper:           payload.Upper,
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// namingIdentifierCaseListener is the listener for identifier case.
type namingIdentifierCaseListener struct {
	*parser.BasePlSqlParserListener

	level           storepb.Advice_Status
	title           string
	currentDatabase string
	adviceList      []*storepb.Advice
	upper           bool
}

func (l *namingIdentifierCaseListener) generateAdvice() ([]*storepb.Advice, error) {
	return l.adviceList, nil
}

// EnterId_expression is called when production id_expression is entered.
func (l *namingIdentifierCaseListener) EnterId_expression(ctx *parser.Id_expressionContext) {
	identifier := normalizeIDExpression(ctx)
	if l.upper {
		if identifier != strings.ToUpper(identifier) {
			l.adviceList = append(l.adviceList, &storepb.Advice{
				Status:        l.level,
				Code:          advisor.NamingCaseMismatch.Int32(),
				Title:         l.title,
				Content:       fmt.Sprintf("Identifier %q should be upper case", identifier),
				StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
			})
		}
	} else {
		if identifier != strings.ToLower(identifier) {
			l.adviceList = append(l.adviceList, &storepb.Advice{
				Status:        l.level,
				Code:          advisor.NamingCaseMismatch.Int32(),
				Title:         l.title,
				Content:       fmt.Sprintf("Identifier %q should be lower case", identifier),
				StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
			})
		}
	}
}
