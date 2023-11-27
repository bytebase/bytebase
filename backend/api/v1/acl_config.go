package v1

import (
	"strings"

	"github.com/bytebase/bytebase/backend/component/iam"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

const (
	apiPackagePrefix = "/bytebase.v1."
)

var ownerAndDBAMethods = map[string]bool{
	v1pb.EnvironmentService_CreateEnvironment_FullMethodName:   true,
	v1pb.EnvironmentService_UpdateEnvironment_FullMethodName:   true,
	v1pb.EnvironmentService_DeleteEnvironment_FullMethodName:   true,
	v1pb.EnvironmentService_UndeleteEnvironment_FullMethodName: true,
	v1pb.EnvironmentService_UpdateBackupSetting_FullMethodName: true,
	v1pb.InstanceService_CreateInstance_FullMethodName:         true,
	v1pb.InstanceService_UpdateInstance_FullMethodName:         true,
	v1pb.InstanceService_DeleteInstance_FullMethodName:         true,
	v1pb.InstanceService_UndeleteInstance_FullMethodName:       true,
	v1pb.InstanceService_AddDataSource_FullMethodName:          true,
	v1pb.InstanceService_RemoveDataSource_FullMethodName:       true,
	v1pb.InstanceService_UpdateDataSource_FullMethodName:       true,
	v1pb.SubscriptionService_TrialSubscription_FullMethodName:  true,
	v1pb.RiskService_CreateRisk_FullMethodName:                 true,
	v1pb.RiskService_UpdateRisk_FullMethodName:                 true,
	v1pb.RiskService_DeleteRisk_FullMethodName:                 true,
	v1pb.SettingService_SetSetting_FullMethodName:              true,
	v1pb.RoleService_CreateRole_FullMethodName:                 true,
	v1pb.RoleService_UpdateRole_FullMethodName:                 true,
	v1pb.RoleService_DeleteRole_FullMethodName:                 true,
	v1pb.ActuatorService_UpdateActuatorInfo_FullMethodName:     true,
	v1pb.ActuatorService_ListDebugLog_FullMethodName:           true,
}

var projectOwnerMethods = map[string]bool{
	v1pb.ProjectService_UpdateProject_FullMethodName:           true,
	v1pb.ProjectService_DeleteProject_FullMethodName:           true,
	v1pb.ProjectService_UndeleteProject_FullMethodName:         true,
	v1pb.ProjectService_SetIamPolicy_FullMethodName:            true,
	v1pb.SubscriptionService_UpdateSubscription_FullMethodName: true,
}

var transferDatabaseMethods = map[string]bool{
	v1pb.DatabaseService_UpdateDatabase_FullMethodName:       true,
	v1pb.DatabaseService_BatchUpdateDatabases_FullMethodName: true,
}

var methodPermissionMap = map[string]iam.Permission{
	"InstanceService/ListInstance":      iam.PermissionInstanceList,
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
