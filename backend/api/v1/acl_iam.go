package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (in *ACLInterceptor) checkIAMPermission(ctx context.Context, fullMethod string, req any, user *store.UserMessage) error {
	p, ok := methodPermissionMap[fullMethod]
	if !ok {
		return nil
	}

	switch fullMethod {
	// skip checking for custom approval.
	case
		v1pb.IssueService_ApproveIssue_FullMethodName,
		v1pb.IssueService_RejectIssue_FullMethodName,
		v1pb.IssueService_RequestIssue_FullMethodName:
		return nil
	// handled in the method because checking is complex.
	case
		v1pb.DatabaseService_ListSlowQueries_FullMethodName,
		v1pb.DatabaseService_ListDatabases_FullMethodName,     // TODO(p0ny): implement
		v1pb.DatabaseService_DiffSchema_FullMethodName,        // TODO(p0ny): implement
		v1pb.IssueService_ListIssues_FullMethodName,           // TODO(p0ny): implement
		v1pb.ProjectService_ListDatabaseGroups_FullMethodName, // TODO(p0ny): implement
		v1pb.ProjectService_ListSchemaGroups_FullMethodName:   // TODO(p0ny): implement
		return nil
	// below are "workspace-level" permissions.
	// we don't have to go down to the project level.
	case
		v1pb.ProjectService_ListProjects_FullMethodName,
		v1pb.ProjectService_CreateProject_FullMethodName,
		v1pb.ProjectService_DeleteProject_FullMethodName,
		v1pb.ProjectService_UndeleteProject_FullMethodName,
		v1pb.InstanceService_ListInstances_FullMethodName,
		v1pb.InstanceService_GetInstance_FullMethodName,
		v1pb.InstanceService_CreateInstance_FullMethodName,
		v1pb.InstanceService_UpdateInstance_FullMethodName,
		v1pb.InstanceService_DeleteInstance_FullMethodName,
		v1pb.InstanceService_UndeleteInstance_FullMethodName,
		v1pb.InstanceService_SyncInstance_FullMethodName,
		v1pb.InstanceService_BatchSyncInstance_FullMethodName,
		v1pb.InstanceService_AddDataSource_FullMethodName,
		v1pb.InstanceService_RemoveDataSource_FullMethodName,
		v1pb.InstanceService_UpdateDataSource_FullMethodName,
		v1pb.InstanceService_SyncSlowQueries_FullMethodName,
		v1pb.EnvironmentService_CreateEnvironment_FullMethodName,
		v1pb.EnvironmentService_UpdateEnvironment_FullMethodName,
		v1pb.EnvironmentService_DeleteEnvironment_FullMethodName,
		v1pb.EnvironmentService_UndeleteEnvironment_FullMethodName,
		v1pb.EnvironmentService_GetEnvironment_FullMethodName,
		v1pb.EnvironmentService_ListEnvironments_FullMethodName,
		v1pb.EnvironmentService_UpdateBackupSetting_FullMethodName:
		ok, err := in.iamManager.CheckPermission(ctx, p, user)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to check permission for method %q, err: %v", fullMethod, err)
		}
		if !ok {
			return status.Errorf(codes.PermissionDenied, "permission denied for method %q, user does not have permission %q", fullMethod, p)
		}
	case
		v1pb.DatabaseService_GetDatabase_FullMethodName,
		v1pb.DatabaseService_UpdateDatabase_FullMethodName,
		v1pb.DatabaseService_BatchUpdateDatabases_FullMethodName,
		v1pb.DatabaseService_SyncDatabase_FullMethodName,
		v1pb.DatabaseService_GetDatabaseMetadata_FullMethodName,
		v1pb.DatabaseService_UpdateDatabaseMetadata_FullMethodName,
		v1pb.DatabaseService_GetDatabaseSchema_FullMethodName,
		v1pb.DatabaseService_GetBackupSetting_FullMethodName,
		v1pb.DatabaseService_UpdateBackupSetting_FullMethodName,
		v1pb.DatabaseService_CreateBackup_FullMethodName,
		v1pb.DatabaseService_ListBackups_FullMethodName,
		v1pb.DatabaseService_ListSecrets_FullMethodName,
		v1pb.DatabaseService_UpdateSecret_FullMethodName,
		v1pb.DatabaseService_DeleteSecret_FullMethodName,
		v1pb.DatabaseService_AdviseIndex_FullMethodName, // TODO(p0ny): implement.
		v1pb.DatabaseService_ListChangeHistories_FullMethodName,
		v1pb.DatabaseService_GetChangeHistory_FullMethodName:
		projectIDs, err := in.getProjectIDsForDatabaseService(ctx, req)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to check permission, err %v", err)
		}
		ok, err = in.iamManager.CheckPermission(ctx, p, user, projectIDs...)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to check permission for method %q, err: %v", fullMethod, err)
		}
		if !ok {
			return status.Errorf(codes.PermissionDenied, "permission denied for method %q, user does not have permission %q", fullMethod, p)
		}
	case
		v1pb.IssueService_GetIssue_FullMethodName,
		v1pb.IssueService_CreateIssue_FullMethodName,
		v1pb.IssueService_UpdateIssue_FullMethodName,
		v1pb.IssueService_CreateIssueComment_FullMethodName,
		v1pb.IssueService_UpdateIssueComment_FullMethodName,
		v1pb.IssueService_BatchUpdateIssuesStatus_FullMethodName:
		projectIDs, err := in.getProjectIDsForProjectService(ctx, req)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to check permission, err %v", err)
		}
		ok, err = in.iamManager.CheckPermission(ctx, p, user, projectIDs...)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to check permission for method %q, err: %v", fullMethod, err)
		}
		if !ok {
			return status.Errorf(codes.PermissionDenied, "permission denied for method %q, user does not have permission %q", fullMethod, p)
		}
	case
		v1pb.ProjectService_GetProject_FullMethodName,
		v1pb.ProjectService_UpdateProject_FullMethodName,
		v1pb.ProjectService_SetIamPolicy_FullMethodName,
		v1pb.ProjectService_GetIamPolicy_FullMethodName,
		v1pb.ProjectService_BatchGetIamPolicy_FullMethodName,
		v1pb.ProjectService_GetDeploymentConfig_FullMethodName,
		v1pb.ProjectService_UpdateDeploymentConfig_FullMethodName,
		v1pb.ProjectService_AddWebhook_FullMethodName,
		v1pb.ProjectService_UpdateWebhook_FullMethodName,
		v1pb.ProjectService_RemoveWebhook_FullMethodName,
		v1pb.ProjectService_TestWebhook_FullMethodName,
		v1pb.ProjectService_UpdateProjectGitOpsInfo_FullMethodName,
		v1pb.ProjectService_UnsetProjectGitOpsInfo_FullMethodName,
		v1pb.ProjectService_GetProjectGitOpsInfo_FullMethodName,
		v1pb.ProjectService_SetupProjectSQLReviewCI_FullMethodName,

		v1pb.ProjectService_GetDatabaseGroup_FullMethodName,
		v1pb.ProjectService_CreateDatabaseGroup_FullMethodName,
		v1pb.ProjectService_UpdateDatabaseGroup_FullMethodName,
		v1pb.ProjectService_DeleteDatabaseGroup_FullMethodName,

		v1pb.ProjectService_GetSchemaGroup_FullMethodName,
		v1pb.ProjectService_CreateSchemaGroup_FullMethodName,
		v1pb.ProjectService_UpdateSchemaGroup_FullMethodName,
		v1pb.ProjectService_DeleteSchemaGroup_FullMethodName,
		v1pb.ProjectService_GetProjectProtectionRules_FullMethodName,
		v1pb.ProjectService_UpdateProjectProtectionRules_FullMethodName:

		projectIDs, err := in.getProjectIDsForIssueService(ctx, req)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to check permission, err %v", err)
		}
		ok, err = in.iamManager.CheckPermission(ctx, p, user, projectIDs...)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to check permission for method %q, err: %v", fullMethod, err)
		}
		if !ok {
			return status.Errorf(codes.PermissionDenied, "permission denied for method %q, user does not have permission %q", fullMethod, p)
		}
	}

	return nil
}

func getDatabaseMessage(ctx context.Context, s *store.Store, databaseResourceName string) (*store.DatabaseMessage, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(databaseResourceName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %q", databaseResourceName)
	}
	find := &store.FindDatabaseMessage{
		ShowDeleted: true,
	}
	databaseUID, isNumber := isNumber(databaseName)
	if isNumber {
		// Expected format: "instances/{ignored_value}/database/{uid}"
		find.UID = &databaseUID
	} else {
		// Expected format: "instances/{instance}/database/{database}"
		find.InstanceID = &instanceID
		find.DatabaseName = &databaseName
		instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
		}
		if instance == nil {
			return nil, errors.Wrapf(err, "instance not found")
		}
		find.IgnoreCaseSensitive = store.IgnoreDatabaseAndTableCaseSensitive(instance)
	}
	database, err := s.GetDatabaseV2(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Wrapf(err, "database %q not found", databaseResourceName)
	}
	return database, nil
}

func (*ACLInterceptor) getProjectIDsForProjectService(_ context.Context, req any) ([]string, error) {
	var projects, projectDeploymentConfigs, projectWebhooks, projectGitopsInfos, databaseGroups, schemaGroups, protectionRules []string

	switch r := req.(type) {
	case *v1pb.GetProjectRequest:
		projects = append(projects, r.GetName())
	case *v1pb.UpdateProjectRequest:
		projects = append(projects, r.GetProject().GetName())
	case *v1pb.SetIamPolicyRequest:
		projects = append(projects, r.GetProject())
	case *v1pb.GetIamPolicyRequest:
		projects = append(projects, r.GetProject())
	case *v1pb.BatchGetIamPolicyRequest:
		projects = append(projects, r.GetNames()...)
	case *v1pb.GetDeploymentConfigRequest:
		projectDeploymentConfigs = append(projectDeploymentConfigs, r.GetName())
	case *v1pb.UpdateDeploymentConfigRequest:
		projectDeploymentConfigs = append(projectDeploymentConfigs, r.GetConfig().GetName())
	case *v1pb.AddWebhookRequest:
		projects = append(projects, r.GetProject())
	case *v1pb.UpdateWebhookRequest:
		projectWebhooks = append(projectWebhooks, r.GetWebhook().GetName())
	case *v1pb.RemoveWebhookRequest:
		projectWebhooks = append(projectWebhooks, r.GetWebhook().GetName())
	case *v1pb.TestWebhookRequest:
		projects = append(projects, r.GetProject())
	case *v1pb.UpdateProjectGitOpsInfoRequest:
		projectGitopsInfos = append(projectGitopsInfos, r.GetProjectGitopsInfo().GetName())
	case *v1pb.UnsetProjectGitOpsInfoRequest:
		projectGitopsInfos = append(projectGitopsInfos, r.GetName())
	case *v1pb.SetupSQLReviewCIRequest:
		projectGitopsInfos = append(projectGitopsInfos, r.GetName())
	case *v1pb.GetProjectGitOpsInfoRequest:
		projectGitopsInfos = append(projectGitopsInfos, r.GetName())
	case *v1pb.GetDatabaseGroupRequest:
		databaseGroups = append(databaseGroups, r.GetName())
	case *v1pb.CreateDatabaseGroupRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.UpdateDatabaseGroupRequest:
		databaseGroups = append(databaseGroups, r.GetDatabaseGroup().GetName())
	case *v1pb.DeleteDatabaseGroupRequest:
		databaseGroups = append(databaseGroups, r.GetName())
	case *v1pb.GetSchemaGroupRequest:
		schemaGroups = append(schemaGroups, r.GetName())
	case *v1pb.CreateSchemaGroupRequest:
		databaseGroups = append(databaseGroups, r.GetParent())
	case *v1pb.UpdateSchemaGroupRequest:
		schemaGroups = append(schemaGroups, r.GetSchemaGroup().GetName())
	case *v1pb.DeleteSchemaGroupRequest:
		schemaGroups = append(schemaGroups, r.GetName())
	case *v1pb.GetProjectProtectionRulesRequest:
		protectionRules = append(protectionRules, r.GetName())
	case *v1pb.UpdateProjectProtectionRulesRequest:
		protectionRules = append(protectionRules, r.GetProtectionRules().GetName())
	}

	var projectIDs []string

	for _, project := range projects {
		projectID, err := common.GetProjectID(project)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q", project)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, projectDeploymentConfig := range projectDeploymentConfigs {
		projectID, _, err := common.GetProjectIDDeploymentConfigID(projectDeploymentConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q", projectDeploymentConfig)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, projectWebhook := range projectWebhooks {
		projectID, _, err := common.GetProjectIDWebhookID(projectWebhook)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q", projectWebhook)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, projectGitopsInfo := range projectGitopsInfos {
		projectID, err := common.TrimSuffixAndGetProjectID(projectGitopsInfo, common.GitOpsInfoSuffix)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q", projectGitopsInfo)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, databaseGroup := range databaseGroups {
		projectID, _, err := common.GetProjectIDDatabaseGroupID(databaseGroup)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q", databaseGroup)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, schemaGroup := range schemaGroups {
		projectID, _, _, err := common.GetProjectIDDatabaseGroupIDSchemaGroupID(schemaGroup)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q", schemaGroup)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, protectionRule := range protectionRules {
		projectID, err := common.TrimSuffixAndGetProjectID(protectionRule, common.ProtectionRulesSuffix)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q", protectionRule)
		}
		projectIDs = append(projectIDs, projectID)
	}
	return projectIDs, nil
}

func (*ACLInterceptor) getProjectIDsForIssueService(_ context.Context, req any) ([]string, error) {
	var issueNames []string

	switch r := req.(type) {
	case *v1pb.GetIssueRequest:
		issueNames = append(issueNames, r.GetName())
	case *v1pb.CreateIssueRequest:
		issueNames = append(issueNames, r.GetIssue().GetName())
	case *v1pb.UpdateIssueRequest:
		issueNames = append(issueNames, r.GetIssue().GetName())
	case *v1pb.CreateIssueCommentRequest:
		issueNames = append(issueNames, r.GetParent())
	case *v1pb.UpdateIssueCommentRequest:
		issueNames = append(issueNames, r.GetParent())
	case *v1pb.BatchUpdateIssuesStatusRequest:
		issueNames = append(issueNames, r.GetIssues()...)
	}

	projectIDsMap := make(map[string]struct{})
	for _, issueName := range issueNames {
		projectID, _, err := common.GetProjectIDIssueID(issueName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get projectID from %q", issueName)
		}
		projectIDsMap[projectID] = struct{}{}
	}

	var projectIDs []string
	for projectID := range projectIDsMap {
		projectIDs = append(projectIDs, projectID)
	}
	return projectIDs, nil
}

func (in *ACLInterceptor) getProjectIDsForDatabaseService(ctx context.Context, req any) ([]string, error) {
	var projectIDs []string

	var databaseNames []string
	switch r := req.(type) {
	case *v1pb.GetDatabaseRequest:
		databaseNames = append(databaseNames, r.GetName())
	case *v1pb.SyncDatabaseRequest:
		databaseNames = append(databaseNames, r.GetName())
	case *v1pb.GetDatabaseMetadataRequest:
		databaseName, err := common.TrimSuffix(r.GetName(), "/metadata")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetName())
		}
		databaseNames = append(databaseNames, databaseName)
	case *v1pb.UpdateDatabaseMetadataRequest:
		databaseName, err := common.TrimSuffix(r.GetDatabaseMetadata().GetName(), "/metadata")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetDatabaseMetadata().GetName())
		}
		databaseNames = append(databaseNames, databaseName)
	case *v1pb.UpdateDatabaseRequest:
		databaseNames = append(databaseNames, r.GetDatabase().GetName())
		if hasPath(r.GetUpdateMask(), "project") {
			projectID, err := common.GetProjectID(r.GetDatabase().GetProject())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get projectID from %q", r.GetDatabase().GetProject())
			}
			projectIDs = append(projectIDs, projectID)
		}
	case *v1pb.BatchUpdateDatabasesRequest:
		for _, request := range r.Requests {
			databaseNames = append(databaseNames, request.GetDatabase().GetName())
			if hasPath(request.GetUpdateMask(), "project") {
				projectID, err := common.GetProjectID(request.GetDatabase().GetProject())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get projectID from %q", request.GetDatabase().GetProject())
				}
				projectIDs = append(projectIDs, projectID)
			}
		}
	case *v1pb.GetDatabaseSchemaRequest:
		databaseName, err := common.TrimSuffix(r.GetName(), "/schema")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetName())
		}
		databaseNames = append(databaseNames, databaseName)
	case *v1pb.GetBackupSettingRequest:
		databaseName, err := common.TrimSuffix(r.GetName(), "/backupSetting")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetName())
		}
		databaseNames = append(databaseNames, databaseName)
	case *v1pb.UpdateBackupSettingRequest:
		databaseName, err := common.TrimSuffix(r.GetSetting().GetName(), "/backupSetting")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetSetting().GetName())
		}
		databaseNames = append(databaseNames, databaseName)
	case *v1pb.CreateBackupRequest:
		databaseNames = append(databaseNames, r.GetParent())
	case *v1pb.ListBackupsRequest:
		databaseNames = append(databaseNames, r.GetParent())
	case *v1pb.ListSecretsRequest:
		databaseNames = append(databaseNames, r.GetParent())
	case *v1pb.UpdateSecretRequest:
		instance, database, _, err := common.GetInstanceDatabaseIDSecretName(r.GetSecret().GetName())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetSecret().GetName())
		}
		databaseNames = append(databaseNames, common.FormatDatabase(instance, database))
	case *v1pb.DeleteSecretRequest:
		instance, database, _, err := common.GetInstanceDatabaseIDSecretName(r.GetName())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetName())
		}
		databaseNames = append(databaseNames, common.FormatDatabase(instance, database))
	case *v1pb.ListChangeHistoriesRequest:
		databaseNames = append(databaseNames, r.GetParent())
	case *v1pb.GetChangeHistoryRequest:
		instance, database, _, err := common.GetInstanceDatabaseIDChangeHistory(r.GetName())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetName())
		}
		databaseNames = append(databaseNames, common.FormatDatabase(instance, database))
	}

	for _, databaseName := range databaseNames {
		database, err := getDatabaseMessage(ctx, in.store, databaseName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database %q", databaseName)
		}
		projectIDs = append(projectIDs, database.ProjectID)
	}

	return projectIDs, nil
}
