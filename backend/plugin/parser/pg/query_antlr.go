package pg

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// validateQueryANTLR validates the SQL statement for SQL editor using ANTLR parser.
// Returns (isReadOnly, allQueriesReturnData, error)
//
// Validation rules:
// 1. Allow: SELECT statements
// 2. Allow: EXPLAIN statements (but not EXPLAIN ANALYZE for non-SELECT)
// 3. Allow: SHOW/SET statements (SET is considered executable)
// 4. Allow: CTEs with only SELECT (reject CTEs with INSERT/UPDATE/DELETE)
// 5. Reject: All other statements (DDL, DML except SELECT)
func validateQueryANTLR(statement string) (bool, bool, error) {
	// Parse with ANTLR
	results, err := ParsePostgreSQL(statement)
	if err != nil {
		// Return syntax error
		if syntaxErr, ok := err.(*base.SyntaxError); ok {
			return false, false, syntaxErr
		}
		return false, false, err
	}

	// Create a single listener to accumulate validation state across all statements
	listener := &queryValidatorListener{
		hasExecute:     false,
		explainAnalyze: false,
		hasCTE:         false,
		hasInvalidStmt: false,
		validStmtCount: 0,
	}

	// Walk through each parsed statement
	for _, result := range results {
		tree := result.Tree

		// Analyze the parse tree
		antlr.ParseTreeWalkerDefault.Walk(listener, tree)

		// If we found invalid statements, reject immediately
		if listener.hasInvalidStmt {
			return false, false, nil
		}

		// Check for DML in CTEs or EXPLAIN ANALYZE
		// This is done in a second pass to detect DML statements anywhere in the tree
		if listener.hasCTE || listener.explainAnalyze {
			dmlDetector := &dmlDetectorListener{}
			antlr.ParseTreeWalkerDefault.Walk(dmlDetector, tree)

			if dmlDetector.hasDML {
				return false, false, nil
			}
		}
	}

	// Return validation results
	// isReadOnly = true (all valid queries are read-only in this context)
	// allQueriesReturnData = !hasExecute (SET statements don't return data)
	return true, !listener.hasExecute, nil
}

// queryValidatorListener walks the ANTLR parse tree to validate query types
type queryValidatorListener struct {
	*parser.BasePostgreSQLParserListener

	hasExecute     bool // True if there are SET statements
	explainAnalyze bool // True if there's EXPLAIN ANALYZE
	hasCTE         bool // True if there's a WITH clause (CTE)
	hasInvalidStmt bool // True if there's an invalid statement type
	validStmtCount int  // Count of valid statements encountered
}

// EnterSelectstmt detects CTEs and marks statement as valid
func (l *queryValidatorListener) EnterSelectstmt(ctx *parser.SelectstmtContext) {
	// Only count top-level SELECT statements
	if isTopLevelStmt(ctx) {
		l.validStmtCount++
	}

	// Check if this SELECT has a WITH clause (CTE)
	// WITH clause is in Select_no_parens
	if ctx.Select_no_parens() != nil {
		selectNoParens := ctx.Select_no_parens()
		if selectNoParens.With_clause() != nil {
			l.hasCTE = true
		}
	}
}

// EnterExplainstmt handles EXPLAIN statements
func (l *queryValidatorListener) EnterExplainstmt(ctx *parser.ExplainstmtContext) {
	if !isTopLevelStmt(ctx) {
		return
	}

	l.validStmtCount++

	// Check if it's EXPLAIN ANALYZE
	if l.isExplainAnalyze(ctx) {
		l.explainAnalyze = true
		// EXPLAIN ANALYZE is only valid for SELECT
		// Check the explained statement
		if ctx.Explainablestmt() != nil {
			explainableStmt := ctx.Explainablestmt()
			// If it's not a SELECT statement, mark as invalid
			if explainableStmt.Selectstmt() == nil {
				l.hasInvalidStmt = true
			}
		}
	}
}

// EnterVariablesetstmt handles SET statements
func (l *queryValidatorListener) EnterVariablesetstmt(ctx *parser.VariablesetstmtContext) {
	if !isTopLevelStmt(ctx) {
		return
	}

	l.validStmtCount++
	l.hasExecute = true
}

// EnterVariableshowstmt handles SHOW statements
func (l *queryValidatorListener) EnterVariableshowstmt(ctx *parser.VariableshowstmtContext) {
	if !isTopLevelStmt(ctx) {
		return
	}

	l.validStmtCount++
}

// Mark DML and DDL statements as invalid (only SELECT, EXPLAIN, SET, SHOW are allowed for SQL editor)
// DML should be detected at top level only, as they may appear in CTEs (which we check separately)
// DDL should always be rejected

// DDL statement handlers - always mark as invalid

// EnterCreatestmt detects CREATE statements (not allowed)
func (l *queryValidatorListener) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if isTopLevelStmt(ctx) {
		l.hasInvalidStmt = true
	}
}

// EnterAltertablestmt detects ALTER TABLE statements (not allowed)
func (l *queryValidatorListener) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if isTopLevelStmt(ctx) {
		l.hasInvalidStmt = true
	}
}

// EnterDropstmt detects DROP statements (not allowed)
func (l *queryValidatorListener) EnterDropstmt(ctx *parser.DropstmtContext) {
	if isTopLevelStmt(ctx) {
		l.hasInvalidStmt = true
	}
}

// EnterTruncatestmt detects TRUNCATE statements (not allowed)
func (l *queryValidatorListener) EnterTruncatestmt(ctx *parser.TruncatestmtContext) {
	if isTopLevelStmt(ctx) {
		l.hasInvalidStmt = true
	}
}

// DML statement handlers - mark as invalid at top level or inside EXPLAIN

// EnterInsertstmt detects INSERT statements (not allowed at top level)
func (l *queryValidatorListener) EnterInsertstmt(ctx *parser.InsertstmtContext) {
	// Check if this is at top level or inside EXPLAIN
	parent := ctx.GetParent()
	for parent != nil {
		switch parent.(type) {
		case *parser.ExplainstmtContext:
			// INSERT inside EXPLAIN - mark as invalid
			l.hasInvalidStmt = true
			return
		case *parser.RootContext, *parser.StmtblockContext, *parser.StmtmultiContext, *parser.StmtContext, *parser.ExplainablestmtContext:
			// Continue walking up - ExplainablestmtContext is the wrapper for statements in EXPLAIN
			parent = parent.GetParent()
		default:
			// Hit some other context - this is nested (e.g., in CTE, subquery)
			// We'll handle CTEs separately
			return
		}
	}
	// Reached root - this is a top-level INSERT
	l.hasInvalidStmt = true
}

// EnterUpdatestmt detects UPDATE statements (not allowed at top level)
func (l *queryValidatorListener) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	// Check if this is at top level or inside EXPLAIN
	parent := ctx.GetParent()
	for parent != nil {
		switch parent.(type) {
		case *parser.ExplainstmtContext:
			// UPDATE inside EXPLAIN - mark as invalid
			l.hasInvalidStmt = true
			return
		case *parser.RootContext, *parser.StmtblockContext, *parser.StmtmultiContext, *parser.StmtContext, *parser.ExplainablestmtContext:
			// Continue walking up - ExplainablestmtContext is the wrapper for statements in EXPLAIN
			parent = parent.GetParent()
		default:
			// Hit some other context - this is nested
			return
		}
	}
	// Reached root - this is a top-level UPDATE
	l.hasInvalidStmt = true
}

// EnterDeletestmt detects DELETE statements (not allowed at top level)
func (l *queryValidatorListener) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	// Check if this is at top level or inside EXPLAIN
	parent := ctx.GetParent()
	for parent != nil {
		switch parent.(type) {
		case *parser.ExplainstmtContext:
			// DELETE inside EXPLAIN - mark as invalid
			l.hasInvalidStmt = true
			return
		case *parser.RootContext, *parser.StmtblockContext, *parser.StmtmultiContext, *parser.StmtContext, *parser.ExplainablestmtContext:
			// Continue walking up - ExplainablestmtContext is the wrapper for statements in EXPLAIN
			parent = parent.GetParent()
		default:
			// Hit some other context - this is nested
			return
		}
	}
	// Reached root - this is a top-level DELETE
	l.hasInvalidStmt = true
}

// isExplainAnalyze checks if an EXPLAIN statement has ANALYZE option
func (*queryValidatorListener) isExplainAnalyze(ctx *parser.ExplainstmtContext) bool {
	// Check for ANALYZE keyword in explain options
	if ctx.Explain_option_list() != nil {
		for _, optCtx := range ctx.Explain_option_list().AllExplain_option_elem() {
			if optCtx.Explain_option_name() != nil {
				optName := optCtx.Explain_option_name().GetText()
				if optName == "ANALYZE" || optName == "analyze" {
					return true
				}
			}
		}
	}

	// Also check for old-style EXPLAIN ANALYZE (without parentheses)
	// Look for ANALYZE keyword directly after EXPLAIN
	for _, child := range ctx.GetChildren() {
		if termNode, ok := child.(antlr.TerminalNode); ok {
			if termNode.GetText() == "ANALYZE" || termNode.GetText() == "analyze" {
				return true
			}
		}
	}

	return false
}

// isTopLevelStmt checks if the statement is at the top level (not nested in CTE, subquery, etc.)
func isTopLevelStmt(ctx antlr.ParserRuleContext) bool {
	parent := ctx.GetParent()
	for parent != nil {
		switch parent.(type) {
		case *parser.RootContext:
			return true
		case *parser.StmtblockContext, *parser.StmtmultiContext, *parser.StmtContext:
			// Continue walking up
			parent = parent.GetParent()
		default:
			// If we hit any other context type, it's nested
			return false
		}
	}
	return true
}

// dmlDetectorListener detects INSERT/UPDATE/DELETE statements anywhere in the tree
type dmlDetectorListener struct {
	*parser.BasePostgreSQLParserListener

	hasDML bool
}

// EnterInsertstmt detects INSERT
func (d *dmlDetectorListener) EnterInsertstmt(_ *parser.InsertstmtContext) {
	d.hasDML = true
}

// EnterUpdatestmt detects UPDATE
func (d *dmlDetectorListener) EnterUpdatestmt(_ *parser.UpdatestmtContext) {
	d.hasDML = true
}

// EnterDeletestmt detects DELETE
func (d *dmlDetectorListener) EnterDeletestmt(_ *parser.DeletestmtContext) {
	d.hasDML = true
}
