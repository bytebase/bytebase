package api

import (
	"context"

	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/db"
)

// ConnectionInfo is the API message for connection infos.
type ConnectionInfo struct {
	Engine           db.Type `jsonapi:"attr,engine"`
	Host             string  `jsonapi:"attr,host"`
	Port             string  `jsonapi:"attr,port"`
	Username         string  `jsonapi:"attr,username"`
	Password         string  `jsonapi:"attr,password"`
	UseEmptyPassword bool    `jsonapi:"attr,useEmptyPassword"`
	InstanceID       *int    `jsonapi:"attr,instanceId"`
	SslCa            *string `jsonapi:"attr,sslCa"`
	SslCert          *string `jsonapi:"attr,sslCert"`
	SslKey           *string `jsonapi:"attr,sslKey"`
}

// SQLSyncSchema is the API message for sync schemas.
type SQLSyncSchema struct {
	InstanceID int `jsonapi:"attr,instanceId"`
}

// SQLExecute is the API message for execute SQL.
// For now, we only support readonly / SELECT.
type SQLExecute struct {
	InstanceID int `jsonapi:"attr,instanceId"`
	// For engines like MySQL, databaseName can be empty.
	DatabaseName string `jsonapi:"attr,databaseName"`
	Statement    string `jsonapi:"attr,statement"`
	// For now, Readonly must be true
	Readonly bool `jsonapi:"attr,readonly"`
	// The maximum row count returned, only applicable to SELECT query.
	// Not enforced if limit <= 0.
	Limit int `jsonapi:"attr,limit"`
}

// SQLResultSet is the API message for SQL results.
type SQLResultSet struct {
	// A list of rows marshalled into a JSON.
	Data string `jsonapi:"attr,data"`
	// SQL operation may fail for connection issue and there is no proper http status code for it, so we return error in the response body.
	Error string `jsonapi:"attr,error"`
	// A list of SQL check advice.
	AdviceList []advisor.Advice `jsonapi:"attr,adviceList"`
}

// SQLService is the service for SQL.
type SQLService interface {
	Ping(ctx context.Context, config *ConnectionInfo) (*SQLResultSet, error)
}
