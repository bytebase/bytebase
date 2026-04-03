package mysql

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*ColumnDisallowDropInIndexAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX, &ColumnDisallowDropInIndexAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX, &ColumnDisallowDropInIndexAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX, &ColumnDisallowDropInIndexAdvisor{})
}

// ColumnDisallowDropInIndexAdvisor is the advisor checking for disallow DROP COLUMN in index.
type ColumnDisallowDropInIndexAdvisor struct {
}

// Check checks for disallow Drop COLUMN in index statement.
func (*ColumnDisallowDropInIndexAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnDisallowDropInIndexOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		tables:           make(tableState),
		originalMetadata: checkCtx.OriginalMetadata,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type columnDisallowDropInIndexOmniRule struct {
	OmniBaseRule
	tables           tableState
	originalMetadata *model.DatabaseMetadata
}

func (*columnDisallowDropInIndexOmniRule) Name() string {
	return "ColumnDisallowDropInIndexRule"
}

func (r *columnDisallowDropInIndexOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnDisallowDropInIndexOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name

	for _, constraint := range n.Constraints {
		if constraint == nil {
			continue
		}
		if constraint.Type != ast.ConstrIndex {
			continue
		}
		for _, col := range constraint.Columns {
			if r.tables[tableName] == nil {
				r.tables[tableName] = make(columnSet)
			}
			r.tables[tableName][col] = true
		}
	}
}

func (r *columnDisallowDropInIndexOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name

	for _, cmd := range n.Commands {
		if cmd == nil || cmd.Type != ast.ATDropColumn {
			continue
		}

		// Load index columns from metadata.
		table := r.originalMetadata.GetSchemaMetadata("").GetTable(tableName)
		if table != nil {
			if r.tables[tableName] == nil {
				r.tables[tableName] = make(columnSet)
			}
			for _, indexColumn := range table.ListIndexes() {
				for _, column := range indexColumn.GetProto().GetExpressions() {
					r.tables[tableName][column] = true
				}
			}
		}

		columnName := cmd.Name
		if _, exists := r.tables[tableName][columnName]; exists {
			r.AddAdvice(&storepb.Advice{
				Status:        r.Level,
				Code:          code.DropIndexColumn.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("`%s`.`%s` cannot drop index column", tableName, columnName),
				StartPosition: common.ConvertANTLRLineToPosition(int(r.LocToLine(cmd.Loc))),
			})
		}
	}
}
