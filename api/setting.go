package api

import (
	"context"
	"encoding/json"
)

type SettingName string

const (
	SettingAuthSecret SettingName = "bb.auth.secret"
	// The console URL, it supports following variables
	// {{DB_NAME}}: the database name
	// e.g. For a phpmyadmin instance running on http://myphpadmin.example.com:8080, the setting would be:
	// http://myphpadmin.example.com:8080/index.php?route=/database/sql&db={{DB_NAME}}
	SettingConsoleURL SettingName = "bb.console.url"
)

type Setting struct {
	ID int `jsonapi:"primary,setting"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Name        SettingName `jsonapi:"attr,name"`
	Value       string      `jsonapi:"attr,value"`
	Description string      `jsonapi:"attr,description"`
}

type SettingCreate struct {
	CreatorID   int
	Name        SettingName
	Value       string
	Description string
}

type SettingFind struct {
	Name *SettingName
}

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

type SettingService interface {
	// Creates new setting and returns if not exist, returns the existing one otherwise.
	CreateSettingIfNotExist(ctx context.Context, create *SettingCreate) (*Setting, error)
	FindSettingList(ctx context.Context, find *SettingFind) ([]*Setting, error)
	FindSetting(ctx context.Context, find *SettingFind) (*Setting, error)
	PatchSetting(ctx context.Context, patch *SettingPatch) (*Setting, error)
}
