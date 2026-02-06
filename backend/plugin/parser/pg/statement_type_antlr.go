package pg

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// statementTypeCollectorWithPosition collects statement types with positions.
type statementTypeCollectorWithPosition struct {
	*parser.BasePostgreSQLParserListener
	tokens   *antlr.CommonTokenStream
	baseLine int
	results  []StatementTypeWithPosition
}

// Helper function to add statement with position.
func (c *statementTypeCollectorWithPosition) addStatement(stmtType storepb.StatementType, ctx antlr.ParserRuleContext) {
	if stmtType == storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED {
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
	c.addStatement(storepb.StatementType_CREATE_TABLE, ctx)
}

// CREATE VIEW statements
func (c *statementTypeCollectorWithPosition) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_CREATE_VIEW, ctx)
}

// CREATE INDEX statements
func (c *statementTypeCollectorWithPosition) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_CREATE_INDEX, ctx)
}

// CREATE SEQUENCE statements
func (c *statementTypeCollectorWithPosition) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_CREATE_SEQUENCE, ctx)
}

// CREATE SCHEMA statements
func (c *statementTypeCollectorWithPosition) EnterCreateschemastmt(ctx *parser.CreateschemastmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_CREATE_SCHEMA, ctx)
}

// CREATE FUNCTION statements
func (c *statementTypeCollectorWithPosition) EnterCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_CREATE_FUNCTION, ctx)
}

// CREATE TRIGGER statements
func (c *statementTypeCollectorWithPosition) EnterCreatetrigstmt(ctx *parser.CreatetrigstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_CREATE_TRIGGER, ctx)
}

// CREATE EXTENSION statements
func (c *statementTypeCollectorWithPosition) EnterCreateextensionstmt(ctx *parser.CreateextensionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_CREATE_EXTENSION, ctx)
}

// CREATE DATABASE statements
func (c *statementTypeCollectorWithPosition) EnterCreatedbstmt(ctx *parser.CreatedbstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_CREATE_DATABASE, ctx)
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
		c.addStatement(storepb.StatementType_ALTER_VIEW, ctx)
		return
	}

	// Always return ALTER_TABLE for ALTER TABLE statements
	// The sub-operations (ADD COLUMN, DROP COLUMN, etc.) are child nodes in the AST,
	// not separate statements
	c.addStatement(storepb.StatementType_ALTER_TABLE, ctx)
}

func (c *statementTypeCollectorWithPosition) EnterAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_ALTER_SEQUENCE, ctx)
}

func (c *statementTypeCollectorWithPosition) EnterAlterenumstmt(ctx *parser.AlterenumstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_ALTER_TYPE, ctx)
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
		c.addStatement(storepb.StatementType_RENAME_INDEX, ctx)
		return
	}
	if ctx.SCHEMA() != nil {
		c.addStatement(storepb.StatementType_RENAME_SCHEMA, ctx)
		return
	}
	if ctx.SEQUENCE() != nil {
		c.addStatement(storepb.StatementType_RENAME_SEQUENCE, ctx)
		return
	}

	// All other RENAME operations (TABLE, COLUMN, CONSTRAINT, VIEW) are wrapped in AlterTableStmt
	// Check if it's a VIEW to return ALTER_VIEW, otherwise return ALTER_TABLE
	if ctx.VIEW() != nil {
		c.addStatement(storepb.StatementType_ALTER_VIEW, ctx)
	} else if ctx.TABLE() != nil {
		// RENAME TABLE, RENAME COLUMN, RENAME CONSTRAINT all use TABLE keyword
		c.addStatement(storepb.StatementType_ALTER_TABLE, ctx)
	}
}

// COMMENT statements
func (c *statementTypeCollectorWithPosition) EnterCommentstmt(ctx *parser.CommentstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_COMMENT, ctx)
}

// CREATE TYPE statements
func (c *statementTypeCollectorWithPosition) EnterDefinestmt(ctx *parser.DefinestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	// Check if this is CREATE TYPE
	if ctx.TYPE_P() != nil {
		c.addStatement(storepb.StatementType_CREATE_TYPE, ctx)
	}
}

// DROP FUNCTION statements
func (c *statementTypeCollectorWithPosition) EnterRemovefuncstmt(ctx *parser.RemovefuncstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_DROP_FUNCTION, ctx)
}

// DROP DATABASE statements
func (c *statementTypeCollectorWithPosition) EnterDropdbstmt(ctx *parser.DropdbstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_DROP_DATABASE, ctx)
}

// DML statements
func (c *statementTypeCollectorWithPosition) EnterInsertstmt(ctx *parser.InsertstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_INSERT, ctx)
}

func (c *statementTypeCollectorWithPosition) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_UPDATE, ctx)
}

func (c *statementTypeCollectorWithPosition) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_DELETE, ctx)
}

// getDropStatementType determines the specific DROP statement type.
func getDropStatementType(ctx *parser.DropstmtContext) storepb.StatementType {
	if ctx == nil {
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}

	// Check TYPE_P for DROP TYPE
	if ctx.TYPE_P() != nil {
		return storepb.StatementType_DROP_TYPE
	}

	// Check DOMAIN_P for DROP DOMAIN (not in legacy, return UNKNOWN)
	if ctx.DOMAIN_P() != nil {
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}

	// Check object_type_any_name (TABLE, SEQUENCE, VIEW, INDEX, etc.)
	if ctx.Object_type_any_name() != nil {
		objType := ctx.Object_type_any_name()
		if objType.TABLE() != nil {
			return storepb.StatementType_DROP_TABLE
		}
		if objType.VIEW() != nil {
			if objType.MATERIALIZED() != nil {
				// DROP MATERIALIZED VIEW holds data — treat as DROP_TABLE for risk assessment.
				return storepb.StatementType_DROP_TABLE
			}
			return storepb.StatementType_DROP_VIEW
		}
		if objType.INDEX() != nil {
			return storepb.StatementType_DROP_INDEX
		}
		if objType.SEQUENCE() != nil {
			return storepb.StatementType_DROP_SEQUENCE
		}
		// Other types like COLLATION, CONVERSION, STATISTICS not in legacy
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}

	// Check drop_type_name (SCHEMA, EXTENSION, etc.)
	if ctx.Drop_type_name() != nil {
		dropType := ctx.Drop_type_name()
		if dropType.SCHEMA() != nil {
			return storepb.StatementType_DROP_SCHEMA
		}
		if dropType.EXTENSION() != nil {
			return storepb.StatementType_DROP_EXTENSION
		}
		// Other types like ACCESS METHOD, EVENT TRIGGER, PUBLICATION not in legacy
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}

	// Check object_type_name_on_any_name (TRIGGER, RULE, POLICY with "ON table" syntax)
	if ctx.Object_type_name_on_any_name() != nil {
		objTypeOn := ctx.Object_type_name_on_any_name()
		if objTypeOn.TRIGGER() != nil {
			return storepb.StatementType_DROP_TRIGGER
		}
		// RULE and POLICY not in legacy
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}

	// Default for unhandled cases
	return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
}
