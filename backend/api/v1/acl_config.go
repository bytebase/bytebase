package v1

import (
	"github.com/bytebase/bytebase/backend/component/iam"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
	v1pb.InstanceService_ListInstances_FullMethodName:     iam.PermissionInstancesList,
	v1pb.InstanceService_GetInstance_FullMethodName:       iam.PermissionInstancesGet,
	v1pb.InstanceService_CreateInstance_FullMethodName:    iam.PermissionInstancesCreate,
	v1pb.InstanceService_UpdateInstance_FullMethodName:    iam.PermissionInstancesUpdate,
	v1pb.InstanceService_DeleteInstance_FullMethodName:    iam.PermissionInstancesDelete,
	v1pb.InstanceService_UndeleteInstance_FullMethodName:  iam.PermissionInstancesUndelete,
	v1pb.InstanceService_SyncInstance_FullMethodName:      iam.PermissionInstancesSync,
	v1pb.InstanceService_BatchSyncInstance_FullMethodName: iam.PermissionInstancesSync,
	v1pb.InstanceService_AddDataSource_FullMethodName:     iam.PermissionInstancesUpdate,
	v1pb.InstanceService_RemoveDataSource_FullMethodName:  iam.PermissionInstancesUpdate,
	v1pb.InstanceService_UpdateDataSource_FullMethodName:  iam.PermissionInstancesUpdate,
	v1pb.InstanceService_SyncSlowQueries_FullMethodName:   iam.PermissionInstancesSync,

	v1pb.DatabaseService_GetDatabase_FullMethodName:            iam.PermissionDatabasesGet,
	v1pb.DatabaseService_ListDatabases_FullMethodName:          iam.PermissionDatabasesList,
	v1pb.DatabaseService_UpdateDatabase_FullMethodName:         iam.PermissionDatabasesUpdate,
	v1pb.DatabaseService_BatchUpdateDatabases_FullMethodName:   iam.PermissionDatabasesUpdate,
	v1pb.DatabaseService_SyncDatabase_FullMethodName:           iam.PermissionDatabasesSync,
	v1pb.DatabaseService_GetDatabaseMetadata_FullMethodName:    iam.PermissionDatabasesGetMetadata,
	v1pb.DatabaseService_UpdateDatabaseMetadata_FullMethodName: iam.PermissionDatabasesUpdateMetadata,
	v1pb.DatabaseService_GetDatabaseSchema_FullMethodName:      iam.PermissionDatabasesGetSchema,
	v1pb.DatabaseService_DiffSchema_FullMethodName:             "", // handled in the method.
	v1pb.DatabaseService_GetBackupSetting_FullMethodName:       iam.PermissionDatabasesGetBackupSetting,
	v1pb.DatabaseService_UpdateBackupSetting_FullMethodName:    iam.PermissionDatabasesUpdateBackupSetting,
	v1pb.DatabaseService_CreateBackup_FullMethodName:           iam.PermissionBackupsCreate,
	v1pb.DatabaseService_ListBackups_FullMethodName:            iam.PermissionBackupsList,
	v1pb.DatabaseService_ListSlowQueries_FullMethodName:        iam.PermissionSlowQueriesList,
	v1pb.DatabaseService_ListSecrets_FullMethodName:            iam.PermissionDatabaseSecretsList,
	v1pb.DatabaseService_UpdateSecret_FullMethodName:           iam.PermissionDatabaseSecretsUpdate,
	v1pb.DatabaseService_DeleteSecret_FullMethodName:           iam.PermissionDatabaseSecretsDelete,
	v1pb.DatabaseService_AdviseIndex_FullMethodName:            "", // TODO(p0ny): not critical, implement later.
	v1pb.DatabaseService_ListChangeHistories_FullMethodName:    iam.PermissionChangeHistoriesList,
	v1pb.DatabaseService_GetChangeHistory_FullMethodName:       iam.PermissionChangeHistoriesGet,
	v1pb.EnvironmentService_CreateEnvironment_FullMethodName:   iam.PermissionEnvironmentsCreate,
	v1pb.EnvironmentService_UpdateEnvironment_FullMethodName:   iam.PermissionEnvironmentsUpdate,
	v1pb.EnvironmentService_DeleteEnvironment_FullMethodName:   iam.PermissionEnvironmentsDelete,
	v1pb.EnvironmentService_UndeleteEnvironment_FullMethodName: iam.PermissionEnvironmentsUndelete,
	v1pb.EnvironmentService_GetEnvironment_FullMethodName:      iam.PermissionEnvironmentsGet,
	v1pb.EnvironmentService_ListEnvironments_FullMethodName:    iam.PermissionEnvironmentsList,
	v1pb.EnvironmentService_UpdateBackupSetting_FullMethodName: iam.PermissionEnvironmentsUpdate,
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

func isOwnerOrDBA(role api.Role) bool {
	return role == api.Owner || role == api.DBA
}
