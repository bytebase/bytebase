package api

import (
	"encoding/json"
	"errors"
)

const (
	// AdminDataSourceName is the name for administrative data source.
	AdminDataSourceName = "Admin data source"
	// ReadOnlyDataSourceName is the name for read-only data source.
	ReadOnlyDataSourceName = "ReadOnly data source"
	// UnknownDataSourceName is the name for unknown data source.
	UnknownDataSourceName = "Unknown data source"
)

// DataSourceNameFromType maps the name from a data source type.
func DataSourceNameFromType(dataSourceType DataSourceType) string {
	switch dataSourceType {
	case Admin:
		return AdminDataSourceName
	case RO:
		return ReadOnlyDataSourceName
	}
	return UnknownDataSourceName
}

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
}

// getDefaultDataSourceOptions returns the default data source options.
func getDefaultDataSourceOptions() DataSourceOptions {
	return DataSourceOptions{
		SRV:                    false,
		AuthenticationDatabase: "",
	}
}

// Scan implements database/sql Scanner interface, converts JSONB to DataSourceOptions struct.
func (d *DataSourceOptions) Scan(src interface{}) error {
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

// DataSourceCreate is the API message for creating a data source.
type DataSourceCreate struct {
	Name     string            `jsonapi:"attr,name"`
	Type     DataSourceType    `jsonapi:"attr,type"`
	Username string            `jsonapi:"attr,username"`
	Password string            `jsonapi:"attr,password"`
	SslCa    string            `jsonapi:"attr,sslCa"`
	SslCert  string            `jsonapi:"attr,sslCert"`
	SslKey   string            `jsonapi:"attr,sslKey"`
	Host     string            `jsonapi:"attr,host"`
	Port     string            `jsonapi:"attr,port"`
	Options  DataSourceOptions `jsonapi:"attr,options"`
	Database string            `jsonapi:"attr,database"`
}

// DataSourcePatch is the API message for data source.
type DataSourcePatch struct {
	Username         *string            `jsonapi:"attr,username"`
	Password         *string            `jsonapi:"attr,password"`
	UseEmptyPassword *bool              `jsonapi:"attr,useEmptyPassword"`
	SslCa            *string            `jsonapi:"attr,sslCa"`
	SslCert          *string            `jsonapi:"attr,sslCert"`
	SslKey           *string            `jsonapi:"attr,sslKey"`
	Host             *string            `jsonapi:"attr,host"`
	Port             *string            `jsonapi:"attr,port"`
	Options          *DataSourceOptions `jsonapi:"attr,options"`
	Database         *string            `jsonapi:"attr,database"`
}
