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
	_ advisor.Advisor = (*NamingUKConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_INDEX_UK, &NamingUKConventionAdvisor{})
}

// NamingUKConventionAdvisor is the advisor checking for unique key naming convention.
type NamingUKConventionAdvisor struct {
}

// Check checks for unique key naming convention.
func (*NamingUKConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	rule := &namingUKConventionRule{
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

type namingUKConventionRule struct {
	OmniBaseRule

	format           string
	maxLength        int
	templateList     []string
	originalMetadata *model.DatabaseMetadata
}

func (*namingUKConventionRule) Name() string {
	return "naming-unique-key-convention"
}

func (r *namingUKConventionRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.IndexStmt:
		r.handleIndexStmt(n)
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	case *ast.RenameStmt:
		r.handleRenameStmt(n)
	default:
	}
}

func (r *namingUKConventionRule) handleIndexStmt(n *ast.IndexStmt) {
	// Only check UNIQUE indexes
	if !n.Unique {
		return
	}

	indexName := n.Idxname
	tableName := omniTableName(n.Relation)
	columnList := omniIndexColumns(n)

	metaData := map[string]string{
		advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
		advisor.TableNameTemplateToken:  tableName,
	}
	r.checkUniqueKeyName(indexName, tableName, metaData)
}

func (r *namingUKConventionRule) handleCreateStmt(n *ast.CreateStmt) {
	tableName := omniTableName(n.Relation)
	cols, constraints := omniTableElements(n)

	// Check table-level UNIQUE constraints
	for _, c := range constraints {
		if c.Contype == ast.CONSTR_UNIQUE && c.Conname != "" {
			columnList := omniConstraintColumns(c)

			// UNIQUE USING INDEX
			if c.Indexname != "" {
				columnList = r.lookupIndexColumns("", tableName, c.Indexname)
			}

			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			r.checkUniqueKeyName(c.Conname, tableName, metaData)
		}
	}

	// Check column-level UNIQUE constraints
	for _, col := range cols {
		for _, c := range omniColumnConstraints(col) {
			if c.Contype == ast.CONSTR_UNIQUE && c.Conname != "" {
				metaData := map[string]string{
					advisor.ColumnListTemplateToken: col.Colname,
					advisor.TableNameTemplateToken:  tableName,
				}
				r.checkUniqueKeyName(c.Conname, tableName, metaData)
			}
		}
	}
}

func (r *namingUKConventionRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Relation)

	for _, cmd := range omniAlterTableCmds(n) {
		if ast.AlterTableType(cmd.Subtype) == ast.AT_AddConstraint {
			c, ok := cmd.Def.(*ast.Constraint)
			if !ok || c.Contype != ast.CONSTR_UNIQUE {
				continue
			}
			if c.Conname == "" {
				continue
			}
			columnList := omniConstraintColumns(c)

			// UNIQUE USING INDEX
			if c.Indexname != "" {
				columnList = r.lookupIndexColumns("", tableName, c.Indexname)
			}

			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			r.checkUniqueKeyName(c.Conname, tableName, metaData)
		}

		// ADD COLUMN with inline UNIQUE constraint
		if ast.AlterTableType(cmd.Subtype) == ast.AT_AddColumn {
			colDef, ok := cmd.Def.(*ast.ColumnDef)
			if !ok {
				continue
			}
			for _, c := range omniColumnConstraints(colDef) {
				if c.Contype == ast.CONSTR_UNIQUE && c.Conname != "" {
					metaData := map[string]string{
						advisor.ColumnListTemplateToken: colDef.Colname,
						advisor.TableNameTemplateToken:  tableName,
					}
					r.checkUniqueKeyName(c.Conname, tableName, metaData)
				}
			}
		}
	}
}

func (r *namingUKConventionRule) handleRenameStmt(n *ast.RenameStmt) {
	// ALTER INDEX ... RENAME TO
	if n.RenameType == ast.OBJECT_INDEX {
		oldIndexName := ""
		if n.Relation != nil {
			oldIndexName = n.Relation.Relname
		}
		newIndexName := n.Newname

		if r.originalMetadata != nil && oldIndexName != "" {
			tableName, index := r.findIndex("", "", oldIndexName)
			if index != nil && index.GetProto().GetUnique() && !index.GetProto().GetPrimary() {
				r.checkUniqueKeyName(newIndexName, tableName, map[string]string{
					advisor.ColumnListTemplateToken: strings.Join(index.GetProto().GetExpressions(), "_"),
					advisor.TableNameTemplateToken:  tableName,
				})
			}
		}
	}

	// ALTER TABLE ... RENAME CONSTRAINT
	if n.RenameType == ast.OBJECT_TABCONSTRAINT && r.originalMetadata != nil {
		tableName := omniTableName(n.Relation)
		schemaName := ""
		if n.Relation != nil {
			schemaName = n.Relation.Schemaname
		}
		oldConstraintName := n.Subname
		newConstraintName := n.Newname

		foundTableName, index := r.findIndex(schemaName, tableName, oldConstraintName)
		if index != nil && index.GetProto().GetUnique() && !index.GetProto().GetPrimary() {
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(index.GetProto().GetExpressions(), "_"),
				advisor.TableNameTemplateToken:  foundTableName,
			}
			r.checkUniqueKeyName(newConstraintName, foundTableName, metaData)
		}
	}
}

func (r *namingUKConventionRule) lookupIndexColumns(schemaName, tableName, indexName string) []string {
	if r.originalMetadata == nil || indexName == "" {
		return nil
	}
	schema := r.originalMetadata.GetSchemaMetadata(normalizeSchemaName(schemaName))
	if schema == nil {
		return nil
	}
	table := schema.GetTable(tableName)
	if table == nil {
		return nil
	}
	index := table.GetIndex(indexName)
	if index != nil {
		return index.GetProto().GetExpressions()
	}
	return nil
}

func (r *namingUKConventionRule) checkUniqueKeyName(indexName, tableName string, metaData map[string]string) {
	line := r.FindLineByName(indexName)

	regex, err := getTemplateRegexp(r.format, r.templateList, metaData)
	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.Internal.Int32(),
			Title:   "Internal error for unique key naming convention rule",
			Content: fmt.Sprintf("Failed to compile regex for unique key naming convention: %v", err),
		})
		return
	}

	if !regex.MatchString(indexName) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.NamingUKConventionMismatch.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf(`Unique key in table "%s" mismatches the naming convention, expect %q but found "%s"`, tableName, regex, indexName),
			StartPosition: &storepb.Position{
				Line:   line,
				Column: 0,
			},
		})
	}

	if r.maxLength > 0 && len(indexName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.NamingUKConventionMismatch.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf(`Unique key "%s" in table "%s" mismatches the naming convention, its length should be within %d characters`, indexName, tableName, r.maxLength),
			StartPosition: &storepb.Position{
				Line:   line,
				Column: 0,
			},
		})
	}
}

// findIndex returns index found in catalogs, nil if not found.
func (r *namingUKConventionRule) findIndex(schemaName string, tableName string, indexName string) (string, *model.IndexMetadata) {
	if r.originalMetadata == nil {
		return "", nil
	}
	schema := r.originalMetadata.GetSchemaMetadata(normalizeSchemaName(schemaName))
	if schema == nil {
		return "", nil
	}
	if tableName != "" {
		table := schema.GetTable(tableName)
		if table != nil {
			index := table.GetIndex(indexName)
			if index != nil {
				return tableName, index
			}
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
