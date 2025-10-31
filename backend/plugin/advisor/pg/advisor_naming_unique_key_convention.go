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

	checker := &namingUKConventionChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		format:                       format,
		maxLength:                    maxLength,
		templateList:                 templateList,
		catalog:                      checkCtx.Catalog,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type namingUKConventionChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList   []*storepb.Advice
	level        storepb.Advice_Status
	title        string
	format       string
	maxLength    int
	templateList []string
	catalog      *catalog.Finder
}

//nolint:unused
type indexMetaData struct {
	indexName string
	tableName string
	line      int
	metaData  map[string]string
}

// EnterIndexstmt handles CREATE UNIQUE INDEX statements
func (c *namingUKConventionChecker) EnterIndexstmt(ctx *parser.IndexstmtContext) {
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

	c.checkUniqueKeyName(indexName, tableName, metaData, ctx.GetStart().GetLine())
}

// EnterCreatestmt handles CREATE TABLE with UNIQUE constraints
func (c *namingUKConventionChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
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
				c.checkTableConstraint(elem.Tableconstraint(), tableName, elem.GetStart().GetLine())
			}
			// Check column-level constraints
			if elem.ColumnDef() != nil {
				c.checkColumnDef(elem.ColumnDef(), tableName)
			}
		}
	}
}

// EnterAltertablestmt handles ALTER TABLE statements
func (c *namingUKConventionChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
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
				c.checkTableConstraint(cmd.Tableconstraint(), tableName, ctx.GetStart().GetLine())
			}
			// ADD COLUMN with constraints
			if cmd.ADD_P() != nil && cmd.ColumnDef() != nil {
				c.checkColumnDef(cmd.ColumnDef(), tableName)
			}
		}
	}

	// Note: ALTER TABLE ... RENAME CONSTRAINT is handled in EnterRenamestmt
	// because PostgreSQL parser treats "ALTER TABLE t RENAME CONSTRAINT old TO new"
	// as a rename statement, not as an alter_table_cmd
}

// EnterRenamestmt handles ALTER INDEX ... RENAME TO and ALTER TABLE ... RENAME CONSTRAINT statements
func (c *namingUKConventionChecker) EnterRenamestmt(ctx *parser.RenamestmtContext) {
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
		if c.catalog != nil && oldIndexName != "" {
			tableName, index := c.findIndex("", "", oldIndexName)
			if index != nil && index.Unique() && !index.Primary() {
				c.checkUniqueKeyName(newIndexName, tableName, map[string]string{
					advisor.ColumnListTemplateToken: strings.Join(index.ExpressionList(), "_"),
					advisor.TableNameTemplateToken:  tableName,
				}, ctx.GetStart().GetLine())
			}
		}
	}

	// Check for ALTER TABLE ... RENAME CONSTRAINT
	if ctx.CONSTRAINT() != nil && ctx.TO() != nil && c.catalog != nil {
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
			foundTableName, index := c.findIndex(schemaName, tableName, oldConstraintName)
			if index != nil && index.Unique() && !index.Primary() {
				metaData := map[string]string{
					advisor.ColumnListTemplateToken: strings.Join(index.ExpressionList(), "_"),
					advisor.TableNameTemplateToken:  foundTableName,
				}
				c.checkUniqueKeyName(newConstraintName, foundTableName, metaData, ctx.GetStart().GetLine())
			}
		}
	}
}

func (c *namingUKConventionChecker) checkTableConstraint(constraint parser.ITableconstraintContext, tableName string, line int) {
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
				foundTableName, index := c.findIndex("", tableName, indexName)
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
				c.checkUniqueKeyName(constraintName, tableName, metaData, line)
			}
		}
	}
}

func (c *namingUKConventionChecker) checkColumnDef(columndef parser.IColumnDefContext, tableName string) {
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
						c.checkUniqueKeyName(constraintName, tableName, metaData, qual.GetStart().GetLine())
					}
				}
			}
		}
	}
}

func (c *namingUKConventionChecker) checkUniqueKeyName(indexName, tableName string, metaData map[string]string, line int) {
	regex, err := c.getTemplateRegexp(metaData)
	if err != nil {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.Internal.Int32(),
			Title:   "Internal error for unique key naming convention rule",
			Content: fmt.Sprintf("Failed to compile regex for unique key naming convention: %v", err),
		})
		return
	}

	if !regex.MatchString(indexName) {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.NamingUKConventionMismatch.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf(`Unique key in table "%s" mismatches the naming convention, expect %q but found "%s"`, tableName, regex, indexName),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}

	if c.maxLength > 0 && len(indexName) > c.maxLength {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    advisor.NamingUKConventionMismatch.Int32(),
			Title:   c.title,
			Content: fmt.Sprintf(`Unique key "%s" in table "%s" mismatches the naming convention, its length should be within %d characters`, indexName, tableName, c.maxLength),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}

func (c *namingUKConventionChecker) getTemplateRegexp(tokens map[string]string) (*regexp.Regexp, error) {
	template := c.format
	for _, key := range c.templateList {
		if token, ok := tokens[key]; ok {
			template = strings.ReplaceAll(template, key, token)
		}
	}
	return regexp.Compile(template)
}

// findIndex returns index found in catalogs, nil if not found.
func (c *namingUKConventionChecker) findIndex(schemaName string, tableName string, indexName string) (string, *catalog.IndexState) {
	if c.catalog == nil {
		return "", nil
	}
	return c.catalog.Origin.FindIndex(&catalog.IndexFind{
		SchemaName: normalizeSchemaName(schemaName),
		TableName:  tableName,
		IndexName:  indexName,
	})
}
