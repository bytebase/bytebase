package redshift

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/redshift"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// StatementTypeWithPosition contains statement type and its position information.
type StatementTypeWithPosition struct {
	Type storepb.StatementType
	// Line is the one-based line number where the statement ends.
	Line int
	Text string
}

// GetStatementTypesWithPosition returns statement types with position information.
// The line numbers are one-based.
func GetStatementTypesWithPosition(asts []base.AST) ([]StatementTypeWithPosition, error) {
	if len(asts) == 0 {
		return []StatementTypeWithPosition{}, nil
	}

	var allResults []StatementTypeWithPosition
	for _, unifiedAST := range asts {
		antlrAST, ok := base.GetANTLRAST(unifiedAST)
		if !ok {
			return nil, errors.New("expected ANTLR AST for Redshift")
		}
		if antlrAST.Tree == nil {
			return nil, errors.New("ANTLR tree is nil")
		}

		collector := &statementTypeCollectorWithPosition{
			tokens:   antlrAST.Tokens,
			baseLine: base.GetLineOffset(antlrAST.StartPosition),
		}

		antlr.ParseTreeWalkerDefault.Walk(collector, antlrAST.Tree)
		allResults = append(allResults, collector.results...)
	}

	return allResults, nil
}

// GetStatementTypes returns only the statement types as strings.
// This is used for registration with base.RegisterGetStatementTypes.
func GetStatementTypes(asts []base.AST) ([]storepb.StatementType, error) {
	results, err := GetStatementTypesWithPosition(asts)
	if err != nil {
		return nil, err
	}
	types := make([]storepb.StatementType, len(results))
	for i, r := range results {
		types[i] = r.Type
	}
	return types, nil
}

// statementTypeCollectorWithPosition collects statement types with positions.
type statementTypeCollectorWithPosition struct {
	*parser.BaseRedshiftParserListener
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

// isTopLevel checks if the context is at top level.
func isTopLevel(ctx antlr.Tree) bool {
	if ctx == nil {
		return true
	}

	switch ctx := ctx.(type) {
	case *parser.RootContext, *parser.StmtblockContext:
		return true
	case *parser.StmtmultiContext, *parser.StmtContext:
		return isTopLevel(ctx.GetParent())
	default:
		return false
	}
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

// CREATE MATERIALIZED VIEW statements
func (c *statementTypeCollectorWithPosition) EnterCreatematviewstmt(ctx *parser.CreatematviewstmtContext) {
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
	c.addStatement(storepb.StatementType_ALTER_TABLE, ctx)
}

// ALTER MATERIALIZED VIEW statements
func (c *statementTypeCollectorWithPosition) EnterAltermaterializedviewstmt(ctx *parser.AltermaterializedviewstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_ALTER_VIEW, ctx)
}

// ALTER EXTERNAL VIEW statements
func (c *statementTypeCollectorWithPosition) EnterAlterexternalviewstmt(ctx *parser.AlterexternalviewstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_ALTER_VIEW, ctx)
}

func (c *statementTypeCollectorWithPosition) EnterAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_ALTER_SEQUENCE, ctx)
}

// RENAME statements
func (c *statementTypeCollectorWithPosition) EnterRenamestmt(ctx *parser.RenamestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Check for top-level RENAME operations
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

	// All other RENAME operations
	if ctx.VIEW() != nil {
		c.addStatement(storepb.StatementType_ALTER_VIEW, ctx)
	} else if ctx.TABLE() != nil {
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

// DROP SCHEMA statements
func (c *statementTypeCollectorWithPosition) EnterDropschemastmt(ctx *parser.DropschemastmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}
	c.addStatement(storepb.StatementType_DROP_SCHEMA, ctx)
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
		return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
	}

	// Note: DROP SCHEMA is handled by DropschemastmtContext, not DropstmtContext
	return storepb.StatementType_STATEMENT_TYPE_UNSPECIFIED
}
