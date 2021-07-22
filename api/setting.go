package api

import (
	"context"
	"encoding/json"
)

type SettingName string

const (
	SettingAuthSecret SettingName = "bb.auth.secret"
)

type Setting struct {
	Name        SettingName
	Value       string
	Description string
}

type SettingCreate struct {
	CreatorId   int
	Name        SettingName
	Value       string
	Description string
}

type SettingFind struct {
	Name SettingName
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
	FindSetting(ctx context.Context, find *SettingFind) (*Setting, error)
}
