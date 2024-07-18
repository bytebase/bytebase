package pg

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NamingTableConventionAdvisor)(nil)
	_ ast.Visitor     = (*namingTableConventionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLNamingTableConvention, &NamingTableConventionAdvisor{})
}

// NamingTableConventionAdvisor is the advisor checking for table naming convention.
type NamingTableConventionAdvisor struct {
}

// Check checks for table naming convention.
func (*NamingTableConventionAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	stmts, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, maxLength, err := advisor.UnmarshalNamingRulePayloadAsRegexp(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingTableConventionChecker{
		level:     level,
		title:     string(ctx.Rule.Type),
		format:    format,
		maxLength: maxLength,
	}

	for _, stmt := range stmts {
		ast.Walk(checker, stmt)
	}

	return checker.adviceList, nil
}

type namingTableConventionChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string
	format     *regexp.Regexp
	maxLength  int
}

// Visit implements the ast.Visitor interface.
func (checker *namingTableConventionChecker) Visit(node ast.Node) ast.Visitor {
	var tableNames []string

	switch n := node.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		tableNames = append(tableNames, n.Name.Name)
	// ALTER TABLE RENAME TABLE
	case *ast.RenameTableStmt:
		tableNames = append(tableNames, n.NewName)
	}

	for _, tableName := range tableNames {
		if !checker.format.MatchString(tableName) {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:  checker.level,
				Code:    advisor.NamingTableConventionMismatch.Int32(),
				Title:   checker.title,
				Content: fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, checker.format),
				StartPosition: &storepb.Position{
					Line: int32(node.LastLine()),
				},
			})
		}
		if checker.maxLength > 0 && len(tableName) > checker.maxLength {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:  checker.level,
				Code:    advisor.NamingTableConventionMismatch.Int32(),
				Title:   checker.title,
				Content: fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, checker.maxLength),
				StartPosition: &storepb.Position{
					Line: int32(node.LastLine()),
				},
			})
		}
	}

	return checker
}
