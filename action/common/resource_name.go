//nolint:revive
package common

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

const (
	ProjectNamePrefix       = "projects/"
	InstanceNamePrefix      = "instances/"
	DatabaseIDPrefix        = "databases/"
	DatabaseGroupNamePrefix = "databaseGroups/"
)

// GetProjectIDDatabaseGroupID returns the project ID and database group ID from a resource name.
func GetProjectIDDatabaseGroupID(name string) (string, string, error) {
	tokens, err := GetNameParentTokens(name, ProjectNamePrefix, DatabaseGroupNamePrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

// GetInstanceDatabaseID returns the instance ID and database ID from a resource name.
func GetInstanceDatabaseID(name string) (string, string, error) {
	// the instance request should be instances/{instance-id}/databases/{database-id}
	tokens, err := GetNameParentTokens(name, InstanceNamePrefix, DatabaseIDPrefix)
	if err != nil {
		return "", "", err
	}
	return tokens[0], tokens[1], nil
}

// GetNameParentTokens returns the tokens from a resource name.
func GetNameParentTokens(name string, tokenPrefixes ...string) ([]string, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 2*len(tokenPrefixes) {
		return nil, errors.Errorf("invalid request %q", name)
	}

	var tokens []string
	for i, tokenPrefix := range tokenPrefixes {
		if fmt.Sprintf("%s/", parts[2*i]) != tokenPrefix {
			return nil, errors.Errorf("invalid prefix %q in request %q", tokenPrefix, name)
		}
		tokens = append(tokens, parts[2*i+1])
	}
	return tokens, nil
}
