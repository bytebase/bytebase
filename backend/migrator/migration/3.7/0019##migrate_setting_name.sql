-- Update setting name values to match proto enum names
UPDATE setting SET name = CASE
    WHEN name = 'bb.auth.secret' THEN 'AUTH_SECRET'
    WHEN name = 'bb.branding.logo' THEN 'BRANDING_LOGO'
    WHEN name = 'bb.workspace.id' THEN 'WORKSPACE_ID'
    WHEN name = 'bb.workspace.profile' THEN 'WORKSPACE_PROFILE'
    WHEN name = 'bb.workspace.approval' THEN 'WORKSPACE_APPROVAL'
    WHEN name = 'bb.workspace.approval.external' THEN 'WORKSPACE_EXTERNAL_APPROVAL'
    WHEN name = 'bb.enterprise.license' THEN 'ENTERPRISE_LICENSE'
    WHEN name = 'bb.app.im' THEN 'APP_IM'
    WHEN name = 'bb.workspace.watermark' THEN 'WATERMARK'
    WHEN name = 'bb.ai' THEN 'AI'
    WHEN name = 'bb.plugin.agent' THEN 'PLUGIN_AGENT'
    WHEN name = 'bb.workspace.mail-delivery' THEN 'WORKSPACE_MAIL_DELIVERY'
    WHEN name = 'bb.workspace.schema-template' THEN 'SCHEMA_TEMPLATE'
    WHEN name = 'bb.workspace.data-classification' THEN 'DATA_CLASSIFICATION'
    WHEN name = 'bb.workspace.semantic-types' THEN 'SEMANTIC_TYPES'
    WHEN name = 'bb.workspace.maximum-sql-result-size' THEN 'SQL_RESULT_SIZE_LIMIT'
    WHEN name = 'bb.workspace.scim' THEN 'SCIM'
    WHEN name = 'bb.workspace.password-restriction' THEN 'PASSWORD_RESTRICTION'
    WHEN name = 'bb.workspace.environment' THEN 'ENVIRONMENT'
    ELSE name
END;