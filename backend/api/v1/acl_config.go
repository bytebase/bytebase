package v1

import (
	"strings"

	"github.com/bytebase/bytebase/backend/component/iam"
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
	"EnvironmentService/UpdateBackupSetting": true,
	"InstanceService/CreateInstance":         true,
	"InstanceService/UpdateInstance":         true,
	"InstanceService/DeleteInstance":         true,
	"InstanceService/UndeleteInstance":       true,
	"InstanceService/AddDataSource":          true,
	"InstanceService/RemoveDataSource":       true,
	"InstanceService/UpdateDataSource":       true,
	"SubscriptionService/TrialSubscription":  true,
	"RiskService/CreateRisk":                 true,
	"RiskService/UpdateRisk":                 true,
	"RiskService/DeleteRisk":                 true,
	"SettingService/SetSetting":              true,
	"RoleService/CreateRole":                 true,
	"RoleService/UpdateRole":                 true,
	"RoleService/DeleteRole":                 true,
	"ActuatorService/UpdateActuatorInfo":     true,
	"ActuatorService/ListDebugLog":           true,
}

var projectOwnerMethods = map[string]bool{
	"ProjectService/UpdateProject":           true,
	"ProjectService/DeleteProject":           true,
	"ProjectService/UndeleteProject":         true,
	"ProjectService/SetIamPolicy":            true,
	"SubscriptionService/UpdateSubscription": true,
}

var transferDatabaseMethods = map[string]bool{
	"DatabaseService/UpdateDatabase":       true,
	"DatabaseService/BatchUpdateDatabases": true,
}

var methodPermissionMap = map[string]iam.Permission{
	"InstanceService/ListInstances":     iam.PermissionInstanceList,
	"InstanceService/GetInstance":       iam.PermissionInstanceGet,
	"InstanceService/CreateInstance":    iam.PermissionInstanceCreate,
	"InstanceService/UpdateInstance":    iam.PermissionInstanceUpdate,
	"InstanceService/DeleteInstance":    iam.PermissionInstanceDelete,
	"InstanceService/UndeleteInstance":  iam.PermissionInstanceUndelete,
	"InstanceService/SyncInstance":      iam.PermissionInstanceSync,
	"InstanceService/BatchSyncInstance": iam.PermissionInstanceSync,
	"InstanceService/AddDataSource":     iam.PermissionInstanceUpdate,
	"InstanceService/RemoveDataSource":  iam.PermissionInstanceUpdate,
	"InstanceService/UpdateDataSource":  iam.PermissionInstanceUpdate,
	"InstanceService/SyncSlowQueries":   iam.PermissionInstanceSync,

	"DatabaseService/GetDatabase":            iam.PermissionDatabasesGet,
	"DatabaseService/ListDatabases":          iam.PermissionDatabasesList,
	"DatabaseService/SearchDatabases":        iam.PermissionDatabasesList,
	"DatabaseService/UpdateDatabase":         iam.PermissionDatabasesUpdate,
	"DatabaseService/BatchUpdateDatabases":   iam.PermissionDatabasesUpdate,
	"DatabaseService/SyncDatabase":           iam.PermissionDatabasesSync,
	"DatabaseService/GetDatabaseMetadata":    iam.PermissionDatabasesGetMetadata,
	"DatabaseService/UpdateDatabaseMetadata": iam.PermissionDatabasesUpdateMetadata,
	"DatabaseService/GetDatabaseSchema":      iam.PermissionDatabasesGetSchema,
	"DatabaseService/DiffSchema":             "", // handled in the method.
	"DatabaseService/GetBackupSetting":       iam.PermissionDatabasesGetBackupSetting,
	"DatabaseService/UpdateBackupSetting":    iam.PermissionDatabasesUpdateBackupSetting,
	"DatabaseService/CreateBackup":           iam.PermissionBackupsCreate,
	"DatabaseService/ListBackups":            iam.PermissionBackupsList,
	"DatabaseService/ListSlowQueries":        "", // TODO(p0ny): slow query is somewhat related to the workspace slow query setting. Do this one together with the workspace slow query.
	"DatabaseService/ListSecrets":            iam.PermissionDatabaseSecretsList,
	"DatabaseService/UpdateSecret":           iam.PermissionDatabaseSecretsUpdate,
	"DatabaseService/DeleteSecret":           iam.PermissionDatabaseSecretsDelete,
	"DatabaseService/AdviseIndex":            "", // TODO(p0ny): not critical, implement later.
	"DatabaseService/ListChangeHistories":    iam.PermissionChangeHistoriesList,
	"DatabaseService/GetChangeHistory":       iam.PermissionChangeHistoriesGet,
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
