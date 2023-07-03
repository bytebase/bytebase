package v1

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	// defaultWorkspaceResourceID is a placeholder for resource id in workspace level IAM policy.
	defaultWorkspaceResourceID = 1
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

	find := &store.FindPolicyMessage{
		ResourceType: &resourceType,
		ResourceUID:  resourceID,
		ShowDeleted:  request.ShowDeleted,
	}

	if v := request.PolicyType; v != nil {
		policyType, err := convertPolicyType(v.String())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		find.Type = &policyType
	}

	policies, err := s.store.ListPoliciesV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	response := &v1pb.ListPoliciesResponse{}
	for _, policy := range policies {
		parentPath, err := s.getPolicyParentPath(ctx, policy)
		if err != nil {
			st := status.Convert(err)
			if st.Code() == codes.NotFound {
				log.Debug("failed to found resource for policy", zap.Error(err), zap.String("resource_type", string(policy.ResourceType)), zap.Int("resource_id", policy.ResourceUID))
				continue
			}
			return nil, err
		}
		p, err := convertToPolicy(parentPath, policy)
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
		case "inherit_from_parent":
			patch.InheritFromParent = &request.Policy.InheritFromParent
		case "payload":
			payloadStr, err := s.convertPolicyPayloadToString(request.Policy)
			if err != nil {
				return nil, err
			}
			patch.Payload = &payloadStr
		case "enforce":
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
	if resourceID == nil {
		return nil, policyParent, status.Errorf(codes.InvalidArgument, "resource id for %s must be specific", resourceType)
	}

	policyType, err := convertPolicyType(tokens[1])
	if err != nil {
		return nil, policyParent, status.Errorf(codes.InvalidArgument, err.Error())
	}

	policy, err := s.store.GetPolicyV2(ctx, &store.FindPolicyMessage{
		ResourceType: &resourceType,
		Type:         &policyType,
		ResourceUID:  resourceID,
	})
	if err != nil {
		return nil, policyParent, status.Errorf(codes.Internal, err.Error())
	}
	if policy == nil {
		return nil, policyParent, status.Errorf(codes.NotFound, "policy %q not found", policyName)
	}

	return policy, policyParent, nil
}

func (s *OrgPolicyService) getPolicyResourceTypeAndID(ctx context.Context, requestName string) (api.PolicyResourceType, *int, error) {
	if requestName == "" {
		return api.PolicyResourceTypeWorkspace, &defaultWorkspaceResourceID, nil
	}

	if strings.HasPrefix(requestName, projectNamePrefix) {
		projectID, err := getProjectID(requestName)
		if err != nil {
			return api.PolicyResourceTypeUnknown, nil, status.Errorf(codes.InvalidArgument, err.Error())
		}

		project, err := s.findActiveProject(ctx, &store.FindProjectMessage{
			ResourceID: &projectID,
		})
		if err != nil {
			return api.PolicyResourceTypeUnknown, nil, status.Errorf(codes.Internal, err.Error())
		}

		return api.PolicyResourceTypeProject, &project.UID, nil
	}

	sections := strings.Split(requestName, "/")

	if strings.HasPrefix(requestName, environmentNamePrefix) && len(sections) == 2 {
		// environment policy request name should be environments/{environment id}
		environmentID, err := getEnvironmentID(requestName)
		if err != nil {
			return api.PolicyResourceTypeUnknown, nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		if environmentID == "-" {
			return api.PolicyResourceTypeEnvironment, nil, nil
		}
		environment, err := s.findActiveEnvironment(ctx, &store.FindEnvironmentMessage{
			ResourceID: &environmentID,
		})
		if err != nil {
			return api.PolicyResourceTypeUnknown, nil, err
		}

		return api.PolicyResourceTypeEnvironment, &environment.UID, nil
	}

	if strings.HasPrefix(requestName, instanceNamePrefix) && len(sections) == 2 {
		// instance policy request name should be instances/{instance id}
		instanceID, err := getInstanceID(requestName)
		if err != nil {
			return api.PolicyResourceTypeUnknown, nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		if instanceID == "-" {
			return api.PolicyResourceTypeInstance, nil, nil
		}

		instance, err := s.findActiveInstance(ctx, &store.FindInstanceMessage{
			ResourceID: &instanceID,
		})
		if err != nil {
			return api.PolicyResourceTypeUnknown, nil, err
		}

		return api.PolicyResourceTypeInstance, &instance.UID, nil
	}

	if strings.HasPrefix(requestName, instanceNamePrefix) && len(sections) == 4 {
		// database policy request name should be instances/{instance id}/databases/{db name}

		instanceID, databaseName, err := getInstanceDatabaseID(requestName)
		if err != nil {
			return api.PolicyResourceTypeUnknown, nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		if databaseName == "-" {
			return api.PolicyResourceTypeDatabase, nil, nil
		}
		database, err := s.findActiveDatabase(ctx, &store.FindDatabaseMessage{
			InstanceID:   &instanceID,
			DatabaseName: &databaseName,
		})
		if err != nil {
			return api.PolicyResourceTypeUnknown, nil, status.Errorf(codes.Internal, err.Error())
		}

		return api.PolicyResourceTypeDatabase, &database.UID, nil
	}

	return api.PolicyResourceTypeUnknown, nil, status.Errorf(codes.InvalidArgument, "unknown request name %s", requestName)
}

func (s *OrgPolicyService) findActiveProject(ctx context.Context, find *store.FindProjectMessage) (*store.ProjectMessage, error) {
	find.ShowDeleted = false
	project, err := s.store.GetProjectV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %v not found", find)
	}
	return project, nil
}

func (s *OrgPolicyService) findActiveEnvironment(ctx context.Context, find *store.FindEnvironmentMessage) (*store.EnvironmentMessage, error) {
	find.ShowDeleted = false
	environment, err := s.store.GetEnvironmentV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if environment == nil {
		return nil, status.Errorf(codes.NotFound, "environment %v not found", find)
	}
	return environment, nil
}

func (s *OrgPolicyService) findActiveInstance(ctx context.Context, find *store.FindInstanceMessage) (*store.InstanceMessage, error) {
	find.ShowDeleted = false
	instance, err := s.store.GetInstanceV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %v not found", find)
	}
	return instance, nil
}

func (s *OrgPolicyService) findActiveDatabase(ctx context.Context, find *store.FindDatabaseMessage) (*store.DatabaseMessage, error) {
	find.ShowDeleted = false
	database, err := s.store.GetDatabaseV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %v not found", find)
	}
	return database, nil
}

func (s *OrgPolicyService) createPolicyMessage(ctx context.Context, creatorID int, parent string, policy *v1pb.Policy) (*v1pb.Policy, error) {
	resourceType, resourceID, err := s.getPolicyResourceTypeAndID(ctx, parent)
	if err != nil {
		return nil, err
	}
	if resourceID == nil {
		return nil, status.Errorf(codes.InvalidArgument, "resource id for %s must be specific", resourceType)
	}

	policyType, err := convertPolicyType(policy.Type.String())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	if err := validatePolicyType(policyType, resourceType); err != nil {
		return nil, err
	}

	payloadStr, err := s.convertPolicyPayloadToString(policy)
	if err != nil {
		return nil, err
	}

	p, err := s.store.CreatePolicyV2(ctx, &store.PolicyMessage{
		ResourceUID:       *resourceID,
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

func (s *OrgPolicyService) getPolicyParentPath(ctx context.Context, policyMessage *store.PolicyMessage) (string, error) {
	switch policyMessage.ResourceType {
	case api.PolicyResourceTypeEnvironment:
		env, err := s.findActiveEnvironment(ctx, &store.FindEnvironmentMessage{
			UID: &policyMessage.ResourceUID,
		})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s%s", environmentNamePrefix, env.ResourceID), nil
	case api.PolicyResourceTypeProject:
		proj, err := s.findActiveProject(ctx, &store.FindProjectMessage{
			UID: &policyMessage.ResourceUID,
		})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s%s", projectNamePrefix, proj.ResourceID), nil
	case api.PolicyResourceTypeInstance:
		ins, err := s.findActiveInstance(ctx, &store.FindInstanceMessage{
			UID: &policyMessage.ResourceUID,
		})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s%s", instanceNamePrefix, ins.ResourceID), nil
	case api.PolicyResourceTypeDatabase:
		db, err := s.findActiveDatabase(ctx, &store.FindDatabaseMessage{
			UID: &policyMessage.ResourceUID,
		})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s%s/%s%s", instanceNamePrefix, db.InstanceID, databaseIDPrefix, db.DatabaseName), nil
	default:
		return "", nil
	}
}

func validatePolicyType(policyType api.PolicyType, policyResourceType api.PolicyResourceType) error {
	for _, rt := range api.AllowedResourceTypes[policyType] {
		if rt == policyResourceType {
			return nil
		}
	}
	return status.Errorf(codes.InvalidArgument, "policy %v is not allowed in resource %v", policyType, policyResourceType)
}

func (s *OrgPolicyService) convertPolicyPayloadToString(policy *v1pb.Policy) (string, error) {
	switch policy.Type {
	case v1pb.PolicyType_WORKSPACE_IAM:
		iamPolicy := convertToStorePBWorkspaceIAMPolicy(policy.GetWorkspaceIamPolicy())
		payloadBytes, err := protojson.Marshal(iamPolicy)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal workspace iam policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_DEPLOYMENT_APPROVAL:
		payload, err := convertToPipelineApprovalPolicyPayload(policy.GetDeploymentApprovalPolicy())
		if err != nil {
			return "", status.Errorf(codes.InvalidArgument, err.Error())
		}
		if payload.Value != api.PipelineApprovalValueManualNever && payload.Value != api.PipelineApprovalValueManualAlways {
			return "", status.Errorf(codes.InvalidArgument, "invalid approval policy value: %q", *payload)
		}
		if err := s.licenseService.IsFeatureEnabled(api.FeatureApprovalPolicy); err != nil {
			if payload.Value != api.PipelineApprovalValueManualNever {
				return "", status.Errorf(codes.PermissionDenied, err.Error())
			}
		}
		issueTypeSeen := make(map[api.IssueType]bool)
		for _, group := range payload.AssigneeGroupList {
			if group.IssueType != api.IssueDatabaseSchemaUpdate &&
				group.IssueType != api.IssueDatabaseSchemaUpdateGhost &&
				group.IssueType != api.IssueDatabaseDataUpdate {
				return "", status.Errorf(codes.InvalidArgument, "invalid assignee group issue type %q", group.IssueType)
			}
			if issueTypeSeen[group.IssueType] {
				return "", status.Errorf(codes.InvalidArgument, "duplicate assignee group issue type %q", group.IssueType)
			}
			issueTypeSeen[group.IssueType] = true
		}
		return payload.String()
	case v1pb.PolicyType_BACKUP_PLAN:
		payload, err := convertToBackupPlanPolicyPayload(policy.GetBackupPlanPolicy())
		if err != nil {
			return "", status.Errorf(codes.InvalidArgument, err.Error())
		}
		if payload.Schedule != api.BackupPlanPolicyScheduleUnset && payload.Schedule != api.BackupPlanPolicyScheduleDaily && payload.Schedule != api.BackupPlanPolicyScheduleWeekly {
			return "", status.Errorf(codes.InvalidArgument, "invalid backup plan policy schedule: %q", payload.Schedule)
		}
		if err := s.licenseService.IsFeatureEnabled(api.FeatureBackupPolicy); err != nil {
			if payload.Schedule != api.BackupPlanPolicyScheduleUnset {
				return "", status.Errorf(codes.PermissionDenied, err.Error())
			}
		}
		return payload.String()
	case v1pb.PolicyType_SQL_REVIEW:
		if err := s.licenseService.IsFeatureEnabled(api.FeatureSQLReview); err != nil {
			return "", status.Errorf(codes.PermissionDenied, err.Error())
		}
		payload, err := convertToSQLReviewPolicyPayload(policy.GetSqlReviewPolicy())
		if err != nil {
			return "", status.Errorf(codes.InvalidArgument, err.Error())
		}
		if err := payload.Validate(); err != nil {
			return "", status.Errorf(codes.InvalidArgument, err.Error())
		}
		return payload.String()
	case v1pb.PolicyType_SENSITIVE_DATA:
		if err := s.licenseService.IsFeatureEnabled(api.FeatureSensitiveData); err != nil {
			return "", status.Errorf(codes.PermissionDenied, err.Error())
		}
		payload, err := convertToSensitiveDataPolicyPayload(policy.GetSensitiveDataPolicy())
		if err != nil {
			return "", status.Errorf(codes.InvalidArgument, err.Error())
		}
		for _, v := range payload.SensitiveDataList {
			if v.Table == "" || v.Column == "" {
				return "", status.Errorf(codes.InvalidArgument, "sensitive data policy rule cannot have empty table or column name")
			}
			if v.Type != api.SensitiveDataMaskTypeDefault {
				return "", status.Errorf(codes.InvalidArgument, "sensitive data policy rule must have mask type %q", api.SensitiveDataMaskTypeDefault)
			}
		}
		return payload.String()
	case v1pb.PolicyType_ACCESS_CONTROL:
		if err := s.licenseService.IsFeatureEnabled(api.FeatureAccessControl); err != nil {
			return "", status.Errorf(codes.PermissionDenied, err.Error())
		}
		payload, err := convertToAccessControlPolicyPayload(policy.GetAccessControlPolicy())
		if err != nil {
			return "", status.Errorf(codes.InvalidArgument, err.Error())
		}
		return payload.String()
	case v1pb.PolicyType_SLOW_QUERY:
		payload, err := convertToSlowQueryPolicyPayload(policy.GetSlowQueryPolicy())
		if err != nil {
			return "", status.Errorf(codes.InvalidArgument, err.Error())
		}
		return payload.String()
	}

	return "", status.Errorf(codes.InvalidArgument, "invalid policy %v", policy.Type)
}

func convertToPolicy(parentPath string, policyMessage *store.PolicyMessage) (*v1pb.Policy, error) {
	resourceType := v1pb.PolicyResourceType_RESOURCE_TYPE_UNSPECIFIED
	switch policyMessage.ResourceType {
	case api.PolicyResourceTypeWorkspace:
		resourceType = v1pb.PolicyResourceType_WORKSPACE
	case api.PolicyResourceTypeEnvironment:
		resourceType = v1pb.PolicyResourceType_ENVIRONMENT
	case api.PolicyResourceTypeProject:
		resourceType = v1pb.PolicyResourceType_PROJECT
	case api.PolicyResourceTypeDatabase:
		resourceType = v1pb.PolicyResourceType_DATABASE
	case api.PolicyResourceTypeInstance:
		resourceType = v1pb.PolicyResourceType_INSTANCE
	}
	policy := &v1pb.Policy{
		Uid:               fmt.Sprintf("%d", policyMessage.UID),
		InheritFromParent: policyMessage.InheritFromParent,
		Enforce:           policyMessage.Enforce,
		ResourceType:      resourceType,
		ResourceUid:       fmt.Sprintf("%d", policyMessage.ResourceUID),
	}

	pType := v1pb.PolicyType_POLICY_TYPE_UNSPECIFIED
	switch policyMessage.Type {
	case api.PolicyTypeWorkspaceIAM:
		pType = v1pb.PolicyType_WORKSPACE_IAM
		storepbIAMPolicy := &storepb.IamPolicy{}
		err := protojson.Unmarshal([]byte(policyMessage.Payload), storepbIAMPolicy)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal workspace IAM policy")
		}
		payload, err := convertToV1PBWorkspaceIAMPolicy(storepbIAMPolicy)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert workspace IAM policy")
		}
		policy.Policy = payload
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
	case api.PolicyTypeSlowQuery:
		pType = v1pb.PolicyType_SLOW_QUERY
		payload, err := convertToV1PBSlowQueryPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	}

	policy.Type = pType
	policy.Name = fmt.Sprintf("%s%s", policyNamePrefix, strings.ToLower(pType.String()))
	if parentPath != "" {
		policy.Name = fmt.Sprintf("%s/%s", parentPath, policy.Name)
	}

	return policy, nil
}

func convertToV1PBWorkspaceIAMPolicy(policy *storepb.IamPolicy) (*v1pb.Policy_WorkspaceIamPolicy, error) {
	iamPolicy := &v1pb.IamPolicy{
		Bindings: []*v1pb.Binding{},
	}
	for _, binding := range policy.Bindings {
		v1pbBinding := v1pb.Binding{
			Role:      binding.Role,
			Members:   binding.Members,
			Condition: binding.Condition,
		}

		env, err := cel.NewEnv(common.QueryExportPolicyCELAttributes...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create cel environment")
		}
		if binding.Condition.Expression != "" {
			ast, issues := env.Parse(binding.Condition.Expression)
			if issues != nil && issues.Err() != nil {
				return nil, errors.Wrap(issues.Err(), "failed to parse expression")
			}
			parsedExpr, err := cel.AstToParsedExpr(ast)
			if err != nil {
				return nil, errors.Wrap(err, "failed to convert ast to parsed expression")
			}
			v1pbBinding.ParsedExpr = parsedExpr
		}
		iamPolicy.Bindings = append(iamPolicy.Bindings, &v1pbBinding)
	}
	return &v1pb.Policy_WorkspaceIamPolicy{
		WorkspaceIamPolicy: iamPolicy,
	}, nil
}

func convertToStorePBWorkspaceIAMPolicy(policy *v1pb.IamPolicy) *storepb.IamPolicy {
	iamPolicy := &storepb.IamPolicy{
		Bindings: []*storepb.Binding{},
	}
	for _, binding := range policy.Bindings {
		iamBinding := storepb.Binding{
			Role:      binding.Role,
			Members:   binding.Members,
			Condition: binding.Condition,
		}
		iamPolicy.Bindings = append(iamPolicy.Bindings, &iamBinding)
	}
	return iamPolicy
}

func convertToV1PBSQLReviewPolicy(payloadStr string) (*v1pb.Policy_SqlReviewPolicy, error) {
	payload, err := api.UnmarshalSQLReviewPolicy(
		api.MergeSQLReviewRulesWithoutEngine(payloadStr),
	)
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
			Comment: rule.Comment,
			Engine:  convertToEngine(db.Type(rule.Engine)),
		})
	}

	return &v1pb.Policy_SqlReviewPolicy{
		SqlReviewPolicy: &v1pb.SQLReviewPolicy{
			Name:  payload.Name,
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
			Comment: rule.Comment,
			// DONOT assign the engine, we will use FlattenSQLReviewRulesWithEngine to map available engine with the rule.
		})
	}

	return api.FlattenSQLReviewRulesWithEngine(&advisor.SQLReviewPolicy{
		Name:     policy.Name,
		RuleList: ruleList,
	}), nil
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

	retentionPeriodTs := 0
	if policy.RetentionDuration != nil {
		retentionPeriodTs = int(policy.RetentionDuration.Seconds)
	}

	return &api.BackupPlanPolicy{
		Schedule:          schedule,
		RetentionPeriodTs: retentionPeriodTs,
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

func convertToV1PBSlowQueryPolicy(payloadStr string) (*v1pb.Policy_SlowQueryPolicy, error) {
	payload, err := api.UnmarshalSlowQueryPolicy(payloadStr)
	if err != nil {
		return nil, err
	}
	return &v1pb.Policy_SlowQueryPolicy{
		SlowQueryPolicy: &v1pb.SlowQueryPolicy{
			Active: payload.Active,
		},
	}, nil
}

func convertToSlowQueryPolicyPayload(policy *v1pb.SlowQueryPolicy) (*api.SlowQueryPolicy, error) {
	return &api.SlowQueryPolicy{
		Active: policy.Active,
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
	case v1pb.PolicyType_WORKSPACE_IAM.String():
		return api.PolicyTypeWorkspaceIAM, nil
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
	case v1pb.PolicyType_SLOW_QUERY.String():
		return api.PolicyTypeSlowQuery, nil
	}
	return policyType, errors.Errorf("invalid policy type %v", pType)
}
