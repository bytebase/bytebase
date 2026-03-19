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
	_ advisor.Advisor = (*NamingPKConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_INDEX_PK, &NamingPKConventionAdvisor{})
}

// NamingPKConventionAdvisor is the advisor checking for primary key naming convention.
type NamingPKConventionAdvisor struct {
}

// Check checks for primary key naming convention.
func (*NamingPKConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	rule := &namingPKConventionRule{
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

type namingPKConventionRule struct {
	OmniBaseRule

	format           string
	maxLength        int
	templateList     []string
	originalMetadata *model.DatabaseMetadata
}

type pkMetaData struct {
	pkName    string
	tableName string
	metaData  map[string]string
}

func (*namingPKConventionRule) Name() string {
	return "naming_primary_key_convention"
}

func (r *namingPKConventionRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	case *ast.RenameStmt:
		r.handleRenameStmt(n)
	default:
	}
}

func (r *namingPKConventionRule) handleCreateStmt(n *ast.CreateStmt) {
	tableName := omniTableName(n.Relation)
	schemaName := omniSchemaName(n.Relation)

	_, constraints := omniTableElements(n)
	for _, c := range constraints {
		if c.Contype == ast.CONSTR_PRIMARY && c.Conname != "" {
			columnList := omniConstraintColumns(c)
			r.checkPKName(&pkMetaData{
				pkName:    c.Conname,
				tableName: tableName,
				metaData: map[string]string{
					advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
					advisor.TableNameTemplateToken:  tableName,
				},
			})
		}
	}

	// Check PRIMARY KEY USING INDEX in table constraints
	for _, c := range constraints {
		if c.Contype == ast.CONSTR_PRIMARY && c.Conname != "" && c.Indexname != "" {
			columnList := r.lookupIndexColumns(schemaName, tableName, c.Indexname)
			r.checkPKName(&pkMetaData{
				pkName:    c.Conname,
				tableName: tableName,
				metaData: map[string]string{
					advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
					advisor.TableNameTemplateToken:  tableName,
				},
			})
		}
	}
}

func (r *namingPKConventionRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Relation)
	schemaName := omniSchemaName(n.Relation)

	for _, cmd := range omniAlterTableCmds(n) {
		if ast.AlterTableType(cmd.Subtype) == ast.AT_AddConstraint {
			c, ok := cmd.Def.(*ast.Constraint)
			if !ok || c.Contype != ast.CONSTR_PRIMARY {
				continue
			}
			pkName := c.Conname
			if pkName == "" {
				continue
			}

			columnList := omniConstraintColumns(c)

			// PRIMARY KEY USING INDEX
			if c.Indexname != "" {
				columnList = r.lookupIndexColumns(schemaName, tableName, c.Indexname)
			}

			r.checkPKName(&pkMetaData{
				pkName:    pkName,
				tableName: tableName,
				metaData: map[string]string{
					advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
					advisor.TableNameTemplateToken:  tableName,
				},
			})
		}
	}
}

func (r *namingPKConventionRule) handleRenameStmt(n *ast.RenameStmt) {
	// ALTER TABLE ... RENAME CONSTRAINT
	if n.RenameType == ast.OBJECT_TABCONSTRAINT {
		tableName := omniTableName(n.Relation)
		schemaName := omniSchemaName(n.Relation)
		oldName := n.Subname
		newName := n.Newname

		if r.originalMetadata != nil && oldName != "" {
			normalizedSchema := normalizeSchemaName(schemaName)
			index := r.originalMetadata.GetSchemaMetadata(normalizedSchema).GetTable(tableName).GetIndex(oldName)
			if index != nil && index.GetProto().GetPrimary() {
				r.checkPKName(&pkMetaData{
					pkName:    newName,
					tableName: tableName,
					metaData: map[string]string{
						advisor.ColumnListTemplateToken: strings.Join(index.GetProto().GetExpressions(), "_"),
						advisor.TableNameTemplateToken:  tableName,
					},
				})
			}
		}
	}

	// ALTER INDEX ... RENAME TO
	if n.RenameType == ast.OBJECT_INDEX {
		oldIndexName := ""
		schemaName := ""
		if n.Relation != nil {
			oldIndexName = n.Relation.Relname
			schemaName = n.Relation.Schemaname
		}
		newIndexName := n.Newname

		if r.originalMetadata != nil && oldIndexName != "" {
			normalizedSchema := normalizeSchemaName(schemaName)
			schema := r.originalMetadata.GetSchemaMetadata(normalizedSchema)
			if schema != nil {
				index := schema.GetIndex(oldIndexName)
				if index != nil && index.GetProto().GetPrimary() {
					tableName := index.GetTableProto().Name
					r.checkPKName(&pkMetaData{
						pkName:    newIndexName,
						tableName: tableName,
						metaData: map[string]string{
							advisor.ColumnListTemplateToken: strings.Join(index.GetProto().GetExpressions(), "_"),
							advisor.TableNameTemplateToken:  tableName,
						},
					})
				}
			}
		}
	}
}

func (r *namingPKConventionRule) lookupIndexColumns(schemaName, tableName, indexName string) []string {
	if r.originalMetadata == nil || indexName == "" {
		return nil
	}
	normalizedSchema := normalizeSchemaName(schemaName)
	schema := r.originalMetadata.GetSchemaMetadata(normalizedSchema)
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

func (r *namingPKConventionRule) checkPKName(pkData *pkMetaData) {
	line := r.FindLineByName(pkData.pkName)

	regex, err := getTemplateRegexp(r.format, r.templateList, pkData.metaData)
	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.Internal.Int32(),
			Title:   "Internal error for primary key naming convention rule",
			Content: fmt.Sprintf("Failed to compile regex: %v", err),
			StartPosition: &storepb.Position{
				Line:   line,
				Column: 0,
			},
		})
		return
	}

	if !regex.MatchString(pkData.pkName) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.NamingPKConventionMismatch.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf(`Primary key in table "%s" mismatches the naming convention, expect %q but found "%s"`, pkData.tableName, regex, pkData.pkName),
			StartPosition: &storepb.Position{
				Line:   line,
				Column: 0,
			},
		})
	}

	if r.maxLength > 0 && len(pkData.pkName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.NamingPKConventionMismatch.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf(`Primary key "%s" in table "%s" mismatches the naming convention, its length should be within %d characters`, pkData.pkName, pkData.tableName, r.maxLength),
			StartPosition: &storepb.Position{
				Line:   line,
				Column: 0,
			},
		})
	}
}
