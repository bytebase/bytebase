package tidb

import (
	"fmt"
	"regexp"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*TableDropNamingConventionAdvisor)(nil)
	_ ast.Visitor     = (*namingDropTableConventionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLTableDropNamingConvention, &TableDropNamingConventionAdvisor{})
}

// TableDropNamingConventionAdvisor is the advisor checking the MySQLTableDropNamingConvention rule.
type TableDropNamingConventionAdvisor struct {
}

// Check checks for drop table naming convention.
func (*TableDropNamingConventionAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	root, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, _, err := advisor.UnmarshalNamingRulePayloadAsRegexp(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &namingDropTableConventionChecker{
		level:  level,
		title:  string(ctx.Rule.Type),
		format: format,
	}
	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type namingDropTableConventionChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	format     *regexp.Regexp
}

// Enter implements the ast.Visitor interface.
func (v *namingDropTableConventionChecker) Enter(in ast.Node) (ast.Node, bool) {
	if node, ok := in.(*ast.DropTableStmt); ok {
		for _, table := range node.Tables {
			if !v.format.MatchString(table.Name.O) {
				v.adviceList = append(v.adviceList, advisor.Advice{
					Status:  v.level,
					Code:    advisor.TableDropNamingConventionMismatch,
					Title:   v.title,
					Content: fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", table.Name.O, v.format),
					Line:    node.OriginTextPosition(),
				})
			}
		}
	}

	return in, false
}

// Leave implements the ast.Visitor interface.
func (*namingDropTableConventionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
