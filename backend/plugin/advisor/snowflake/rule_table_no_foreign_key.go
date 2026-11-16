// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"

	omniast "github.com/bytebase/omni/snowflake/ast"
	snowflakev1review "github.com/bytebase/omni/snowflake/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*TableNoForeignKeyAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY, &TableNoForeignKeyAdvisor{})
}

// TableNoForeignKeyAdvisor is the advisor checking for table disallow
// foreign key. The judgment lives in omni
// (snowflake/v1review.TableNoForeignKey); this adapter renders the findings
// as advice.
type TableNoForeignKeyAdvisor struct {
}

// Check checks for table disallow foreign key.
func (*TableNoForeignKeyAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stmts, nodes := tableStateStatements(checkCtx.ParsedStatements)
	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice
	for _, f := range snowflakev1review.TableNoForeignKey(nodes) {
		if f.Code != v1review.TableForeignKey {
			continue
		}
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.TableHasFK.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("FOREIGN KEY is not allowed in the table %s.", f.Name),
			StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine() + findingLine(stmt.Text, f)),
		})
	}
	return adviceList, nil
}

// tableStateStatements returns the statements with an omni AST and their
// nodes, index for index, for the rules that judge across statements.
func tableStateStatements(parsed []base.ParsedStatement) ([]base.ParsedStatement, []omniast.Node) {
	var stmts []base.ParsedStatement
	var nodes []omniast.Node
	for _, stmt := range parsed {
		if stmt.AST == nil {
			continue
		}
		node, ok := snowsqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		stmts = append(stmts, stmt)
		nodes = append(nodes, node)
	}
	return stmts, nodes
}
