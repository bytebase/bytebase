package pg

import "slices"

var (
	comparisonFunctions = []string{
		"num_nonnulls",
		"num_nulls",
	}

	mathematicalFunctions = []string{
		"abs",
		"cbrt",
		"ceil",
		"ceiling",
		"degrees",
		"div",
		"erf",
		"erfc",
		"exp",
		"factorial",
		"floor",
		"gcd",
		"lcm",
		"ln",
		"log",
		"log10",
		"min_scale",
		"mod",
		"pi",
		"power",
		"radians",
		"round",
		"scale",
		"sign",
		"sqrt",
		"trim_scale",
		"trunc",
		"width_bucket",
		"random",
		"random_normal",
		"setseed",
		"acos",
		"acosd",
		"asin",
		"asind",
		"atan",
		"atand",
		"atan2",
		"atan2d",
		"cos",
		"cosd",
		"cot",
		"cotd",
		"sin",
		"sind",
		"tan",
		"tand",
		"sinh",
		"cosh",
		"tanh",
		"asinh",
		"acosh",
		"atanh",
	}

	stringFunctions = []string{
		"btrim",
		"bit_length",
		"char_length",
		"character_length",
		"lower",
		"lpad",
		"ltrim",
		"normalize",
		"octet_length",
		"overlay",
		"position",
		"rpad",
		"rtrim",
		"substring",
		"trim",
		"upper",
		"ascii",
		"chr",
		"concat",
		"concat_ws",
		"format",
		"initcap",
		"left",
		"length",
		"md5",
		"parse_ident",
		"pg_client_encoding",
		"quote_ident",
		"quote_literal",
		"quote_nullable",
		"regexp_count",
		"regexp_instr",
		"regexp_like",
		"regexp_match",
		"regexp_matches",
		"regexp_replace",
		"regexp_split_to_array",
		"regexp_split_to_table",
		"regexp_substr",
		"repeat",
		"replace",
		"reverse",
		"right",
		"split_part",
		"starts_with",
		"string_to_array",
		"string_to_table",
		"strpos",
		"substr",
		"to_ascii",
		"to_hex",
		"translate",
		"unistr",
	}

	binaryStringFunctions = []string{
		"bit_length",
		"btrim",
		"ltrim",
		"octet_length",
		"overlay",
		"position",
		"rtrim",
		"substring",
		"trim",
		"bit_count",
		"get_bit",
		"get_byte",
		"length",
		"md5",
		"set_bit",
		"set_byte",
		"sha224",
		"sha256",
		"sha384",
		"sha512",
		"substr",
		"convert",
		"convert_from",
		"convert_to",
		"encode",
		"decode",
	}

	bitStringFunctions = []string{
		"bit_count",
		"bit_length",
		"length",
		"octet_length",
		"overlay",
		"position",
		"substring",
		"get_bit",
		"set_bit",
	}

	dataTypeFormattingFunctions = []string{
		"to_char",
		"to_date",
		"to_number",
		"to_timestamp",
	}

	dateTimeFunctions = []string{
		"age",
		"clock_timestamp",
		"current_time",
		"current_timestamp",
		"date_add",
		"date_bin",
		"date_part",
		"date_subtract",
		"date_trunc",
		"extract",
		"isfinite",
		"justify_days",
		"justify_hours",
		"justify_interval",
		"localtime",
		"localtimestamp",
		"make_date",
		"make_interval",
		"make_time",
		"make_timestamp",
		"make_timestamptz",
		"now",
		"statement_timestamp",
		"timeofday",
		"transaction_timestamp",
		"to_timestamp",
	}

	enumSupportFunctions = []string{
		"enum_first",
		"enum_last",
		"enum_range",
	}

	geometricFunctions = []string{
		"area",
		"center",
		"diagonal",
		"diameter",
		"height",
		"isclosed",
		"isopen",
		"length",
		"npoints",
		"pclose",
		"popen",
		"radius",
		"slope",
		"width",
		"box",
		"bound_box",
		"circle",
		"line",
		"lseg",
		"path",
		"point",
		"polygon",
	}

	networkAddressFunctions = []string{
		"abbrev",
		"broadcast",
		"family",
		"host",
		"hostmask",
		"inet_merge",
		"inet_same_family",
		"masklen",
		"netmask",
		"network",
		"set_masklen",
		"text",
		"trunc",
		"macaddr8_set7bit",
	}

	textSearchFunctions = []string{
		"array_to_tsvector",
		"get_current_ts_config",
		"length",
		"numnode",
		"plainto_tsquery",
		"phraseto_tsquery",
		"websearch_to_tsquery",
		"querytree",
		"setweight",
		"strip",
		"to_tsquery",
		"to_tsvector",
		"json_to_tsvector",
		"jsonb_to_tsvector",
		"ts_delete",
		"ts_filter",
		"ts_headline",
		"ts_rank",
		"ts_rank_cd",
		"ts_rewrite",
		"tsquery_phrase",
		"tsvector_to_array",
		"unnest",
		"ts_debug",
		"ts_lexize",
		"ts_parse",
		"ts_token_type",
		"ts_stat",
	}

	uuidFunctions = []string{
		"gen_random_uuid",
	}

	xmlFunctions = []string{
		"xmlcomment",
		"xmlconcat",
		"xmlelement",
		"xmlforest",
		"xmlpi",
		"xmlroot",
		"xmlagg",
		"XMLEXISTS",
		"xml_is_well_formed",
		"xml_is_well_formed_document",
		"xml_is_well_formed_content",
		"xpath",
		"xpath_exists",
		"table_to_xml",
		"query_to_xml",
		"cursor_to_xml",
		"table_to_xmlschema",
		"query_to_xmlschema",
		"cursor_to_xmlschema",
		"table_to_xml_and_xmlschema",
		"query_to_xml_and_xmlschema",
		"schema_to_xml",
		"schema_to_xmlschema",
		"schema_to_xml_and_xmlschema",
		"database_to_xml",
		"database_to_xmlschema",
		"database_to_xml_and_xmlschema",
	}

	jsonFunctions = []string{
		"to_json",
		"to_jsonb",
		"array_to_json",
		"json_array",
		"row_to_json",
		"json_build_array",
		"jsonb_build_array",
		"json_build_object",
		"jsonb_build_object",
		"json_object",
		"jsonb_object",
		"json_array_elements",
		"jsonb_array_elements",
		"json_array_elements_text",
		"jsonb_array_elements_text",
		"json_array_length",
		"jsonb_array_length",
		"json_each",
		"jsonb_each",
		"json_each_text",
		"jsonb_each_text",
		"json_extract_path",
		"jsonb_extract_path",
		"json_extract_path_text",
		"jsonb_extract_path_text",
		"json_object_keys",
		"jsonb_object_keys",
		"json_populate_record",
		"jsonb_populate_record",
		"json_populate_recordset",
		"jsonb_populate_recordset",
		"json_to_record",
		"jsonb_to_record",
		"json_to_recordset",
		"jsonb_to_recordset",
		"jsonb_set",
		"jsonb_set_lax",
		"jsonb_insert",
		"json_strip_nulls",
		"jsonb_strip_nulls",
		"jsonb_path_exists",
		"jsonb_path_match",
		"jsonb_path_query",
		"jsonb_path_query_array",
		"jsonb_path_query_first",
		"jsonb_path_exists_tz",
		"jsonb_path_match_tz",
		"jsonb_path_query_tz",
		"jsonb_path_query_array_tz",
		"jsonb_path_query_first_tz",
		"jsonb_pretty",
		"json_typeof",
		"jsonb_typeof",
		"exists",
	}

	sequenceManipulationFunctions = []string{
		"nextval",
		"setval",
		"currval",
		"lastval",
	}

	conditionalFunctions = []string{
		"COALESCE",
		"NULLIF",
		"GREATEST",
		"LEAST",
	}

	arrayFunctions = []string{
		"array_append",
		"array_cat",
		"array_dims",
		"array_fill",
		"array_length",
		"array_lower",
		"array_ndims",
		"array_position",
		"array_positions",
		"array_prepend",
		"array_remove",
		"array_replace",
		"array_sample",
		"array_shuffle",
		"array_to_string",
		"array_upper",
		"cardinality",
		"trim_array",
		"unnest",
	}

	rangeFunctions = []string{
		"lower",
		"upper",
		"isempty",
		"lower_inc",
		"upper_inc",
		"lower_inf",
		"upper_inf",
		"range_merge",
		"multirange",
		"unnest",
	}

	aggregateFunctions = []string{
		"any_value",
		"array_agg",
		"avg",
		"bit_and",
		"bit_or",
		"bit_xor",
		"bool_and",
		"bool_or",
		"count",
		"every",
		"json_agg",
		"jsonb_agg",
		"json_objectagg",
		"json_object_agg",
		"jsonb_object_agg",
		"json_object_agg_strict",
		"jsonb_object_agg_strict",
		"json_object_agg_unique",
		"jsonb_object_agg_unique",
		"json_arrayagg",
		"json_object_agg_unique_strict",
		"jsonb_object_agg_unique_strict",
		"max",
		"min",
		"range_agg",
		"range_intersect_agg",
		"json_agg_strict",
		"jsonb_agg_strict",
		"string_agg",
		"sum",
		"xmlagg",
		"corr",
		"covar_pop",
		"covar_samp",
		"regr_avgx",
		"regr_avgy",
		"regr_count",
		"regr_intercept",
		"regr_r2",
		"regr_slope",
		"regr_sxx",
		"regr_sxy",
		"regr_syy",
		"stddev",
		"stddev_pop",
		"stddev_samp",
		"variance",
		"var_pop",
		"var_samp",
		"mode",
		"percentile_cont",
		"percentile_disc",
		"rank",
		"dense_rank",
		"percent_rank",
		"cume_dist",
		"GROUPING",
	}

	windowFunctions = []string{
		"row_number",
		"rank",
		"dense_rank",
		"percent_rank",
		"cume_dist",
		"ntile",
		"lag",
		"lead",
		"first_value",
		"last_value",
		"nth_value",
	}

	setReturningFunctions = []string{
		"generate_series",
		"generate_subscripts",
	}

	systemInformationFunctions = []string{
		"current_database",
		"current_query",
		"current_schema",
		"current_schemas",
		"inet_client_addr",
		"inet_client_port",
		"inet_server_addr",
		"inet_server_port",
		"pg_backend_pid",
		"pg_blocking_pids",
		"pg_conf_load_time",
		"pg_current_logfile",
		"pg_my_temp_schema",
		"pg_is_other_temp_schema",
		"pg_jit_available",
		"pg_listening_channels",
		"pg_notification_queue_usage",
		"pg_postmaster_start_time",
		"pg_safe_snapshot_blocking_pids",
		"pg_trigger_depth",
		"version",
		"has_any_column_privilege",
		"has_column_privilege",
		"has_database_privilege",
		"has_foreign_data_wrapper_privilege",
		"has_function_privilege",
		"has_language_privilege",
		"has_parameter_privilege",
		"has_schema_privilege",
		"has_sequence_privilege",
		"has_server_privilege",
		"has_table_privilege",
		"has_tablespace_privilege",
		"has_type_privilege",
		"pg_has_role",
		"row_security_active",
		"acldefault",
		"aclexplode",
		"makeaclitem",
		"pg_collation_is_visible",
		"pg_conversion_is_visible",
		"pg_function_is_visible",
		"pg_opclass_is_visible",
		"pg_operator_is_visible",
		"pg_opfamily_is_visible",
		"pg_statistics_obj_is_visible",
		"pg_table_is_visible",
		"pg_ts_config_is_visible",
		"pg_ts_dict_is_visible",
		"pg_ts_parser_is_visible",
		"pg_ts_template_is_visible",
		"pg_type_is_visible",
		"format_type",
		"pg_char_to_encoding",
		"pg_encoding_to_char",
		"pg_get_catalog_foreign_keys",
		"pg_get_constraintdef",
		"pg_get_expr",
		"pg_get_functiondef",
		"pg_get_function_arguments",
		"pg_get_function_identity_arguments",
		"pg_get_function_result",
		"pg_get_indexdef",
		"pg_get_keywords",
		"pg_get_partkeydef",
		"pg_get_ruledef",
		"pg_get_serial_sequence",
		"pg_get_statisticsobjdef",
		"pg_get_triggerdef",
		"pg_get_userbyid",
		"pg_get_viewdef",
		"pg_index_column_has_property",
		"pg_index_has_property",
		"pg_indexam_has_property",
		"pg_options_to_table",
		"pg_settings_get_flags",
		"pg_tablespace_databases",
		"pg_tablespace_location",
		"pg_typeof",
		"COLLATION FOR",
		"to_regclass",
		"to_regcollation",
		"to_regnamespace",
		"to_regoper",
		"to_regoperator",
		"to_regproc",
		"to_regprocedure",
		"to_regrole",
		"to_regtype",
		"pg_describe_object",
		"pg_identify_object",
		"pg_identify_object_as_address",
		"pg_get_object_address",
		"col_description",
		"obj_description",
		"shobj_description",
		"pg_input_is_valid",
		"pg_input_error_info",
		"pg_current_xact_id",
		"pg_current_xact_id_if_assigned",
		"pg_xact_status",
		"pg_current_snapshot",
		"pg_snapshot_xip",
		"pg_snapshot_xmax",
		"pg_snapshot_xmin",
		"pg_visible_in_snapshot",
		"txid_current",
		"txid_current_if_assigned",
		"txid_current_snapshot",
		"txid_snapshot_xip",
		"txid_snapshot_xmax",
		"txid_snapshot_xmin",
		"txid_visible_in_snapshot",
		"txid_status",
		"pg_xact_commit_timestamp",
		"pg_xact_commit_timestamp_origin",
		"pg_last_committed_xact",
		"pg_control_checkpoint",
		"pg_control_system",
		"pg_control_init",
		"pg_control_recovery",
	}

	systemAdministrationFunctions = []string{
		"current_setting",
		"set_config",
		"pg_cancel_backend",
		"pg_log_backend_memory_contexts",
		"pg_reload_conf",
		"pg_rotate_logfile",
		"pg_terminate_backend",
		"pg_create_restore_point",
		"pg_current_wal_flush_lsn",
		"pg_current_wal_insert_lsn",
		"pg_current_wal_lsn",
		"pg_backup_start",
		"pg_backup_stop",
		"pg_switch_wal",
		"pg_walfile_name",
		"pg_walfile_name_offset",
		"pg_split_walfile_name",
		"pg_wal_lsn_diff",
		"pg_is_in_recovery",
		"pg_last_wal_receive_lsn",
		"pg_last_wal_replay_lsn",
		"pg_last_xact_replay_timestamp",
		"pg_get_wal_resource_managers",
		"pg_is_wal_replay_paused",
		"pg_get_wal_replay_pause_state",
		"pg_promote",
		"pg_wal_replay_pause",
		"pg_wal_replay_resume",
		"pg_export_snapshot",
		"pg_log_standby_snapshot",
		"pg_create_physical_replication_slot",
		"pg_drop_replication_slot",
		"pg_create_logical_replication_slot",
		"pg_copy_physical_replication_slot",
		"pg_copy_logical_replication_slot",
		"pg_logical_slot_get_changes",
		"pg_logical_slot_peek_changes",
		"pg_logical_slot_get_binary_changes",
		"pg_logical_slot_peek_binary_changes",
		"pg_replication_slot_advance",
		"pg_replication_origin_create",
		"pg_replication_origin_drop",
		"pg_replication_origin_oid",
		"pg_replication_origin_session_setup",
		"pg_replication_origin_session_reset",
		"pg_replication_origin_session_is_setup",
		"pg_replication_origin_session_progress",
		"pg_replication_origin_xact_setup",
		"pg_replication_origin_xact_reset",
		"pg_replication_origin_advance",
		"pg_replication_origin_progress",
		"pg_logical_emit_message",
		"pg_column_size",
		"pg_column_compression",
		"pg_database_size",
		"pg_indexes_size",
		"pg_relation_size",
		"pg_size_bytes",
		"pg_size_pretty",
		"pg_table_size",
		"pg_tablespace_size",
		"pg_total_relation_size",
		"pg_relation_filenode",
		"pg_relation_filepath",
		"pg_filenode_relation",
		"pg_collation_actual_version",
		"pg_database_collation_actual_version",
		"pg_import_system_collations",
		"pg_partition_tree",
		"pg_partition_ancestors",
		"pg_partition_root",
		"brin_summarize_new_values",
		"brin_summarize_range",
		"brin_desummarize_range",
		"gin_clean_pending_list",
		"pg_ls_dir",
		"pg_ls_logdir",
		"pg_ls_waldir",
		"pg_ls_logicalmapdir",
		"pg_ls_logicalsnapdir",
		"pg_ls_replslotdir",
		"pg_ls_archive_statusdir",
		"pg_ls_tmpdir",
		"pg_read_file",
		"pg_read_binary_file",
		"pg_stat_file",
		"pg_advisory_lock",
		"pg_advisory_lock_shared",
		"pg_advisory_unlock",
		"pg_advisory_unlock_all",
		"pg_advisory_unlock_shared",
		"pg_advisory_xact_lock",
		"pg_advisory_xact_lock_shared",
		"pg_try_advisory_lock",
		"pg_try_advisory_lock_shared",
		"pg_try_advisory_xact_lock",
		"pg_try_advisory_xact_lock_shared",
	}

	triggerFunctions = []string{
		"suppress_redundant_updates_trigger",
		"tsvector_update_trigger",
		"tsvector_update_trigger_column",
	}

	eventTriggerFunctions = []string{
		"pg_event_trigger_ddl_commands",
		"pg_event_trigger_dropped_objects",
		"pg_event_trigger_table_rewrite_oid",
		"pg_event_trigger_table_rewrite_reason",
	}

	statisticsFunctions = []string{
		"pg_event_trigger_table_rewrite_reason",
	}

	builtinFunctions = []string{}
)

func init() {
	funcMap := make(map[string]bool)

	for _, f := range comparisonFunctions {
		funcMap[f] = true
	}
	for _, f := range mathematicalFunctions {
		funcMap[f] = true
	}
	for _, f := range stringFunctions {
		funcMap[f] = true
	}
	for _, f := range binaryStringFunctions {
		funcMap[f] = true
	}
	for _, f := range bitStringFunctions {
		funcMap[f] = true
	}
	for _, f := range dataTypeFormattingFunctions {
		funcMap[f] = true
	}
	for _, f := range dateTimeFunctions {
		funcMap[f] = true
	}
	for _, f := range enumSupportFunctions {
		funcMap[f] = true
	}
	for _, f := range geometricFunctions {
		funcMap[f] = true
	}
	for _, f := range networkAddressFunctions {
		funcMap[f] = true
	}
	for _, f := range textSearchFunctions {
		funcMap[f] = true
	}
	for _, f := range uuidFunctions {
		funcMap[f] = true
	}
	for _, f := range xmlFunctions {
		funcMap[f] = true
	}
	for _, f := range jsonFunctions {
		funcMap[f] = true
	}
	for _, f := range sequenceManipulationFunctions {
		funcMap[f] = true
	}
	for _, f := range conditionalFunctions {
		funcMap[f] = true
	}
	for _, f := range arrayFunctions {
		funcMap[f] = true
	}
	for _, f := range rangeFunctions {
		funcMap[f] = true
	}
	for _, f := range aggregateFunctions {
		funcMap[f] = true
	}
	for _, f := range windowFunctions {
		funcMap[f] = true
	}
	for _, f := range setReturningFunctions {
		funcMap[f] = true
	}
	for _, f := range systemInformationFunctions {
		funcMap[f] = true
	}
	for _, f := range systemAdministrationFunctions {
		funcMap[f] = true
	}
	for _, f := range triggerFunctions {
		funcMap[f] = true
	}
	for _, f := range eventTriggerFunctions {
		funcMap[f] = true
	}
	for _, f := range statisticsFunctions {
		funcMap[f] = true
	}

	for f := range funcMap {
		builtinFunctions = append(builtinFunctions, f)
	}

	slices.Sort(builtinFunctions)
}

func getBuiltinFunctions() []string {
	var result []string
	result = append(result, builtinFunctions...)
	return result
}
