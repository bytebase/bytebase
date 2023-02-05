package api

import (
	"encoding/json"
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
	// SettingEnterpriseLicense is the setting name for enterprise license.
	SettingEnterpriseLicense SettingName = "bb.enterprise.license"
	// SettingEnterpriseTrial is the setting name for free trial.
	SettingEnterpriseTrial SettingName = "bb.enterprise.trial"
	// SettingAppIM is the setting name for IM applications.
	SettingAppIM SettingName = "bb.app.im"
	// SettingWatermark is the setting name for watermark displaying.
	SettingWatermark SettingName = "bb.workspace.watermark"
)

// IMType is the type of IM.
type IMType string

// IMTypeFeishu is IM feishu.
const IMTypeFeishu IMType = "im.feishu"

// Setting is the API message for a setting.
type Setting struct {
	// Domain specific fields
	Name        SettingName `jsonapi:"attr,name"`
	Value       string      `jsonapi:"attr,value"`
	Description string      `jsonapi:"attr,description"`
}

// SettingCreate is the API message for creating a setting.
type SettingCreate struct {
	CreatorID   int
	Name        SettingName
	Value       string
	Description string
}

// SettingFind is the API message for finding settings.
type SettingFind struct {
	Name *SettingName
}

// SettingPatch is the API message for patching a setting.
type SettingPatch struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	Name  SettingName
	Value string `jsonapi:"attr,value"`
}

func (find *SettingFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// SettingAppIMValue is the setting value of SettingAppIM type setting.
type SettingAppIMValue struct {
	IMType           IMType `json:"imType"`
	AppID            string `json:"appId"`
	AppSecret        string `json:"appSecret"`
	ExternalApproval struct {
		Enabled              bool   `json:"enabled"`
		ApprovalDefinitionID string `json:"approvalDefinitionID"`
	} `json:"externalApproval"`
}
