package mariadb

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mariadb/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mariadbparser "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
)

var (
	_ advisor.Advisor = (*ColumnMaximumVarcharLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH, &ColumnMaximumVarcharLengthAdvisor{})
}

// ColumnMaximumVarcharLengthAdvisor is the advisor checking for max varchar length.
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
		return nil, errors.New("number_payload is required for column maximum varchar length rule")
	}

	rule := &columnMaximumVarcharLengthRule{
		BaseRule: BaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		maximum: int(numberPayload.Number),
	}

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mariadbparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		rule.SetStatement(stmt.BaseLine(), stmt.Text)
		rule.OnStatement(node)
	}

	return rule.GetAdviceList(), nil
}

type columnMaximumVarcharLengthRule struct {
	BaseRule
	maximum int
}

func (r *columnMaximumVarcharLengthRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnMaximumVarcharLengthRule) checkCreateTable(n *ast.CreateTableStmt) {
	tableName := tableName(n.Table)
	if tableName == "" {
		return
	}
	for _, col := range n.Columns {
		length := getVarcharLengthFrom(col.TypeName)
		if r.maximum > 0 && length > r.maximum {
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    code.VarcharLengthExceedsLimit.Int32(),
				Title:   r.Title,
				Content: fmt.Sprintf("The length of the VARCHAR column `%s.%s` is bigger than %d", tableName, col.Name, r.maximum),
				StartPosition: &storepb.Position{
					Line: r.LocToLine(col.Loc),
				},
			})
		}
	}
}

func (r *columnMaximumVarcharLengthRule) checkAlterTable(n *ast.AlterTableStmt) {
	tableName := tableName(n.Table)
	if tableName == "" {
		return
	}
	for _, cmd := range n.Commands {
		cols := getColumnsFromCmd(cmd)
		for _, col := range cols {
			length := getVarcharLengthFrom(col.TypeName)
			if r.maximum > 0 && length > r.maximum {
				r.AddAdvice(&storepb.Advice{
					Status:  r.Level,
					Code:    code.VarcharLengthExceedsLimit.Int32(),
					Title:   r.Title,
					Content: fmt.Sprintf("The length of the VARCHAR column `%s.%s` is bigger than %d", tableName, col.Name, r.maximum),
					StartPosition: &storepb.Position{
						Line: r.LocToLine(col.Loc),
					},
				})
			}
		}
	}
}

func getVarcharLengthFrom(dt *ast.DataType) int {
	if dt == nil {
		return 0
	}
	if !strings.EqualFold(dt.Name, "VARCHAR") {
		return 0
	}
	// MySQL default: VARCHAR without length = VARCHAR(1)
	if dt.Length == 0 {
		return 1
	}
	return dt.Length
}
