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
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

var (
	_ advisor.Advisor = (*FullyQualifiedObjectNameAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_FULLY_QUALIFIED, &FullyQualifiedObjectNameAdvisor{})
}

// FullyQualifiedObjectNameAdvisor is the advisor checking for fully qualified object names.
type FullyQualifiedObjectNameAdvisor struct {
}

// Check checks for fully qualified object names.
func (*FullyQualifiedObjectNameAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	var adviceList []*storepb.Advice
	for _, stmtInfo := range checkCtx.ParsedStatements {
		if stmtInfo.AST == nil {
			continue
		}
		antlrAST, ok := base.GetANTLRAST(stmtInfo.AST)
		if !ok {
			continue
		}
		rule := &fullyQualifiedObjectNameRule{
			BaseRule: BaseRule{
				level: level,
				title: checkCtx.Rule.Type.String(),
			},
			dbMetadata: checkCtx.DBSchema,
			tokens:     antlrAST.Tokens,
		}
		rule.SetBaseLine(stmtInfo.BaseLine())

		checker := NewGenericChecker([]Rule{rule})
		antlr.ParseTreeWalkerDefault.Walk(checker, antlrAST.Tree)
		adviceList = append(adviceList, checker.GetAdviceList()...)
	}

	return adviceList, nil
}

type fullyQualifiedObjectNameRule struct {
	BaseRule

	dbMetadata *storepb.DatabaseSchemaMetadata
	tokens     *antlr.CommonTokenStream
}

func (*fullyQualifiedObjectNameRule) Name() string {
	return "naming_fully_qualified"
}

func (r *fullyQualifiedObjectNameRule) OnEnter(ctx antlr.ParserRuleContext, nodeType string) error {
	switch nodeType {
	case "Createstmt":
		if c, ok := ctx.(*parser.CreatestmtContext); ok {
			r.handleCreatestmt(c)
		}
	case "Createseqstmt":
		if c, ok := ctx.(*parser.CreateseqstmtContext); ok {
			r.handleCreateseqstmt(c)
		}
	case "Createtrigstmt":
		if c, ok := ctx.(*parser.CreatetrigstmtContext); ok {
			r.handleCreatetrigstmt(c)
		}
	case "Indexstmt":
		if c, ok := ctx.(*parser.IndexstmtContext); ok {
			r.handleIndexstmt(c)
		}
	case "Dropstmt":
		if c, ok := ctx.(*parser.DropstmtContext); ok {
			r.handleDropstmt(c)
		}
	case "Altertablestmt":
		if c, ok := ctx.(*parser.AltertablestmtContext); ok {
			r.handleAltertablestmt(c)
		}
	case "Alterseqstmt":
		if c, ok := ctx.(*parser.AlterseqstmtContext); ok {
			r.handleAlterseqstmt(c)
		}
	case "Renamestmt":
		if c, ok := ctx.(*parser.RenamestmtContext); ok {
			r.handleRenamestmt(c)
		}
	case "Insertstmt":
		if c, ok := ctx.(*parser.InsertstmtContext); ok {
			r.handleInsertstmt(c)
		}
	case "Updatestmt":
		if c, ok := ctx.(*parser.UpdatestmtContext); ok {
			r.handleUpdatestmt(c)
		}
	case "Selectstmt":
		if c, ok := ctx.(*parser.SelectstmtContext); ok {
			r.handleSelectstmt(c)
		}
	default:
		// Do nothing for other node types
	}
	return nil
}

func (*fullyQualifiedObjectNameRule) OnExit(_ antlr.ParserRuleContext, _ string) error {
	return nil
}

// handleCreatestmt handles CREATE TABLE
func (r *fullyQualifiedObjectNameRule) handleCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	allQualifiedNames := ctx.AllQualified_name()
	if len(allQualifiedNames) > 0 {
		r.checkQualifiedName(allQualifiedNames[0], ctx.GetStop().GetLine())
	}
}

// handleCreateseqstmt handles CREATE SEQUENCE
func (r *fullyQualifiedObjectNameRule) handleCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Qualified_name() != nil {
		r.checkQualifiedName(ctx.Qualified_name(), ctx.GetStop().GetLine())
	}
}

// handleCreatetrigstmt handles CREATE TRIGGER
func (r *fullyQualifiedObjectNameRule) handleCreatetrigstmt(ctx *parser.CreatetrigstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check the table name in the ON clause
	if ctx.Qualified_name() != nil {
		r.checkQualifiedName(ctx.Qualified_name(), ctx.GetStop().GetLine())
	}
}

// handleIndexstmt handles CREATE INDEX
func (r *fullyQualifiedObjectNameRule) handleIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check the table name in the ON clause
	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		r.checkQualifiedName(ctx.Relation_expr().Qualified_name(), ctx.GetStop().GetLine())
	}
}

// handleDropstmt handles DROP TABLE, DROP SEQUENCE, DROP INDEX
func (r *fullyQualifiedObjectNameRule) handleDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check all qualified names in the drop statement
	if ctx.Any_name_list() != nil {
		for _, anyName := range ctx.Any_name_list().AllAny_name() {
			r.checkAnyName(anyName, ctx.GetStop().GetLine())
		}
	}
}

// handleAltertablestmt handles ALTER TABLE
func (r *fullyQualifiedObjectNameRule) handleAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		r.checkQualifiedName(ctx.Relation_expr().Qualified_name(), ctx.GetStop().GetLine())
	}
}

// handleAlterseqstmt handles ALTER SEQUENCE
func (r *fullyQualifiedObjectNameRule) handleAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Qualified_name() != nil {
		r.checkQualifiedName(ctx.Qualified_name(), ctx.GetStop().GetLine())
	}
}

// handleRenamestmt handles ALTER TABLE RENAME
func (r *fullyQualifiedObjectNameRule) handleRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		r.checkQualifiedName(ctx.Relation_expr().Qualified_name(), ctx.GetStop().GetLine())
	}
}

// handleInsertstmt handles INSERT
func (r *fullyQualifiedObjectNameRule) handleInsertstmt(ctx *parser.InsertstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Insert_target() != nil && ctx.Insert_target().Qualified_name() != nil {
		r.checkQualifiedName(ctx.Insert_target().Qualified_name(), ctx.GetStop().GetLine())
	}
}

// handleUpdatestmt handles UPDATE
func (r *fullyQualifiedObjectNameRule) handleUpdatestmt(ctx *parser.UpdatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Relation_expr_opt_alias() != nil && ctx.Relation_expr_opt_alias().Relation_expr() != nil {
		if ctx.Relation_expr_opt_alias().Relation_expr().Qualified_name() != nil {
			r.checkQualifiedName(ctx.Relation_expr_opt_alias().Relation_expr().Qualified_name(), ctx.GetStop().GetLine())
		}
	}
}

// handleSelectstmt handles SELECT
func (r *fullyQualifiedObjectNameRule) handleSelectstmt(ctx *parser.SelectstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// For SELECT statements, we need to extract tables from the query
	// and check them against the database schema
	if r.dbMetadata == nil {
		return
	}

	statementText := getTextFromTokens(r.tokens, ctx)

	// Find all table references in the SELECT statement
	for _, tableName := range r.findAllTablesInSelect(statementText) {
		objName := tableName.String()
		if !r.isFullyQualified(objName) {
			r.AddAdvice(&storepb.Advice{
				Status:  r.level,
				Code:    int32(code.NamingNotFullyQualifiedName),
				Title:   r.title,
				Content: fmt.Sprintf("unqualified object name: '%s'", objName),
				StartPosition: &storepb.Position{
					Line:   int32(ctx.GetStop().GetLine()),
					Column: 0,
				},
			})
		}
	}
}

// checkQualifiedName checks if a qualified name is fully qualified
func (r *fullyQualifiedObjectNameRule) checkQualifiedName(ctx parser.IQualified_nameContext, line int) {
	if ctx == nil {
		return
	}

	parts := pgparser.NormalizePostgreSQLQualifiedName(ctx)
	objName := strings.Join(parts, ".")

	if !r.isFullyQualified(objName) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    int32(code.NamingNotFullyQualifiedName),
			Title:   r.title,
			Content: fmt.Sprintf("unqualified object name: '%s'", objName),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}

// checkAnyName checks if an any_name is fully qualified
func (r *fullyQualifiedObjectNameRule) checkAnyName(ctx parser.IAny_nameContext, line int) {
	if ctx == nil {
		return
	}

	// Extract parts from any_name (schema.object or object)
	parts := pgparser.NormalizePostgreSQLAnyName(ctx)
	objName := strings.Join(parts, ".")

	if !r.isFullyQualified(objName) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.level,
			Code:    int32(code.NamingNotFullyQualifiedName),
			Title:   r.title,
			Content: fmt.Sprintf("unqualified object name: '%s'", objName),
			StartPosition: &storepb.Position{
				Line:   int32(line),
				Column: 0,
			},
		})
	}
}

// isFullyQualified checks if an object name is fully qualified (contains a dot)
func (*fullyQualifiedObjectNameRule) isFullyQualified(objName string) bool {
	if objName == "" {
		return true
	}
	re := regexp.MustCompile(`.+\..+`)
	return re.MatchString(objName)
}

// findAllTablesInSelect finds all table references in a SELECT statement
func (r *fullyQualifiedObjectNameRule) findAllTablesInSelect(statement string) []base.ColumnResource {
	// Parse the statement to extract table references
	parsedStatements, err := pgparser.ParsePostgreSQL(statement)
	if err != nil {
		return nil
	}

	// Use a visitor to find all table references
	collector := &tableReferenceCollector{
		schemaNameMap: r.getSchemaNameMapFromPublic(),
	}

	for _, antlrAST := range parsedStatements {
		if antlrAST != nil && antlrAST.Tree != nil {
			antlr.ParseTreeWalkerDefault.Walk(collector, antlrAST.Tree)
		}
	}

	return collector.tables
}

// getSchemaNameMapFromPublic creates a map of table names from the database schema
func (r *fullyQualifiedObjectNameRule) getSchemaNameMapFromPublic() map[string]bool {
	if r.dbMetadata == nil || r.dbMetadata.Schemas == nil {
		return nil
	}

	filterMap := map[string]bool{}
	for _, schema := range r.dbMetadata.Schemas {
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
