package tidb

import (
	"context"
	"fmt"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
	_ ast.Visitor     = (*columnNoNullChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.SchemaRuleColumnNotNull, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &columnNoNullChecker{
		level: level,
		title: string(checkCtx.Rule.Type),
	}

	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	return checker.adviceList, nil
}

type columnNoNullChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
}

// Enter implements the ast.Visitor interface.
func (checker *columnNoNullChecker) Enter(in ast.Node) (ast.Node, bool) {
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		for _, column := range node.Cols {
			if canNull(column) {
				checker.adviceList = append(checker.adviceList, &storepb.Advice{
					Status:        checker.level,
					Code:          advisor.ColumnCannotNull.Int32(),
					Title:         checker.title,
					Content:       fmt.Sprintf("`%s`.`%s` cannot have NULL value", node.Table.Name.O, column.Name.Name.O),
					StartPosition: common.ConvertANTLRLineToPosition(column.OriginTextPosition()),
				})
			}
		}
	// ALTER TABLE
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			switch spec.Tp {
			// ADD COLUMNS
			case ast.AlterTableAddColumns:
				for _, column := range spec.NewColumns {
					if canNull(column) {
						checker.adviceList = append(checker.adviceList, &storepb.Advice{
							Status:        checker.level,
							Code:          advisor.ColumnCannotNull.Int32(),
							Title:         checker.title,
							Content:       fmt.Sprintf("`%s`.`%s` cannot have NULL value", node.Table.Name.O, column.Name.Name.O),
							StartPosition: common.ConvertANTLRLineToPosition(node.OriginTextPosition()),
						})
					}
				}
			// CHANGE COLUMN
			case ast.AlterTableChangeColumn:
				if len(spec.NewColumns) > 0 && canNull(spec.NewColumns[0]) {
					checker.adviceList = append(checker.adviceList, &storepb.Advice{
						Status:        checker.level,
						Code:          advisor.ColumnCannotNull.Int32(),
						Title:         checker.title,
						Content:       fmt.Sprintf("`%s`.`%s` cannot have NULL value", node.Table.Name.O, spec.NewColumns[0].Name.Name.O),
						StartPosition: common.ConvertANTLRLineToPosition(node.OriginTextPosition()),
					})
				}
			default:
				// Skip other alter table specification types
			}
		}
	}

	return in, false
}

// Leave implements the ast.Visitor interface.
func (*columnNoNullChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

func canNull(column *ast.ColumnDef) bool {
	for _, option := range column.Options {
		if option.Tp == ast.ColumnOptionNotNull || option.Tp == ast.ColumnOptionPrimaryKey {
			return false
		}
	}
	return true
}
