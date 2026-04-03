package mysql

import (
	"context"
	"fmt"
	"regexp"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*IndexNoDuplicateColumnAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, &IndexNoDuplicateColumnAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, &IndexNoDuplicateColumnAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN, &IndexNoDuplicateColumnAdvisor{})
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

	rule := &indexNoDuplicateColumnOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type indexNoDuplicateColumnOmniRule struct {
	OmniBaseRule
}

func (*indexNoDuplicateColumnOmniRule) Name() string {
	return "IndexNoDuplicateColumnRule"
}

func (r *indexNoDuplicateColumnOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	case *ast.CreateIndexStmt:
		r.checkCreateIndex(n)
	default:
	}
}

func (r *indexNoDuplicateColumnOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	for _, constraint := range n.Constraints {
		if constraint == nil {
			continue
		}
		r.handleConstraint(tableName, constraint, r.LocToLine(constraint.Loc))
	}
}

func (r *indexNoDuplicateColumnOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		switch cmd.Type {
		case ast.ATAddConstraint, ast.ATAddIndex:
			if cmd.Constraint != nil {
				r.handleConstraint(tableName, cmd.Constraint, r.LocToLine(n.Loc))
			}
		default:
		}
	}
}

func (r *indexNoDuplicateColumnOmniRule) checkCreateIndex(n *ast.CreateIndexStmt) {
	if n.Table == nil {
		return
	}
	if n.Fulltext || n.Spatial {
		return
	}

	tableName := n.Table.Name
	indexName := n.IndexName

	indexType := "INDEX "
	if n.Unique {
		indexType = "UNIQUE INDEX "
	}

	columnList := omniIndexColumns(n.Columns)
	if column, duplicate := hasDuplicateColumn(columnList); duplicate {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.DuplicateColumnInIndex.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("%s`%s` has duplicate column `%s`.`%s`", indexType, indexName, tableName, column),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(r.LocToLine(n.Loc))),
		})
	}
}

func (r *indexNoDuplicateColumnOmniRule) handleConstraint(tableName string, constraint *ast.Constraint, line int32) {
	var columnList []string
	indexType := ""
	switch constraint.Type {
	case ast.ConstrPrimaryKey:
		columnList = constraint.Columns
		indexType = "PRIMARY KEY "
	case ast.ConstrUnique:
		columnList = constraint.Columns
		indexType = "UNIQUE KEY "
	case ast.ConstrIndex:
		columnList = constraint.Columns
		indexType = "INDEX "
	case ast.ConstrForeignKey:
		columnList = constraint.Columns
		indexType = "FOREIGN KEY "
	case ast.ConstrFulltextIndex, ast.ConstrSpatialIndex:
		return
	default:
		return
	}

	indexName := constraint.Name
	// Workaround: omni parser doesn't capture PRIMARY KEY constraint names.
	// Extract from statement text as a fallback.
	if indexName == "" && constraint.Type == ast.ConstrPrimaryKey {
		indexName = extractPKNameFromText(r.StmtText)
	}
	if column, duplicate := hasDuplicateColumn(columnList); duplicate {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.DuplicateColumnInIndex.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("%s`%s` has duplicate column `%s`.`%s`", indexType, indexName, tableName, column),
			StartPosition: common.ConvertANTLRLineToPosition(r.BaseLine + int(line)),
		})
	}
}

// primaryKeyNameRe extracts the optional constraint name from a PRIMARY KEY clause.
// Matches patterns like: PRIMARY KEY pk_a (...), PRIMARY KEY `pk_a` (...)
var primaryKeyNameRe = regexp.MustCompile(`(?i)PRIMARY\s+KEY\s+` + "`?" + `(\w+)` + "`?" + `\s*\(`)

// extractPKNameFromText extracts the PRIMARY KEY constraint name from the statement text.
// Returns empty string if no name is found.
func extractPKNameFromText(stmtText string) string {
	m := primaryKeyNameRe.FindStringSubmatch(stmtText)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

func hasDuplicateColumn(keyList []string) (string, bool) {
	listMap := make(map[string]struct{})
	for _, keyName := range keyList {
		if _, exists := listMap[keyName]; exists {
			return keyName, true
		}
		listMap[keyName] = struct{}{}
	}
	return "", false
}
