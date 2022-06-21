package api

import (
	"encoding/json"
)

const (
	// AdminDataSourceName is the name for administrative data source.
	AdminDataSourceName = "Admin data source"
	// ReadWriteDataSourceName is the name for read/write data source.
	ReadWriteDataSourceName = "Read/Write data source"
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
	case RW:
		return ReadWriteDataSourceName
	}
	return UnknownDataSourceName
}

// DataSourceType is the type of data source.
type DataSourceType string

const (
	// Admin is the ADMIN type of data source.
	Admin DataSourceType = "ADMIN"
	// RW is the read/write type of data source.
	RW DataSourceType = "RW"
	// RO is the read-only type of data source.
	RO DataSourceType = "RO"
)

func (e DataSourceType) String() string {
	switch e {
	case Admin:
		return "ADMIN"
	case RW:
		return "RW"
	case RO:
		return "RO"
	}
	return ""
}

// DataSource is the API message for a data source.
type DataSource struct {
	ID int `jsonapi:"primary,dataSource"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	// Just returns InstanceID and DatabaseID otherwise would cause circular dependency.
	InstanceID int `jsonapi:"attr,instanceId"`
	DatabaseID int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name     string         `jsonapi:"attr,name"`
	Type     DataSourceType `jsonapi:"attr,type"`
	Username string         `jsonapi:"attr,username"`
	// Do not return the password to client
	Password string
	SslCa    string
	SslCert  string
	SslKey   string
}

// DataSourceCreate is the API message for creating a data source.
type DataSourceCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	InstanceID int `jsonapi:"attr,instanceId"`
	DatabaseID int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name     string         `jsonapi:"attr,name"`
	Type     DataSourceType `jsonapi:"attr,type"`
	Username string         `jsonapi:"attr,username"`
	Password string         `jsonapi:"attr,password"`
	SslCa    string         `jsonapi:"attr,sslCa"`
	SslCert  string         `jsonapi:"attr,sslCert"`
	SslKey   string         `jsonapi:"attr,sslKey"`
	// If true, syncs the schema after creating the data source. The client
	// may set to false if the target data source's instance contains too many databases
	// to avoid the request timeout.
	SyncSchema bool `jsonapi:"attr,syncSchema"`
}

// DataSourceFind is the API message for finding data sources.
type DataSourceFind struct {
	ID *int

	// Related fields
	InstanceID *int
	DatabaseID *int

	// Domain specific fields
	Type *DataSourceType
}

func (find *DataSourceFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// DataSourcePatch is the API message for data source.
type DataSourcePatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Username         *string `jsonapi:"attr,username"`
	Password         *string `jsonapi:"attr,password"`
	UseEmptyPassword *bool   `jsonapi:"attr,useEmptyPassword"`
	SslCa            *string `jsonapi:"attr,sslCa"`
	SslCert          *string `jsonapi:"attr,sslCert"`
	SslKey           *string `jsonapi:"attr,sslKey"`
	// If true, syncs the schema after patching the data source. The client
	// may set to false if the target data source's instance contains too many databases
	// to avoid the request timeout.
	SyncSchema bool `jsonapi:"attr,syncSchema"`
}
