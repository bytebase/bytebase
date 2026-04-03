package mysql

import (
	"context"
	"fmt"
	"slices"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_NO_NULL, &ColumnNoNullAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_NO_NULL, &ColumnNoNullAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_NO_NULL, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &columnNoNullOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		columnSet:     make(map[string]columnName),
		finalMetadata: checkCtx.FinalMetadata,
	}

	// Walk all statements to collect columns.
	RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule})

	// Generate advice after walking all statements.
	rule.generateAdvice()

	return rule.GetAdviceList(), nil
}

type columnName struct {
	tableName  string
	columnName string
	line       int
}

func (c columnName) name() string {
	return fmt.Sprintf("%s.%s", c.tableName, c.columnName)
}

type columnNoNullOmniRule struct {
	OmniBaseRule
	columnSet     map[string]columnName
	finalMetadata *model.DatabaseMetadata
}

func (*columnNoNullOmniRule) Name() string {
	return "ColumnNoNullRule"
}

func (r *columnNoNullOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnNoNullOmniRule) generateAdvice() {
	var columnList []columnName
	for _, column := range r.columnSet {
		columnList = append(columnList, column)
	}
	slices.SortFunc(columnList, func(a, b columnName) int {
		if a.line != b.line {
			if a.line < b.line {
				return -1
			}
			return 1
		}
		if a.columnName < b.columnName {
			return -1
		}
		if a.columnName > b.columnName {
			return 1
		}
		return 0
	})

	for _, column := range columnList {
		schema := r.finalMetadata.GetSchemaMetadata("")
		if schema == nil {
			continue
		}
		table := schema.GetTable(column.tableName)
		if table == nil {
			continue
		}
		col := table.GetColumn(column.columnName)
		if col != nil && col.GetProto().Nullable {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.ColumnCannotNull.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("`%s`.`%s` cannot have NULL value", column.tableName, column.columnName),
				StartPosition: common.ConvertANTLRLineToPosition(column.line),
			})
		}
	}
}

func (r *columnNoNullOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	for _, col := range n.Columns {
		if col == nil {
			continue
		}
		c := columnName{
			tableName:  tableName,
			columnName: col.Name,
			line:       r.BaseLine + int(r.LocToLine(col.Loc)),
		}
		if _, exists := r.columnSet[c.name()]; !exists {
			r.columnSet[c.name()] = c
		}
	}
}

func (r *columnNoNullOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		var columns []string
		switch cmd.Type {
		case ast.ATAddColumn:
			if cmd.Column != nil {
				columns = append(columns, cmd.Column.Name)
			}
			for _, col := range cmd.Columns {
				columns = append(columns, col.Name)
			}
		case ast.ATChangeColumn:
			// Only care about the new column name.
			if cmd.Column != nil {
				columns = append(columns, cmd.Column.Name)
			}
		default:
		}
		for _, column := range columns {
			c := columnName{
				tableName:  tableName,
				columnName: column,
				line:       r.BaseLine + int(r.LocToLine(cmd.Loc)),
			}
			if _, exists := r.columnSet[c.name()]; !exists {
				r.columnSet[c.name()] = c
			}
		}
	}
}
