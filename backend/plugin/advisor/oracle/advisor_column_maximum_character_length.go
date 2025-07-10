// Package oracle is the advisor for oracle database.
package oracle

import (
	"context"
	"fmt"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*ColumnMaximumCharacterLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleColumnMaximumCharacterLength, &ColumnMaximumCharacterLengthAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleColumnMaximumCharacterLength, &ColumnMaximumCharacterLengthAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleColumnMaximumCharacterLength, &ColumnMaximumCharacterLengthAdvisor{})
}

// ColumnMaximumCharacterLengthAdvisor is the advisor checking for maximum character length.
type ColumnMaximumCharacterLengthAdvisor struct {
}

// Check checks for maximum character length.
func (*ColumnMaximumCharacterLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := checkCtx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	listener := &columnMaximumCharacterLengthListener{
		level:   level,
		title:   string(checkCtx.Rule.Type),
		maximum: payload.Number,
	}

	if listener.maximum > 0 {
		antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	}

	return listener.generateAdvice()
}

// columnMaximumCharacterLengthListener is the listener for maximum character length.
type columnMaximumCharacterLengthListener struct {
	*parser.BasePlSqlParserListener

	level      storepb.Advice_Status
	title      string
	maximum    int
	adviceList []*storepb.Advice
}

func (l *columnMaximumCharacterLengthListener) generateAdvice() ([]*storepb.Advice, error) {
	return l.adviceList, nil
}

// EnterDatatype is called when production datatype is entered.
func (l *columnMaximumCharacterLengthListener) EnterDatatype(ctx *parser.DatatypeContext) {
	if ctx.Native_datatype_element() == nil {
		return
	}

	if ctx.Native_datatype_element().CHAR() == nil && ctx.Native_datatype_element().CHARACTER() == nil {
		return
	}

	if ctx.Precision_part() == nil {
		return
	}

	if ctx.Precision_part().Numeric(0) != nil {
		lengthText := ctx.Precision_part().Numeric(0).GetText()
		length, err := strconv.Atoi(lengthText)
		if err != nil || length <= l.maximum {
			return
		}
	}

	l.adviceList = append(l.adviceList, &storepb.Advice{
		Status:        l.level,
		Code:          advisor.CharLengthExceedsLimit.Int32(),
		Title:         l.title,
		Content:       fmt.Sprintf("The maximum character length is %d.", l.maximum),
		StartPosition: common.ConvertANTLRLineToPosition(ctx.GetStart().GetLine()),
	})
}
