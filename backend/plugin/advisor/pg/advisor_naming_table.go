package pg

import (
	"context"
	"fmt"
	"regexp"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
)

var (
	_ advisor.Advisor = (*NamingTableConventionAdvisor)(nil)
	_ ast.Visitor     = (*namingTableConventionChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleTableNaming, &NamingTableConventionAdvisor{})
}

// NamingTableConventionAdvisor is the advisor checking for table naming convention.
type NamingTableConventionAdvisor struct {
}

// Check checks for table naming convention.
func (*NamingTableConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, maxLength, err := advisor.UnmarshalNamingRulePayloadAsRegexp(checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingTableConventionChecker{
		level:     level,
		title:     string(checkCtx.Rule.Type),
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
				Status:        checker.level,
				Code:          advisor.NamingTableConventionMismatch.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, tableName, checker.format),
				StartPosition: common.ConvertPGParserLineToPosition(node.LastLine()),
			})
		}
		if checker.maxLength > 0 && len(tableName) > checker.maxLength {
			checker.adviceList = append(checker.adviceList, &storepb.Advice{
				Status:        checker.level,
				Code:          advisor.NamingTableConventionMismatch.Int32(),
				Title:         checker.title,
				Content:       fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", tableName, checker.maxLength),
				StartPosition: common.ConvertPGParserLineToPosition(node.LastLine()),
			})
		}
	}

	return checker
}
