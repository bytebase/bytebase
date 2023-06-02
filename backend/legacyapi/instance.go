package api

import (
	"encoding/json"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

// Instance is the API message for an instance.
type Instance struct {
	ID         int       `jsonapi:"primary,instance"`
	ResourceID string    `jsonapi:"attr,resourceId"`
	RowStatus  RowStatus `jsonapi:"attr,rowStatus"`

	// Related fields
	EnvironmentID  int
	Environment    *Environment  `jsonapi:"relation,environment"`
	DataSourceList []*DataSource `jsonapi:"relation,dataSourceList"`

	// Domain specific fields
	Name          string  `jsonapi:"attr,name"`
	Engine        db.Type `jsonapi:"attr,engine"`
	EngineVersion string  `jsonapi:"attr,engineVersion"`
	ExternalLink  string  `jsonapi:"attr,externalLink"`
	Host          string  `jsonapi:"attr,host"`
	Port          string  `jsonapi:"attr,port"`
	// Database is the initial connection database for PostgreSQL only.
	Database string `jsonapi:"attr,database"`
	Username string `jsonapi:"attr,username"`
	// Password is not returned to the client
	Password string
}

// InstanceFind is the API message for finding instances.
type InstanceFind struct {
	ID *int

	// Standard fields
	RowStatus *RowStatus
}

func (find *InstanceFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}
