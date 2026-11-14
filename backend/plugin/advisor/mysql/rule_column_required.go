package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	mysqlv1review "github.com/bytebase/omni/mysql/v1review"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

var (
	_ advisor.Advisor = (*ColumnRequirementAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_COLUMN_REQUIRED, &ColumnRequirementAdvisor{})
}

// ColumnRequirementAdvisor is the advisor checking for column requirement.
// The judgment lives in omni (mysql/v1review.ColumnRequired); this adapter
// passes the required columns and renders one advice per table.
type ColumnRequirementAdvisor struct {
}

// Check checks for the column requirement.
func (*ColumnRequirementAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	if stringArrayPayload == nil {
		return nil, errors.New("string_array_payload is required for column required rule")
	}

	stmts, nodes := parsedNodes(checkCtx.ParsedStatements)
	findings := mysqlv1review.ColumnRequired(nodes, stringArrayPayload.List)

	var adviceList []*storepb.Advice
	for i := 0; i < len(findings); {
		f := findings[i]
		var missingColumns []string
		for ; i < len(findings) && findings[i].Table == f.Table; i++ {
			missingColumns = append(missingColumns, findings[i].Name)
		}
		stmt := stmts[f.Statement]
		adviceList = append(adviceList, &storepb.Advice{
			Status:  level,
			Code:    code.NoRequiredColumn.Int32(),
			Title:   checkCtx.Rule.Type.String(),
			Content: fmt.Sprintf("Table `%s` requires columns: %s", f.Table, strings.Join(missingColumns, ", ")),
			StartPosition: &storepb.Position{
				Line:   int32(stmt.BaseLine() + int(findingLine(stmt.Text, f))),
				Column: 0,
			},
		})
	}
	return adviceList, nil
}

// parsedNodes returns the statements that carry an omni node and their nodes,
// index for index, for a rule that judges the statements together.
func parsedNodes(parsed []base.ParsedStatement) ([]base.ParsedStatement, []ast.Node) {
	var stmts []base.ParsedStatement
	var nodes []ast.Node
	for _, stmt := range parsed {
		if stmt.AST == nil {
			continue
		}
		node, ok := mysqlparser.GetNode(stmt.AST)
		if !ok {
			continue
		}
		stmts = append(stmts, stmt)
		nodes = append(nodes, node)
	}
	return stmts, nodes
}
