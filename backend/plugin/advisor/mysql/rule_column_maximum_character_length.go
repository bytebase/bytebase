package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnMaximumCharacterLengthAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH, &ColumnMaximumCharacterLengthAdvisor{})
}

// ColumnMaximumCharacterLengthAdvisor is the advisor checking for max character length.
type ColumnMaximumCharacterLengthAdvisor struct {
}

// Check checks for maximum character length.
func (*ColumnMaximumCharacterLengthAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for column maximum character length rule")
	}

	rule := &columnMaximumCharacterLengthRule{
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
		node, ok := mysqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		rule.SetStatement(stmt.BaseLine(), stmt.Text)
		rule.OnStatement(node)
	}

	return rule.GetAdviceList(), nil
}

type columnMaximumCharacterLengthRule struct {
	BaseRule
	maximum int
}

func (*columnMaximumCharacterLengthRule) Name() string {
	return "ColumnMaximumCharacterLengthRule"
}

func (r *columnMaximumCharacterLengthRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnMaximumCharacterLengthRule) checkCreateTable(n *ast.CreateTableStmt) {
	tableName := tableName(n.Table)
	if tableName == "" {
		return
	}
	for _, col := range n.Columns {
		charLength := getCharLengthFrom(col.TypeName)
		if r.maximum > 0 && charLength > r.maximum {
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Code:    code.CharLengthExceedsLimit.Int32(),
				Title:   r.Title,
				Content: fmt.Sprintf("The length of the CHAR column `%s.%s` is bigger than %d, please use VARCHAR instead", tableName, col.Name, r.maximum),
				StartPosition: &storepb.Position{
					Line: r.LocToLine(col.Loc),
				},
			})
		}
	}
}

func (r *columnMaximumCharacterLengthRule) checkAlterTable(n *ast.AlterTableStmt) {
	tableName := tableName(n.Table)
	if tableName == "" {
		return
	}
	for _, cmd := range n.Commands {
		cols := getColumnsFromCmd(cmd)
		for _, col := range cols {
			charLength := getCharLengthFrom(col.TypeName)
			if r.maximum > 0 && charLength > r.maximum {
				r.AddAdvice(&storepb.Advice{
					Status:  r.Level,
					Code:    code.CharLengthExceedsLimit.Int32(),
					Title:   r.Title,
					Content: fmt.Sprintf("The length of the CHAR column `%s.%s` is bigger than %d, please use VARCHAR instead", tableName, col.Name, r.maximum),
					StartPosition: &storepb.Position{
						Line: r.LocToLine(col.Loc),
					},
				})
			}
		}
	}
}

func getCharLengthFrom(dt *ast.DataType) int {
	if dt == nil {
		return 0
	}
	if !strings.EqualFold(dt.Name, "CHAR") {
		return 0
	}
	// MySQL default: CHAR without length = CHAR(1)
	if dt.Length == 0 {
		return 1
	}
	return dt.Length
}
