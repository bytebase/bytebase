package iam

import (
	_ "embed"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

//go:embed acl.yaml
var aclYaml []byte

type acl struct {
	Allowlist []Binding `yaml:"allowlist"`
}

type Binding struct {
	Role        string   `yaml:"role"`
	Permissions []string `yaml:"permissions"`
}

type Manager struct {
	Allowlist map[string][]Permission
}

func NewManager() (*Manager, error) {
	predefinedACL := new(acl)
	if err := yaml.Unmarshal(aclYaml, predefinedACL); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal predefined acl")
	}

	allowlist := make(map[string][]Permission)
	for _, binding := range predefinedACL.Allowlist {
		allowlist[binding.Role] = make([]Permission, len(binding.Permissions))
		for i, permission := range binding.Permissions {
			allowlist[binding.Role][i] = Permission(permission)
		}
	}

	return &Manager{
		Allowlist: allowlist,
	}, nil
}
