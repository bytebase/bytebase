package pg

import "strings"

var (
	// excludedDatabaseList is the list of system or internal databases.
	excludedDatabaseList = map[string]bool{
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

	// SystemSchemaList is the list of system schemas that we will exclude from the schema sync.
	SystemSchemaList = map[string]bool{
		"information_schema":       true,
		"pg_catalog":               true,
		"pg_toast":                 true,
		"_timescaledb_cache":       true,
		"_timescaledb_catalog":     true,
		"_timescaledb_internal":    true,
		"_timescaledb_config":      true,
		"timescaledb_information":  true,
		"timescaledb_experimental": true,
	}

	// SystemTableList is the list of system tables that we will exclude from the schema sync.
	SystemTableList = []string{
		"pg_aggregate",
		"pg_am",
		"pg_amop",
		"pg_amproc",
		"pg_attrdef",
		"pg_attribute",
		"pg_authid",
		"pg_auth_members",
		"pg_cast",
		"pg_class",
		"pg_collation",
		"pg_constraint",
		"pg_conversion",
		"pg_database",
		"pg_db_role_setting",
		"pg_default_acl",
		"pg_depend",
		"pg_description",
		"pg_enum",
		"pg_event_trigger",
		"pg_extension",
		"pg_foreign_data_wrapper",
		"pg_foreign_server",
		"pg_foreign_table",
		"pg_index",
		"pg_inherits",
		"pg_init_privs",
		"pg_language",
		"pg_largeobject",
		"pg_largeobject_metadata",
		"pg_namespace",
		"pg_opclass",
		"pg_operator",
		"pg_opfamily",
		"pg_parameter_acl",
		"pg_partitioned_table",
		"pg_policy",
		"pg_proc",
		"pg_publication",
		"pg_publication_namespace",
		"pg_publication_rel",
		"pg_range",
		"pg_replication_origin",
		"pg_rewrite",
		"pg_seclabel",
		"pg_sequence",
		"pg_shdepend",
		"pg_shdescription",
		"pg_shseclabel",
		"pg_statistic",
		"pg_statistic_ext",
		"pg_statistic_ext_data",
		"pg_subscription",
		"pg_subscription_rel",
		"pg_tablespace",
		"pg_transform",
		"pg_trigger",
		"pg_ts_config",
		"pg_ts_config_map",
		"pg_ts_dict",
		"pg_ts_parser",
		"pg_ts_template",
		"pg_type",
		"pg_user_mapping",
		"pg_stat_activity",
		"pg_stat_replication",
		"pg_stat_replication_slots",
		"pg_stat_wal_receiver",
		"pg_stat_recovery_prefetch",
		"pg_stat_subscription",
		"pg_stat_subscription_stats",
		"pg_stat_ssl",
		"pg_stat_gssapi",
		"pg_stat_archiver",
		"pg_stat_bgwriter",
		"pg_stat_wal",
		"pg_stat_database",
		"pg_stat_database_conflicts",
		"pg_stat_all_tables",
		"pg_stat_all_indexes",
		"pg_statio_all_tables",
		"pg_statio_all_indexes",
		"pg_statio_all_sequences",
		"pg_stat_user_functions",
		"pg_stat_slru",
	}
)

const systemSchemas = "'information_schema', 'pg_catalog', 'pg_toast', '_timescaledb_cache', '_timescaledb_catalog', '_timescaledb_internal', '_timescaledb_config', 'timescaledb_information', 'timescaledb_experimental'"

func IsSystemDatabase(database string) bool {
	_, ok := excludedDatabaseList[database]
	return ok
}

func IsSystemSchema(schema string) bool {
	_, ok := SystemSchemaList[schema]
	return ok
}

func IsSystemUser(user string) bool {
	return strings.HasPrefix(user, "alloydb")
}

func IsSystemView(view string) bool {
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

func IsSystemFunctions(function string) bool {
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
	return false
}
