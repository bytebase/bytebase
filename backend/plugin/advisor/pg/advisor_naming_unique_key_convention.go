package pg

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
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
		BaseRule: BaseRule{
			level: level,
			title: checkCtx.Rule.Type.String(),
		},
		format:           format,
		maxLength:        maxLength,
		templateList:     templateList,
		originalMetadata: checkCtx.OriginalMetadata,
	}

	checker := NewGenericChecker([]Rule{rule})

	for _, stmt := range checkCtx.ParsedStatements {
		if stmt.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmt.AST)
		if !ok {
			continue
		}
		rule.SetBaseLine(stmt.BaseLine())
		checker.SetBaseLine(stmt.BaseLine())
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
	}

	return checker.GetAdviceList(), nil
}

type namingUKConventionRule struct {
	BaseRule

	format           string
	maxLength        int
	templateList     []string
	originalMetadata *model.DatabaseMetadata
}

func (*namingUKConventionRule) Name() string {
	return "naming-unique-key-convention"
}

func (r *namingUKConventionRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Indexstmt":
		r.handleIndexstmt(ctx.(*parser.IndexstmtContext))
	case "Createstmt":
		r.handleCreatestmt(ctx.(*parser.CreatestmtContext))
	case "Altertablestmt":
		r.handleAltertablestmt(ctx.(*parser.AltertablestmtContext))
	case "Renamestmt":
		r.handleRenamestmt(ctx.(*parser.RenamestmtContext))
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*namingUKConventionRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleIndexstmt handles CREATE UNIQUE INDEX statements
func (r *namingUKConventionRule) handleIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Only check UNIQUE indexes
	if ctx.Opt_unique() == nil || ctx.Opt_unique().UNIQUE() == nil {
		return
	}

	indexName := ""
	if ctx.Name() != nil {
		indexName = pgparser.NormalizePostgreSQLName(ctx.Name())
	}

	tableName := ""
	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		tableName = extractTableName(ctx.Relation_expr().Qualified_name())
	}

	// Extract column list
	var columnList []string
	if ctx.Index_params() != nil {
		allParams := ctx.Index_params().AllIndex_elem()
		for _, param := range allParams {
			if param.Colid() != nil {
				colName := pgparser.NormalizePostgreSQLColid(param.Colid())
				columnList = append(columnList, colName)
			}
		}
	}

	metaData := map[string]string{
		advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
		advisor.TableNameTemplateToken:  tableName,
	}

	r.checkUniqueKeyName(indexName, tableName, metaData, ctx.GetStart().GetLine())
}

// handleCreatestmt handles CREATE TABLE with UNIQUE constraints
func (r *namingUKConventionRule) handleCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	qualifiedNames := ctx.AllQualified_name()
	if len(qualifiedNames) == 0 {
		return
	}

	tableName := extractTableName(qualifiedNames[0])
	if tableName == "" {
		return
	}

	// Check table-level constraints
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			if elem.Tableconstraint() != nil {
				r.checkTableConstraint(elem.Tableconstraint(), tableName, elem.GetStart().GetLine())
			}
			// Check column-level constraints
			if elem.ColumnDef() != nil {
				r.checkColumnDef(elem.ColumnDef(), tableName)
			}
		}
	}
}

// handleAltertablestmt handles ALTER TABLE statements
func (r *namingUKConventionRule) handleAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() == nil || ctx.Relation_expr().Qualified_name() == nil {
		return
	}

	tableName := extractTableName(ctx.Relation_expr().Qualified_name())
	if tableName == "" {
		return
	}

	if ctx.Alter_table_cmds() != nil {
		allCmds := ctx.Alter_table_cmds().AllAlter_table_cmd()
		for _, cmd := range allCmds {
			// ADD CONSTRAINT
			if cmd.ADD_P() != nil && cmd.Tableconstraint() != nil {
				r.checkTableConstraint(cmd.Tableconstraint(), tableName, ctx.GetStart().GetLine())
			}
			// ADD COLUMN with constraints
			if cmd.ADD_P() != nil && cmd.ColumnDef() != nil {
				r.checkColumnDef(cmd.ColumnDef(), tableName)
			}
		}
	}

	// Note: ALTER TABLE ... RENAME CONSTRAINT is handled in handleRenamestmt
	// because PostgreSQL parser treats "ALTER TABLE t RENAME CONSTRAINT old TO new"
	// as a rename statement, not as an alter_table_cmd
}

// handleRenamestmt handles ALTER INDEX ... RENAME TO and ALTER TABLE ... RENAME CONSTRAINT statements
func (r *namingUKConventionRule) handleRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check for ALTER INDEX ... RENAME TO
	if ctx.INDEX() != nil && ctx.TO() != nil {
		allNames := ctx.AllName()
		if len(allNames) < 1 {
			return
		}

		// Get old index name from qualified_name
		var oldIndexName string
		if ctx.Qualified_name() != nil {
			parts := pgparser.NormalizePostgreSQLQualifiedName(ctx.Qualified_name())
			if len(parts) > 0 {
				oldIndexName = parts[len(parts)-1]
			}
		}

		// Get new index name from the name after TO
		newIndexName := pgparser.NormalizePostgreSQLName(allNames[0])

		// Look up the index in catalog to determine if it's a unique key
		if r.originalMetadata != nil && oldIndexName != "" {
			tableName, index := r.findIndex("", "", oldIndexName)
			if index != nil && index.GetProto().GetUnique() && !index.GetProto().GetPrimary() {
				r.checkUniqueKeyName(newIndexName, tableName, map[string]string{
					advisor.ColumnListTemplateToken: strings.Join(index.GetProto().GetExpressions(), "_"),
					advisor.TableNameTemplateToken:  tableName,
				}, ctx.GetStart().GetLine())
			}
		}
	}

	// Check for ALTER TABLE ... RENAME CONSTRAINT
	if ctx.CONSTRAINT() != nil && ctx.TO() != nil && r.originalMetadata != nil {
		allNames := ctx.AllName()
		if len(allNames) >= 2 {
			oldConstraintName := pgparser.NormalizePostgreSQLName(allNames[0])
			newConstraintName := pgparser.NormalizePostgreSQLName(allNames[1])

			// Get table name from the statement
			var tableName, schemaName string
			if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
				tableName = extractTableName(ctx.Relation_expr().Qualified_name())
				schemaName = extractSchemaName(ctx.Relation_expr().Qualified_name())
			}

			// Check if this is a unique key constraint in catalog
			foundTableName, index := r.findIndex(schemaName, tableName, oldConstraintName)
			if index != nil && index.GetProto().GetUnique() && !index.GetProto().GetPrimary() {
				metaData := map[string]string{
					advisor.ColumnListTemplateToken: strings.Join(index.GetProto().GetExpressions(), "_"),
					advisor.TableNameTemplateToken:  foundTableName,
				}
				r.checkUniqueKeyName(newConstraintName, foundTableName, metaData, ctx.GetStart().GetLine())
			}
		}
	}
}

func (r *namingUKConventionRule) checkTableConstraint(constraint parser.ITableconstraintContext, tableName string, line int) {
	if constraint == nil {
		return
	}

	constraintName := ""
	if constraint.Name() != nil {
		constraintName = pgparser.NormalizePostgreSQLName(constraint.Name())
	}

	if constraint.Constraintelem() != nil {
		elem := constraint.Constraintelem()

		// UNIQUE constraint
		if elem.UNIQUE() != nil {
			var columnList []string
			if elem.Columnlist() != nil {
				allColumns := elem.Columnlist().AllColumnElem()
				for _, col := range allColumns {
					if col.Colid() != nil {
						colName := pgparser.NormalizePostgreSQLColid(col.Colid())
						columnList = append(columnList, colName)
					}
				}
			} else if elem.Existingindex() != nil && elem.Existingindex().Name() != nil {
				// Handle UNIQUE USING INDEX - the column list is in the existing index
				indexName := pgparser.NormalizePostgreSQLName(elem.Existingindex().Name())
				schema := r.originalMetadata.GetSchemaMetadata(normalizeSchemaName(""))
				var index *model.IndexMetadata
				if schema != nil {
					table := schema.GetTable(tableName)
					if table != nil {
						index = table.GetIndex(indexName)
					}
				}
				if index != nil {
					columnList = index.GetProto().GetExpressions()
				}
			}

			// Only check if we have a constraint name (unnamed constraints are auto-generated)
			if constraintName != "" {
				metaData := map[string]string{
					advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
					advisor.TableNameTemplateToken:  tableName,
				}
				r.checkUniqueKeyName(constraintName, tableName, metaData, line)
			}
		}
	}
}

func (r *namingUKConventionRule) checkColumnDef(columndef parser.IColumnDefContext, tableName string) {
	if columndef == nil {
		return
	}

	colName := ""
	if columndef.Colid() != nil {
		colName = pgparser.NormalizePostgreSQLColid(columndef.Colid())
	}

	// Check column-level constraints
	if columndef.Colquallist() != nil {
		allQuals := columndef.Colquallist().AllColconstraint()
		for _, qual := range allQuals {
			if qual.Colconstraintelem() != nil {
				elem := qual.Colconstraintelem()
				// Check for UNIQUE constraint
				if elem.UNIQUE() != nil {
					// Column-level unique constraints with names
					constraintName := ""
					if qual.Name() != nil {
						constraintName = pgparser.NormalizePostgreSQLName(qual.Name())
					}

					// Only check if we have a constraint name
					if constraintName != "" {
						metaData := map[string]string{
							advisor.ColumnListTemplateToken: colName,
							advisor.TableNameTemplateToken:  tableName,
						}
						r.checkUniqueKeyName(constraintName, tableName, metaData, qual.GetStart().GetLine())
					}
				}
			}
		}
	}
}

func (r *namingUKConventionRule) checkUniqueKeyName(indexName, tableName string, metaData map[string]string, line int) {
	regex, err := r.getTemplateRegexp(metaData)
	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.Internal.Int32(),
			Title:   "Internal error for unique key naming convention rule",
			Content: fmt.Sprintf("Failed to compile regex for unique key naming convention: %v", err),
		})
		return
	}

	if !regex.MatchString(indexName) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NamingUKConventionMismatch.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf(`Unique key in table "%s" mismatches the naming convention, expect %q but found "%s"`, tableName, regex, indexName),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}

	if r.maxLength > 0 && len(indexName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    code.NamingUKConventionMismatch.Int32(),
			Title:   r.title,
			Content: fmt.Sprintf(`Unique key "%s" in table "%s" mismatches the naming convention, its length should be within %d characters`, indexName, tableName, r.maxLength),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}

func (r *namingUKConventionRule) getTemplateRegexp(tokens map[string]string) (*regexp.Regexp, error) {
	template := r.format
	for _, key := range r.templateList {
		if token, ok := tokens[key]; ok {
			template = strings.ReplaceAll(template, key, token)
		}
	}
	return regexp.Compile(template)
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
