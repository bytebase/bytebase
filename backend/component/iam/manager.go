package iam

import (
	_ "embed"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

//go:embed acl.yaml
var aclYaml []byte

type acl struct {
	Roles []Role `yaml:"roles"`
}

type Role struct {
	Name        string   `yaml:"name"`
	Permissions []string `yaml:"permissions"`
}

type Manager struct {
	roles map[string][]Permission
}

func NewManager() (*Manager, error) {
	predefinedACL := new(acl)
	if err := yaml.Unmarshal(aclYaml, predefinedACL); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal predefined acl")
	}

	roles := make(map[string][]Permission)
	for _, binding := range predefinedACL.Roles {
		for _, permission := range binding.Permissions {
			roles[binding.Name] = append(roles[binding.Name], Permission(permission))
		}
	}

	return &Manager{
		roles: roles,
	}, nil
}
