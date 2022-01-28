package api

import (
	"context"
	"encoding/json"
)

// SettingName is the name of a setting.
type SettingName string

const (
	// SettingAuthSecret is the setting name for auth secret.
	SettingAuthSecret SettingName = "bb.auth.secret"
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

// SettingService is the service for settings.
type SettingService interface {
	// Creates new setting and returns if not exist, returns the existing one otherwise.
	CreateSettingIfNotExist(ctx context.Context, create *SettingCreate) (*Setting, error)
	FindSettingList(ctx context.Context, find *SettingFind) ([]*Setting, error)
	FindSetting(ctx context.Context, find *SettingFind) (*Setting, error)
	PatchSetting(ctx context.Context, patch *SettingPatch) (*Setting, error)
}
