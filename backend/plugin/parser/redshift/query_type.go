package redshift

import (
	"strings"

	"github.com/pkg/errors"

	parser "github.com/bytebase/parser/redshift"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type queryTypeListener struct {
	*parser.BaseRedshiftParserListener
	result           base.QueryType
	allSystems       bool
	isExplainAnalyze bool
	err              error
}

func (q *queryTypeListener) EnterStmtblock(ctx *parser.StmtblockContext) {
	multi := ctx.Stmtmulti()
	if multi == nil || len(multi.AllStmt()) == 0 {
		q.result = base.QueryTypeUnknown
		return
	}
	if len(multi.AllStmt()) > 1 {
		q.err = errors.Errorf("only single statement is allowed, but got %d", len(multi.AllStmt()))
		return
	}

	stmt := multi.AllStmt()[0]
	q.result = q.getStatementType(stmt)
}

func (q *queryTypeListener) getStatementType(stmt parser.IStmtContext) base.QueryType {
	if stmt == nil {
		return base.QueryTypeUnknown
	}

	// DML statements
	if stmt.Insertstmt() != nil {
		return base.DML
	}
	if stmt.Updatestmt() != nil {
		return base.DML
	}
	if stmt.Deletestmt() != nil {
		return base.DML
	}
	if stmt.Mergestmt() != nil {
		return base.DML
	}
	if stmt.Copystmt() != nil {
		return base.DML
	}

	// Special case for SELECT with INTO clause (becomes DDL)
	if selectStmt := stmt.Selectstmt(); selectStmt != nil {
		// Check if it's a SELECT INTO statement
		if q.hasIntoClause(selectStmt) {
			return base.DDL
		}
		if q.allSystems {
			return base.SelectInfoSchema
		}
		return base.Select
	}

	// Variable SET/SHOW statements
	if stmt.Variablesetstmt() != nil {
		// Treat SET as select (safe operation)
		return base.Select
	}
	if stmt.Variableshowstmt() != nil {
		return base.SelectInfoSchema
	}
	// Note: SHOW is handled via Variableshowstmt

	// EXPLAIN statement
	if explainStmt := stmt.Explainstmt(); explainStmt != nil {
		// Check if it has ANALYZE option
		if q.hasAnalyzeOption(explainStmt) {
			q.isExplainAnalyze = true
			// For EXPLAIN ANALYZE, return the type of the explained statement
			if explainedStmt := explainStmt.Explainablestmt(); explainedStmt != nil {
				// Check what kind of statement is being explained
				if explainedStmt.Selectstmt() != nil {
					// Check for SELECT INTO which is DDL
					if q.hasIntoClause(explainedStmt.Selectstmt()) {
						return base.DDL
					}
					return base.Select
				}
				if explainedStmt.Insertstmt() != nil {
					return base.DML
				}
				if explainedStmt.Updatestmt() != nil {
					return base.DML
				}
				if explainedStmt.Deletestmt() != nil {
					return base.DML
				}
				if explainedStmt.Declarecursorstmt() != nil {
					return base.Select
				}
				if explainedStmt.Createasstmt() != nil {
					return base.DDL
				}
				if explainedStmt.Refreshmatviewstmt() != nil {
					return base.DML
				}
				if explainedStmt.Executestmt() != nil {
					// EXECUTE statements are harder to classify without context
					return base.Select
				}
			}
			// Default to Select if we can't determine the explained statement type
			return base.Select
		}
		return base.Explain
	}

	// DDL statements - Direct CREATE/DROP statements
	if stmt.Createstmt() != nil || stmt.Dropstmt() != nil {
		return base.DDL
	}

	// CREATE statements
	if stmt.Createasstmt() != nil || stmt.Createseqstmt() != nil ||
		stmt.Createschemastmt() != nil || stmt.Createdbstmt() != nil ||
		stmt.Createfunctionstmt() != nil || stmt.Createrolestmt() != nil ||
		stmt.Indexstmt() != nil || stmt.Createextensionstmt() != nil ||
		stmt.Createtrigstmt() != nil || stmt.Createeventtrigstmt() != nil ||
		stmt.Createdomainstmt() != nil || stmt.Createconversionstmt() != nil ||
		stmt.Createcaststmt() != nil || stmt.Createopclassstmt() != nil ||
		stmt.Createopfamilystmt() != nil || stmt.Createpolicystmt() != nil ||
		stmt.Createamstmt() != nil || stmt.Createtransformstmt() != nil ||
		stmt.Createstatsstmt() != nil || stmt.Createtablespacestmt() != nil ||
		stmt.Createfdwstmt() != nil || stmt.Createforeignserverstmt() != nil ||
		stmt.Createforeigntablestmt() != nil || stmt.Createplangstmt() != nil ||
		stmt.Createpublicationstmt() != nil || stmt.Createsubscriptionstmt() != nil ||
		stmt.Createusermappingstmt() != nil {
		return base.DDL
	}

	// VIEW statements (DDL)
	if stmt.Viewstmt() != nil {
		return base.DDL
	}

	// ALTER statements
	if stmt.Altertablestmt() != nil || stmt.Alterseqstmt() != nil ||
		stmt.Alterdatabasestmt() != nil ||
		stmt.Alterdatabasesetstmt() != nil ||
		stmt.Alterfunctionstmt() != nil || stmt.Alterrolestmt() != nil ||
		stmt.Alterrolesetstmt() != nil || stmt.Altercollationstmt() != nil ||
		stmt.Alterdomainstmt() != nil || stmt.Alterextensionstmt() != nil ||
		stmt.Alterextensioncontentsstmt() != nil || stmt.Alterfdwstmt() != nil ||
		stmt.Alterforeignserverstmt() != nil || stmt.Alteropfamilystmt() != nil ||
		stmt.Alterpolicystmt() != nil || stmt.Altereventtrigstmt() != nil ||
		stmt.Alterobjectdependsstmt() != nil || stmt.Alterobjectschemastmt() != nil ||
		stmt.Alterownerstmt() != nil || stmt.Alteroperatorstmt() != nil ||
		stmt.Altertypestmt() != nil || stmt.Alterenumstmt() != nil ||
		stmt.Alterstatsstmt() != nil || stmt.Altertblspcstmt() != nil ||
		stmt.Altersystemstmt() != nil ||
		stmt.Alterpublicationstmt() != nil || stmt.Altersubscriptionstmt() != nil ||
		stmt.Alterusermappingstmt() != nil || stmt.Altercompositetypestmt() != nil ||
		stmt.Alterdefaultprivilegesstmt() != nil || stmt.Altergroupstmt() != nil ||
		stmt.Altertsconfigurationstmt() != nil || stmt.Altertsdictionarystmt() != nil {
		return base.DDL
	}

	// DROP statements (in addition to generic Dropstmt)
	if stmt.Dropdbstmt() != nil || stmt.Droprolestmt() != nil ||
		stmt.Dropownedstmt() != nil || stmt.Dropsubscriptionstmt() != nil ||
		stmt.Dropusermappingstmt() != nil {
		return base.DDL
	}

	// Other DDL statements
	if stmt.Truncatestmt() != nil || stmt.Commentstmt() != nil ||
		stmt.Grantstmt() != nil || stmt.Revokestmt() != nil ||
		stmt.Clusterstmt() != nil || stmt.Vacuumstmt() != nil ||
		stmt.Analyzestmt() != nil || stmt.Lockstmt() != nil ||
		stmt.Reindexstmt() != nil || stmt.Rulestmt() != nil ||
		stmt.Renamestmt() != nil || stmt.Reassignownedstmt() != nil ||
		stmt.Loadstmt() != nil ||
		stmt.Importforeignschemastmt() != nil || stmt.Seclabelstmt() != nil ||
		stmt.Dostmt() != nil ||
		stmt.Creatematviewstmt() != nil || stmt.Discardstmt() != nil ||
		stmt.Fetchstmt() != nil || stmt.Constraintssetstmt() != nil ||
		stmt.Checkpointstmt() != nil || stmt.Createtablespacestmt() != nil {
		return base.DDL
	}

	// REFRESH MATERIALIZED VIEW is DML
	if stmt.Refreshmatviewstmt() != nil {
		return base.DML
	}

	// Default to unknown for any unhandled statement types
	return base.QueryTypeUnknown
}

func (q *queryTypeListener) hasIntoClause(selectStmt parser.ISelectstmtContext) bool {
	// Check if SELECT has INTO clause by traversing the AST
	if selectStmt == nil {
		return false
	}

	// Check select_no_parens path
	if selectNoParens := selectStmt.Select_no_parens(); selectNoParens != nil {
		if selectClause := selectNoParens.Select_clause(); selectClause != nil {
			// Traverse through select_clause -> simple_select_intersect -> simple_select_pramary
			for _, intersect := range selectClause.AllSimple_select_intersect() {
				if intersect != nil {
					for _, primary := range intersect.AllSimple_select_pramary() {
						if primary != nil {
							// Check if this simple_select_pramary has INTO clause
							if len(primary.AllInto_clause()) > 0 {
								return true
							}
						}
					}
				}
			}
		}
	}

	// Check select_with_parens path
	if selectWithParens := selectStmt.Select_with_parens(); selectWithParens != nil {
		// Recursively check nested select statements
		if selectWithParens.Select_with_parens() != nil {
			// For nested parentheses, we need to check recursively
			// This is a simplified approach - a full implementation would need
			// to handle all nesting cases
			if selectWithParens.Select_no_parens() != nil {
				return q.hasIntoClauseInSelectNoParens(selectWithParens.Select_no_parens())
			}
		}
	}

	return false
}

func (*queryTypeListener) hasIntoClauseInSelectNoParens(selectNoParens parser.ISelect_no_parensContext) bool {
	if selectNoParens == nil {
		return false
	}

	if selectClause := selectNoParens.Select_clause(); selectClause != nil {
		// Traverse through select_clause -> simple_select_intersect -> simple_select_pramary
		for _, intersect := range selectClause.AllSimple_select_intersect() {
			if intersect != nil {
				for _, primary := range intersect.AllSimple_select_pramary() {
					if primary != nil {
						// Check if this simple_select_pramary has INTO clause
						if len(primary.AllInto_clause()) > 0 {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

func (*queryTypeListener) hasAnalyzeOption(explainStmt parser.IExplainstmtContext) bool {
	if explainStmt == nil {
		return false
	}

	// Check if EXPLAIN has ANALYZE keyword via AST
	// Method 1: Check if Analyze_keyword() exists
	if explainStmt.Analyze_keyword() != nil {
		return true
	}

	// Method 2: Check in the explain option list
	if optionList := explainStmt.Explain_option_list(); optionList != nil {
		// Check each option element
		for _, optionElem := range optionList.AllExplain_option_elem() {
			if optionElem != nil {
				// Check if the option name is "analyze"
				if optionName := optionElem.Explain_option_name(); optionName != nil {
					// Check if it's the ANALYZE keyword
					if optionName.Analyze_keyword() != nil {
						return true
					}
					// Also check if it's specified as a nonreserved word
					// Note: We need to check the text here because ANALYZE can be specified
					// as a non-reserved word in PostgreSQL, and we need to verify it's actually
					// the ANALYZE option and not some other non-reserved word
					if optionName.Nonreservedword() != nil {
						optionText := optionName.Nonreservedword().GetText()
						if strings.EqualFold(optionText, "analyze") {
							return true
						}
					}
				}
			}
		}
	}

	return false
}
