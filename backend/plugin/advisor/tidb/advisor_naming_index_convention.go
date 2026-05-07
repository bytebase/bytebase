package tidb

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
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
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for this rule")
	}

	format := namingPayload.Format
	templateList, _ := advisor.ParseTemplateTokens(format)

	for _, key := range templateList {
		if _, ok := advisor.TemplateNamingTokens[checkCtx.Rule.Type][key]; !ok {
			return nil, errors.Errorf("invalid template %s for rule %s", key, checkCtx.Rule.Type)
		}
	}

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}

	title := checkCtx.Rule.Type.String()
	originalMetadata := checkCtx.OriginalMetadata

	var adviceList []*storepb.Advice
	for _, ostmt := range stmts {
		var indexDataList []*indexMetaData
		switch n := ostmt.Node.(type) {
		case *ast.CreateTableStmt:
			indexDataList = collectIndexCreateTable(ostmt, n)
		case *ast.AlterTableStmt:
			indexDataList = collectIndexAlterTable(ostmt, n, originalMetadata)
		case *ast.CreateIndexStmt:
			indexDataList = collectIndexCreateIndex(ostmt, n)
		default:
		}

		for _, indexData := range indexDataList {
			regex, err := getTemplateRegexp(format, templateList, indexData.metaData)
			if err != nil {
				adviceList = append(adviceList, &storepb.Advice{
					Status:  level,
					Code:    code.Internal.Int32(),
					Title:   "Internal error for index naming convention rule",
					Content: fmt.Sprintf("%q meet internal error %q", ostmt.TrimmedText(), err.Error()),
				})
				continue
			}
			if !regex.MatchString(indexData.indexName) {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Code:          code.NamingIndexConventionMismatch.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("Index in table `%s` mismatches the naming convention, expect %q but found `%s`", indexData.tableName, regex, indexData.indexName),
					StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
				})
			}
			if maxLength > 0 && len(indexData.indexName) > maxLength {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Code:          code.NamingIndexConventionMismatch.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("Index `%s` in table `%s` mismatches the naming convention, its length should be within %d characters", indexData.indexName, indexData.tableName, maxLength),
					StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
				})
			}
		}
	}

	return adviceList, nil
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
