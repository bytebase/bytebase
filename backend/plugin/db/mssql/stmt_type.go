package mssql

import (
	"github.com/antlr4-go/antlr/v4"
	rawparser "github.com/bytebase/parser/tsql"
	"github.com/pkg/errors"

	parser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

type stmtType = int

const (
	stmtTypeUnknown = 1 << iota
	stmtTypeResultSetGenerating
	stmtTypeRowCountGenerating
)

type stmtTypeListener struct {
	*rawparser.BaseTSqlParserListener
	stmtType stmtType
	err      error
}

func getStmtType(stmt string) (stmtType, error) {
	parseResults, err := parser.ParseTSQL(stmt)
	if err != nil {
		return stmtTypeUnknown, err
	}

	if len(parseResults) != 1 {
		return stmtTypeUnknown, errors.Errorf("expected exactly 1 statement, got %d", len(parseResults))
	}

	l := &stmtTypeListener{}
	antlr.ParseTreeWalkerDefault.Walk(l, parseResults[0].Tree)
	if l.err != nil {
		return stmtTypeUnknown, l.err
	}
	return l.stmtType, nil
}

func (l *stmtTypeListener) EnterBatch_without_go(ctx *rawparser.Batch_without_goContext) {
	switch {
	case len(ctx.AllSql_clauses()) > 0:
		if len(ctx.AllSql_clauses()) > 1 {
			l.err = errors.Errorf("unexpected multiple SQL clauses")
		}
		l.stmtType, l.err = getStmtTypeFromSQLClauses(ctx.AllSql_clauses()[0])
	case ctx.Batch_level_statement() != nil:
		l.stmtType = stmtTypeUnknown
	case ctx.Execute_body_batch() != nil:
		l.err = errors.Errorf("unsupported execute func proc")
	default:
		// For any other unhandled cases, set the statement type to unknown
		l.stmtType = stmtTypeUnknown
	}
}

func getStmtTypeFromSQLClauses(ctx rawparser.ISql_clausesContext) (stmtType, error) {
	switch {
	case ctx.Dml_clause() != nil:
		if v := ctx.Dml_clause().Select_statement_standalone(); v != nil {
			if v.Select_statement().Query_expression().Query_specification() != nil && v.Select_statement().Query_expression().Query_specification().INTO() != nil {
				// SELECT INTO will generate the row count only.
				return stmtTypeRowCountGenerating, nil
			}
			return stmtTypeResultSetGenerating | stmtTypeRowCountGenerating, nil
		}
		return stmtTypeRowCountGenerating, nil
	case ctx.Cfl_statement() != nil:
		return stmtTypeUnknown, errors.Errorf("unsupported control flow statement")
	case ctx.Another_statement() != nil:
		return stmtTypeUnknown, nil
	case ctx.Ddl_clause() != nil:
		return stmtTypeUnknown, nil
	case ctx.Dbcc_clause() != nil:
		return stmtTypeUnknown, nil
	case ctx.Backup_statement() != nil:
		return stmtTypeUnknown, nil
	default:
		// For any unhandled SQL clause types, return unknown
		return stmtTypeUnknown, nil
	}
}
