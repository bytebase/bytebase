package tidb

import (
	"context"
	"strings"

	"github.com/bytebase/omni/tidb/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*NamingUKConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_INDEX_UK, &NamingUKConventionAdvisor{})
}

// NamingUKConventionAdvisor is the advisor checking for unique key naming convention.
type NamingUKConventionAdvisor struct {
}

// Check checks for unique key naming convention.
func (*NamingUKConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	// Capture originalMetadata in a local so the collector closure can read
	// it without runNamingConventionRule needing advisor.Context in its
	// callback signature — the FK rule doesn't need metadata, so keeping
	// the helper signature uniform is cleaner than threading a Context arg.
	originalMetadata := checkCtx.OriginalMetadata
	return runNamingConventionRule(checkCtx, namingRuleConfig{
		mismatchCode:       code.NamingUKConventionMismatch,
		typeNoun:           "Unique key",
		internalErrorTitle: "Internal error for unique key naming convention rule",
	}, func(ostmt OmniStmt) []*indexMetaData {
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			return collectUKCreateTable(ostmt, n)
		case *ast.AlterTableStmt:
			return collectUKAlterTable(ostmt, n, originalMetadata)
		case *ast.CreateIndexStmt:
			return collectUKCreateIndex(ostmt, n)
		}
		return nil
	})
}

// Pingcap had three distinct constraint enums (ConstraintUniq /
// ConstraintUniqKey / ConstraintUniqIndex) for the syntactic forms
// `UNIQUE`, `UNIQUE KEY`, and `UNIQUE INDEX`. Omni unifies all three under
// `ast.ConstrUnique`. The single switch arm here covers all three forms.
// Per Phase 1.5 cumulative shape-divergence table item #2.
func collectUKCreateTable(ostmt OmniStmt, n *ast.CreateTableStmt) []*indexMetaData {
	if n.Table == nil {
		return nil
	}
	tableName := n.Table.Name
	var res []*indexMetaData
	for _, constraint := range n.Constraints {
		if constraint == nil || constraint.Type != ast.ConstrUnique {
			continue
		}
		columnList := constraint.Columns
		if len(columnList) == 0 {
			columnList = omniIndexColumns(constraint.IndexColumns)
		}
		metaData := map[string]string{
			advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
			advisor.TableNameTemplateToken:  tableName,
		}
		res = append(res, &indexMetaData{
			indexName: constraint.Name,
			tableName: tableName,
			metaData:  metaData,
			line:      ostmt.AbsoluteLine(constraint.Loc.Start),
		})
	}
	return res
}

func collectUKAlterTable(ostmt OmniStmt, n *ast.AlterTableStmt, originalMetadata *model.DatabaseMetadata) []*indexMetaData {
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
		switch cmd.Type {
		// Mirror mysql omni: ATAddIndex covers `ALTER TABLE ... ADD UNIQUE
		// INDEX uk (col)`, ATAddConstraint covers `ADD CONSTRAINT uk UNIQUE`.
		case ast.ATAddConstraint, ast.ATAddIndex:
			if cmd.Constraint == nil || cmd.Constraint.Type != ast.ConstrUnique {
				continue
			}
			columnList := cmd.Constraint.Columns
			if len(columnList) == 0 {
				columnList = omniIndexColumns(cmd.Constraint.IndexColumns)
			}
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			res = append(res, &indexMetaData{
				indexName: cmd.Constraint.Name,
				tableName: tableName,
				metaData:  metaData,
				line:      stmtLine,
			})
		case ast.ATRenameIndex:
			schema := originalMetadata.GetSchemaMetadata("")
			if schema == nil {
				continue
			}
			index := schema.GetIndex(cmd.Name)
			if index == nil {
				continue
			}
			if !index.GetProto().GetUnique() {
				// Non-unique index naming convention is handled by
				// advisor_naming_index_convention.go.
				continue
			}
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(index.GetProto().GetExpressions(), "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			res = append(res, &indexMetaData{
				indexName: cmd.NewName,
				tableName: tableName,
				metaData:  metaData,
				line:      stmtLine,
			})
		default:
		}
	}
	return res
}

func collectUKCreateIndex(ostmt OmniStmt, n *ast.CreateIndexStmt) []*indexMetaData {
	if !n.Unique {
		return nil
	}
	if n.Table == nil {
		return nil
	}
	tableName := n.Table.Name
	columnList := omniIndexColumns(n.Columns)
	metaData := map[string]string{
		advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
		advisor.TableNameTemplateToken:  tableName,
	}
	return []*indexMetaData{{
		indexName: n.IndexName,
		tableName: tableName,
		metaData:  metaData,
		line:      ostmt.FirstTokenLine(),
	}}
}
