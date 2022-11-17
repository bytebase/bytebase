package v1

import (
	"github.com/bytebase/bytebase/plugin/db"
)

// Instance is the API message for an instance.
type Instance struct {
	ID int `jsonapi:"primary,instance" json:"id"`

	// Related fields
	Environment string `json:"environment"`

	// Domain specific fields
	Name          string  `json:"name"`
	Engine        db.Type `json:"engine"`
	EngineVersion string  `json:"engineVersion"`
	ExternalLink  string  `json:"externalLink"`
	Host          string  `json:"host"`
	Port          string  `json:"port"`
	Username      string  `json:"username"`
}
