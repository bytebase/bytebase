package tsql

import (
	parser "github.com/bytebase/tsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type queryTypeListener struct {
	*parser.BaseTSqlParserListener

	allSystems bool
	result     base.QueryType
	err        error
}

func (l *queryTypeListener) EnterTsql_file(ctx *parser.Tsql_fileContext) {
	if l.err != nil {
		return
	}

	l.result, l.err = l.getQueryTypeForTSqlFile(ctx)
}

func (l *queryTypeListener) getQueryTypeForTSqlFile(file parser.ITsql_fileContext) (base.QueryType, error) {
	if len(file.AllBatch_without_go()) == 0 {
		// Multiple go statement only.
		return base.QueryTypeUnknown, nil
	}

	// TODO(zp): Make sure the splitter had handled the GO statement.
	return l.getQueryTypeForBatchWithoutGo(file.Batch_without_go(0))
}

func (l *queryTypeListener) getQueryTypeForBatchWithoutGo(batch parser.IBatch_without_goContext) (base.QueryType, error) {
	// TODO(zp): Make sure the splitter had handled the SEMICOLON.
	switch {
	case len(batch.AllSql_clauses()) != 0 && batch.Execute_body_batch() == nil:
		return l.getQueryTypeForSQLClause(batch.Sql_clauses(0))
	case batch.Batch_level_statement() != nil:
	case batch.Execute_body_batch() != nil:
	}

	return base.QueryTypeUnknown, nil
}

func (l *queryTypeListener) getQueryTypeForSQLClause(clause parser.ISql_clausesContext) (base.QueryType, error) {
	// We only care about the first clause.
	switch {
	case clause.Dml_clause() != nil:
		if clause.Dml_clause().Select_statement_standalone() != nil {
			if l.allSystems {
				return base.SelectInfoSchema, nil
			}
			return base.Select, nil
		}
		return base.DML, nil
	case clause.Ddl_clause() != nil:
		return base.DDL, nil
	case clause.Cfl_statement() != nil, clause.Another_statement() != nil, clause.Dbcc_clause() != nil, clause.Backup_statement() != nil:
		return base.QueryTypeUnknown, nil
	}
	return base.QueryTypeUnknown, nil
}

func (l *queryTypeListener) getQueryTypeForBatchLevelStatement(parser.IBatch_level_statementContext) (base.QueryType, error) {
	return base.DDL, nil
}

func (l *queryTypeListener) getQueryTypeForExecuteBodyBatch(parser.IExecute_bodyContext) (base.QueryType, error) {
	// Call stored procedure or function. Do not detect the type for now.
	return base.QueryTypeUnknown, nil
}
