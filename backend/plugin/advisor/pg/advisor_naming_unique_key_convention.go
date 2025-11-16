package pg

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*NamingUKConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleUKNaming, &NamingUKConventionAdvisor{})
}

// NamingUKConventionAdvisor is the advisor checking for unique key naming convention.
type NamingUKConventionAdvisor struct {
}

// Check checks for unique key naming convention.
func (*NamingUKConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	format, templateList, maxLength, err := advisor.UnmarshalNamingRulePayloadAsTemplate(advisor.SQLReviewRuleType(checkCtx.Rule.Type), checkCtx.Rule.Payload)
	if err != nil {
		return nil, err
	}

	rule := &namingUKConventionRule{
		BaseRule: BaseRule{
			level: level,
			title: string(checkCtx.Rule.Type),
		},
		format:        format,
		maxLength:     maxLength,
		templateList:  templateList,
		originCatalog: checkCtx.OriginCatalog,
	}

	checker := NewGenericChecker([]Rule{rule})
	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.GetAdviceList(), nil
}

type namingUKConventionRule struct {
	BaseRule

	format        string
	maxLength     int
	templateList  []string
	originCatalog *catalog.DatabaseState
}

//nolint:unused
type indexMetaData struct {
	indexName string
	tableName string
	line      int
	metaData  map[string]string
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
		if r.originCatalog != nil && oldIndexName != "" {
			tableName, index := r.findIndex("", "", oldIndexName)
			if index != nil && index.Unique() && !index.Primary() {
				r.checkUniqueKeyName(newIndexName, tableName, map[string]string{
					advisor.ColumnListTemplateToken: strings.Join(index.ExpressionList(), "_"),
					advisor.TableNameTemplateToken:  tableName,
				}, ctx.GetStart().GetLine())
			}
		}
	}

	// Check for ALTER TABLE ... RENAME CONSTRAINT
	if ctx.CONSTRAINT() != nil && ctx.TO() != nil && r.originCatalog != nil {
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
			if index != nil && index.Unique() && !index.Primary() {
				metaData := map[string]string{
					advisor.ColumnListTemplateToken: strings.Join(index.ExpressionList(), "_"),
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
				foundTableName, index := r.findIndex("", tableName, indexName)
				if index != nil {
					columnList = index.ExpressionList()
					tableName = foundTableName
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
func (r *namingUKConventionRule) findIndex(schemaName string, tableName string, indexName string) (string, *catalog.IndexState) {
	if r.originCatalog == nil {
		return "", nil
	}
	return r.originCatalog.GetIndex(normalizeSchemaName(schemaName), tableName, indexName)
}
