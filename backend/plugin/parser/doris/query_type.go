package doris

import (
	parser "github.com/bytebase/doris-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type queryTypeListener struct {
	*parser.BaseDorisSQLListener

	result base.QueryType
}

func (l *queryTypeListener) EnterSingleStatement(ctx *parser.SingleStatementContext) {
	if ctx == nil {
		return
	}

	s := ctx.Statement()
	if s == nil {
		return
	}

	switch {
	case s.QueryStatement() != nil:
		l.result = base.Select
	case s.InsertStatement() != nil, s.UpdateStatement() != nil, s.DeleteStatement() != nil:
		l.result = base.DML
	case s.ShowAlterStatement() != nil,
		s.ShowAnalyzeStatement() != nil,
		s.ShowAuthenticationStatement() != nil,
		s.ShowAuthorStatement() != nil,
		s.ShowBackendBlackListStatement() != nil,
		s.ShowCatalogsStatement() != nil,
		s.ShowDatabasesStatement() != nil,
		s.ShowEnginesStatement() != nil,
		s.ShowFunctionsStatement() != nil,
		s.ShowGrantsStatement() != nil,
		s.ShowIndexStatement() != nil,
		s.ShowPartitionsStatement() != nil,
		s.ShowProcesslistStatement() != nil,
		s.ShowRolesStatement() != nil,
		s.ShowTransactionStatement() != nil,
		s.ShowTriggersStatement() != nil,
		s.ShowUserStatement() != nil,
		s.ShowVariablesStatement() != nil,
		s.ShowTableStatement() != nil,
		s.ShowTableStatusStatement() != nil,
		s.ShowCreateDbStatement() != nil,
		s.ShowCreateTableStatement() != nil:
		l.result = base.SelectInfoSchema
	default:
		l.result = base.DDL
	}
}
