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
	_ advisor.Advisor = (*NamingIndexConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_NAMING_INDEX_IDX, &NamingIndexConventionAdvisor{})
}

// NamingIndexConventionAdvisor is the advisor checking for index naming convention.
type NamingIndexConventionAdvisor struct {
}

// Check checks for index naming convention.
func (*NamingIndexConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	// Capture originalMetadata in a local so the collector closure can read
	// it without runNamingConventionRule needing advisor.Context in its
	// callback signature — the FK rule doesn't need metadata, so keeping
	// the helper signature uniform is cleaner than threading a Context arg.
	originalMetadata := checkCtx.OriginalMetadata
	return runNamingConventionRule(checkCtx, namingRuleConfig{
		mismatchCode:       code.NamingIndexConventionMismatch,
		typeNoun:           "Index",
		internalErrorTitle: "Internal error for index naming convention rule",
	}, func(ostmt OmniStmt) []*indexMetaData {
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			return collectIndexCreateTable(ostmt, n)
		case *ast.AlterTableStmt:
			return collectIndexAlterTable(ostmt, n, originalMetadata)
		case *ast.CreateIndexStmt:
			return collectIndexCreateIndex(ostmt, n)
		}
		return nil
	})
}

func collectIndexCreateTable(ostmt OmniStmt, n *ast.CreateTableStmt) []*indexMetaData {
	if n.Table == nil {
		return nil
	}
	tableName := n.Table.Name
	var res []*indexMetaData
	for _, constraint := range n.Constraints {
		if constraint == nil || constraint.Type != ast.ConstrIndex {
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

func collectIndexAlterTable(ostmt OmniStmt, n *ast.AlterTableStmt, originalMetadata *model.DatabaseMetadata) []*indexMetaData {
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
		// ATAddIndex (omni) and ATAddConstraint (omni) are split where pingcap
		// collapsed both ALTER TABLE ADD INDEX and ALTER TABLE ADD CONSTRAINT
		// INDEX into AlterTableAddConstraint. Mirror the mysql omni branching
		// to cover both syntactic forms.
		case ast.ATAddConstraint, ast.ATAddIndex:
			if cmd.Constraint == nil || cmd.Constraint.Type != ast.ConstrIndex {
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
			if index.GetProto().GetUnique() {
				// Unique index naming convention is handled by
				// advisor_naming_unique_key_convention.go.
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

func collectIndexCreateIndex(ostmt OmniStmt, n *ast.CreateIndexStmt) []*indexMetaData {
	// Preserve pingcap behavior: only exclude unique indexes. Fulltext/spatial
	// CREATE INDEX statements were checked by the pingcap-typed advisor and
	// remain checked here. The mysql omni rule additionally excludes Fulltext
	// and Spatial; aligning tidb with mysql on that is a separate behavior
	// change, deferred from this AST migration.
	if n.Unique {
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
