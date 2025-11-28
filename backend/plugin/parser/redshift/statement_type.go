package redshift

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/redshift"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// GetStatementTypes returns the statement types from the given AST.
func GetStatementTypes(asts []base.AST) ([]string, error) {
	listener := &statementTypeListener{
		types: make(map[string]bool),
	}

	for _, ast := range asts {
		antlrAST, ok := base.GetANTLRAST(ast)
		if !ok {
			return nil, errors.New("expected ANTLR AST for Redshift")
		}
		antlr.ParseTreeWalkerDefault.Walk(listener, antlrAST.Tree)
	}

	var sqlTypes []string
	for sqlType := range listener.types {
		sqlTypes = append(sqlTypes, sqlType)
	}
	return sqlTypes, nil
}

type statementTypeListener struct {
	*parser.BaseRedshiftParserListener
	types map[string]bool
}

func (l *statementTypeListener) EnterStmt(ctx *parser.StmtContext) {
	if ctx == nil {
		return
	}

	// Determine statement type based on context
	switch {
	case ctx.Selectstmt() != nil:
		l.types["SELECT"] = true
	case ctx.Insertstmt() != nil:
		l.types["INSERT"] = true
	case ctx.Updatestmt() != nil:
		l.types["UPDATE"] = true
	case ctx.Deletestmt() != nil:
		l.types["DELETE"] = true
	case ctx.Createstmt() != nil:
		l.types["CREATE_TABLE"] = true
	case ctx.Dropstmt() != nil:
		l.types["DROP"] = true
	case ctx.Altertablestmt() != nil:
		l.types["ALTER"] = true
	case ctx.Indexstmt() != nil:
		l.types["CREATE_INDEX"] = true
	case ctx.Createdbstmt() != nil:
		l.types["CREATE_DATABASE"] = true
	case ctx.Dropdbstmt() != nil:
		l.types["DROP_DATABASE"] = true
	case ctx.Grantstmt() != nil:
		l.types["GRANT"] = true
	case ctx.Revokestmt() != nil:
		l.types["REVOKE"] = true
	case ctx.Analyzestmt() != nil:
		l.types["ANALYZE"] = true
	case ctx.Vacuumstmt() != nil:
		l.types["VACUUM"] = true
	case ctx.Copystmt() != nil:
		l.types["COPY"] = true
	case ctx.Explainstmt() != nil:
		l.types["EXPLAIN"] = true
	case ctx.Createexternalfunctionstmt() != nil || ctx.Createfunctionstmt() != nil:
		l.types["CREATE_FUNCTION"] = true
	case ctx.Createprocedurestmt() != nil:
		l.types["CREATE_PROCEDURE"] = true
	case ctx.Creatematviewstmt() != nil:
		l.types["CREATE_VIEW"] = true
	case ctx.Createexternalviewstmt() != nil:
		l.types["CREATE_VIEW"] = true
	case ctx.Createschemastmt() != nil:
		l.types["CREATE_SCHEMA"] = true
	case ctx.Createuserstmt() != nil || ctx.Createrolestmt() != nil:
		l.types["CREATE_USER"] = true
	case ctx.Alteruserstmt() != nil || ctx.Alterrolestmt() != nil:
		l.types["ALTER_USER"] = true
	case ctx.Dropuserstmt() != nil || ctx.Droprolestmt() != nil:
		l.types["DROP_USER"] = true
	case ctx.Truncatestmt() != nil:
		l.types["TRUNCATE"] = true
	case ctx.Commentstmt() != nil:
		l.types["COMMENT"] = true
	case ctx.Variablesetstmt() != nil:
		l.types["SET"] = true
	case ctx.Variableshowstmt() != nil:
		l.types["SHOW"] = true
	case ctx.Callstmt() != nil:
		l.types["CALL"] = true
	case ctx.Transactionstmt() != nil:
		// Check transaction type
		if tStmt := ctx.Transactionstmt(); tStmt != nil {
			text := strings.ToUpper(tStmt.GetText())
			if strings.Contains(text, "BEGIN") || strings.Contains(text, "START") {
				l.types["BEGIN"] = true
			} else if strings.Contains(text, "COMMIT") {
				l.types["COMMIT"] = true
			} else if strings.Contains(text, "ROLLBACK") {
				l.types["ROLLBACK"] = true
			}
		}
	default:
		// For any unrecognized statement type
		l.types["UNKNOWN"] = true
	}
}
