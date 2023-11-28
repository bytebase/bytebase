package pg

import (
	"fmt"
	"strings"
)

var (
	// systemDatabases is the list of system or internal databases.
	systemDatabases = map[string]bool{
		// Skip our internal "bytebase" database
		"bytebase": true,
		// Skip internal databases from cloud service providers
		// see https://github.com/bytebase/bytebase/issues/30
		// aws
		"rdsadmin": true,
		// gcp
		"cloudsql":      true,
		"cloudsqladmin": true,
		"alloydbadmin":  true,
		// system templates.
		"template0": true,
		"template1": true,
	}

	// systemViews is the list of system views that we will exclude from the schema sync.
	// https://www.postgresql.org/docs/16/views.html
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
		"pg_matview":                      true,
		"pg_policies":                     true,
		"pg_prepared_statement":           true,
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
	}

	// systemSchemas is the list of system schemas that we will exclude from the schema sync.
	systemSchemas = map[string]bool{
		"information_schema":       true,
		"pg_catalog":               true,
		"pg_toast":                 true,
		"rw_catalog":               true,
		"timescaledb_information":  true,
		"timescaledb_experimental": true,
		"_timescaledb_cache":       true,
		"_timescaledb_catalog":     true,
		"_timescaledb_internal":    true,
		"_timescaledb_config":      true,
	}

	// systemTables is the list of system tables that we will exclude from the schema sync.
	// https://www.postgresql.org/docs/16/catalogs.html
	systemTables = map[string]bool{
		"pg_aggregate":               true,
		"pg_am":                      true,
		"pg_amop":                    true,
		"pg_amproc":                  true,
		"pg_attrdef":                 true,
		"pg_attribute":               true,
		"pg_authid":                  true,
		"pg_auth_members":            true,
		"pg_cast":                    true,
		"pg_class":                   true,
		"pg_collation":               true,
		"pg_constraint":              true,
		"pg_conversion":              true,
		"pg_database":                true,
		"pg_db_role_setting":         true,
		"pg_default_acl":             true,
		"pg_depend":                  true,
		"pg_description":             true,
		"pg_enum":                    true,
		"pg_event_trigger":           true,
		"pg_extension":               true,
		"pg_foreign_data_wrapper":    true,
		"pg_foreign_server":          true,
		"pg_foreign_table":           true,
		"pg_index":                   true,
		"pg_inherits":                true,
		"pg_init_privs":              true,
		"pg_language":                true,
		"pg_largeobject":             true,
		"pg_largeobject_metadata":    true,
		"pg_namespace":               true,
		"pg_opclass":                 true,
		"pg_operator":                true,
		"pg_opfamily":                true,
		"pg_parameter_acl":           true,
		"pg_partitioned_table":       true,
		"pg_policy":                  true,
		"pg_proc":                    true,
		"pg_publication":             true,
		"pg_publication_namespace":   true,
		"pg_publication_rel":         true,
		"pg_range":                   true,
		"pg_replication_origin":      true,
		"pg_rewrite":                 true,
		"pg_seclabel":                true,
		"pg_sequence":                true,
		"pg_shdepend":                true,
		"pg_shdescription":           true,
		"pg_shseclabel":              true,
		"pg_statistic":               true,
		"pg_statistic_ext":           true,
		"pg_statistic_ext_data":      true,
		"pg_subscription":            true,
		"pg_subscription_rel":        true,
		"pg_tablespace":              true,
		"pg_transform":               true,
		"pg_trigger":                 true,
		"pg_ts_config":               true,
		"pg_ts_config_map":           true,
		"pg_ts_dict":                 true,
		"pg_ts_parser":               true,
		"pg_ts_template":             true,
		"pg_type":                    true,
		"pg_user_mapping":            true,
		"pg_stat_activity":           true,
		"pg_stat_replication":        true,
		"pg_stat_replication_slots":  true,
		"pg_stat_wal_receiver":       true,
		"pg_stat_recovery_prefetch":  true,
		"pg_stat_subscription":       true,
		"pg_stat_subscription_stats": true,
		"pg_stat_ssl":                true,
		"pg_stat_gssapi":             true,
		"pg_stat_archiver":           true,
		"pg_stat_bgwriter":           true,
		"pg_stat_wal":                true,
		"pg_stat_database":           true,
		"pg_stat_database_conflicts": true,
		"pg_stat_all_tables":         true,
		"pg_stat_all_indexes":        true,
		"pg_statio_all_tables":       true,
		"pg_statio_all_indexes":      true,
		"pg_statio_all_sequences":    true,
		"pg_stat_user_functions":     true,
		"pg_stat_slru":               true,
	}

	// SystemSchemaWhereClause is an optimization for getting less schema objects.
	SystemSchemaWhereClause = func() string {
		var schemas []string
		for schema := range systemSchemas {
			schemas = append(schemas, fmt.Sprintf("'%s'", schema))
		}
		return strings.Join(schemas, ",")
	}()
)

func IsSystemUser(user string) bool {
	return strings.HasPrefix(user, "alloydb")
}

func IsSystemDatabase(database string) bool {
	_, ok := systemDatabases[database]
	return ok
}

func IsSystemSchema(schema string) bool {
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

func IsSystemTable(table string) bool {
	_, ok := systemTables[table]
	return ok
}

func IsSystemView(view string) bool {
	if _, ok := systemViews[view]; ok {
		return true
	}
	if strings.HasPrefix(view, "g_columnar_") {
		return true
	}
	if strings.HasPrefix(view, "google_db_advisor_") {
		return true
	}
	if strings.HasPrefix(view, "g_agg_stat_") {
		return true
	}
	if strings.HasPrefix(view, "g_agg_stat_") {
		return true
	}
	if strings.HasPrefix(view, "hypopg") {
		return true
	}
	return false
}

func IsSystemFunctions(function, definition string) bool {
	if strings.HasPrefix(function, "g_columnar_") {
		return true
	}
	if strings.HasPrefix(function, "google_columnar_") {
		return true
	}
	if strings.HasPrefix(function, "google_db_advisor_") {
		return true
	}
	if strings.HasPrefix(function, "g_agg_stat_") {
		return true
	}
	if strings.HasPrefix(function, "hypopg") {
		return true
	}
	if function == "pg_stat_statements_wrapper" {
		return true
	}
	if strings.Contains(definition, "$libdir/timescaledb") {
		return true
	}
	return false
}
