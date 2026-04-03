package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*TableNoDuplicateIndexAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_TABLE_NO_DUPLICATE_INDEX, &TableNoDuplicateIndexAdvisor{})
}

// TableNoDuplicateIndexAdvisor is the advisor checking for no duplicate index in table.
type TableNoDuplicateIndexAdvisor struct {
}

// Check checks for no duplicate index in table.
func (*TableNoDuplicateIndexAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	rule := &tableNoDuplicateIndexOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type duplicateIndex struct {
	indexName string
	indexType string
	unique    bool
	fulltext  bool
	spatial   bool
	table     string
	columns   []string
	line      int
}

type tableNoDuplicateIndexOmniRule struct {
	OmniBaseRule
	indexList []duplicateIndex
}

func (*tableNoDuplicateIndexOmniRule) Name() string {
	return "TableNoDuplicateIndexRule"
}

func (r *tableNoDuplicateIndexOmniRule) OnStatement(node ast.Node) {
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

func (r *tableNoDuplicateIndexOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
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
	if index := hasDuplicateIndexes(r.indexList); index != nil {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.DuplicateIndexInTable.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("`%s` has duplicate index `%s`", tableName, index.indexName),
			StartPosition: common.ConvertANTLRLineToPosition(index.line),
		})
	}
}

func (r *tableNoDuplicateIndexOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
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
	if index := hasDuplicateIndexes(r.indexList); index != nil {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.DuplicateIndexInTable.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("`%s` has duplicate index `%s`", tableName, index.indexName),
			StartPosition: common.ConvertANTLRLineToPosition(index.line),
		})
	}
}

func (r *tableNoDuplicateIndexOmniRule) handleConstraint(tableName string, constraint *ast.Constraint, line int32) {
	switch constraint.Type {
	case ast.ConstrPrimaryKey, ast.ConstrUnique, ast.ConstrIndex, ast.ConstrFulltextIndex, ast.ConstrSpatialIndex:
	default:
		return
	}

	index := duplicateIndex{
		indexType: "BTREE",
		line:      r.BaseLine + int(line),
		table:     tableName,
		columns:   constraint.Columns,
		indexName: constraint.Name,
	}
	if constraint.IndexType != "" {
		index.indexType = constraint.IndexType
	}
	switch constraint.Type {
	case ast.ConstrUnique:
		index.unique = true
	case ast.ConstrFulltextIndex:
		index.fulltext = true
	case ast.ConstrSpatialIndex:
		index.spatial = true
	default:
	}
	r.indexList = append(r.indexList, index)
}

func (r *tableNoDuplicateIndexOmniRule) checkCreateIndex(n *ast.CreateIndexStmt) {
	if n.Table == nil {
		return
	}

	tableName := n.Table.Name
	index := duplicateIndex{
		indexType: "BTREE",
		line:      r.BaseLine + int(r.LocToLine(n.Loc)),
		table:     tableName,
		indexName: n.IndexName,
		columns:   omniIndexColumns(n.Columns),
	}
	if n.IndexType != "" {
		index.indexType = n.IndexType
	}
	if n.Unique {
		index.unique = true
	} else if n.Fulltext {
		index.fulltext = true
	} else if n.Spatial {
		index.spatial = true
	}

	r.indexList = append(r.indexList, index)
	if dup := hasDuplicateIndexes(r.indexList); dup != nil {
		r.AddAdviceAbsolute(&storepb.Advice{
			Status:        r.Level,
			Code:          code.DuplicateIndexInTable.Int32(),
			Title:         r.Title,
			Content:       fmt.Sprintf("`%s` has duplicate index `%s`", tableName, dup.indexName),
			StartPosition: common.ConvertANTLRLineToPosition(dup.line),
		})
	}
}

// hasDuplicateIndexes returns the first duplicate index if found, otherwise nil.
func hasDuplicateIndexes(indexList []duplicateIndex) *duplicateIndex {
	seen := make(map[string]struct{})
	for _, index := range indexList {
		key := indexKey(index)
		if _, exists := seen[key]; exists {
			return &index
		}
		seen[key] = struct{}{}
	}
	return nil
}

// indexKey returns a string key for the index with the index type and columns.
func indexKey(index duplicateIndex) string {
	parts := []string{}
	if index.unique {
		parts = append(parts, "unique")
	}
	if index.fulltext {
		parts = append(parts, "fulltext")
	}
	if index.spatial {
		parts = append(parts, "spatial")
	}
	parts = append(parts, index.indexType)
	parts = append(parts, index.table)
	parts = append(parts, index.columns...)
	return strings.Join(parts, "-")
}
