package tsql

import (
	parser "github.com/bytebase/parser/tsql"

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
		return base.Select, nil
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
		return l.getQueryTypeForBatchLevelStatement(batch.Batch_level_statement())
	case batch.Execute_body_batch() != nil:
		return l.getQueryTypeForExecuteBodyBatch(batch.Execute_body_batch())
	default:
		return base.QueryTypeUnknown, nil
	}
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
	case clause.Another_statement() != nil:
		// Treat SAFE SET as select statement.
		if clause.Another_statement().Set_statement() != nil || clause.Another_statement().Setuser_statement() != nil || clause.Another_statement().Declare_statement() != nil {
			return base.Select, nil
		}
		return base.QueryTypeUnknown, nil
	case clause.Cfl_statement() != nil, clause.Dbcc_clause() != nil, clause.Backup_statement() != nil:
		return base.QueryTypeUnknown, nil
	default:
		return base.QueryTypeUnknown, nil
	}
}

func (*queryTypeListener) getQueryTypeForBatchLevelStatement(parser.IBatch_level_statementContext) (base.QueryType, error) {
	return base.DDL, nil
}

func (*queryTypeListener) getQueryTypeForExecuteBodyBatch(parser.IExecute_body_batchContext) (base.QueryType, error) {
	// Call stored procedure or function. Do not detect the type for now.
	return base.QueryTypeUnknown, nil
}
