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
	// SettingSchemaSystemBoot it the setting name for schema system that needs to boot.
	SettingSchemaSystemBoot SettingName = "bb.schema-system.boot"
)

// Setting is the API message for a setting.
type Setting struct {
	ID int `jsonapi:"primary,setting"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

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
