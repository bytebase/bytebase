package api

import (
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

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
	// SettingEnterpriseTrial is the setting name for free trial.
	SettingEnterpriseTrial SettingName = "bb.enterprise.trial"
	// SettingAppIM is the setting name for IM applications.
	SettingAppIM SettingName = "bb.app.im"
	// SettingWatermark is the setting name for watermark displaying.
	SettingWatermark SettingName = "bb.workspace.watermark"
	// SettingPluginOpenAIKey is used for OpenAI's API key.
	// For AI-related features.
	SettingPluginOpenAIKey SettingName = "bb.plugin.openai.key"
	// SettingPluginOpenAIEndpoint is used for OpenAI's API endpoint.
	SettingPluginOpenAIEndpoint SettingName = "bb.plugin.openai.endpoint"
	// SettingPluginAgent is the setting name for the internal agent API.
	// For now we will call the hub to fetch the subscription license.
	SettingPluginAgent SettingName = "bb.plugin.agent"
	// SettingWorkspaceMailDelivery is the setting name for workspace mail delivery.
	SettingWorkspaceMailDelivery SettingName = "bb.workspace.mail-delivery"
	// SettingSchemaTemplate is the setting name for schema template.
	SettingSchemaTemplate SettingName = "bb.workspace.schema-template"
	// SettingDataClassification is the setting name for data classification.
	SettingDataClassification SettingName = "bb.workspace.data-classification"
)

// IMType is the type of IM.
type IMType string

// IMTypeFeishu is IM feishu.
const IMTypeFeishu IMType = "im.feishu"

// ExternalApproval is the external approval setting for app IM.
type ExternalApproval struct {
	Enabled              bool   `json:"enabled"`
	ApprovalDefinitionID string `json:"approvalDefinitionID"`
}

// SettingAppIMValue is the setting value of SettingAppIM type setting.
type SettingAppIMValue struct {
	IMType           IMType           `json:"imType"`
	AppID            string           `json:"appId"`
	AppSecret        string           `json:"appSecret"`
	ExternalApproval ExternalApproval `json:"externalApproval"`
}

// SettingWorkspaceMailDeliveryValue is the setting value of SettingMailDelivery type setting.
type SettingWorkspaceMailDeliveryValue struct {
	SMTPServerHost         string                                         `json:"smtpServerHost"`
	SMTPServerPort         int                                            `json:"smtpServerPort"`
	SMTPUsername           string                                         `json:"smtpUsername"`
	SMTPPassword           *string                                        `json:"smtpPassword"`
	SMTPFrom               string                                         `json:"smtpFrom"`
	SMTPAuthenticationType storepb.SMTPMailDeliverySetting_Authentication `json:"smtpAuthenticationType"`
	SMTPEncryptionType     storepb.SMTPMailDeliverySetting_Encryption     `json:"smtpEncryptionType"`
	SMTPTo                 string                                         `json:"sendTo"`
}
