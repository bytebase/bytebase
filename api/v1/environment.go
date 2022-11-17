package v1

import "github.com/bytebase/bytebase/api"

// Environment is the API message for an environment.
type Environment struct {
	ID int `json:"id"`

	// Domain specific fields
	Name  string                   `json:"name"`
	Order int                      `json:"order"`
	Tier  api.EnvironmentTierValue `json:"tier"`
}

// EnvironmentPatch is the API message for patching an environment.
type EnvironmentPatch struct {
	// Domain specific fields
	Name  *string `json:"name"`
	Order *int    `json:"order"`
}

// EnvironmentCreate is the API message for creating an environment.
type EnvironmentCreate struct {
	// Domain specific fields
	Name string `json:"name"`
}
