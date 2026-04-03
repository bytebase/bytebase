package mysql

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnAutoIncrementInitialValueAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE, &ColumnAutoIncrementInitialValueAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE, &ColumnAutoIncrementInitialValueAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE, &ColumnAutoIncrementInitialValueAdvisor{})
}

// ColumnAutoIncrementInitialValueAdvisor is the advisor checking for auto-increment column initial value.
type ColumnAutoIncrementInitialValueAdvisor struct {
}

// Check checks for auto-increment column initial value.
func (*ColumnAutoIncrementInitialValueAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	numberPayload := checkCtx.Rule.GetNumberPayload()
	if numberPayload == nil {
		return nil, errors.New("number_payload is required for column auto increment initial value rule")
	}

	rule := &columnAutoIncrementInitialValueOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		value: int(numberPayload.Number),
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

type columnAutoIncrementInitialValueOmniRule struct {
	OmniBaseRule
	value int
}

func (*columnAutoIncrementInitialValueOmniRule) Name() string {
	return "ColumnAutoIncrementInitialValueRule"
}

func (r *columnAutoIncrementInitialValueOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	default:
	}
}

func (r *columnAutoIncrementInitialValueOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}
	r.checkOptions(tableName, n.Options, r.LocToLine(n.Loc))
}

func (r *columnAutoIncrementInitialValueOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Table)
	if tableName == "" {
		return
	}
	for _, cmd := range n.Commands {
		if cmd.Type == ast.ATTableOption && cmd.Option != nil && strings.EqualFold(cmd.Option.Name, "AUTO_INCREMENT") {
			r.checkOptionValue(tableName, cmd.Option.Value, r.LocToLine(n.Loc))
		}
	}
}

func (r *columnAutoIncrementInitialValueOmniRule) checkOptions(tableName string, opts []*ast.TableOption, line int32) {
	for _, opt := range opts {
		if strings.EqualFold(opt.Name, "AUTO_INCREMENT") {
			r.checkOptionValue(tableName, opt.Value, line)
		}
	}
}

func (r *columnAutoIncrementInitialValueOmniRule) checkOptionValue(tableName, valueStr string, line int32) {
	value, err := strconv.ParseUint(valueStr, 10, 0)
	if err != nil {
		return
	}
	if value != uint64(r.value) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.AutoIncrementColumnInitialValueNotMatch.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("The initial auto-increment value in table `%s` is %v, which doesn't equal %v", tableName, value, r.value),
			StartPosition: &storepb.Position{
				Line: line,
			},
		})
	}
}
