package pg

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"
	"github.com/pkg/errors"
)

// GetStatementTypesANTLR returns statement types from ANTLR parse result.
func GetStatementTypesANTLR(parseResult *ParseResult) ([]string, error) {
	if parseResult == nil || parseResult.Tree == nil {
		return nil, errors.New("invalid parse result")
	}

	collector := &statementTypeCollector{
		types: make(map[string]bool),
	}

	antlr.ParseTreeWalkerDefault.Walk(collector, parseResult.Tree)

	var sqlTypes []string
	for sqlType := range collector.types {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

// GetStatementTypesWithPositionsANTLR returns statement types with position information from ANTLR parse result.
func GetStatementTypesWithPositionsANTLR(parseResult *ParseResult) ([]StatementTypeWithPosition, error) {
	if parseResult == nil || parseResult.Tree == nil {
		return nil, errors.New("invalid parse result")
	}

	collector := &statementTypeCollectorWithPosition{
		tokens: parseResult.Tokens,
	}

	antlr.ParseTreeWalkerDefault.Walk(collector, parseResult.Tree)

	return collector.results, nil
}

// statementTypeCollector collects unique statement types.
type statementTypeCollector struct {
	*parser.BasePostgreSQLParserListener
	types map[string]bool
}

// statementTypeCollectorWithPosition collects statement types with positions.
type statementTypeCollectorWithPosition struct {
	*parser.BasePostgreSQLParserListener
	tokens  *antlr.CommonTokenStream
	results []StatementTypeWithPosition
}

// Helper function to add statement type.
func (c *statementTypeCollector) addType(stmtType string) {
	if stmtType != "" && stmtType != "UNKNOWN" {
		c.types[stmtType] = true
	}
}

// Helper function to add statement with position.
func (c *statementTypeCollectorWithPosition) addStatement(stmtType string, ctx antlr.ParserRuleContext) {
	if stmtType == "" || stmtType == "UNKNOWN" {
		return
	}

	// Get line number from the stop token
	line := 0
	if ctx.GetStop() != nil {
		line = ctx.GetStop().GetLine()
	}

	// Get statement text
	text := ""
	if c.tokens != nil && ctx.GetStart() != nil && ctx.GetStop() != nil {
		text = c.tokens.GetTextFromInterval(antlr.NewInterval(
			ctx.GetStart().GetTokenIndex(),
			ctx.GetStop().GetTokenIndex(),
		))
	}

	c.results = append(c.results, StatementTypeWithPosition{
		Type: stmtType,
		Line: line,
		Text: text,
	})
}

// CREATE TABLE statements
func (c *statementTypeCollector) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("CREATE_TABLE")
}

func (c *statementTypeCollectorWithPosition) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_TABLE", ctx)
}

// CREATE VIEW statements
func (c *statementTypeCollector) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("CREATE_VIEW")
}

func (c *statementTypeCollectorWithPosition) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_VIEW", ctx)
}

// CREATE INDEX statements
func (c *statementTypeCollector) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("CREATE_INDEX")
}

func (c *statementTypeCollectorWithPosition) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_INDEX", ctx)
}

// CREATE SEQUENCE statements
func (c *statementTypeCollector) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("CREATE_SEQUENCE")
}

func (c *statementTypeCollectorWithPosition) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_SEQUENCE", ctx)
}

// CREATE SCHEMA statements
func (c *statementTypeCollector) EnterCreateschemastmt(ctx *parser.CreateschemastmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("CREATE_SCHEMA")
}

func (c *statementTypeCollectorWithPosition) EnterCreateschemastmt(ctx *parser.CreateschemastmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_SCHEMA", ctx)
}

// CREATE FUNCTION statements
func (c *statementTypeCollector) EnterCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("CREATE_FUNCTION")
}

func (c *statementTypeCollectorWithPosition) EnterCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_FUNCTION", ctx)
}

// CREATE TRIGGER statements
func (c *statementTypeCollector) EnterCreatetrigstmt(ctx *parser.CreatetrigstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("CREATE_TRIGGER")
}

func (c *statementTypeCollectorWithPosition) EnterCreatetrigstmt(ctx *parser.CreatetrigstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_TRIGGER", ctx)
}

// CREATE EXTENSION statements
func (c *statementTypeCollector) EnterCreateextensionstmt(ctx *parser.CreateextensionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("CREATE_EXTENSION")
}

func (c *statementTypeCollectorWithPosition) EnterCreateextensionstmt(ctx *parser.CreateextensionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_EXTENSION", ctx)
}

// CREATE DATABASE statements
func (c *statementTypeCollector) EnterCreatedbstmt(ctx *parser.CreatedbstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("CREATE_DATABASE")
}

func (c *statementTypeCollectorWithPosition) EnterCreatedbstmt(ctx *parser.CreatedbstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_DATABASE", ctx)
}

// DROP statements
func (c *statementTypeCollector) EnterDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType(getDropStatementType(ctx))
}

func (c *statementTypeCollectorWithPosition) EnterDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(getDropStatementType(ctx), ctx)
}

// ALTER statements
func (c *statementTypeCollector) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("ALTER_TABLE")
}

func (c *statementTypeCollectorWithPosition) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("ALTER_TABLE", ctx)
}

func (c *statementTypeCollector) EnterAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("ALTER_SEQUENCE")
}

func (c *statementTypeCollectorWithPosition) EnterAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("ALTER_SEQUENCE", ctx)
}

func (c *statementTypeCollector) EnterAlterenumstmt(ctx *parser.AlterenumstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("ALTER_TYPE")
}

func (c *statementTypeCollectorWithPosition) EnterAlterenumstmt(ctx *parser.AlterenumstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("ALTER_TYPE", ctx)
}

// RENAME statements
func (c *statementTypeCollector) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType(getRenameStatementType(ctx))
}

func (c *statementTypeCollectorWithPosition) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(getRenameStatementType(ctx), ctx)
}

// COMMENT statements
func (c *statementTypeCollector) EnterCommentstmt(ctx *parser.CommentstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("COMMENT")
}

func (c *statementTypeCollectorWithPosition) EnterCommentstmt(ctx *parser.CommentstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("COMMENT", ctx)
}

// DML statements
func (c *statementTypeCollector) EnterInsertstmt(ctx *parser.InsertstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("INSERT")
}

func (c *statementTypeCollectorWithPosition) EnterInsertstmt(ctx *parser.InsertstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("INSERT", ctx)
}

func (c *statementTypeCollector) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("UPDATE")
}

func (c *statementTypeCollectorWithPosition) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("UPDATE", ctx)
}

func (c *statementTypeCollector) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addType("DELETE")
}

func (c *statementTypeCollectorWithPosition) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("DELETE", ctx)
}

// getDropStatementType determines the specific DROP statement type.
func getDropStatementType(ctx *parser.DropstmtContext) string {
	if ctx == nil {
		return ""
	}

	// Check object type
	if ctx.Object_type_any_name() != nil {
		objType := ctx.Object_type_any_name()
		if objType.TABLE() != nil {
			return "DROP_TABLE"
		}
		if objType.VIEW() != nil {
			return "DROP_TABLE" // Views treated as tables in legacy
		}
		if objType.INDEX() != nil {
			return "DROP_INDEX"
		}
		if objType.SEQUENCE() != nil {
			return "DROP_SEQUENCE"
		}
	}

	// Check drop type name for SCHEMA
	if ctx.Drop_type_name() != nil && ctx.Drop_type_name().SCHEMA() != nil {
		return "DROP_SCHEMA"
	}

	// Default - could be DROP FUNCTION, DROP TYPE, etc.
	// We return a generic DROP for unhandled cases
	return "DROP"
}

// getRenameStatementType determines the specific RENAME statement type.
func getRenameStatementType(ctx *parser.RenamestmtContext) string {
	if ctx == nil {
		return ""
	}

	// Check for ALTER TABLE variants
	if ctx.TABLE() != nil {
		// ALTER TABLE ... RENAME CONSTRAINT ... TO ...
		if ctx.CONSTRAINT() != nil {
			return "RENAME_CONSTRAINT"
		}
		// ALTER TABLE ... RENAME [COLUMN] ... TO ...
		// RENAME_COLUMN has 2 name elements (old_name, new_name)
		// RENAME_TABLE has 1 name element (new_table_name)
		// The table name is in relation_expr, not counted in AllName()
		if ctx.Opt_column() != nil || len(ctx.AllName()) >= 2 {
			return "RENAME_COLUMN"
		}
		// ALTER TABLE ... RENAME TO ...
		return "RENAME_TABLE"
	}

	// Check for ALTER INDEX
	if ctx.INDEX() != nil {
		return "RENAME_INDEX"
	}

	// Check for ALTER SCHEMA
	if ctx.SCHEMA() != nil {
		return "RENAME_SCHEMA"
	}

	// Check for ALTER SEQUENCE
	if ctx.SEQUENCE() != nil {
		return "RENAME_SEQUENCE"
	}

	// Check for ALTER VIEW (includes MATERIALIZED VIEW)
	if ctx.VIEW() != nil {
		return "RENAME_VIEW"
	}

	// Default for other RENAME types (AGGREGATE, COLLATION, DOMAIN, FUNCTION, etc.)
	return "RENAME"
}
