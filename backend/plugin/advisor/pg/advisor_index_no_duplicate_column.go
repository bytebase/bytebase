package pg

import (
	"context"
	"fmt"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*IndexNoDuplicateColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, &IndexNoDuplicateColumnAdvisor{})
}

// IndexNoDuplicateColumnAdvisor is the advisor checking for no duplicate columns in index.
type IndexNoDuplicateColumnAdvisor struct {
}

// Check checks for no duplicate columns in index.
func (*IndexNoDuplicateColumnAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &indexNoDuplicateColumnRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type indexNoDuplicateColumnRule struct {
	OmniBaseRule
}

func (*indexNoDuplicateColumnRule) Name() string {
	return "index_no_duplicate_column"
}

func (r *indexNoDuplicateColumnRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.IndexStmt:
		r.handleIndexStmt(n)
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	default:
	}
}

func (r *indexNoDuplicateColumnRule) handleIndexStmt(n *ast.IndexStmt) {
	indexName := n.Idxname
	tableName := omniTableName(n.Relation)
	columns := omniIndexColumns(n)
	if dupCol := findDuplicate(columns); dupCol != "" {
		r.addDuplicateAdvice("INDEX", indexName, tableName, dupCol, r.ContentStartLine())
	}
}

func (r *indexNoDuplicateColumnRule) handleCreateStmt(n *ast.CreateStmt) {
	tableName := omniTableName(n.Relation)
	_, constraints := omniTableElements(n)
	for _, c := range constraints {
		r.checkConstraint(c, tableName)
	}
}

func (r *indexNoDuplicateColumnRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Relation)
	for _, cmd := range omniAlterTableCmds(n) {
		if cmd.Subtype == int(ast.AT_AddConstraint) {
			if c, ok := cmd.Def.(*ast.Constraint); ok {
				r.checkConstraint(c, tableName)
			}
		}
	}
}

func (r *indexNoDuplicateColumnRule) checkConstraint(c *ast.Constraint, tableName string) {
	var columns []string
	var constraintType, keyword string
	switch c.Contype {
	case ast.CONSTR_PRIMARY:
		constraintType = "PRIMARY KEY"
		keyword = "PRIMARY KEY"
		columns = omniConstraintColumns(c)
	case ast.CONSTR_UNIQUE:
		constraintType = "UNIQUE KEY"
		keyword = "UNIQUE"
		columns = omniConstraintColumns(c)
	case ast.CONSTR_FOREIGN:
		constraintType = "FOREIGN KEY"
		keyword = "FOREIGN KEY"
		columns = omniListStrings(c.FkAttrs)
	default:
		return
	}
	if dupCol := findDuplicate(columns); dupCol != "" {
		r.addDuplicateAdvice(constraintType, c.Conname, tableName, dupCol, r.FindLineByName(keyword))
	}
}

func (r *indexNoDuplicateColumnRule) addDuplicateAdvice(constraintType, constraintName, tableName, duplicateColumn string, line int32) {
	r.AddAdvice(&storepb.Advice{
		Status:  r.Level,
		Code:    code.DuplicateColumnInIndex.Int32(),
		Title:   r.Title,
		Content: fmt.Sprintf("%s %q has duplicate column %q.%q", constraintType, constraintName, tableName, duplicateColumn),
		StartPosition: &storepb.Position{
			Line:   line,
			Column: 0,
		},
	})
}

func findDuplicate(columns []string) string {
	seen := make(map[string]bool)
	for _, col := range columns {
		if seen[col] {
			return col
		}
		seen[col] = true
	}
	return ""
}
