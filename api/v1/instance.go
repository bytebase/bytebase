package v1

import (
	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
)

// Instance is the API message for an instance.
type Instance struct {
	ID int `jsonapi:"primary,instance" json:"id"`

	// Standard fields
	RowStatus api.RowStatus `json:"rowStatus"`
	CreatorID int           `json:"creatorId"`
	UpdaterID int           `json:"updaterId"`
	CreatedTs int64         `json:"createdTs"`
	UpdatedTs int64         `json:"updatedTs"`

	// Related fields
	EnvironmentName string `json:"environmentName"`

	// Domain specific fields
	Name          string  `json:"name"`
	Engine        db.Type `json:"engine"`
	EngineVersion string  `json:"engineVersion"`
	ExternalLink  string  `json:"externalLink"`
	Host          string  `json:"host"`
	Port          string  `json:"port"`
	Username      string  `json:"username"`
}
