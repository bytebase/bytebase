package v1

import (
	"github.com/bytebase/bytebase/plugin/db"
)

// Instance is the API message for an instance.
type Instance struct {
	ID int `json:"id"`

	// Related fields
	Environment string `json:"environment"`

	// Domain specific fields
	Name          string  `json:"name"`
	Engine        db.Type `json:"engine"`
	EngineVersion string  `json:"engineVersion"`
	ExternalLink  string  `json:"externalLink"`
	Host          string  `json:"host"`
	Port          string  `json:"port"`
	Database      string  `json:"database"`
	Username      string  `json:"username"`
}

// InstanceCreate is the API message for creating an instance.
type InstanceCreate struct {
	// Related fields
	Environment string `json:"environment"`

	// Domain specific fields
	Name         string  `json:"name"`
	Engine       db.Type `json:"engine"`
	ExternalLink string  `json:"externalLink"`
	Host         string  `json:"host"`
	Port         string  `json:"port"`
	Database     string  `json:"database"`
	Username     string  `json:"username"`
	Password     string  `json:"password"`
	SslCa        string  `json:"sslCa"`
	SslCert      string  `json:"sslCert"`
	SslKey       string  `json:"sslKey"`
}

// InstancePatch is the API message for patching an instance.
type InstancePatch struct {
	// Domain specific fields
	Name         *string `json:"name"`
	ExternalLink *string `json:"externalLink"`
	Host         *string `json:"host"`
	Port         *string `json:"port"`
	Database     *string `json:"database"`
}
