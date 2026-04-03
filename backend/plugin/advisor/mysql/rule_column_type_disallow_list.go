package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnTypeDisallowListAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, &ColumnTypeDisallowListAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, &ColumnTypeDisallowListAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST, &ColumnTypeDisallowListAdvisor{})
}

// ColumnTypeDisallowListAdvisor is the advisor checking for column type restriction.
type ColumnTypeDisallowListAdvisor struct {
}

// Check checks for column type restriction.
func (*ColumnTypeDisallowListAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()

	typeRestriction := make(map[string]bool)
	for _, tp := range stringArrayPayload.List {
		typeRestriction[strings.ToUpper(tp)] = true
	}

	rule := &columnTypeDisallowListOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		typeRestriction: typeRestriction,
	}

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		rule.SetStatement(stmt.BaseLine(), stmt.Text)
		rule.OnStatement(node)
	}

	return rule.GetAdviceList(), nil
}

type columnTypeDisallowListOmniRule struct {
	OmniBaseRule
	typeRestriction map[string]bool
}

func (*columnTypeDisallowListOmniRule) Name() string {
	return "ColumnTypeDisallowListRule"
}

func (r *columnTypeDisallowListOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnTypeDisallowListOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}
	for _, col := range n.Columns {
		r.checkColumnType(tableName, col.Name, col.TypeName, col.Loc)
	}
}

func (r *columnTypeDisallowListOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}
	for _, cmd := range n.Commands {
		for _, col := range omniGetColumnsFromCmd(cmd) {
			r.checkColumnType(tableName, col.Name, col.TypeName, col.Loc)
		}
	}
}

func (r *columnTypeDisallowListOmniRule) checkColumnType(tableName, colName string, dt *ast.DataType, loc ast.Loc) {
	if dt == nil {
		return
	}
	columnType := strings.ToUpper(omniDataTypeNameCompact(dt))
	if _, exists := r.typeRestriction[columnType]; exists {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.DisabledColumnType.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Disallow column type %s but column `%s`.`%s` is", columnType, tableName, colName),
			StartPosition: &storepb.Position{
				Line: r.LocToLine(loc),
			},
		})
	}
}
