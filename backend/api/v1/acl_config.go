package v1

import (
	"strings"

	api "github.com/bytebase/bytebase/backend/legacyapi"
)

const (
	apiPackagePrefix = "/bytebase.v1."
)

var ownerAndDBAMethods = map[string]bool{
	"EnvironmentService/CreateEnvironment":   true,
	"EnvironmentService/UpdateEnvironment":   true,
	"EnvironmentService/DeleteEnvironment":   true,
	"EnvironmentService/UndeleteEnvironment": true,
	"InstanceService/CreateInstance":         true,
	"InstanceService/UpdateInstance":         true,
	"InstanceService/DeleteInstance":         true,
	"InstanceService/UndeleteInstance":       true,
	"InstanceService/AddDataSource":          true,
	"InstanceService/RemoveDataSource":       true,
	"InstanceService/UpdateDataSource":       true,
}

var projectOwnerMethods = map[string]bool{
	"ProjectService/UpdateProject":   true,
	"ProjectService/DeleteProject":   true,
	"ProjectService/UndeleteProject": true,
}

var transferDatabaseMethods = map[string]bool{
	"DatabaseService/UpdateDatabase":       true,
	"DatabaseService/BatchUpdateDatabases": true,
}

func isOwnerAndDBAMethod(methodName string) bool {
	return ownerAndDBAMethods[methodName]
}

func isProjectOwnerMethod(methodName string) bool {
	return projectOwnerMethods[methodName]
}

func isTransferDatabaseMethods(methodName string) bool {
	return transferDatabaseMethods[methodName]
}

// getShortMethodName gets the short method name from v1 API.
func getShortMethodName(fullMethod string) string {
	return strings.TrimPrefix(fullMethod, apiPackagePrefix)
}

func isOwnerOrDBA(role api.Role) bool {
	return role == api.Owner || role == api.DBA
}
