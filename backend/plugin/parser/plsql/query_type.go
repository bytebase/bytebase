package plsql

import (
	"github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type queryTypeListener struct {
	*plsql.BasePlSqlParserListener

	allSystems bool
	result     base.QueryType
}

func (l *queryTypeListener) EnterUnit_statement(ctx *plsql.Unit_statementContext) {
	switch {
	case ctx.Alter_analytic_view() != nil,
		ctx.Alter_attribute_dimension() != nil,
		ctx.Alter_audit_policy() != nil,
		ctx.Alter_cluster() != nil,
		ctx.Alter_database() != nil,
		ctx.Alter_database_link() != nil,
		ctx.Alter_dimension() != nil,
		ctx.Alter_diskgroup() != nil,
		ctx.Alter_flashback_archive() != nil,
		ctx.Alter_function() != nil,
		ctx.Alter_hierarchy() != nil,
		ctx.Alter_index() != nil,
		ctx.Alter_inmemory_join_group() != nil,
		ctx.Alter_java() != nil,
		ctx.Alter_library() != nil,
		ctx.Alter_lockdown_profile() != nil,
		ctx.Alter_materialized_view() != nil,
		ctx.Alter_materialized_view_log() != nil,
		ctx.Alter_materialized_zonemap() != nil,
		ctx.Alter_operator() != nil,
		ctx.Alter_outline() != nil,
		ctx.Alter_package() != nil,
		ctx.Alter_pmem_filestore() != nil,
		ctx.Alter_procedure() != nil,
		ctx.Alter_resource_cost() != nil,
		ctx.Alter_role() != nil,
		ctx.Alter_rollback_segment() != nil,
		ctx.Alter_sequence() != nil,
		ctx.Alter_session() != nil,
		ctx.Alter_synonym() != nil,
		ctx.Alter_table() != nil,
		ctx.Alter_tablespace() != nil,
		ctx.Alter_tablespace_set() != nil,
		ctx.Alter_trigger() != nil,
		ctx.Alter_type() != nil,
		ctx.Alter_user() != nil,
		ctx.Alter_view() != nil,
		ctx.Create_analytic_view() != nil,
		ctx.Create_attribute_dimension() != nil,
		ctx.Create_audit_policy() != nil,
		ctx.Create_cluster() != nil,
		ctx.Create_context() != nil,
		ctx.Create_controlfile() != nil,
		ctx.Create_database() != nil,
		ctx.Create_database_link() != nil,
		ctx.Create_dimension() != nil,
		ctx.Create_directory() != nil,
		ctx.Create_diskgroup() != nil,
		ctx.Create_edition() != nil,
		ctx.Create_flashback_archive() != nil,
		ctx.Create_function_body() != nil,
		ctx.Create_hierarchy() != nil,
		ctx.Create_index() != nil,
		ctx.Create_inmemory_join_group() != nil,
		ctx.Create_java() != nil,
		ctx.Create_library() != nil,
		ctx.Create_lockdown_profile() != nil,
		ctx.Create_materialized_view() != nil,
		ctx.Create_materialized_view_log() != nil,
		ctx.Create_materialized_zonemap() != nil,
		ctx.Create_operator() != nil,
		ctx.Create_outline() != nil,
		ctx.Create_package() != nil,
		ctx.Create_package_body() != nil,
		ctx.Create_pmem_filestore() != nil,
		ctx.Create_procedure_body() != nil,
		ctx.Create_profile() != nil,
		ctx.Create_restore_point() != nil,
		ctx.Create_role() != nil,
		ctx.Create_rollback_segment() != nil,
		ctx.Create_sequence() != nil,
		ctx.Create_spfile() != nil,
		ctx.Create_synonym() != nil,
		ctx.Create_table() != nil,
		ctx.Create_tablespace() != nil,
		ctx.Create_tablespace_set() != nil,
		ctx.Create_trigger() != nil,
		ctx.Create_type() != nil,
		ctx.Create_user() != nil,
		ctx.Create_view() != nil,
		ctx.Drop_analytic_view() != nil,
		ctx.Drop_attribute_dimension() != nil,
		ctx.Drop_audit_policy() != nil,
		ctx.Drop_cluster() != nil,
		ctx.Drop_context() != nil,
		ctx.Drop_database() != nil,
		ctx.Drop_database_link() != nil,
		ctx.Drop_directory() != nil,
		ctx.Drop_diskgroup() != nil,
		ctx.Drop_edition() != nil,
		ctx.Drop_flashback_archive() != nil,
		ctx.Drop_function() != nil,
		ctx.Drop_hierarchy() != nil,
		ctx.Drop_index() != nil,
		ctx.Drop_indextype() != nil,
		ctx.Drop_inmemory_join_group() != nil,
		ctx.Drop_java() != nil,
		ctx.Drop_library() != nil,
		ctx.Drop_lockdown_profile() != nil,
		ctx.Drop_materialized_view() != nil,
		ctx.Drop_materialized_zonemap() != nil,
		ctx.Drop_operator() != nil,
		ctx.Drop_outline() != nil,
		ctx.Drop_package() != nil,
		ctx.Drop_pmem_filestore() != nil,
		ctx.Drop_procedure() != nil,
		ctx.Drop_restore_point() != nil,
		ctx.Drop_role() != nil,
		ctx.Drop_rollback_segment() != nil,
		ctx.Drop_sequence() != nil,
		ctx.Drop_synonym() != nil,
		ctx.Drop_table() != nil,
		ctx.Drop_tablespace() != nil,
		ctx.Drop_tablespace_set() != nil,
		ctx.Drop_trigger() != nil,
		ctx.Drop_type() != nil,
		ctx.Drop_user() != nil,
		ctx.Drop_view() != nil:
		l.result = base.DDL
	case ctx.Data_manipulation_language_statements() != nil:
		dml := ctx.Data_manipulation_language_statements()
		if dml.Explain_statement() != nil {
			l.result = base.Explain
			return
		}
		if dml.Select_statement() != nil {
			if l.allSystems {
				l.result = base.SelectInfoSchema
			} else {
				l.result = base.Select
			}
			return
		}
		l.result = base.DML
	default:
		// Other statement types
	}
}
