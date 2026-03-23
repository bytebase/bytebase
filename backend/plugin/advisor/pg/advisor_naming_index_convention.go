package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*NamingIndexConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_INDEX_IDX, &NamingIndexConventionAdvisor{})
}

// NamingIndexConventionAdvisor is the advisor checking for index naming convention.
type NamingIndexConventionAdvisor struct {
}

// Check checks for index naming convention.
func (*NamingIndexConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	rule := &namingIndexConventionRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		format:           format,
		maxLength:        maxLength,
		templateList:     templateList,
		originalMetadata: checkCtx.OriginalMetadata,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type namingIndexConventionRule struct {
	OmniBaseRule

	format           string
	maxLength        int
	templateList     []string
	originalMetadata *model.DatabaseMetadata
}

// Name returns the rule name.
func (*namingIndexConventionRule) Name() string {
	return "naming-index-convention"
}

func (r *namingIndexConventionRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.IndexStmt:
		r.handleIndexStmt(n)
	case *ast.RenameStmt:
		r.handleRenameStmt(n)
	default:
	}
}

func (r *namingIndexConventionRule) handleIndexStmt(n *ast.IndexStmt) {
	// Skip UNIQUE indexes - they are handled by the UK naming rule
	if n.Unique {
		return
	}

	indexName := n.Idxname
	if indexName == "" {
		return
	}

	tableName := omniTableName(n.Relation)
	if tableName == "" {
		return
	}

	columnList := omniIndexColumns(n)
	r.checkIndexName(indexName, tableName, columnList)
}

func (r *namingIndexConventionRule) handleRenameStmt(n *ast.RenameStmt) {
	// ALTER INDEX ... RENAME TO
	if n.RenameType == ast.OBJECT_INDEX {
		oldIndexName := ""
		if n.Relation != nil {
			oldIndexName = n.Relation.Relname
		}
		newIndexName := n.Newname

		if r.originalMetadata != nil && oldIndexName != "" {
			tableName, index := r.findIndex("", "", oldIndexName)
			if index != nil && !index.GetProto().GetUnique() && !index.GetProto().GetPrimary() {
				r.checkIndexName(newIndexName, tableName, index.GetProto().GetExpressions())
			}
		}
	}
}

func (r *namingIndexConventionRule) checkIndexName(indexName, tableName string, columnList []string) {
	metaData := map[string]string{
		advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
		advisor.TableNameTemplateToken:  tableName,
	}

	regex, err := getTemplateRegexp(r.format, r.templateList, metaData)
	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.Internal.Int32(),
			Title:   "Internal error for index naming convention rule",
			Content: fmt.Sprintf("Failed to compile regex: %v", err),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
		return
	}

	if !regex.MatchString(indexName) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.NamingIndexConventionMismatch.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Index in table %q mismatches the naming convention, expect %q but found %q", tableName, regex, indexName),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}

	if r.maxLength > 0 && len(indexName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.NamingIndexConventionMismatch.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf("Index %q in table %q mismatches the naming convention, its length should be within %d characters", indexName, tableName, r.maxLength),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}

// findIndex returns index found in catalogs, nil if not found.
func (r *namingIndexConventionRule) findIndex(schemaName string, tableName string, indexName string) (string, *model.IndexMetadata) {
	if r.originalMetadata == nil {
		return "", nil
	}
	schema := r.originalMetadata.GetSchemaMetadata(normalizeSchemaName(schemaName))
	if schema == nil {
		return "", nil
	}
	if tableName != "" {
		index := schema.GetTable(tableName).GetIndex(indexName)
		if index != nil {
			return tableName, index
		}
		return "", nil
	}
	// tableName is empty, search all tables
	index := schema.GetIndex(indexName)
	if index != nil {
		return index.GetTableProto().Name, index
	}
	return "", nil
}
