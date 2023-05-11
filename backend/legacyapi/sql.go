package api

import (
	"context"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// ConnectionInfo is the API message for connection infos.
type ConnectionInfo struct {
	Engine           db.Type `jsonapi:"attr,engine"`
	Host             string  `jsonapi:"attr,host"`
	Port             string  `jsonapi:"attr,port"`
	Username         string  `jsonapi:"attr,username"`
	Password         string  `jsonapi:"attr,password"`
	UseEmptyPassword bool    `jsonapi:"attr,useEmptyPassword"`
	Database         string  `jsonapi:"attr,database"`
	InstanceID       *int    `jsonapi:"attr,instanceId"`
	SslCa            *string `jsonapi:"attr,sslCa"`
	SslCert          *string `jsonapi:"attr,sslCert"`
	SslKey           *string `jsonapi:"attr,sslKey"`
	// SRV is used for MongoDB only.
	SRV bool `jsonapi:"attr,srv"`
	// AuthenticationDatabase is used for MongoDB only.
	AuthenticationDatabase string `json:"authenticationDatabase" jsonapi:"attr,authenticationDatabase"`
	SID                    string `json:"sid" jsonapi:"attr,sid"`
	ServiceName            string `json:"serviceName" jsonapi:"attr,serviceName"`
	// SSH configuration.
	UseSSHConfig  bool   `jsonapi:"attr,useSSHConfig"`
	SSHHost       string `json:"sshHost" jsonapi:"attr,sshHost"`
	SSHPort       string `json:"sshPort" jsonapi:"attr,sshPort"`
	SSHUser       string `json:"sshUser" jsonapi:"attr,sshUser"`
	SSHPassword   string `json:"sshPassword" jsonapi:"attr,sshPassword"`
	SSHPrivateKey string `json:"sshPrivateKey" jsonapi:"attr,sshPrivateKey"`
}

// SQLSyncSchema is the API message for sync schemas.
type SQLSyncSchema struct {
	InstanceID *int `jsonapi:"attr,instanceId"`
	DatabaseID *int `jsonapi:"attr,databaseId"`
}

// SQLExecute is the API message for execute SQL.
// For now, we only support readonly / SELECT.
type SQLExecute struct {
	InstanceID int `jsonapi:"attr,instanceId"`
	// For engines such as MySQL, databaseName can be empty.
	DatabaseName string `jsonapi:"attr,databaseName"`
	Statement    string `jsonapi:"attr,statement"`
	// For now, Readonly must be true
	Readonly bool `jsonapi:"attr,readonly"`
	// The maximum row count returned, only applicable to SELECT query.
	// Not enforced if limit <= 0.
	Limit int `jsonapi:"attr,limit"`
	// ExportFomat includes QUERY, CSV, JSON.
	// QUERY is used for querying database. CSV and JSON are the formats used for exporting data.
	ExportFormat string `jsonapi:"attr,exportFormat"`
}

// SingleSQLResult is the API message for single SQL result.
type SingleSQLResult struct {
	// A list of rows marshalled into a JSON.
	Data string `jsonapi:"attr,data" json:"data"`
	// SQL operation may fail for connection issue and there is no proper http status code for it, so we return error in the response body.
	Error string `jsonapi:"attr,error" json:"error"`
}

// SQLResultSet is the API message for SQL results.
type SQLResultSet struct {
	// A list of SQL results.
	SingleSQLResultList []SingleSQLResult `jsonapi:"attr,singleSQLResultList"`
	// Error of the whole SQL execution.
	Error string `jsonapi:"attr,error"`
	// A list of SQL check advice.
	AdviceList []advisor.Advice `jsonapi:"attr,adviceList"`
}

// SQLService is the service for SQL.
type SQLService interface {
	Ping(ctx context.Context, config *ConnectionInfo) (*SQLResultSet, error)
}
