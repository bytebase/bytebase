package spanner

import (
	"fmt"

	parser "github.com/bytebase/google-sql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type queryTypeListener struct {
	*parser.BaseGoogleSQLParserListener

	allSystems bool
	result     base.QueryType
	err        error
}

func (l *queryTypeListener) EnterStmts(ctx *parser.StmtsContext) {
	// Assume that the stmts contains only one unterminated statement.
	if len(ctx.AllUnterminated_sql_statement()) != 1 {
		l.err = fmt.Errorf("expecting 1 unterminated sql statement, but got %d", len(ctx.AllUnterminated_sql_statement()))
		return
	}
	unterminatedStatement := ctx.AllUnterminated_sql_statement()[0]
	l.result, l.err = l.getQueryTypeForUnterminatedSqlStatement(unterminatedStatement)
}

func (l *queryTypeListener) getQueryTypeForUnterminatedSqlStatement(u parser.IUnterminated_sql_statementContext) (base.QueryType, error) {
	body := u.Sql_statement_body()
	switch {
	case body.Query_statement() != nil:
		if l.allSystems {
			return base.SelectInfoSchema, nil
		}
		return base.Select, nil
	case body.Alter_statement() != nil, body.Create_constant_statement() != nil, body.Create_connection_statement() != nil, body.Create_database_statement() != nil,
		body.Create_function_statement() != nil, body.Create_procedure_statement() != nil, body.Create_index_statement() != nil, body.Create_privilege_restriction_statement() != nil, body.Create_row_access_policy_statement() != nil,
		body.Create_external_table_statement() != nil, body.Create_external_table_function_statement() != nil, body.Create_model_statement() != nil, body.Create_property_graph_statement() != nil,
		body.Create_schema_statement() != nil, body.Create_external_schema_statement() != nil, body.Create_snapshot_statement() != nil, body.Create_table_function_statement() != nil, body.Create_table_statement() != nil,
		body.Create_view_statement() != nil, body.Create_entity_statement() != nil, body.Rename_statement() != nil, body.Drop_all_row_access_policies_statement() != nil, body.Drop_statement() != nil, body.Undrop_statement() != nil:
		return base.DDL, nil
	case body.Dml_statement() != nil, body.Merge_statement() != nil:
		return base.DML, nil
	case body.Explain_statement() != nil:
		return base.Explain, nil
	// Treat SAFE SET as select statement.
	case body.Set_statement() != nil:
		return base.Select, nil
	default:
		return base.QueryTypeUnknown, nil
	}
}
