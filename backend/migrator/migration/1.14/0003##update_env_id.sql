UPDATE repository
SET
	file_path_template=REPLACE(repository.file_path_template, '{{ENV_NAME}}', '{{ENV_ID}}'),
	schema_path_template=REPLACE(repository.schema_path_template, '{{ENV_NAME}}', '{{ENV_ID}}'),
	sheet_path_template=REPLACE(repository.sheet_path_template, '{{ENV_NAME}}', '{{ENV_ID}}')
;
