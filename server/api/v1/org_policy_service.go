package v1

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/plugin/advisor"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/store"
)

// OrgPolicyService implements the workspace policy service.
type OrgPolicyService struct {
	v1pb.UnimplementedOrgPolicyServiceServer
	store          *store.Store
	licenseService enterpriseAPI.LicenseService
}

// NewOrgPolicyService creates a new OrgPolicyService.
func NewOrgPolicyService(store *store.Store, licenseService enterpriseAPI.LicenseService) *OrgPolicyService {
	return &OrgPolicyService{
		store:          store,
		licenseService: licenseService,
	}
}

// GetPolicy gets a policy in a specific resource.
func (s *OrgPolicyService) GetPolicy(ctx context.Context, request *v1pb.GetPolicyRequest) (*v1pb.Policy, error) {
	tokens := strings.Split(request.Name, policyNamePrefix)
	if len(tokens) != 2 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request %s", request.Name)
	}

	token := tokens[0]
	if strings.HasSuffix(token, "/") {
		token = token[:(len(token) - 1)]
	}
	resourceType, resourceID, err := s.getPolicyResourceTypeAndID(ctx, token)
	if err != nil {
		return nil, err
	}

	policyType, err := convertPolicyType(tokens[1])
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	policy, err := s.store.GetPolicyV2(ctx, &store.FindPolicyMessage{
		ResourceType: &resourceType,
		Type:         &policyType,
		ResourceUID:  &resourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if policy == nil {
		return nil, status.Errorf(codes.NotFound, "policy %q not found", request.Name)
	}

	response, err := convertToPolicy(token, policy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return response, nil
}

// ListPolicies lists policies in a specific resource.
func (s *OrgPolicyService) ListPolicies(ctx context.Context, request *v1pb.ListPoliciesRequest) (*v1pb.ListPoliciesResponse, error) {
	resourceType, resourceID, err := s.getPolicyResourceTypeAndID(ctx, request.Parent)
	if err != nil {
		return nil, err
	}

	policies, err := s.store.ListPoliciesV2(ctx, &store.FindPolicyMessage{
		ResourceType: &resourceType,
		ResourceUID:  &resourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	response := &v1pb.ListPoliciesResponse{}
	for _, policy := range policies {
		p, err := convertToPolicy(request.Parent, policy)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		if p.Type == v1pb.PolicyType_POLICY_TYPE_UNSPECIFIED {
			// skip unknown type policy and environment tier policy
			continue
		}
		response.Policies = append(response.Policies, p)
	}
	return response, nil
}

func (s *OrgPolicyService) getPolicyResourceTypeAndID(ctx context.Context, requestName string) (api.PolicyResourceType, int, error) {
	if requestName == "" {
		return api.PolicyResourceTypeWorkspace, 0, nil
	}

	if strings.HasPrefix(requestName, projectNamePrefix) {
		projectID, err := getProjectID(requestName)
		if err != nil {
			return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, err.Error())
		}

		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID:  &projectID,
			ShowDeleted: true,
		})
		if err != nil {
			return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.Internal, err.Error())
		}
		if project == nil {
			return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.NotFound, "project %q not found", projectID)
		}
		if project.Deleted {
			return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, "project %q has been deleted", projectID)
		}

		return api.PolicyResourceTypeProject, project.UID, nil
	}

	if strings.HasPrefix(requestName, environmentNamePrefix) {
		sections := strings.Split(requestName, "/")

		// environment policy request name should be environments/{environment id}
		if len(sections) == 2 {
			environmentID, err := getEnvironmentID(requestName)
			if err != nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, err.Error())
			}
			environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
				ResourceID: &environmentID,
			})
			if err != nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.Internal, err.Error())
			}
			if environment == nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.NotFound, "environment %q not found", environmentID)
			}
			if environment.Deleted {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, "environment %q has been deleted", environmentID)
			}

			return api.PolicyResourceTypeEnvironment, environment.UID, nil
		}

		// instance policy request name should be environments/{environment id}/instances/{instance id}
		if len(sections) == 4 {
			environmentID, instanceID, err := getEnvironmentInstanceID(requestName)
			if err != nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, err.Error())
			}

			instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
				EnvironmentID: &environmentID,
				ResourceID:    &instanceID,
				ShowDeleted:   true,
			})
			if err != nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.Internal, err.Error())
			}
			if instance == nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
			}
			if instance.Deleted {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, "instance %q has been deleted", instanceID)
			}

			return api.PolicyResourceTypeInstance, instance.UID, nil
		}

		// database policy request name should be environments/{environment id}/instances/{instance id}/databases/{db name}
		if len(sections) == 6 {
			environmentID, instanceID, databaseName, err := getEnvironmentInstanceDatabaseID(requestName)
			if err != nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, err.Error())
			}
			database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				EnvironmentID: &environmentID,
				InstanceID:    &instanceID,
				DatabaseName:  &databaseName,
			})
			if err != nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.Internal, err.Error())
			}
			if database == nil {
				return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.NotFound, "database %q not found", databaseName)
			}

			return api.PolicyResourceTypeDatabase, database.UID, nil
		}
	}

	return api.PolicyResourceTypeUnknown, 0, status.Errorf(codes.InvalidArgument, "unknown request name %s", requestName)
}

func convertToPolicy(prefix string, policyMessage *store.PolicyMessage) (*v1pb.Policy, error) {
	policy := &v1pb.Policy{
		Uid:               fmt.Sprintf("%d", policyMessage.UID),
		State:             convertDeletedToState(policyMessage.Deleted),
		InheritFromParent: policyMessage.InheritFromParent,
	}

	pType := v1pb.PolicyType_POLICY_TYPE_UNSPECIFIED
	switch policyMessage.Type {
	case api.PolicyTypePipelineApproval:
		pType = v1pb.PolicyType_DEPLOYMENT_APPROVAL
		payload, err := convertPipelineApprovalPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case api.PolicyTypeBackupPlan:
		pType = v1pb.PolicyType_BACKUP_PLAN
		payload, err := convertBackupPlanPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case api.PolicyTypeSQLReview:
		pType = v1pb.PolicyType_SQL_REVIEW
		payload, err := convertSQLReviewPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case api.PolicyTypeSensitiveData:
		pType = v1pb.PolicyType_SENSITIVE_DATA
		payload, err := convertSensitiveDataPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case api.PolicyTypeAccessControl:
		pType = v1pb.PolicyType_ACCESS_CONTROL
		payload, err := convertAccessControlPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	}

	policy.Type = pType
	policy.Name = fmt.Sprintf("%s/%s%s", prefix, policyNamePrefix, strings.ToLower(pType.String()))

	return policy, nil
}

func convertSQLReviewPolicy(payloadStr string) (*v1pb.Policy_SqlReviewPolicy, error) {
	payload, err := api.UnmarshalSQLReviewPolicy(payloadStr)
	if err != nil {
		return nil, err
	}

	var rules []*v1pb.SQLReviewRule
	for _, rule := range payload.RuleList {
		level := v1pb.SQLReviewRuleLevel_LEVEL_UNSPECIFIED
		switch rule.Level {
		case advisor.SchemaRuleLevelError:
			level = v1pb.SQLReviewRuleLevel_ERROR
		case advisor.SchemaRuleLevelWarning:
			level = v1pb.SQLReviewRuleLevel_WARNING
		case advisor.SchemaRuleLevelDisabled:
			level = v1pb.SQLReviewRuleLevel_DISABLED
		}
		rules = append(rules, &v1pb.SQLReviewRule{
			Level:   level,
			Type:    string(rule.Type),
			Payload: rule.Payload,
		})
	}

	return &v1pb.Policy_SqlReviewPolicy{
		SqlReviewPolicy: &v1pb.SQLReviewPolicy{
			Title: payload.Name,
			Rules: rules,
		},
	}, nil
}

func convertAccessControlPolicy(payloadStr string) (*v1pb.Policy_AccessControlPolicy, error) {
	payload, err := api.UnmarshalAccessControlPolicy(payloadStr)
	if err != nil {
		return nil, err
	}

	var disallowRules []*v1pb.AccessControlRule
	for _, rule := range payload.DisallowRuleList {
		disallowRules = append(disallowRules, &v1pb.AccessControlRule{
			FullDatabase: rule.FullDatabase,
		})
	}
	return &v1pb.Policy_AccessControlPolicy{
		AccessControlPolicy: &v1pb.AccessControlPolicy{
			DisallowRules: disallowRules,
		},
	}, nil
}

func convertSensitiveDataPolicy(payloadStr string) (*v1pb.Policy_SensitiveDataPolicy, error) {
	payload, err := api.UnmarshalSensitiveDataPolicy(payloadStr)
	if err != nil {
		return nil, err
	}

	var sensitiveDataList []*v1pb.SensitiveData
	for _, data := range payload.SensitiveDataList {
		maskType := v1pb.SensitiveDataMaskType_MASK_TYPE_UNSPECIFIED
		if data.Type == api.SensitiveDataMaskTypeDefault {
			maskType = v1pb.SensitiveDataMaskType_DEFAULT
		}
		sensitiveDataList = append(sensitiveDataList, &v1pb.SensitiveData{
			Table:    data.Table,
			Column:   data.Column,
			MaskType: maskType,
		})
	}

	return &v1pb.Policy_SensitiveDataPolicy{
		SensitiveDataPolicy: &v1pb.SensitiveDataPolicy{
			SensitiveData: sensitiveDataList,
		},
	}, nil
}

func convertBackupPlanPolicy(payloadStr string) (*v1pb.Policy_BackupPlanPolicy, error) {
	payload, err := api.UnmarshalBackupPlanPolicy(payloadStr)
	if err != nil {
		return nil, err
	}

	schedule := v1pb.BackupPlanSchedule_SCHEDULE_UNSPECIFIED
	switch payload.Schedule {
	case api.BackupPlanPolicyScheduleUnset:
		schedule = v1pb.BackupPlanSchedule_UNSET
	case api.BackupPlanPolicyScheduleDaily:
		schedule = v1pb.BackupPlanSchedule_DAILY
	case api.BackupPlanPolicyScheduleWeekly:
		schedule = v1pb.BackupPlanSchedule_WEEKLY
	}

	return &v1pb.Policy_BackupPlanPolicy{
		BackupPlanPolicy: &v1pb.BackupPlanPolicy{
			Schedule:          schedule,
			RetentionDuration: &durationpb.Duration{Seconds: int64(payload.RetentionPeriodTs)},
		},
	}, nil
}

func convertPipelineApprovalPolicy(payloadStr string) (*v1pb.Policy_DeploymentApprovalPolicy, error) {
	payload, err := api.UnmarshalPipelineApprovalPolicy(payloadStr)
	if err != nil {
		return nil, err
	}

	approvalStrategy := v1pb.ApprovalStrategy_APPROVAL_STRATEGY_UNSPECIFIED
	switch payload.Value {
	case api.PipelineApprovalValueManualAlways:
		approvalStrategy = v1pb.ApprovalStrategy_MANUAL
	case api.PipelineApprovalValueManualNever:
		approvalStrategy = v1pb.ApprovalStrategy_AUTOMATIC
	}

	approvalStrategies := make([]*v1pb.DeploymentApprovalStrategy, 0)
	for _, group := range payload.AssigneeGroupList {
		assigneeGroupValue := v1pb.ApprovalGroup_ASSIGNEE_GROUP_UNSPECIFIED
		switch group.Value {
		case api.AssigneeGroupValueProjectOwner:
			assigneeGroupValue = v1pb.ApprovalGroup_APPROVAL_GROUP_PROJECT_OWNER
		case api.AssigneeGroupValueWorkspaceOwnerOrDBA:
			assigneeGroupValue = v1pb.ApprovalGroup_APPROVAL_GROUP_DBA
		}

		approvalStrategies = append(approvalStrategies, &v1pb.DeploymentApprovalStrategy{
			ApprovalGroup:    assigneeGroupValue,
			DeploymentType:   convertIssueTypeToDeplymentType(group.IssueType),
			ApprovalStrategy: approvalStrategy,
		})
	}

	return &v1pb.Policy_DeploymentApprovalPolicy{
		DeploymentApprovalPolicy: &v1pb.DeploymentApprovalPolicy{
			DefaultStrategy:              approvalStrategy,
			DeploymentApprovalStrategies: approvalStrategies,
		},
	}, nil
}

func convertIssueTypeToDeplymentType(issueType api.IssueType) v1pb.DeploymentType {
	res := v1pb.DeploymentType_ISSUE_TYPE_UNSPECIFIED
	switch issueType {
	case api.IssueDatabaseCreate:
		res = v1pb.DeploymentType_DATABASE_CREATE
	case api.IssueDatabaseSchemaUpdate:
		res = v1pb.DeploymentType_DATABASE_DDL
	case api.IssueDatabaseSchemaUpdateGhost:
		res = v1pb.DeploymentType_DATABASE_DDL_GHOST
	case api.IssueDatabaseDataUpdate:
		res = v1pb.DeploymentType_DATABASE_DML
	case api.IssueDatabaseRestorePITR:
		res = v1pb.DeploymentType_DATABASE_RESTORE_PITR
	case api.IssueDatabaseRollback:
		res = v1pb.DeploymentType_DATABASE_DML_ROLLBACK
	}

	return res
}

func convertPolicyType(pType string) (api.PolicyType, error) {
	var policyType api.PolicyType
	switch strings.ToUpper(pType) {
	case v1pb.PolicyType_DEPLOYMENT_APPROVAL.String():
		return api.PolicyTypePipelineApproval, nil
	case v1pb.PolicyType_BACKUP_PLAN.String():
		return api.PolicyTypeBackupPlan, nil
	case v1pb.PolicyType_SQL_REVIEW.String():
		return api.PolicyTypeSQLReview, nil
	case v1pb.PolicyType_SENSITIVE_DATA.String():
		return api.PolicyTypeSensitiveData, nil
	case v1pb.PolicyType_ACCESS_CONTROL.String():
		return api.PolicyTypeAccessControl, nil
	}
	return policyType, errors.Errorf("invalid policy type %v", pType)
}
