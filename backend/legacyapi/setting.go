package api

// SettingName is the name of a setting.
type SettingName string

const (
	// SettingAuthSecret is the setting name for auth secret.
	SettingAuthSecret SettingName = "bb.auth.secret"
	// SettingBrandingLogo is the setting name for branding logo.
	SettingBrandingLogo SettingName = "bb.branding.logo"
	// SettingWorkspaceID is the setting name for workspace identifier.
	SettingWorkspaceID SettingName = "bb.workspace.id"
	// SettingWorkspaceProfile is the setting name for workspace profile settings.
	SettingWorkspaceProfile SettingName = "bb.workspace.profile"
	// SettingWorkspaceApproval is the setting name for workspace approval config.
	SettingWorkspaceApproval SettingName = "bb.workspace.approval"
	// SettingWorkspaceExternalApproval is the setting name for workspace external approval config.
	SettingWorkspaceExternalApproval SettingName = "bb.workspace.approval.external"
	// SettingEnterpriseLicense is the setting name for enterprise license.
	SettingEnterpriseLicense SettingName = "bb.enterprise.license"
	// SettingAppIM is the setting name for IM applications.
	SettingAppIM SettingName = "bb.app.im"
	// SettingWatermark is the setting name for watermark displaying.
	SettingWatermark SettingName = "bb.workspace.watermark"
	// SettingAI is used for AI setting.
	SettingAI SettingName = "bb.ai"
	// SettingPluginAgent is the setting name for the internal agent API.
	// For now we will call the hub to fetch the subscription license.
	SettingPluginAgent SettingName = "bb.plugin.agent"
	// SettingWorkspaceMailDelivery is the setting name for workspace mail delivery.
	SettingWorkspaceMailDelivery SettingName = "bb.workspace.mail-delivery"
	// SettingSchemaTemplate is the setting name for schema template.
	SettingSchemaTemplate SettingName = "bb.workspace.schema-template"
	// SettingDataClassification is the setting name for data classification.
	SettingDataClassification SettingName = "bb.workspace.data-classification"
	// SettingSemanticTypes is the setting name for semantic types.
	SettingSemanticTypes SettingName = "bb.workspace.semantic-types"
	// SettingSQLResultSizeLimit is the setting name for SQL query result size limit.
	SettingSQLResultSizeLimit SettingName = "bb.workspace.maximum-sql-result-size"
	// SettingSCIM is the setting name for SCIM config.
	SettingSCIM SettingName = "bb.workspace.scim"
	// SettingPasswordRestriction is the setting name for password.
	SettingPasswordRestriction SettingName = "bb.workspace.password-restriction"
	// SettingEnvironment is the setting name for environments.
	SettingEnvironment SettingName = "bb.workspace.environment"
)
