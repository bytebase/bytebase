package redshift

import "strings"

var (
	// systemSchemas is the list of system schemas that we will exclude.
	systemSchemas = map[string]bool{
		"information_schema": true,
		"pg_catalog":         true,
		"pg_internal":        true,
		"pg_toast":           true,
	}

	// systemViews is the list of system views.
	systemViews = map[string]bool{
		"pg_available_extensions":         true,
		"pg_available_extension_versions": true,
		"pg_backend_memory_contexts":      true,
		"pg_config":                       true,
		"pg_cursors":                      true,
		"pg_file_settings":                true,
		"pg_group":                        true,
		"pg_hba_file_rules":               true,
		"pg_ident_file_mappings":          true,
		"pg_indexes":                      true,
		"pg_locks":                        true,
		"pg_matviews":                     true,
		"pg_policies":                     true,
		"pg_prepared_statements":          true,
		"pg_prepared_xacts":               true,
		"pg_publication_tables":           true,
		"pg_replication_origin_status":    true,
		"pg_replication_slots":            true,
		"pg_roles":                        true,
		"pg_rules":                        true,
		"pg_seclabels":                    true,
		"pg_sequences":                    true,
		"pg_settings":                     true,
		"pg_shadow":                       true,
		"pg_shmem_allocations":            true,
		"pg_stats":                        true,
		"pg_stats_ext":                    true,
		"pg_stats_ext_exprs":              true,
		"pg_tables":                       true,
		"pg_timezone_abbrevs":             true,
		"pg_timezone_names":               true,
		"pg_user":                         true,
		"pg_user_mappings":                true,
		"pg_views":                        true,
		"pg_stat_statements":              true,
		// Redshift-specific system views
		"stl_query":                   true,
		"stl_querytext":               true,
		"stl_wlm_query":               true,
		"stl_ddltext":                 true,
		"stl_utilitytext":             true,
		"stl_load_commits":            true,
		"stl_load_errors":             true,
		"svv_tables":                  true,
		"svv_table_info":              true,
		"svv_columns":                 true,
		"svv_external_tables":         true,
		"svv_external_columns":        true,
		"svv_external_schemas":        true,
		"svv_redshift_tables":         true,
		"svv_redshift_columns":        true,
		"svv_all_tables":              true,
		"svv_all_columns":             true,
		"svl_statementtext":           true,
		"svl_user_info":               true,
		"svl_stored_proc_call":        true,
		"stv_recents":                 true,
		"stv_inflight":                true,
		"stv_sessions":                true,
		"stv_wlm_query_state":         true,
		"stv_wlm_service_class_state": true,
	}

	// systemTables is the list of system tables.
	systemTables = map[string]bool{
		"pg_aggregate":             true,
		"pg_am":                    true,
		"pg_amop":                  true,
		"pg_amproc":                true,
		"pg_attrdef":               true,
		"pg_attribute":             true,
		"pg_authid":                true,
		"pg_auth_members":          true,
		"pg_cast":                  true,
		"pg_class":                 true,
		"pg_collation":             true,
		"pg_constraint":            true,
		"pg_conversion":            true,
		"pg_database":              true,
		"pg_db_role_setting":       true,
		"pg_default_acl":           true,
		"pg_depend":                true,
		"pg_description":           true,
		"pg_enum":                  true,
		"pg_event_trigger":         true,
		"pg_extension":             true,
		"pg_foreign_data_wrapper":  true,
		"pg_foreign_server":        true,
		"pg_foreign_table":         true,
		"pg_index":                 true,
		"pg_inherits":              true,
		"pg_init_privs":            true,
		"pg_language":              true,
		"pg_largeobject":           true,
		"pg_largeobject_metadata":  true,
		"pg_namespace":             true,
		"pg_opclass":               true,
		"pg_operator":              true,
		"pg_opfamily":              true,
		"pg_parameter_acl":         true,
		"pg_partitioned_table":     true,
		"pg_policy":                true,
		"pg_proc":                  true,
		"pg_publication":           true,
		"pg_publication_namespace": true,
		"pg_publication_rel":       true,
		"pg_range":                 true,
		"pg_replication_origin":    true,
		"pg_rewrite":               true,
		"pg_seclabel":              true,
		"pg_sequence":              true,
		"pg_shdepend":              true,
		"pg_shdescription":         true,
		"pg_shseclabel":            true,
		"pg_statistic":             true,
		"pg_statistic_ext":         true,
		"pg_statistic_ext_data":    true,
		"pg_subscription":          true,
		"pg_subscription_rel":      true,
		"pg_tablespace":            true,
		"pg_transform":             true,
		"pg_trigger":               true,
		"pg_ts_config":             true,
		"pg_ts_config_map":         true,
		"pg_ts_dict":               true,
		"pg_ts_parser":             true,
		"pg_ts_template":           true,
		"pg_type":                  true,
		"pg_user_mapping":          true,
	}
)

func isSystemSchema(schema string) bool {
	_, ok := systemSchemas[schema]
	if ok {
		return true
	}
	if strings.HasPrefix(schema, "pg_temp") {
		return true
	}
	if strings.HasPrefix(schema, "pg_toast") {
		return true
	}
	return false
}

func isSystemTable(table string) bool {
	_, ok := systemTables[table]
	return ok
}

func isSystemView(view string) bool {
	_, ok := systemViews[view]
	return ok
}
