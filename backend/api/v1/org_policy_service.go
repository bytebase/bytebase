package v1

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisordb "github.com/bytebase/bytebase/backend/plugin/advisor/db"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
	policy, parent, err := s.findPolicyMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}

	response, err := convertToPolicy(parent, policy)
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

// CreatePolicy creates a policy in a specific resource.
func (s *OrgPolicyService) CreatePolicy(ctx context.Context, request *v1pb.CreatePolicyRequest) (*v1pb.Policy, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	if request.Policy == nil {
		return nil, status.Errorf(codes.InvalidArgument, "policy must be set")
	}

	// TODO(d): validate policy.
	return s.createPolicyMessage(ctx, principalID, request.Parent, request.Policy)
}

// UpdatePolicy updates a policy in a specific resource.
func (s *OrgPolicyService) UpdatePolicy(ctx context.Context, request *v1pb.UpdatePolicyRequest) (*v1pb.Policy, error) {
	if request.Policy == nil {
		return nil, status.Errorf(codes.InvalidArgument, "policy must be set")
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	policy, parent, err := s.findPolicyMessage(ctx, request.Policy.Name)
	if err != nil {
		st := status.Convert(err)
		if st.Code() == codes.NotFound && request.AllowMissing {
			return s.createPolicyMessage(ctx, principalID, parent, request.Policy)
		}
		return nil, err
	}

	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	patch := &store.UpdatePolicyMessage{
		UpdaterID:    principalID,
		ResourceType: policy.ResourceType,
		Type:         policy.Type,
		ResourceUID:  policy.ResourceUID,
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "policy.inherit_from_parent":
			patch.InheritFromParent = &request.Policy.InheritFromParent
		case "policy.payload":
			payloadStr, err := convertPolicyPayloadToString(request.Policy)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid policy %v", err.Error())
			}
			patch.Payload = &payloadStr
		case "policy.enforce":
			patch.Enforce = &request.Policy.Enforce
		}
	}

	p, err := s.store.UpdatePolicyV2(ctx, patch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	response, err := convertToPolicy(parent, p)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return response, nil
}

// DeletePolicy deletes a policy for a specific resource.
func (s *OrgPolicyService) DeletePolicy(ctx context.Context, request *v1pb.DeletePolicyRequest) (*emptypb.Empty, error) {
	policy, _, err := s.findPolicyMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}

	if err := s.store.DeletePolicyV2(ctx, policy); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

// findPolicyMessage finds the policy and the parent name by the policy name.
func (s *OrgPolicyService) findPolicyMessage(ctx context.Context, policyName string) (*store.PolicyMessage, string, error) {
	tokens := strings.Split(policyName, policyNamePrefix)
	if len(tokens) != 2 {
		return nil, "", status.Errorf(codes.InvalidArgument, "invalid request %s", policyName)
	}

	policyParent := tokens[0]
	if strings.HasSuffix(policyParent, "/") {
		policyParent = policyParent[:(len(policyParent) - 1)]
	}
	resourceType, resourceID, err := s.getPolicyResourceTypeAndID(ctx, policyParent)
	if err != nil {
		return nil, policyParent, err
	}

	policyType, err := convertPolicyType(tokens[1])
	if err != nil {
		return nil, policyParent, status.Errorf(codes.InvalidArgument, err.Error())
	}

	policy, err := s.store.GetPolicyV2(ctx, &store.FindPolicyMessage{
		ResourceType: &resourceType,
		Type:         &policyType,
		ResourceUID:  &resourceID,
	})
	if err != nil {
		return nil, policyParent, status.Errorf(codes.Internal, err.Error())
	}
	if policy == nil {
		return nil, policyParent, status.Errorf(codes.NotFound, "policy %q not found", policyName)
	}

	return policy, policyParent, nil
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

func (s *OrgPolicyService) createPolicyMessage(ctx context.Context, creatorID int, parent string, policy *v1pb.Policy) (*v1pb.Policy, error) {
	resourceType, resourceID, err := s.getPolicyResourceTypeAndID(ctx, parent)
	if err != nil {
		return nil, err
	}

	policyType, err := convertPolicyType(policy.Type.String())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	payloadStr, err := convertPolicyPayloadToString(policy)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid policy %v", err.Error())
	}

	p, err := s.store.CreatePolicyV2(ctx, &store.PolicyMessage{
		ResourceUID:       resourceID,
		ResourceType:      resourceType,
		Payload:           payloadStr,
		Type:              policyType,
		InheritFromParent: policy.InheritFromParent,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	}, creatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	response, err := convertToPolicy(parent, p)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return response, nil
}

func convertPolicyPayloadToString(policy *v1pb.Policy) (string, error) {
	switch policy.Type {
	case v1pb.PolicyType_DEPLOYMENT_APPROVAL:
		payload, err := convertToPipelineApprovalPolicyPayload(policy.GetDeploymentApprovalPolicy())
		if err != nil {
			return "", err
		}
		return payload.String()
	case v1pb.PolicyType_BACKUP_PLAN:
		payload, err := convertToBackupPlanPolicyPayload(policy.GetBackupPlanPolicy())
		if err != nil {
			return "", err
		}
		return payload.String()
	case v1pb.PolicyType_SQL_REVIEW:
		payload, err := convertToSQLReviewPolicyPayload(policy.GetSqlReviewPolicy())
		if err != nil {
			return "", err
		}
		return payload.String()
	case v1pb.PolicyType_SENSITIVE_DATA:
		payload, err := convertToSensitiveDataPolicyPayload(policy.GetSensitiveDataPolicy())
		if err != nil {
			return "", err
		}
		return payload.String()
	case v1pb.PolicyType_ACCESS_CONTROL:
		payload, err := convertToAccessControlPolicyPayload(policy.GetAccessControlPolicy())
		if err != nil {
			return "", err
		}
		return payload.String()
	}

	return "", errors.Errorf("invalid policy %v", policy.Type)
}

func convertToPolicy(prefix string, policyMessage *store.PolicyMessage) (*v1pb.Policy, error) {
	policy := &v1pb.Policy{
		Uid:               fmt.Sprintf("%d", policyMessage.UID),
		InheritFromParent: policyMessage.InheritFromParent,
		Enforce:           policyMessage.Enforce,
	}

	pType := v1pb.PolicyType_POLICY_TYPE_UNSPECIFIED
	switch policyMessage.Type {
	case api.PolicyTypePipelineApproval:
		pType = v1pb.PolicyType_DEPLOYMENT_APPROVAL
		payload, err := convertToV1PBDeploymentApprovalPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case api.PolicyTypeBackupPlan:
		pType = v1pb.PolicyType_BACKUP_PLAN
		payload, err := convertToV1PBBackupPlanPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case api.PolicyTypeSQLReview:
		pType = v1pb.PolicyType_SQL_REVIEW
		payload, err := convertToV1PBSQLReviewPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case api.PolicyTypeSensitiveData:
		pType = v1pb.PolicyType_SENSITIVE_DATA
		payload, err := convertToV1PBSensitiveDataPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case api.PolicyTypeAccessControl:
		pType = v1pb.PolicyType_ACCESS_CONTROL
		payload, err := convertToV1PBAccessControlPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	}

	policy.Type = pType
	policy.Name = fmt.Sprintf("%s%s", policyNamePrefix, strings.ToLower(pType.String()))
	if prefix != "" {
		policy.Name = fmt.Sprintf("%s/%s", prefix, policy.Name)
	}

	return policy, nil
}

func convertToV1PBSQLReviewPolicy(payloadStr string) (*v1pb.Policy_SqlReviewPolicy, error) {
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
			Engine:  convertToEngine(db.Type(rule.Engine)),
		})
	}

	return &v1pb.Policy_SqlReviewPolicy{
		SqlReviewPolicy: &v1pb.SQLReviewPolicy{
			Title: payload.Name,
			Rules: rules,
		},
	}, nil
}

func convertToSQLReviewPolicyPayload(policy *v1pb.SQLReviewPolicy) (*advisor.SQLReviewPolicy, error) {
	var ruleList []*advisor.SQLReviewRule
	for _, rule := range policy.Rules {
		var level advisor.SQLReviewRuleLevel
		switch rule.Level {
		case v1pb.SQLReviewRuleLevel_ERROR:
			level = advisor.SchemaRuleLevelError
		case v1pb.SQLReviewRuleLevel_WARNING:
			level = advisor.SchemaRuleLevelWarning
		case v1pb.SQLReviewRuleLevel_DISABLED:
			level = advisor.SchemaRuleLevelDisabled
		default:
			return nil, errors.Errorf("invalid rule level %v", rule.Level)
		}
		ruleList = append(ruleList, &advisor.SQLReviewRule{
			Level:   level,
			Payload: rule.Payload,
			Type:    advisor.SQLReviewRuleType(rule.Type),
			Engine:  advisordb.Type(convertEngine(rule.Engine)),
		})
	}

	return &advisor.SQLReviewPolicy{
		Name:     policy.Title,
		RuleList: ruleList,
	}, nil
}

func convertToV1PBAccessControlPolicy(payloadStr string) (*v1pb.Policy_AccessControlPolicy, error) {
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

func convertToAccessControlPolicyPayload(policy *v1pb.AccessControlPolicy) (*api.AccessControlPolicy, error) {
	var disallowRuleList []api.AccessControlRule
	for _, rule := range policy.DisallowRules {
		disallowRuleList = append(disallowRuleList, api.AccessControlRule{
			FullDatabase: rule.FullDatabase,
		})
	}

	return &api.AccessControlPolicy{
		DisallowRuleList: disallowRuleList,
	}, nil
}

func convertToV1PBSensitiveDataPolicy(payloadStr string) (*v1pb.Policy_SensitiveDataPolicy, error) {
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
			Schema:   data.Schema,
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

func convertToSensitiveDataPolicyPayload(policy *v1pb.SensitiveDataPolicy) (*api.SensitiveDataPolicy, error) {
	var sensitiveDataList []api.SensitiveData
	for _, data := range policy.SensitiveData {
		if data.MaskType != v1pb.SensitiveDataMaskType_DEFAULT {
			return nil, errors.Errorf("invalid sensitive data mask type %v", data.MaskType)
		}
		sensitiveDataList = append(sensitiveDataList, api.SensitiveData{
			Schema: data.Schema,
			Table:  data.Table,
			Column: data.Column,
			Type:   api.SensitiveDataMaskTypeDefault,
		})
	}
	return &api.SensitiveDataPolicy{
		SensitiveDataList: sensitiveDataList,
	}, nil
}

func convertToV1PBBackupPlanPolicy(payloadStr string) (*v1pb.Policy_BackupPlanPolicy, error) {
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

func convertToBackupPlanPolicyPayload(policy *v1pb.BackupPlanPolicy) (*api.BackupPlanPolicy, error) {
	var schedule api.BackupPlanPolicySchedule
	switch policy.Schedule {
	case v1pb.BackupPlanSchedule_UNSET:
		schedule = api.BackupPlanPolicyScheduleUnset
	case v1pb.BackupPlanSchedule_DAILY:
		schedule = api.BackupPlanPolicyScheduleDaily
	case v1pb.BackupPlanSchedule_WEEKLY:
		schedule = api.BackupPlanPolicyScheduleWeekly
	default:
		return nil, errors.Errorf("invalid backup plan schedule %v", policy.Schedule)
	}

	return &api.BackupPlanPolicy{
		Schedule:          schedule,
		RetentionPeriodTs: int(policy.RetentionDuration.Seconds),
	}, nil
}

func convertToV1PBDeploymentApprovalPolicy(payloadStr string) (*v1pb.Policy_DeploymentApprovalPolicy, error) {
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
			ApprovalGroup:  assigneeGroupValue,
			DeploymentType: convertIssueTypeToDeplymentType(group.IssueType),
			// TODO: support using different strategy for different assignee group.
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

func convertToPipelineApprovalPolicyPayload(policy *v1pb.DeploymentApprovalPolicy) (*api.PipelineApprovalPolicy, error) {
	var strategy api.PipelineApprovalValue
	switch policy.DefaultStrategy {
	case v1pb.ApprovalStrategy_MANUAL:
		strategy = api.PipelineApprovalValueManualAlways
	case v1pb.ApprovalStrategy_AUTOMATIC:
		strategy = api.PipelineApprovalValueManualNever
	default:
		return nil, errors.Errorf("invalid default strategy %v", policy.DefaultStrategy)
	}

	var assigneeGroupList []api.AssigneeGroup
	for _, group := range policy.DeploymentApprovalStrategies {
		var assigneeGroup api.AssigneeGroupValue
		switch group.ApprovalGroup {
		case v1pb.ApprovalGroup_APPROVAL_GROUP_PROJECT_OWNER:
			assigneeGroup = api.AssigneeGroupValueProjectOwner
		case v1pb.ApprovalGroup_APPROVAL_GROUP_DBA:
			assigneeGroup = api.AssigneeGroupValueWorkspaceOwnerOrDBA
		default:
			return nil, errors.Errorf("invalid assignee group %v", group.ApprovalGroup)
		}

		var issueType api.IssueType
		switch group.DeploymentType {
		case v1pb.DeploymentType_DATABASE_CREATE:
			issueType = api.IssueDatabaseCreate
		case v1pb.DeploymentType_DATABASE_DDL:
			issueType = api.IssueDatabaseSchemaUpdate
		case v1pb.DeploymentType_DATABASE_DDL_GHOST:
			issueType = api.IssueDatabaseSchemaUpdateGhost
		case v1pb.DeploymentType_DATABASE_DML:
			issueType = api.IssueDatabaseDataUpdate
		case v1pb.DeploymentType_DATABASE_RESTORE_PITR:
			issueType = api.IssueDatabaseRestorePITR
		default:
			return nil, errors.Errorf("invalid deployment type %v", group.DeploymentType)
		}

		assigneeGroupList = append(assigneeGroupList, api.AssigneeGroup{
			Value:     assigneeGroup,
			IssueType: issueType,
		})
	}

	return &api.PipelineApprovalPolicy{
		Value:             strategy,
		AssigneeGroupList: assigneeGroupList,
	}, nil
}

func convertIssueTypeToDeplymentType(issueType api.IssueType) v1pb.DeploymentType {
	res := v1pb.DeploymentType_DEPLOYMENT_TYPE_UNSPECIFIED
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
