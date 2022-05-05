package mysql

import (
	"fmt"
	"regexp"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/pingcap/tidb/parser/ast"
)

var (
	_ advisor.Advisor = (*TableNamingConventionAdvisor)(nil)
)

func init() {
	advisor.Register(db.MySQL, advisor.MySQLTableNamingConvention, &TableNamingConventionAdvisor{})
	advisor.Register(db.TiDB, advisor.MySQLTableNamingConvention, &TableNamingConventionAdvisor{})
}

// TableNamingConventionAdvisor is the advisor checking for table naming convention.
type TableNamingConventionAdvisor struct {
}

// Check checks for table naming convention.
func (adv *TableNamingConventionAdvisor) Check(ctx advisor.Context, statement string) ([]advisor.Advice, error) {
	p := newParser()

	root, _, err := p.Parse(statement, ctx.Charset, ctx.Collation)
	if err != nil {
		return []advisor.Advice{
			{
				Status:  advisor.Error,
				Title:   "Syntax error",
				Content: err.Error(),
			},
		}, nil
	}
	level, err := advisor.NewStatusBySchemaReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	payload, err := api.UnmarshalNamingRulePayload(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	format, err := regexp.Compile(payload.Format)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regular expression: %v, err: %v", payload.Format, err)
	}
	checker := &tableNamingConventionChecker{
		level:  level,
		format: format,
	}
	for _, stmtNode := range root {
		(stmtNode).Accept(checker)
	}

	if len(checker.advisorList) == 0 {
		checker.advisorList = append(checker.advisorList, advisor.Advice{
			Status:  advisor.Success,
			Code:    common.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.advisorList, nil
}

type tableNamingConventionChecker struct {
	advisorList []advisor.Advice
	level       advisor.Status
	format      *regexp.Regexp
}

func (v *tableNamingConventionChecker) Enter(in ast.Node) (ast.Node, bool) {
	code := common.Ok
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		// Original string
		if !v.format.MatchString(node.Table.Name.O) {
			code = common.TableNamingConventionMismatch
		}
	// ALTER TABLE
	case *ast.AlterTableStmt:
		for _, spec := range node.Specs {
			// RENAME TABLE
			if spec.Tp == ast.AlterTableRenameTable {
				if !v.format.MatchString(spec.NewTable.Name.O) {
					code = common.TableNamingConventionMismatch
					break
				}
			}
		}
	// RENAME TABLE
	case *ast.RenameTableStmt:
		for _, table2Table := range node.TableToTables {
			if !v.format.MatchString(table2Table.NewTable.Name.O) {
				code = common.TableNamingConventionMismatch
				break
			}
		}
	}

	if code != common.Ok {
		v.advisorList = append(v.advisorList, advisor.Advice{
			Status:  v.level,
			Code:    code,
			Title:   "Mismatch table naming convention",
			Content: fmt.Sprintf("%q mismatches table naming convention", in.Text()),
		})
	}
	return in, false
}

func (v *tableNamingConventionChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
