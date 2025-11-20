package pg

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"
)

// Statement type constants.
const (
	// CREATE statements.
	stmtTypeCreateTable     = "CREATE_TABLE"
	stmtTypeCreateView      = "CREATE_VIEW"
	stmtTypeCreateIndex     = "CREATE_INDEX"
	stmtTypeCreateSequence  = "CREATE_SEQUENCE"
	stmtTypeCreateSchema    = "CREATE_SCHEMA"
	stmtTypeCreateFunction  = "CREATE_FUNCTION"
	stmtTypeCreateTrigger   = "CREATE_TRIGGER"
	stmtTypeCreateExtension = "CREATE_EXTENSION"
	stmtTypeCreateDatabase  = "CREATE_DATABASE"
	stmtTypeCreateType      = "CREATE_TYPE"

	// DROP statements.
	stmtTypeDropTable     = "DROP_TABLE"
	stmtTypeDropIndex     = "DROP_INDEX"
	stmtTypeDropSchema    = "DROP_SCHEMA"
	stmtTypeDropSequence  = "DROP_SEQUENCE"
	stmtTypeDropExtension = "DROP_EXTENSION"
	stmtTypeDropDatabase  = "DROP_DATABASE"
	stmtTypeDropType      = "DROP_TYPE"
	stmtTypeDropTrigger   = "DROP_TRIGGER"
	stmtTypeDropFunction  = "DROP_FUNCTION"

	// ALTER statements.
	stmtTypeAlterTable    = "ALTER_TABLE"
	stmtTypeAlterView     = "ALTER_VIEW"
	stmtTypeAlterSequence = "ALTER_SEQUENCE"
	stmtTypeAlterType     = "ALTER_TYPE"

	// RENAME statements.
	stmtTypeRenameIndex    = "RENAME_INDEX"
	stmtTypeRenameSchema   = "RENAME_SCHEMA"
	stmtTypeRenameSequence = "RENAME_SEQUENCE"

	// DML statements.
	stmtTypeInsert = "INSERT"
	stmtTypeUpdate = "UPDATE"
	stmtTypeDelete = "DELETE"

	// Other statements.
	stmtTypeComment = "COMMENT"

	// Special value for filtering.
	stmtTypeUnknown = "UNKNOWN"
)

// statementTypeCollectorWithPosition collects statement types with positions.
type statementTypeCollectorWithPosition struct {
	*parser.BasePostgreSQLParserListener
	tokens   *antlr.CommonTokenStream
	baseLine int
	results  []StatementTypeWithPosition
}

// Helper function to add statement with position.
func (c *statementTypeCollectorWithPosition) addStatement(stmtType string, ctx antlr.ParserRuleContext) {
	if stmtType == "" || stmtType == stmtTypeUnknown {
		return
	}

	// Get line number from the stop token
	line := 0
	if ctx.GetStop() != nil {
		line = ctx.GetStop().GetLine() + c.baseLine
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
	c.addStatement(stmtTypeCreateTable, ctx)
}

// CREATE VIEW statements
func (c *statementTypeCollectorWithPosition) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeCreateView, ctx)
}

// CREATE INDEX statements
func (c *statementTypeCollectorWithPosition) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeCreateIndex, ctx)
}

// CREATE SEQUENCE statements
func (c *statementTypeCollectorWithPosition) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeCreateSequence, ctx)
}

// CREATE SCHEMA statements
func (c *statementTypeCollectorWithPosition) EnterCreateschemastmt(ctx *parser.CreateschemastmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeCreateSchema, ctx)
}

// CREATE FUNCTION statements
func (c *statementTypeCollectorWithPosition) EnterCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeCreateFunction, ctx)
}

// CREATE TRIGGER statements
func (c *statementTypeCollectorWithPosition) EnterCreatetrigstmt(ctx *parser.CreatetrigstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeCreateTrigger, ctx)
}

// CREATE EXTENSION statements
func (c *statementTypeCollectorWithPosition) EnterCreateextensionstmt(ctx *parser.CreateextensionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeCreateExtension, ctx)
}

// CREATE DATABASE statements
func (c *statementTypeCollectorWithPosition) EnterCreatedbstmt(ctx *parser.CreatedbstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeCreateDatabase, ctx)
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
		c.addStatement(stmtTypeAlterView, ctx)
		return
	}

	// Always return ALTER_TABLE for ALTER TABLE statements
	// The sub-operations (ADD COLUMN, DROP COLUMN, etc.) are child nodes in the AST,
	// not separate statements
	c.addStatement(stmtTypeAlterTable, ctx)
}

func (c *statementTypeCollectorWithPosition) EnterAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeAlterSequence, ctx)
}

func (c *statementTypeCollectorWithPosition) EnterAlterenumstmt(ctx *parser.AlterenumstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeAlterType, ctx)
}

// RENAME statements
func (c *statementTypeCollectorWithPosition) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// In legacy implementation:
	// - RENAME INDEX, RENAME SCHEMA, RENAME SEQUENCE are standalone top-level nodes
	// - RENAME TABLE, RENAME COLUMN, RENAME CONSTRAINT, RENAME VIEW are wrapped in AlterTableStmt
	//   - When wrapped, AlterTableStmt.Table.Type determines the statement type:
	//     - TableTypeView → "ALTER_VIEW"
	//     - TableTypeBaseTable → "ALTER_TABLE"

	// Check for top-level RENAME operations that are NOT wrapped in AlterTableStmt
	if ctx.INDEX() != nil {
		c.addStatement(stmtTypeRenameIndex, ctx)
		return
	}
	if ctx.SCHEMA() != nil {
		c.addStatement(stmtTypeRenameSchema, ctx)
		return
	}
	if ctx.SEQUENCE() != nil {
		c.addStatement(stmtTypeRenameSequence, ctx)
		return
	}

	// All other RENAME operations (TABLE, COLUMN, CONSTRAINT, VIEW) are wrapped in AlterTableStmt
	// Check if it's a VIEW to return ALTER_VIEW, otherwise return ALTER_TABLE
	if ctx.VIEW() != nil {
		c.addStatement(stmtTypeAlterView, ctx)
	} else if ctx.TABLE() != nil {
		// RENAME TABLE, RENAME COLUMN, RENAME CONSTRAINT all use TABLE keyword
		c.addStatement(stmtTypeAlterTable, ctx)
	}
}

// COMMENT statements
func (c *statementTypeCollectorWithPosition) EnterCommentstmt(ctx *parser.CommentstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeComment, ctx)
}

// CREATE TYPE statements
func (c *statementTypeCollectorWithPosition) EnterDefinestmt(ctx *parser.DefinestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	// Check if this is CREATE TYPE
	if ctx.TYPE_P() != nil {
		c.addStatement(stmtTypeCreateType, ctx)
	}
}

// DROP FUNCTION statements
func (c *statementTypeCollectorWithPosition) EnterRemovefuncstmt(ctx *parser.RemovefuncstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeDropFunction, ctx)
}

// DROP DATABASE statements
func (c *statementTypeCollectorWithPosition) EnterDropdbstmt(ctx *parser.DropdbstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeDropDatabase, ctx)
}

// DML statements
func (c *statementTypeCollectorWithPosition) EnterInsertstmt(ctx *parser.InsertstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeInsert, ctx)
}

func (c *statementTypeCollectorWithPosition) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeUpdate, ctx)
}

func (c *statementTypeCollectorWithPosition) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(stmtTypeDelete, ctx)
}

// getDropStatementType determines the specific DROP statement type.
func getDropStatementType(ctx *parser.DropstmtContext) string {
	if ctx == nil {
		return ""
	}

	// Check TYPE_P for DROP TYPE
	if ctx.TYPE_P() != nil {
		return stmtTypeDropType
	}

	// Check DOMAIN_P for DROP DOMAIN (not in legacy, return UNKNOWN)
	if ctx.DOMAIN_P() != nil {
		return stmtTypeUnknown
	}

	// Check object_type_any_name (TABLE, SEQUENCE, VIEW, INDEX, etc.)
	if ctx.Object_type_any_name() != nil {
		objType := ctx.Object_type_any_name()
		if objType.TABLE() != nil {
			return stmtTypeDropTable
		}
		if objType.VIEW() != nil {
			// Check if MATERIALIZED VIEW or regular VIEW
			// Both treated as DROP_TABLE in legacy
			return stmtTypeDropTable
		}
		if objType.INDEX() != nil {
			return stmtTypeDropIndex
		}
		if objType.SEQUENCE() != nil {
			return stmtTypeDropSequence
		}
		// Other types like COLLATION, CONVERSION, STATISTICS not in legacy
		return stmtTypeUnknown
	}

	// Check drop_type_name (SCHEMA, EXTENSION, etc.)
	if ctx.Drop_type_name() != nil {
		dropType := ctx.Drop_type_name()
		if dropType.SCHEMA() != nil {
			return stmtTypeDropSchema
		}
		if dropType.EXTENSION() != nil {
			return stmtTypeDropExtension
		}
		// Other types like ACCESS METHOD, EVENT TRIGGER, PUBLICATION not in legacy
		return stmtTypeUnknown
	}

	// Check object_type_name_on_any_name (TRIGGER, RULE, POLICY with "ON table" syntax)
	if ctx.Object_type_name_on_any_name() != nil {
		objTypeOn := ctx.Object_type_name_on_any_name()
		if objTypeOn.TRIGGER() != nil {
			return stmtTypeDropTrigger
		}
		// RULE and POLICY not in legacy
		return stmtTypeUnknown
	}

	// Default for unhandled cases
	return stmtTypeUnknown
}
