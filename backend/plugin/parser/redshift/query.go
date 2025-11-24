package redshift

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/redshift"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	// Override the existing PG-based validator with Redshift-specific one
	base.RegisterQueryValidator(storepb.Engine_REDSHIFT, ValidateSQLForEditor)
}

// ValidateSQLForEditor validates the SQL statement for SQL editor.
// We support the following SQLs:
// 1. EXPLAIN statement, except EXPLAIN ANALYZE (unless it's EXPLAIN ANALYZE SELECT)
// 2. SELECT statement
// We also support CTE with SELECT statements, but not with DML statements.
// Returns (canRunInReadOnly, returnsData, error):
// - canRunInReadOnly: whether all queries can run in read-only mode
// - returnsData: whether all queries return data
// - error: parsing error if the statement is invalid
func ValidateSQLForEditor(statement string) (bool, bool, error) {
	// Parse the statement using the Redshift parser
	parseResults, err := ParseRedshift(statement)
	if err != nil {
		return false, false, err
	}

	// Create a validator listener to walk the parse tree
	validator := &queryValidatorListener{
		canRunInReadOnly: true,
		returnsData:      true,
	}

	// Validate each statement
	for _, parseResult := range parseResults {
		antlr.ParseTreeWalkerDefault.Walk(validator, parseResult.Tree)
		// If any statement fails validation, return immediately
		if !validator.canRunInReadOnly {
			break
		}
	}

	return validator.canRunInReadOnly, validator.returnsData, nil
}

type queryValidatorListener struct {
	*parser.BaseRedshiftParserListener

	canRunInReadOnly bool
	returnsData      bool
}

// EnterStmt is called when entering a statement
func (l *queryValidatorListener) EnterStmt(ctx *parser.StmtContext) {
	if ctx == nil {
		return
	}

	// Check each statement type
	switch {
	case ctx.Selectstmt() != nil:
		// SELECT is allowed
		return

	case ctx.Explainstmt() != nil:
		// EXPLAIN is allowed, but check if it's EXPLAIN ANALYZE
		if explainCtx, ok := ctx.Explainstmt().(*parser.ExplainstmtContext); ok {
			l.handleExplainStmt(explainCtx)
		}
		return

	// SHOW statements - different types of SHOW commands
	case ctx.Variableshowstmt() != nil,
		ctx.Showcolumnsstmt() != nil,
		ctx.Showdatabasesstmt() != nil,
		ctx.Showdatasharesstmt() != nil,
		ctx.Showexternaltablestmt() != nil,
		ctx.Showgrantsstmt() != nil,
		ctx.Showmodelstmt() != nil,
		ctx.Showprocedurestmt() != nil,
		ctx.Showschemasstmt() != nil,
		ctx.Showtablestmt() != nil,
		ctx.Showtablesstmt() != nil,
		ctx.Showviewstmt() != nil:
		// All SHOW statements are allowed and return data
		return

	case ctx.Variablesetstmt() != nil:
		// SET statements can run in read-only but don't return data
		l.returnsData = false
		return

	// DML statements are not allowed in read-only mode
	case ctx.Insertstmt() != nil,
		ctx.Updatestmt() != nil,
		ctx.Deletestmt() != nil,
		ctx.Truncatestmt() != nil:
		l.canRunInReadOnly = false
		l.returnsData = false
		return

	// DDL statements are not allowed in read-only mode
	case ctx.Createstmt() != nil,
		ctx.Dropstmt() != nil,
		ctx.Altertablestmt() != nil,
		ctx.Indexstmt() != nil,
		ctx.Createdbstmt() != nil,
		ctx.Dropdbstmt() != nil,
		ctx.Createfunctionstmt() != nil,
		ctx.Createexternalfunctionstmt() != nil,
		ctx.Createprocedurestmt() != nil,
		ctx.Creatematviewstmt() != nil,
		ctx.Createexternalviewstmt() != nil,
		ctx.Createschemastmt() != nil,
		ctx.Createuserstmt() != nil,
		ctx.Createrolestmt() != nil,
		ctx.Alteruserstmt() != nil,
		ctx.Alterrolestmt() != nil,
		ctx.Dropuserstmt() != nil,
		ctx.Droprolestmt() != nil:
		l.canRunInReadOnly = false
		l.returnsData = false
		return

	// Administrative statements are not allowed in read-only mode
	case ctx.Grantstmt() != nil,
		ctx.Revokestmt() != nil,
		ctx.Analyzestmt() != nil,
		ctx.Vacuumstmt() != nil,
		ctx.Copystmt() != nil,
		ctx.Commentstmt() != nil,
		ctx.Callstmt() != nil:
		l.canRunInReadOnly = false
		l.returnsData = false
		return

	// Transaction statements are not allowed in read-only mode
	case ctx.Transactionstmt() != nil:
		l.canRunInReadOnly = false
		l.returnsData = false
		return

	default:
		// For any unrecognized statement, be conservative and reject it
		l.canRunInReadOnly = false
		l.returnsData = false
	}
}

// handleExplainStmt handles EXPLAIN statements
func (l *queryValidatorListener) handleExplainStmt(ctx *parser.ExplainstmtContext) {
	if ctx == nil {
		return
	}

	// Check if it's EXPLAIN ANALYZE by looking for the Analyze_keyword
	if ctx.Analyze_keyword() != nil {
		// EXPLAIN ANALYZE actually executes the query
		// Check what statement is being explained using AST
		if explainableStmt := ctx.Explainablestmt(); explainableStmt != nil {
			// Use type assertion to check the actual explainable statement
			if explainableCtx, ok := explainableStmt.(*parser.ExplainablestmtContext); ok {
				// Check if it's a SELECT statement using the AST
				if explainableCtx.Selectstmt() != nil {
					// EXPLAIN ANALYZE SELECT is allowed in read-only but doesn't return data
					l.returnsData = false
				} else {
					// EXPLAIN ANALYZE of non-SELECT (INSERT/UPDATE/DELETE/DECLARE CURSOR) is not allowed
					l.canRunInReadOnly = false
					l.returnsData = false
				}
			}
		}
	}
	// Regular EXPLAIN (without ANALYZE) is always allowed and returns data
}

// EnterCommon_table_expr is called when entering a CTE
func (l *queryValidatorListener) EnterCommon_table_expr(ctx *parser.Common_table_exprContext) {
	if ctx != nil {
		// Check if the CTE contains DML statements
		if l.hasDMLInContext(ctx) {
			l.canRunInReadOnly = false
			l.returnsData = false
		}
	}
}

// hasDMLInContext checks if the given context contains DML statements
func (*queryValidatorListener) hasDMLInContext(ctx antlr.ParseTree) bool {
	if ctx == nil {
		return false
	}

	// Create a DML detector listener
	dmlDetector := &dmlDetectorListener{
		hasDML: false,
	}

	antlr.ParseTreeWalkerDefault.Walk(dmlDetector, ctx)
	return dmlDetector.hasDML
}

type dmlDetectorListener struct {
	*parser.BaseRedshiftParserListener
	hasDML bool
}

// Check for DELETE statement
func (l *dmlDetectorListener) EnterDeletestmt(_ *parser.DeletestmtContext) {
	l.hasDML = true
}

// Check for INSERT statement
func (l *dmlDetectorListener) EnterInsertstmt(_ *parser.InsertstmtContext) {
	l.hasDML = true
}

// Check for UPDATE statement
func (l *dmlDetectorListener) EnterUpdatestmt(_ *parser.UpdatestmtContext) {
	l.hasDML = true
}
