// Package oracle is the advisor for oracle database.
package oracle

import (
	"fmt"
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnMaximumVarcharLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_ORACLE, advisor.OracleColumnMaximumVarcharLength, &ColumnMaximumVarcharLengthAdvisor{})
	advisor.Register(storepb.Engine_DM, advisor.OracleColumnMaximumVarcharLength, &ColumnMaximumVarcharLengthAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE_ORACLE, advisor.OracleColumnMaximumVarcharLength, &ColumnMaximumVarcharLengthAdvisor{})
}

// ColumnMaximumVarcharLengthAdvisor is the advisor checking for maximum varchar length.
type ColumnMaximumVarcharLengthAdvisor struct {
}

// Check checks for maximum varchar length.
func (*ColumnMaximumVarcharLengthAdvisor) Check(ctx advisor.Context) ([]*storepb.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := advisor.UnmarshalNumberTypeRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	listener := &columnMaximumVarcharLengthListener{
		level:   level,
		title:   string(ctx.Rule.Type),
		maximum: payload.Number,
	}

	if listener.maximum > 0 {
		antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	}

	return listener.generateAdvice()
}

// columnMaximumVarcharLengthListener is the listener for maximum varchar length.
type columnMaximumVarcharLengthListener struct {
	*parser.BasePlSqlParserListener

	level      storepb.Advice_Status
	title      string
	maximum    int
	adviceList []*storepb.Advice
}

func (l *columnMaximumVarcharLengthListener) generateAdvice() ([]*storepb.Advice, error) {
	return l.adviceList, nil
}

// EnterDatatype is called when production datatype is entered.
func (l *columnMaximumVarcharLengthListener) EnterDatatype(ctx *parser.DatatypeContext) {
	if ctx.Native_datatype_element() == nil {
		return
	}

	if ctx.Native_datatype_element().VARCHAR() == nil && ctx.Native_datatype_element().VARCHAR2() == nil {
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
		Status:  l.level,
		Code:    advisor.VarcharLengthExceedsLimit.Int32(),
		Title:   l.title,
		Content: fmt.Sprintf("The maximum varchar length is %d.", l.maximum),
		StartPosition: &storepb.Position{
			Line: int32(ctx.GetStart().GetLine()),
		},
	})
}
