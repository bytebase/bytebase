package api

import (
	"encoding/json"
	"errors"
)

// DataSourceType is the type of data source.
type DataSourceType string

const (
	// Admin is the ADMIN type of data source.
	Admin DataSourceType = "ADMIN"
	// RO is the read-only type of data source.
	RO DataSourceType = "RO"
)

// DataSourceOptions is the options for a data source.
type DataSourceOptions struct {
	// SRV is used for MongoDB only.
	SRV bool `json:"srv" jsonapi:"attr,srv"`
	// AuthenticationDatabase is used for MongoDB only.
	AuthenticationDatabase string `json:"authenticationDatabase" jsonapi:"attr,authenticationDatabase"`
	// SID and ServiceName are used for Oracle.
	SID           string `json:"sid" jsonapi:"attr,sid"`
	ServiceName   string `json:"serviceName" jsonapi:"attr,serviceName"`
	SSHHost       string `json:"sshHost" jsonapi:"attr,sshHost"`
	SSHPort       string `json:"sshPort" jsonapi:"attr,sshPort"`
	SSHUser       string `json:"sshUser" jsonapi:"attr,sshUser"`
	SSHPassword   string `json:"sshPassword" jsonapi:"attr,sshPassword"`
	SSHPrivateKey string `json:"sshPrivateKey" jsonapi:"attr,sshPrivateKey"`
}

// getDefaultDataSourceOptions returns the default data source options.
func getDefaultDataSourceOptions() DataSourceOptions {
	return DataSourceOptions{}
}

// Scan implements database/sql Scanner interface, converts JSONB to DataSourceOptions struct.
func (d *DataSourceOptions) Scan(src any) error {
	if bs, ok := src.([]byte); ok {
		if string(bs) == "{}" {
			// handle '{}', return default values
			*d = getDefaultDataSourceOptions()
			return nil
		}
		return json.Unmarshal(bs, d)
	}
	return errors.New("failed to scan data source options")
}

// DataSource is the API message for a data source.
type DataSource struct {
	ID int `jsonapi:"primary,dataSource"`

	// Related fields
	// Just returns InstanceID and DatabaseID otherwise would cause circular dependency.
	InstanceID int `jsonapi:"attr,instanceId"`
	DatabaseID int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name     string            `jsonapi:"attr,name"`
	Type     DataSourceType    `jsonapi:"attr,type"`
	Username string            `jsonapi:"attr,username"`
	Host     string            `jsonapi:"attr,host"`
	Port     string            `jsonapi:"attr,port"`
	Options  DataSourceOptions `jsonapi:"attr,options"`
	Database string            `jsonapi:"attr,database"`
}
