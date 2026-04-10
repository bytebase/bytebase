package mssql

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/bytebase/omni/mssql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*ColumnMaximumVarcharLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MSSQL, storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH, &ColumnMaximumVarcharLengthAdvisor{})
}

// ColumnMaximumVarcharLengthAdvisor is the advisor checking for maximum varchar length..
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

	rule := &columnMaximumVarcharLengthRule{
		OmniBaseRule: OmniBaseRule{Level: level, Title: checkCtx.Rule.Type.String()},
		maximum:      int(numberPayload.Number),
		checkTypeString: map[string]bool{
			"varchar":  true,
			"nvarchar": true,
			"char":     true,
			"nchar":    true,
		},
	}
	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnMaximumVarcharLengthRule struct {
	OmniBaseRule
	checkTypeString map[string]bool
	maximum         int
}

func (*columnMaximumVarcharLengthRule) Name() string {
	return "ColumnMaximumVarcharLengthRule"
}

func (r *columnMaximumVarcharLengthRule) OnStatement(node ast.Node) {
	seen := make(map[*ast.DataType]bool)
	ast.Inspect(node, func(n ast.Node) bool {
		dt, ok := n.(*ast.DataType)
		if !ok || dt == nil {
			return true
		}
		if seen[dt] {
			return true
		}
		seen[dt] = true

		normalizedType := strings.ToLower(dt.Name)
		if !r.checkTypeString[normalizedType] {
			return true
		}

		currentLength := 0
		if dt.MaxLength {
			currentLength = math.MaxInt32
		} else if dt.Length != nil {
			if lit, ok := dt.Length.(*ast.Literal); ok {
				currentLength = int(lit.Ival)
			}
		}

		if currentLength > r.maximum {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.VarcharLengthExceedsLimit.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("The maximum varchar length is %d.", r.maximum),
				StartPosition: &storepb.Position{Line: r.LocToLine(dt.Loc)},
			})
		}
		return true
	})
}
