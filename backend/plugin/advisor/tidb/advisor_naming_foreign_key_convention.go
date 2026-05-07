package tidb

import (
	"context"
	"strings"

	"github.com/bytebase/omni/tidb/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NamingFKConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_INDEX_FK, &NamingFKConventionAdvisor{})
}

// NamingFKConventionAdvisor is the advisor checking for foreign key naming convention.
type NamingFKConventionAdvisor struct {
}

// Check checks for foreign key naming convention.
func (*NamingFKConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	return runNamingConventionRule(checkCtx, namingRuleConfig{
		mismatchCode:       code.NamingFKConventionMismatch,
		typeNoun:           "Foreign key",
		internalErrorTitle: "Internal error for foreign key naming convention rule",
	}, func(ostmt OmniStmt) []*indexMetaData {
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			return collectFKCreateTable(ostmt, n)
		case *ast.AlterTableStmt:
			return collectFKAlterTable(ostmt, n)
		}
		return nil
	})
}

func collectFKCreateTable(ostmt OmniStmt, n *ast.CreateTableStmt) []*indexMetaData {
	if n.Table == nil {
		return nil
	}
	tableName := n.Table.Name
	var res []*indexMetaData
	for _, constraint := range n.Constraints {
		// Nil-check before evaluating constraint.Loc.Start; otherwise the
		// line-computation argument would panic ahead of buildFKMetaData's
		// own nil guard. Mirrors the index/UK CreateTable collectors.
		if constraint == nil {
			continue
		}
		if metaData := buildFKMetaData(tableName, constraint, ostmt.AbsoluteLine(constraint.Loc.Start)); metaData != nil {
			res = append(res, metaData)
		}
	}
	return res
}

func collectFKAlterTable(ostmt OmniStmt, n *ast.AlterTableStmt) []*indexMetaData {
	if n.Table == nil {
		return nil
	}
	tableName := n.Table.Name
	stmtLine := ostmt.FirstTokenLine()
	var res []*indexMetaData
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		// FOREIGN KEY definitions land on ATAddConstraint only — no
		// ATAddIndex equivalent (mirroring mysql omni rule).
		if cmd.Type != ast.ATAddConstraint {
			continue
		}
		if metaData := buildFKMetaData(tableName, cmd.Constraint, stmtLine); metaData != nil {
			res = append(res, metaData)
		}
	}
	return res
}

func buildFKMetaData(tableName string, constraint *ast.Constraint, line int) *indexMetaData {
	if constraint == nil || constraint.Type != ast.ConstrForeignKey {
		return nil
	}
	referencingColumnList := constraint.Columns
	referencedTable := ""
	// constraint.RefTable can be nil even on FK constraints in pathological
	// inputs; guard defensively per the mysql omni analog.
	if constraint.RefTable != nil {
		referencedTable = constraint.RefTable.Name
	}
	referencedColumnList := constraint.RefColumns
	metaData := map[string]string{
		advisor.ReferencingTableNameTemplateToken:  tableName,
		advisor.ReferencingColumnNameTemplateToken: strings.Join(referencingColumnList, "_"),
		advisor.ReferencedTableNameTemplateToken:   referencedTable,
		advisor.ReferencedColumnNameTemplateToken:  strings.Join(referencedColumnList, "_"),
	}
	return &indexMetaData{
		indexName: constraint.Name,
		tableName: tableName,
		metaData:  metaData,
		line:      line,
	}
}
