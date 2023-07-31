package api

import (
	"github.com/bytebase/bytebase/backend/plugin/advisor"
)

// SQLSyncSchema is the API message for sync schemas.
type SQLSyncSchema struct {
	InstanceID *int `jsonapi:"attr,instanceId"`
	DatabaseID *int `jsonapi:"attr,databaseId"`
}

// SQLExecute is the API message for execute SQL.
// We only support readonly / SELECT.
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
