package snowflake

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	omniast "github.com/bytebase/omni/snowflake/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

const (
	// varcharDefaultLength is the default length of varchar in Snowflake.
	// https://docs.snowflake.com/en/sql-reference/data-types-text
	varcharDefaultLength = 16_777_216
)

var (
	_ advisor.Advisor = (*ColumnMaximumVarcharLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH, &ColumnMaximumVarcharLengthAdvisor{})
}

// ColumnMaximumVarcharLengthAdvisor is the advisor checking for maximum varchar length.
type ColumnMaximumVarcharLengthAdvisor struct {
}

// Check checks for maximum varchar length.
func (*ColumnMaximumVarcharLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for this rule")
	}

	if int(numberPayload.Number) <= 0 {
		return nil, nil
	}

	// Create the rule
	rule := NewColumnMaximumVarcharLengthRule(level, checkCtx.Rule.Type.String(), int(numberPayload.Number))

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := snowsqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		rule.checkStatement(node, stmt.Text)
	}

	return rule.GetAdviceList(), nil
}

// ColumnMaximumVarcharLengthRule checks for maximum varchar length.
type ColumnMaximumVarcharLengthRule struct {
	BaseRule
	maximum int
	// stmtText is the SQL text of the statement currently being checked; node
	// Loc offsets are relative to it.
	stmtText string
}

// NewColumnMaximumVarcharLengthRule creates a new ColumnMaximumVarcharLengthRule.
func NewColumnMaximumVarcharLengthRule(level storepb.Advice_Status, title string, maximum int) *ColumnMaximumVarcharLengthRule {
	return &ColumnMaximumVarcharLengthRule{
		BaseRule: BaseRule{
			level: level,
			title: title,
		},
		maximum: maximum,
	}
}

// Name returns the rule name.
func (*ColumnMaximumVarcharLengthRule) Name() string {
	return "ColumnMaximumVarcharLengthRule"
}

// checkStatement walks one statement's omni AST and checks every data type.
// This mirrors the legacy ANTLR listener that fired on every Data_type context
// anywhere in the statement.
func (r *ColumnMaximumVarcharLengthRule) checkStatement(node omniast.Node, stmtText string) {
	r.stmtText = stmtText
	omniast.Inspect(node, func(n omniast.Node) bool {
		if typeName, ok := n.(*omniast.TypeName); ok {
			r.checkTypeName(typeName)
		}
		return true
	})
}

func (r *ColumnMaximumVarcharLengthRule) checkTypeName(typeName *omniast.TypeName) {
	// The legacy rule only fired on the literal VARCHAR keyword: aliases like
	// STRING, TEXT, NVARCHAR (all TypeVarchar in omni) did not trigger it.
	if !strings.EqualFold(typeName.Name, "VARCHAR") {
		return
	}

	length := varcharDefaultLength
	if len(typeName.Params) > 0 {
		length = typeName.Params[0]
	}

	if length > r.maximum {
		r.AddAdvice(&storepb.Advice{
			Status:        r.level,
			Code:          code.VarcharLengthExceedsLimit.Int32(),
			Title:         r.title,
			Content:       fmt.Sprintf("The maximum varchar length is %d.", r.maximum),
			StartPosition: common.ConvertANTLRLineToPosition(r.baseLine + r.line(typeName.Loc.Start)),
		})
	}
}

// line converts a byte offset within the current statement text to a 1-based
// line number, mirroring the ANTLR token line the legacy advisor reported.
func (r *ColumnMaximumVarcharLengthRule) line(offset int) int {
	if offset < 0 {
		return 1
	}
	if offset > len(r.stmtText) {
		offset = len(r.stmtText)
	}
	return 1 + strings.Count(r.stmtText[:offset], "\n")
}
