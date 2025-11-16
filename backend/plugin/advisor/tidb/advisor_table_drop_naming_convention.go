package tidb

import (
	"context"
	"fmt"
	"regexp"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

var (
	_ advisor.Advisor = (*TableDropNamingConventionAdvisor)(nil)
	_ ast.Visitor     = (*namingDropTableConventionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.SchemaRuleTableDropNamingConvention, &TableDropNamingConventionAdvisor{})
}

// TableDropNamingConventionAdvisor is the advisor checking the MySQLTableDropNamingConvention rule.
type TableDropNamingConventionAdvisor struct {
}

// Check checks for drop table naming convention.
func (*TableDropNamingConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	root, ok := checkCtx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, _, err := advisor.UnmarshalNamingRulePayloadAsRegexp(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	checker := &namingDropTableConventionChecker{
		level:  level,
		title:  string(checkCtx.Rule.Type),
		format: format,
	}
	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	return checker.adviceList, nil
}

type namingDropTableConventionChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	format     *regexp.Regexp
}

// Enter implements the ast.Visitor interface.
func (v *namingDropTableConventionChecker) Enter(in ast.Node) (ast.Node, bool) {
	if node, ok := in.(*ast.DropTableStmt); ok {
		for _, table := range node.Tables {
			if !v.format.MatchString(table.Name.O) {
				v.adviceList = append(v.adviceList, &storepb.Advice{
					Status:        v.level,
					Code:          code.TableDropNamingConventionMismatch.Int32(),
					Title:         v.title,
					Content:       fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", table.Name.O, v.format),
					StartPosition: common.ConvertANTLRLineToPosition(node.OriginTextPosition()),
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
