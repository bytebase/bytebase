package pgantlr

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*FullyQualifiedObjectNameAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleFullyQualifiedObjectName, &FullyQualifiedObjectNameAdvisor{})
}

// FullyQualifiedObjectNameAdvisor is the advisor checking for fully qualified object names.
type FullyQualifiedObjectNameAdvisor struct {
}

// Check checks for fully qualified object names.
func (*FullyQualifiedObjectNameAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	tree, err := getANTLRTree(checkCtx)
	if err != nil {
		return nil, err
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &fullyQualifiedObjectNameChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
		level:                        level,
		title:                        string(checkCtx.Rule.Type),
		dbSchema:                     checkCtx.DBSchema,
		statementsText:               checkCtx.Statements,
	}

	antlr.ParseTreeWalkerDefault.Walk(checker, tree.Tree)

	return checker.adviceList, nil
}

type fullyQualifiedObjectNameChecker struct {
	*parser.BasePostgreSQLParserListener

	adviceList     []*storepb.Advice
	level          storepb.Advice_Status
	title          string
	dbSchema       *storepb.DatabaseSchemaMetadata
	statementsText string
}

// EnterCreatestmt handles CREATE TABLE
func (c *fullyQualifiedObjectNameChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	allQualifiedNames := ctx.AllQualified_name()
	if len(allQualifiedNames) > 0 {
		c.checkQualifiedName(allQualifiedNames[0], ctx.GetStop().GetLine())
	}
}

// EnterCreateseqstmt handles CREATE SEQUENCE
func (c *fullyQualifiedObjectNameChecker) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Qualified_name() != nil {
		c.checkQualifiedName(ctx.Qualified_name(), ctx.GetStop().GetLine())
	}
}

// EnterCreatetrigstmt handles CREATE TRIGGER
func (c *fullyQualifiedObjectNameChecker) EnterCreatetrigstmt(ctx *parser.CreatetrigstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check the table name in the ON clause
	if ctx.Qualified_name() != nil {
		c.checkQualifiedName(ctx.Qualified_name(), ctx.GetStop().GetLine())
	}
}

// EnterIndexstmt handles CREATE INDEX
func (c *fullyQualifiedObjectNameChecker) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check the table name in the ON clause
	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		c.checkQualifiedName(ctx.Relation_expr().Qualified_name(), ctx.GetStop().GetLine())
	}
}

// EnterDropstmt handles DROP TABLE, DROP SEQUENCE, DROP INDEX
func (c *fullyQualifiedObjectNameChecker) EnterDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check all qualified names in the drop statement
	if ctx.Any_name_list() != nil {
		for _, anyName := range ctx.Any_name_list().AllAny_name() {
			c.checkAnyName(anyName, ctx.GetStop().GetLine())
		}
	}
}

// EnterAltertablestmt handles ALTER TABLE
func (c *fullyQualifiedObjectNameChecker) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		c.checkQualifiedName(ctx.Relation_expr().Qualified_name(), ctx.GetStop().GetLine())
	}
}

// EnterAlterseqstmt handles ALTER SEQUENCE
func (c *fullyQualifiedObjectNameChecker) EnterAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Qualified_name() != nil {
		c.checkQualifiedName(ctx.Qualified_name(), ctx.GetStop().GetLine())
	}
}

// EnterRenamestmt handles ALTER TABLE RENAME
func (c *fullyQualifiedObjectNameChecker) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		c.checkQualifiedName(ctx.Relation_expr().Qualified_name(), ctx.GetStop().GetLine())
	}
}

// EnterInsertstmt handles INSERT
func (c *fullyQualifiedObjectNameChecker) EnterInsertstmt(ctx *parser.InsertstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Insert_target() != nil && ctx.Insert_target().Qualified_name() != nil {
		c.checkQualifiedName(ctx.Insert_target().Qualified_name(), ctx.GetStop().GetLine())
	}
}

// EnterUpdatestmt handles UPDATE
func (c *fullyQualifiedObjectNameChecker) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr_opt_alias() != nil && ctx.Relation_expr_opt_alias().Relation_expr() != nil {
		if ctx.Relation_expr_opt_alias().Relation_expr().Qualified_name() != nil {
			c.checkQualifiedName(ctx.Relation_expr_opt_alias().Relation_expr().Qualified_name(), ctx.GetStop().GetLine())
		}
	}
}

// EnterSelectstmt handles SELECT
func (c *fullyQualifiedObjectNameChecker) EnterSelectstmt(ctx *parser.SelectstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// For SELECT statements, we need to extract tables from the query
	// and check them against the database schema
	if c.dbSchema == nil {
		return
	}

	// Extract the statement text for this SELECT
	startLine := ctx.GetStop().GetLine()
	endLine := ctx.GetStop().GetLine()
	statementText := extractStatementText(c.statementsText, startLine, endLine)

	if statementText == "" {
		return
	}

	// Find all table references in the SELECT statement
	for _, tableName := range c.findAllTablesInSelect(statementText) {
		objName := tableName.String()
		if !c.isFullyQualified(objName) {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  c.level,
				Code:    int32(advisor.NamingNotFullyQualifiedName),
				Title:   c.title,
				Content: fmt.Sprintf("unqualified object name: '%s'", objName),
				StartPosition: &storepb.Position{
					Line:   int32(startLine),
					Column: 0,
				},
			})
		}
	}
}

// checkQualifiedName checks if a qualified name is fully qualified
func (c *fullyQualifiedObjectNameChecker) checkQualifiedName(ctx parser.IQualified_nameContext, line int) {
	if ctx == nil {
		return
	}

	parts := pgparser.NormalizePostgreSQLQualifiedName(ctx)
	objName := strings.Join(parts, ".")

	if !c.isFullyQualified(objName) {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    int32(advisor.NamingNotFullyQualifiedName),
			Title:   c.title,
			Content: fmt.Sprintf("unqualified object name: '%s'", objName),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}

// checkAnyName checks if an any_name is fully qualified
func (c *fullyQualifiedObjectNameChecker) checkAnyName(ctx parser.IAny_nameContext, line int) {
	if ctx == nil {
		return
	}

	// Extract parts from any_name (schema.object or object)
	parts := pgparser.NormalizePostgreSQLAnyName(ctx)
	objName := strings.Join(parts, ".")

	if !c.isFullyQualified(objName) {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  c.level,
			Code:    int32(advisor.NamingNotFullyQualifiedName),
			Title:   c.title,
			Content: fmt.Sprintf("unqualified object name: '%s'", objName),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}

// isFullyQualified checks if an object name is fully qualified (contains a dot)
func (*fullyQualifiedObjectNameChecker) isFullyQualified(objName string) bool {
	if objName == "" {
		return true
	}
	re := regexp.MustCompile(`.+\..+`)
	return re.MatchString(objName)
}

// findAllTablesInSelect finds all table references in a SELECT statement
func (c *fullyQualifiedObjectNameChecker) findAllTablesInSelect(statement string) []base.ColumnResource {
	// Parse the statement to extract table references
	result, err := pgparser.ParsePostgreSQL(statement)
	if err != nil {
		return nil
	}

	// Use a visitor to find all table references
	collector := &tableReferenceCollector{
		schemaNameMap: c.getSchemaNameMapFromPublic(),
	}

	if result.Tree != nil {
		antlr.ParseTreeWalkerDefault.Walk(collector, result.Tree)
	}

	return collector.tables
}

// getSchemaNameMapFromPublic creates a map of table names from the database schema
func (c *fullyQualifiedObjectNameChecker) getSchemaNameMapFromPublic() map[string]bool {
	if c.dbSchema == nil || c.dbSchema.Schemas == nil {
		return nil
	}

	filterMap := map[string]bool{}
	for _, schema := range c.dbSchema.Schemas {
		// Tables
		for _, tbl := range schema.Tables {
			filterMap[tbl.Name] = true
		}
		// External Tables
		for _, tbl := range schema.ExternalTables {
			filterMap[tbl.Name] = true
		}
	}
	return filterMap
}

// tableReferenceCollector collects table references from a SELECT statement
type tableReferenceCollector struct {
	*parser.BasePostgreSQLParserListener

	tables        []base.ColumnResource
	schemaNameMap map[string]bool
}

// EnterTable_ref collects table references
func (c *tableReferenceCollector) EnterTable_ref(ctx *parser.Table_refContext) {
	// Look for relation_expr in the table_ref
	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		parts := pgparser.NormalizePostgreSQLQualifiedName(ctx.Relation_expr().Qualified_name())

		resource := base.ColumnResource{}
		if len(parts) == 2 {
			resource.Schema = parts[0]
			resource.Table = parts[1]
		} else if len(parts) == 1 {
			resource.Table = parts[0]
		}

		// Only add if the table exists in the schema
		if c.schemaNameMap == nil || c.schemaNameMap[resource.Table] {
			c.tables = append(c.tables, resource)
		}
	}
}
