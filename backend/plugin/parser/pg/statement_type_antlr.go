package pg

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"
)

// statementTypeCollectorWithPosition collects statement types with positions.
type statementTypeCollectorWithPosition struct {
	*parser.BasePostgreSQLParserListener
	tokens  *antlr.CommonTokenStream
	results []StatementTypeWithPosition
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
func (c *statementTypeCollectorWithPosition) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_TABLE", ctx)
}

// CREATE VIEW statements
func (c *statementTypeCollectorWithPosition) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_VIEW", ctx)
}

// CREATE INDEX statements
func (c *statementTypeCollectorWithPosition) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_INDEX", ctx)
}

// CREATE SEQUENCE statements
func (c *statementTypeCollectorWithPosition) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_SEQUENCE", ctx)
}

// CREATE SCHEMA statements
func (c *statementTypeCollectorWithPosition) EnterCreateschemastmt(ctx *parser.CreateschemastmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_SCHEMA", ctx)
}

// CREATE FUNCTION statements
func (c *statementTypeCollectorWithPosition) EnterCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_FUNCTION", ctx)
}

// CREATE TRIGGER statements
func (c *statementTypeCollectorWithPosition) EnterCreatetrigstmt(ctx *parser.CreatetrigstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_TRIGGER", ctx)
}

// CREATE EXTENSION statements
func (c *statementTypeCollectorWithPosition) EnterCreateextensionstmt(ctx *parser.CreateextensionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_EXTENSION", ctx)
}

// CREATE DATABASE statements
func (c *statementTypeCollectorWithPosition) EnterCreatedbstmt(ctx *parser.CreatedbstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("CREATE_DATABASE", ctx)
}

// DROP statements
func (c *statementTypeCollectorWithPosition) EnterDropstmt(ctx *parser.DropstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(getDropStatementType(ctx), ctx)
}

// ALTER statements
func (c *statementTypeCollectorWithPosition) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check if this is ALTER VIEW
	if ctx.VIEW() != nil {
		c.addStatement("ALTER_VIEW", ctx)
		return
	}

	// Always return ALTER_TABLE for ALTER TABLE statements
	// The sub-operations (ADD COLUMN, DROP COLUMN, etc.) are child nodes in the AST,
	// not separate statements
	c.addStatement("ALTER_TABLE", ctx)
}

func (c *statementTypeCollectorWithPosition) EnterAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("ALTER_SEQUENCE", ctx)
}

func (c *statementTypeCollectorWithPosition) EnterAlterenumstmt(ctx *parser.AlterenumstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("ALTER_TYPE", ctx)
}

// RENAME statements
func (c *statementTypeCollectorWithPosition) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(getRenameStatementType(ctx), ctx)
}

// COMMENT statements
func (c *statementTypeCollectorWithPosition) EnterCommentstmt(ctx *parser.CommentstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("COMMENT", ctx)
}

// CREATE TYPE statements
func (c *statementTypeCollectorWithPosition) EnterDefinestmt(ctx *parser.DefinestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	// Check if this is CREATE TYPE
	if ctx.TYPE_P() != nil {
		c.addStatement("CREATE_TYPE", ctx)
	}
}

// DROP FUNCTION statements
func (c *statementTypeCollectorWithPosition) EnterRemovefuncstmt(ctx *parser.RemovefuncstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("DROP_FUNCTION", ctx)
}

// DROP DATABASE statements
func (c *statementTypeCollectorWithPosition) EnterDropdbstmt(ctx *parser.DropdbstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("DROP_DATABASE", ctx)
}

// DML statements
func (c *statementTypeCollectorWithPosition) EnterInsertstmt(ctx *parser.InsertstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("INSERT", ctx)
}

func (c *statementTypeCollectorWithPosition) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement("UPDATE", ctx)
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

	// Check TYPE_P for DROP TYPE
	if ctx.TYPE_P() != nil {
		return "DROP_TYPE"
	}

	// Check DOMAIN_P for DROP DOMAIN (not in legacy, return UNKNOWN)
	if ctx.DOMAIN_P() != nil {
		return "UNKNOWN"
	}

	// Check object_type_any_name (TABLE, SEQUENCE, VIEW, INDEX, etc.)
	if ctx.Object_type_any_name() != nil {
		objType := ctx.Object_type_any_name()
		if objType.TABLE() != nil {
			return "DROP_TABLE"
		}
		if objType.VIEW() != nil {
			// Check if MATERIALIZED VIEW or regular VIEW
			// Both treated as DROP_TABLE in legacy
			return "DROP_TABLE"
		}
		if objType.INDEX() != nil {
			return "DROP_INDEX"
		}
		if objType.SEQUENCE() != nil {
			return "DROP_SEQUENCE"
		}
		// Other types like COLLATION, CONVERSION, STATISTICS not in legacy
		return "UNKNOWN"
	}

	// Check drop_type_name (SCHEMA, EXTENSION, etc.)
	if ctx.Drop_type_name() != nil {
		dropType := ctx.Drop_type_name()
		if dropType.SCHEMA() != nil {
			return "DROP_SCHEMA"
		}
		if dropType.EXTENSION() != nil {
			return "DROP_EXTENSION"
		}
		// Other types like ACCESS METHOD, EVENT TRIGGER, PUBLICATION not in legacy
		return "UNKNOWN"
	}

	// Check object_type_name_on_any_name (TRIGGER, RULE, POLICY with "ON table" syntax)
	if ctx.Object_type_name_on_any_name() != nil {
		objTypeOn := ctx.Object_type_name_on_any_name()
		if objTypeOn.TRIGGER() != nil {
			return "DROP_TRIGGER"
		}
		// RULE and POLICY not in legacy
		return "UNKNOWN"
	}

	// Default for unhandled cases
	return "UNKNOWN"
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
	// Return UNKNOWN to maintain backward compatibility
	return "UNKNOWN"
}
