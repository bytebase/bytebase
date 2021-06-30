package api

import (
	"context"
	"encoding/json"
)

type Config struct {
	Name        string
	Value       string
	Description string
}

type ConfigCreate struct {
	CreatorId   int
	Name        string
	Value       string
	Description string
}

type ConfigFind struct {
	Name string
}

func (find *ConfigFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type ConfigService interface {
	CreateConfigIfNotExist(ctx context.Context, create *ConfigCreate) (*Config, error)
	FindConfig(ctx context.Context, find *ConfigFind) (*Config, error)
}
