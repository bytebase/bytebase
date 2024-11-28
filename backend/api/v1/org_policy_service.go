package v1

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// OrgPolicyService implements the workspace policy service.
type OrgPolicyService struct {
	v1pb.UnimplementedOrgPolicyServiceServer
	store          *store.Store
	licenseService enterprise.LicenseService
}

// NewOrgPolicyService creates a new OrgPolicyService.
func NewOrgPolicyService(store *store.Store, licenseService enterprise.LicenseService) *OrgPolicyService {
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

	response, err := s.convertToPolicy(ctx, parent, policy)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
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
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		find.Type = &policyType
	}

	policies, err := s.store.ListPoliciesV2(ctx, find)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response := &v1pb.ListPoliciesResponse{}
	for _, policy := range policies {
		parentPath, err := s.getPolicyParentPath(ctx, policy)
		if err != nil {
			st := status.Convert(err)
			if st.Code() == codes.NotFound {
				slog.Debug("failed to found resource for policy", log.BBError(err), slog.String("resource_type", string(policy.ResourceType)), slog.Int("resource_id", policy.ResourceUID))
				continue
			}
			return nil, err
		}
		p, err := s.convertToPolicy(ctx, parentPath, policy)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
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
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
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

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
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
			if err := validatePolicyPayload(policy.Type, request.Policy); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid policy: %v", err)
			}
			payloadStr, err := s.convertPolicyPayloadToString(ctx, request.Policy)
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
		return nil, status.Error(codes.Internal, err.Error())
	}

	response, err := s.convertToPolicy(ctx, parent, p)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
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
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

// findPolicyMessage finds the policy and the parent name by the policy name.
func (s *OrgPolicyService) findPolicyMessage(ctx context.Context, policyName string) (*store.PolicyMessage, string, error) {
	tokens := strings.Split(policyName, common.PolicyNamePrefix)
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
	if resourceID == nil && resourceType != api.PolicyResourceTypeWorkspace {
		return nil, policyParent, status.Errorf(codes.InvalidArgument, "resource id for %s must be specific", resourceType)
	}

	policyType, err := convertPolicyType(tokens[1])
	if err != nil {
		return nil, policyParent, status.Error(codes.InvalidArgument, err.Error())
	}

	policy, err := s.store.GetPolicyV2(ctx, &store.FindPolicyMessage{
		ResourceType: &resourceType,
		Type:         &policyType,
		ResourceUID:  resourceID,
	})
	if err != nil {
		return nil, policyParent, status.Error(codes.Internal, err.Error())
	}
	if policy == nil {
		return nil, policyParent, status.Errorf(codes.NotFound, "policy %q not found", policyName)
	}

	return policy, policyParent, nil
}

func (s *OrgPolicyService) getPolicyResourceTypeAndID(ctx context.Context, requestName string) (api.PolicyResourceType, *int, error) {
	if requestName == "" {
		return api.PolicyResourceTypeWorkspace, nil, nil
	}

	if strings.HasPrefix(requestName, common.ProjectNamePrefix) {
		projectID, err := common.GetProjectID(requestName)
		if err != nil {
			return api.PolicyResourceTypeUnknown, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if projectID == "-" {
			return api.PolicyResourceTypeProject, nil, nil
		}
		project, err := s.findActiveProject(ctx, &store.FindProjectMessage{
			ResourceID: &projectID,
		})
		if err != nil {
			return api.PolicyResourceTypeUnknown, nil, status.Error(codes.Internal, err.Error())
		}

		return api.PolicyResourceTypeProject, &project.UID, nil
	}

	sections := strings.Split(requestName, "/")

	if strings.HasPrefix(requestName, common.EnvironmentNamePrefix) && len(sections) == 2 {
		// environment policy request name should be environments/{environment id}
		environmentID, err := common.GetEnvironmentID(requestName)
		if err != nil {
			return api.PolicyResourceTypeUnknown, nil, status.Error(codes.InvalidArgument, err.Error())
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

	if strings.HasPrefix(requestName, common.InstanceNamePrefix) && len(sections) == 2 {
		// instance policy request name should be instances/{instance id}
		instanceID, err := common.GetInstanceID(requestName)
		if err != nil {
			return api.PolicyResourceTypeUnknown, nil, status.Error(codes.InvalidArgument, err.Error())
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

	if strings.HasPrefix(requestName, common.InstanceNamePrefix) && len(sections) == 4 {
		// database policy request name should be instances/{instance id}/databases/{db name}

		instanceID, databaseName, err := common.GetInstanceDatabaseID(requestName)
		if err != nil {
			return api.PolicyResourceTypeUnknown, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if databaseName == "-" {
			return api.PolicyResourceTypeDatabase, nil, nil
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return api.PolicyResourceTypeUnknown, nil, status.Error(codes.Internal, err.Error())
		}
		if instance == nil {
			return api.PolicyResourceTypeUnknown, nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
		}
		database, err := s.findActiveDatabase(ctx, &store.FindDatabaseMessage{
			InstanceID:          &instanceID,
			DatabaseName:        &databaseName,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		})
		if err != nil {
			return api.PolicyResourceTypeUnknown, nil, status.Error(codes.Internal, err.Error())
		}
		if database == nil {
			return api.PolicyResourceTypeUnknown, nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
		}

		return api.PolicyResourceTypeDatabase, &database.UID, nil
	}

	return api.PolicyResourceTypeUnknown, nil, status.Errorf(codes.InvalidArgument, "unknown request name %s", requestName)
}

func (s *OrgPolicyService) findActiveProject(ctx context.Context, find *store.FindProjectMessage) (*store.ProjectMessage, error) {
	find.ShowDeleted = false
	project, err := s.store.GetProjectV2(ctx, find)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project not found")
	}
	return project, nil
}

func (s *OrgPolicyService) findActiveEnvironment(ctx context.Context, find *store.FindEnvironmentMessage) (*store.EnvironmentMessage, error) {
	find.ShowDeleted = false
	environment, err := s.store.GetEnvironmentV2(ctx, find)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
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
		return nil, status.Error(codes.Internal, err.Error())
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
		return nil, status.Error(codes.Internal, err.Error())
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
	if resourceID == nil && resourceType != api.PolicyResourceTypeWorkspace {
		return nil, status.Errorf(codes.InvalidArgument, "resource id for %s must be specific", resourceType)
	}

	policyType, err := convertPolicyType(policy.Type.String())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := validatePolicyType(policyType, resourceType); err != nil {
		return nil, err
	}

	if err := validatePolicyPayload(policyType, policy); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid policy: %v", err)
	}

	payloadStr, err := s.convertPolicyPayloadToString(ctx, policy)
	if err != nil {
		return nil, err
	}

	create := &store.PolicyMessage{
		ResourceType:      resourceType,
		Payload:           payloadStr,
		Type:              policyType,
		InheritFromParent: policy.InheritFromParent,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	}
	if resourceID != nil {
		create.ResourceUID = *resourceID
	}

	p, err := s.store.CreatePolicyV2(ctx, create, creatorID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response, err := s.convertToPolicy(ctx, parent, p)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
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
		return common.FormatEnvironment(env.ResourceID), nil
	case api.PolicyResourceTypeProject:
		proj, err := s.findActiveProject(ctx, &store.FindProjectMessage{
			UID: &policyMessage.ResourceUID,
		})
		if err != nil {
			return "", err
		}
		return common.FormatProject(proj.ResourceID), nil
	case api.PolicyResourceTypeInstance:
		ins, err := s.findActiveInstance(ctx, &store.FindInstanceMessage{
			UID: &policyMessage.ResourceUID,
		})
		if err != nil {
			return "", err
		}
		return common.FormatInstance(ins.ResourceID), nil
	case api.PolicyResourceTypeDatabase:
		db, err := s.findActiveDatabase(ctx, &store.FindDatabaseMessage{
			UID: &policyMessage.ResourceUID,
		})
		if err != nil {
			return "", err
		}
		return common.FormatDatabase(db.InstanceID, db.DatabaseName), nil
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

func validatePolicyPayload(policyType api.PolicyType, policy *v1pb.Policy) error {
	switch policyType {
	case api.PolicyTypeMasking:
		maskingPolicy, ok := policy.Policy.(*v1pb.Policy_MaskingPolicy)
		if !ok {
			return status.Errorf(codes.InvalidArgument, "unmatched policy type %v and policy %v", policyType, policy.Policy)
		}
		if maskingPolicy.MaskingPolicy == nil {
			return status.Errorf(codes.InvalidArgument, "masking policy must be set")
		}
		for _, maskData := range maskingPolicy.MaskingPolicy.MaskData {
			if maskData.Column == "" || maskData.Table == "" {
				return status.Errorf(codes.InvalidArgument, "masking column and table must be set")
			}
		}
	case api.PolicyTypeMaskingRule:
		maskingRulePolicy, ok := policy.Policy.(*v1pb.Policy_MaskingRulePolicy)
		if !ok {
			return status.Errorf(codes.InvalidArgument, "unmatched policy type %v and policy %v", policyType, policy.Policy)
		}
		if maskingRulePolicy.MaskingRulePolicy == nil {
			return status.Errorf(codes.InvalidArgument, "masking rule policy must be set")
		}
		for _, rule := range maskingRulePolicy.MaskingRulePolicy.Rules {
			if rule.Id == "" {
				return status.Errorf(codes.InvalidArgument, "masking rule must have ID set")
			}
			if _, err := common.ValidateMaskingRuleCELExpr(rule.Condition.Expression); err != nil {
				return status.Errorf(codes.InvalidArgument, "invalid masking rule expression: %v", err)
			}
		}
	case api.PolicyTypeMaskingException:
		maskingExceptionPolicy, ok := policy.Policy.(*v1pb.Policy_MaskingExceptionPolicy)
		if !ok {
			return status.Errorf(codes.InvalidArgument, "unmatched policy type %v and policy %v", policyType, policy.Policy)
		}
		if maskingExceptionPolicy.MaskingExceptionPolicy == nil {
			return status.Errorf(codes.InvalidArgument, "masking exception policy must be set")
		}
		for _, exception := range maskingExceptionPolicy.MaskingExceptionPolicy.MaskingExceptions {
			if exception.Action == v1pb.MaskingExceptionPolicy_MaskingException_ACTION_UNSPECIFIED {
				return status.Errorf(codes.InvalidArgument, "masking exception must have action set")
			}
			if exception.MaskingLevel == v1pb.MaskingLevel_FULL {
				return status.Errorf(codes.InvalidArgument, "masking exception cannot have full masking level")
			}
			if exception.MaskingLevel == v1pb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED {
				return status.Errorf(codes.InvalidArgument, "masking exception must have masking level set")
			}
			if _, err := common.ValidateMaskingExceptionCELExpr(exception.Condition); err != nil {
				return status.Error(codes.InvalidArgument, fmt.Sprintf("invalid masking exception expression: %v", err))
			}
			if err := validateMember(exception.Member); err != nil {
				return err
			}
		}
	default:
	}
	return nil
}

func (s *OrgPolicyService) convertPolicyPayloadToString(ctx context.Context, policy *v1pb.Policy) (string, error) {
	switch policy.Type {
	case v1pb.PolicyType_ROLLOUT_POLICY:
		rolloutPolicy := convertToStorePBRolloutPolicy(policy.GetRolloutPolicy())
		if !rolloutPolicy.Automatic {
			if err := s.licenseService.IsFeatureEnabled(api.FeatureRolloutPolicy); err != nil {
				return "", status.Error(codes.PermissionDenied, err.Error())
			}
		}
		payloadBytes, err := protojson.Marshal(rolloutPolicy)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal rollout policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_TAG:
		tagPolicy := &storepb.TagPolicy{
			Tags: policy.GetTagPolicy().Tags,
		}
		payloadBytes, err := protojson.Marshal(tagPolicy)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal tag policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_MASKING:
		if err := s.licenseService.IsFeatureEnabled(api.FeatureSensitiveData); err != nil {
			return "", status.Error(codes.PermissionDenied, err.Error())
		}
		payload, err := convertToStorePBMaskingPolicyPayload(policy.GetMaskingPolicy())
		if err != nil {
			return "", status.Error(codes.InvalidArgument, err.Error())
		}
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal masking policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_SLOW_QUERY:
		payload, err := convertToSlowQueryPolicyPayload(policy.GetSlowQueryPolicy())
		if err != nil {
			return "", status.Error(codes.InvalidArgument, err.Error())
		}
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal masking policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_DISABLE_COPY_DATA:
		if err := s.licenseService.IsFeatureEnabled(api.FeatureAccessControl); err != nil {
			return "", status.Error(codes.PermissionDenied, err.Error())
		}
		payload, err := convertToDisableCopyDataPolicyPayload(policy.GetDisableCopyDataPolicy())
		if err != nil {
			return "", status.Error(codes.InvalidArgument, err.Error())
		}
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_DATA_EXPORT:
		if err := s.licenseService.IsFeatureEnabled(api.FeatureAccessControl); err != nil {
			return "", status.Error(codes.PermissionDenied, err.Error())
		}
		payload, err := convertToExportDataPolicyPayload(policy.GetExportDataPolicy())
		if err != nil {
			return "", status.Error(codes.InvalidArgument, err.Error())
		}
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_MASKING_RULE:
		if err := s.licenseService.IsFeatureEnabled(api.FeatureSensitiveData); err != nil {
			return "", status.Error(codes.PermissionDenied, err.Error())
		}
		payload, err := convertToStorePBMskingRulePolicy(policy.GetMaskingRulePolicy())
		if err != nil {
			return "", status.Error(codes.InvalidArgument, err.Error())
		}
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal masking rule policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_MASKING_EXCEPTION:
		if err := s.licenseService.IsFeatureEnabled(api.FeatureSensitiveData); err != nil {
			return "", status.Error(codes.PermissionDenied, err.Error())
		}
		payload, err := s.convertToStorePBMaskingExceptionPolicyPayload(ctx, policy.GetMaskingExceptionPolicy())
		if err != nil {
			return "", status.Error(codes.InvalidArgument, err.Error())
		}
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal masking exception policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW:
		if err := s.licenseService.IsFeatureEnabled(api.FeatureAccessControl); err != nil {
			return "", status.Error(codes.PermissionDenied, err.Error())
		}
		payload, err := convertToRestrictIssueCreationForSQLReviewPayload(policy.GetRestrictIssueCreationForSqlReviewPolicy())
		if err != nil {
			return "", status.Error(codes.InvalidArgument, err.Error())
		}
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal restrict issue creation for SQL review policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_DATA_SOURCE_QUERY:
		payload, err := convertToDataSourceQueryPayload(policy.GetDataSourceQueryPolicy())
		if err != nil {
			return "", status.Error(codes.InvalidArgument, err.Error())
		}
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal data source query policy")
		}
		return string(payloadBytes), nil
	}

	return "", status.Errorf(codes.InvalidArgument, "invalid policy %v", policy.Type)
}

func (s *OrgPolicyService) convertToPolicy(ctx context.Context, parentPath string, policyMessage *store.PolicyMessage) (*v1pb.Policy, error) {
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
		InheritFromParent: policyMessage.InheritFromParent,
		Enforce:           policyMessage.Enforce,
		ResourceType:      resourceType,
	}

	pType := v1pb.PolicyType_POLICY_TYPE_UNSPECIFIED
	switch policyMessage.Type {
	case api.PolicyTypeRollout:
		pType = v1pb.PolicyType_ROLLOUT_POLICY
		payload, err := convertToV1RolloutPolicyPayload(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case api.PolicyTypeTag:
		pType = v1pb.PolicyType_TAG
		p := &v1pb.TagPolicy{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policyMessage.Payload), p); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal rollout policy payload")
		}
		policy.Policy = &v1pb.Policy_TagPolicy{
			TagPolicy: p,
		}
	case api.PolicyTypeMasking:
		pType = v1pb.PolicyType_MASKING
		payload, err := convertToV1PBMaskingPolicy(policyMessage.Payload)
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
	case api.PolicyTypeDisableCopyData:
		pType = v1pb.PolicyType_DISABLE_COPY_DATA
		payload, err := convertToV1PBDisableCopyDataPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case api.PolicyTypeExportData:
		pType = v1pb.PolicyType_DATA_EXPORT
		payload, err := convertToV1PBExportDataPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case api.PolicyTypeMaskingRule:
		pType = v1pb.PolicyType_MASKING_RULE
		maskingRulePolicy := &storepb.MaskingRulePolicy{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policyMessage.Payload), maskingRulePolicy); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal masking rule policy")
		}
		payload, err := convertToV1PBMaskingRulePolicy(maskingRulePolicy)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert masking rule policy")
		}
		policy.Policy = &v1pb.Policy_MaskingRulePolicy{
			MaskingRulePolicy: payload,
		}
	case api.PolicyTypeMaskingException:
		pType = v1pb.PolicyType_MASKING_EXCEPTION
		maskingRulePolicy := &storepb.MaskingExceptionPolicy{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policyMessage.Payload), maskingRulePolicy); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal masking exception policy")
		}
		payload, err := s.convertToV1PBMaskingExceptionPolicyPayload(ctx, maskingRulePolicy)
		if err != nil {
			return nil, errors.Wrap(err, "failed to convert masking exception policy")
		}
		policy.Policy = &v1pb.Policy_MaskingExceptionPolicy{
			MaskingExceptionPolicy: payload,
		}
	case api.PolicyTypeRestrictIssueCreationForSQLReview:
		pType = v1pb.PolicyType_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW
		payload, err := convertToV1PBRestrictIssueCreationForSQLReviewPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case api.PolicyTypeDataSourceQuery:
		pType = v1pb.PolicyType_DATA_SOURCE_QUERY
		payload, err := convertToV1PBDataSourceQueryPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	}

	policy.Type = pType
	policy.Name = fmt.Sprintf("%s%s", common.PolicyNamePrefix, strings.ToLower(pType.String()))
	if parentPath != "" {
		policy.Name = fmt.Sprintf("%s/%s", parentPath, policy.Name)
	}

	return policy, nil
}

func convertToV1PBSQLReviewRules(ruleList []*storepb.SQLReviewRule) []*v1pb.SQLReviewRule {
	var rules []*v1pb.SQLReviewRule
	for _, rule := range ruleList {
		level := v1pb.SQLReviewRuleLevel_LEVEL_UNSPECIFIED
		switch rule.Level {
		case storepb.SQLReviewRuleLevel_ERROR:
			level = v1pb.SQLReviewRuleLevel_ERROR
		case storepb.SQLReviewRuleLevel_WARNING:
			level = v1pb.SQLReviewRuleLevel_WARNING
		case storepb.SQLReviewRuleLevel_DISABLED:
			level = v1pb.SQLReviewRuleLevel_DISABLED
		}
		rules = append(rules, &v1pb.SQLReviewRule{
			Level:   level,
			Type:    string(rule.Type),
			Payload: rule.Payload,
			Comment: rule.Comment,
			Engine:  convertToEngine(rule.Engine),
		})
	}

	return rules
}

func convertToSQLReviewRules(rules []*v1pb.SQLReviewRule) ([]*storepb.SQLReviewRule, error) {
	var ruleList []*storepb.SQLReviewRule
	for _, rule := range rules {
		var level storepb.SQLReviewRuleLevel
		switch rule.Level {
		case v1pb.SQLReviewRuleLevel_ERROR:
			level = storepb.SQLReviewRuleLevel_ERROR
		case v1pb.SQLReviewRuleLevel_WARNING:
			level = storepb.SQLReviewRuleLevel_WARNING
		case v1pb.SQLReviewRuleLevel_DISABLED:
			level = storepb.SQLReviewRuleLevel_DISABLED
		default:
			return nil, errors.Errorf("invalid rule level %v", rule.Level)
		}
		ruleList = append(ruleList, &storepb.SQLReviewRule{
			Level:   level,
			Payload: rule.Payload,
			Type:    rule.Type,
			Comment: rule.Comment,
			Engine:  convertEngine(rule.Engine),
		})
	}

	return ruleList, nil
}

func convertToV1PBMaskingPolicy(payloadStr string) (*v1pb.Policy_MaskingPolicy, error) {
	var maskingPolicy storepb.MaskingPolicy
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadStr), &maskingPolicy); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal masking policy")
	}

	var maskDataList []*v1pb.MaskData
	for _, data := range maskingPolicy.MaskData {
		maskDataList = append(maskDataList, &v1pb.MaskData{
			Schema:                    data.Schema,
			Table:                     data.Table,
			Column:                    data.Column,
			MaskingLevel:              convertToV1PBMaskingLevel(data.MaskingLevel),
			FullMaskingAlgorithmId:    data.FullMaskingAlgorithmId,
			PartialMaskingAlgorithmId: data.PartialMaskingAlgorithmId,
		})
	}

	return &v1pb.Policy_MaskingPolicy{
		MaskingPolicy: &v1pb.MaskingPolicy{
			MaskData: maskDataList,
		},
	}, nil
}

func convertToStorePBMaskingPolicyPayload(policy *v1pb.MaskingPolicy) (*storepb.MaskingPolicy, error) {
	var maskData []*storepb.MaskData

	for _, data := range policy.MaskData {
		maskData = append(maskData, &storepb.MaskData{
			Schema:                    data.Schema,
			Table:                     data.Table,
			Column:                    data.Column,
			MaskingLevel:              convertToStorePBMaskingLevel(data.MaskingLevel),
			FullMaskingAlgorithmId:    data.FullMaskingAlgorithmId,
			PartialMaskingAlgorithmId: data.PartialMaskingAlgorithmId,
		})
	}

	return &storepb.MaskingPolicy{
		MaskData: maskData,
	}, nil
}

func convertToV1PBAction(action storepb.MaskingExceptionPolicy_MaskingException_Action) v1pb.MaskingExceptionPolicy_MaskingException_Action {
	switch action {
	case storepb.MaskingExceptionPolicy_MaskingException_ACTION_UNSPECIFIED:
		return v1pb.MaskingExceptionPolicy_MaskingException_ACTION_UNSPECIFIED
	case storepb.MaskingExceptionPolicy_MaskingException_QUERY:
		return v1pb.MaskingExceptionPolicy_MaskingException_QUERY
	case storepb.MaskingExceptionPolicy_MaskingException_EXPORT:
		return v1pb.MaskingExceptionPolicy_MaskingException_EXPORT
	}
	return v1pb.MaskingExceptionPolicy_MaskingException_ACTION_UNSPECIFIED
}

func convertToStorePBAction(action v1pb.MaskingExceptionPolicy_MaskingException_Action) storepb.MaskingExceptionPolicy_MaskingException_Action {
	switch action {
	case v1pb.MaskingExceptionPolicy_MaskingException_ACTION_UNSPECIFIED:
		return storepb.MaskingExceptionPolicy_MaskingException_ACTION_UNSPECIFIED
	case v1pb.MaskingExceptionPolicy_MaskingException_QUERY:
		return storepb.MaskingExceptionPolicy_MaskingException_QUERY
	case v1pb.MaskingExceptionPolicy_MaskingException_EXPORT:
		return storepb.MaskingExceptionPolicy_MaskingException_EXPORT
	}
	return storepb.MaskingExceptionPolicy_MaskingException_ACTION_UNSPECIFIED
}

func convertToV1PBMaskingLevel(level storepb.MaskingLevel) v1pb.MaskingLevel {
	switch level {
	case storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED:
		return v1pb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED
	case storepb.MaskingLevel_NONE:
		return v1pb.MaskingLevel_NONE
	case storepb.MaskingLevel_PARTIAL:
		return v1pb.MaskingLevel_PARTIAL
	case storepb.MaskingLevel_FULL:
		return v1pb.MaskingLevel_FULL
	default:
		return v1pb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED
	}
}

func convertToStorePBMaskingLevel(level v1pb.MaskingLevel) storepb.MaskingLevel {
	switch level {
	case v1pb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED:
		return storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED
	case v1pb.MaskingLevel_NONE:
		return storepb.MaskingLevel_NONE
	case v1pb.MaskingLevel_PARTIAL:
		return storepb.MaskingLevel_PARTIAL
	case v1pb.MaskingLevel_FULL:
		return storepb.MaskingLevel_FULL
	default:
		return storepb.MaskingLevel_MASKING_LEVEL_UNSPECIFIED
	}
}

func convertToV1RolloutPolicyPayload(payloadStr string) (*v1pb.Policy_RolloutPolicy, error) {
	p := &v1pb.RolloutPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadStr), p); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal rollout policy payload")
	}
	return &v1pb.Policy_RolloutPolicy{
		RolloutPolicy: p,
	}, nil
}

func convertToStorePBRolloutPolicy(policy *v1pb.RolloutPolicy) *storepb.RolloutPolicy {
	return &storepb.RolloutPolicy{
		Automatic:      policy.Automatic,
		WorkspaceRoles: policy.WorkspaceRoles,
		ProjectRoles:   policy.ProjectRoles,
		IssueRoles:     policy.IssueRoles,
	}
}

func convertToV1PBSlowQueryPolicy(payloadStr string) (*v1pb.Policy_SlowQueryPolicy, error) {
	payload := &storepb.SlowQueryPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadStr), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal slow query policy payload")
	}
	return &v1pb.Policy_SlowQueryPolicy{
		SlowQueryPolicy: &v1pb.SlowQueryPolicy{
			Active: payload.Active,
		},
	}, nil
}

func convertToSlowQueryPolicyPayload(policy *v1pb.SlowQueryPolicy) (*storepb.SlowQueryPolicy, error) {
	return &storepb.SlowQueryPolicy{
		Active: policy.Active,
	}, nil
}

func convertToV1PBDisableCopyDataPolicy(payloadStr string) (*v1pb.Policy_DisableCopyDataPolicy, error) {
	payload := &storepb.DisableCopyDataPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadStr), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal disable copy policy payload")
	}
	return &v1pb.Policy_DisableCopyDataPolicy{
		DisableCopyDataPolicy: &v1pb.DisableCopyDataPolicy{
			Active: payload.Active,
		},
	}, nil
}

func convertToV1PBExportDataPolicy(payloadStr string) (*v1pb.Policy_ExportDataPolicy, error) {
	payload := &storepb.ExportDataPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadStr), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal disable copy policy payload")
	}
	return &v1pb.Policy_ExportDataPolicy{
		ExportDataPolicy: &v1pb.ExportDataPolicy{
			Disable: payload.Disable,
		},
	}, nil
}

func convertToDisableCopyDataPolicyPayload(policy *v1pb.DisableCopyDataPolicy) (*storepb.DisableCopyDataPolicy, error) {
	return &storepb.DisableCopyDataPolicy{
		Active: policy.Active,
	}, nil
}

func convertToExportDataPolicyPayload(policy *v1pb.ExportDataPolicy) (*storepb.ExportDataPolicy, error) {
	return &storepb.ExportDataPolicy{
		Disable: policy.Disable,
	}, nil
}

func convertToStorePBMskingRulePolicy(policy *v1pb.MaskingRulePolicy) (*storepb.MaskingRulePolicy, error) {
	var rules []*storepb.MaskingRulePolicy_MaskingRule
	for _, rule := range policy.Rules {
		rules = append(rules, &storepb.MaskingRulePolicy_MaskingRule{
			Id: rule.Id,
			Condition: &expr.Expr{
				Title:       rule.Condition.Title,
				Expression:  rule.Condition.Expression,
				Description: rule.Condition.Description,
				Location:    rule.Condition.Location,
			},
			MaskingLevel: convertToStorePBMaskingLevel(rule.MaskingLevel),
		})
	}

	return &storepb.MaskingRulePolicy{
		Rules: rules,
	}, nil
}

func convertToV1PBMaskingRulePolicy(policy *storepb.MaskingRulePolicy) (*v1pb.MaskingRulePolicy, error) {
	var rules []*v1pb.MaskingRulePolicy_MaskingRule
	for _, rule := range policy.Rules {
		rules = append(rules, &v1pb.MaskingRulePolicy_MaskingRule{
			Id: rule.Id,
			Condition: &expr.Expr{
				Title:       rule.Condition.Title,
				Expression:  rule.Condition.Expression,
				Description: rule.Condition.Description,
				Location:    rule.Condition.Location,
			},
			MaskingLevel: convertToV1PBMaskingLevel(rule.MaskingLevel),
		})
	}

	return &v1pb.MaskingRulePolicy{
		Rules: rules,
	}, nil
}

func (s *OrgPolicyService) convertToStorePBMaskingExceptionPolicyPayload(ctx context.Context, policy *v1pb.MaskingExceptionPolicy) (*storepb.MaskingExceptionPolicy, error) {
	var exceptions []*storepb.MaskingExceptionPolicy_MaskingException
	for _, exception := range policy.MaskingExceptions {
		member, err := convertToStoreIamPolicyMember(ctx, s.store, exception.Member)
		if err != nil {
			return nil, err
		}
		exceptions = append(exceptions, &storepb.MaskingExceptionPolicy_MaskingException{
			Action:       convertToStorePBAction(exception.Action),
			MaskingLevel: convertToStorePBMaskingLevel(exception.MaskingLevel),
			Member:       member,
			Condition: &expr.Expr{
				Title:       exception.Condition.Title,
				Expression:  exception.Condition.Expression,
				Description: exception.Condition.Description,
				Location:    exception.Condition.Location,
			},
		})
	}

	return &storepb.MaskingExceptionPolicy{
		MaskingExceptions: exceptions,
	}, nil
}

func (s *OrgPolicyService) convertToV1PBMaskingExceptionPolicyPayload(ctx context.Context, policy *storepb.MaskingExceptionPolicy) (*v1pb.MaskingExceptionPolicy, error) {
	var exceptions []*v1pb.MaskingExceptionPolicy_MaskingException
	for _, exception := range policy.MaskingExceptions {
		memberInBinding := convertToV1MemberInBinding(ctx, s.store, exception.Member)
		if memberInBinding == "" {
			continue
		}

		exceptions = append(exceptions, &v1pb.MaskingExceptionPolicy_MaskingException{
			Action:       convertToV1PBAction(exception.Action),
			MaskingLevel: convertToV1PBMaskingLevel(exception.MaskingLevel),
			Member:       memberInBinding,
			Condition: &expr.Expr{
				Title:       exception.Condition.Title,
				Expression:  exception.Condition.Expression,
				Description: exception.Condition.Description,
				Location:    exception.Condition.Location,
			},
		})
	}

	return &v1pb.MaskingExceptionPolicy{
		MaskingExceptions: exceptions,
	}, nil
}

func convertToV1PBRestrictIssueCreationForSQLReviewPolicy(payloadStr string) (*v1pb.Policy_RestrictIssueCreationForSqlReviewPolicy, error) {
	payload := &storepb.RestrictIssueCreationForSQLReviewPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadStr), payload); err != nil {
		return nil, err
	}
	return &v1pb.Policy_RestrictIssueCreationForSqlReviewPolicy{
		RestrictIssueCreationForSqlReviewPolicy: &v1pb.RestrictIssueCreationForSQLReviewPolicy{
			Disallow: payload.Disallow,
		},
	}, nil
}

func convertToRestrictIssueCreationForSQLReviewPayload(policy *v1pb.RestrictIssueCreationForSQLReviewPolicy) (*storepb.RestrictIssueCreationForSQLReviewPolicy, error) {
	return &storepb.RestrictIssueCreationForSQLReviewPolicy{
		Disallow: policy.Disallow,
	}, nil
}

func convertToV1PBDataSourceQueryPolicy(payloadStr string) (*v1pb.Policy_DataSourceQueryPolicy, error) {
	payload := &storepb.DataSourceQueryPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadStr), payload); err != nil {
		return nil, err
	}

	return &v1pb.Policy_DataSourceQueryPolicy{
		DataSourceQueryPolicy: &v1pb.DataSourceQueryPolicy{
			AdminDataSourceRestriction: v1pb.DataSourceQueryPolicy_Restriction(payload.AdminDataSourceRestriction),
			DisallowDdl:                payload.DisallowDdl,
			DisallowDml:                payload.DisallowDml,
		},
	}, nil
}

func convertToDataSourceQueryPayload(policy *v1pb.DataSourceQueryPolicy) (*storepb.DataSourceQueryPolicy, error) {
	return &storepb.DataSourceQueryPolicy{
		AdminDataSourceRestriction: storepb.DataSourceQueryPolicy_Restriction(policy.AdminDataSourceRestriction),
		DisallowDdl:                policy.DisallowDdl,
		DisallowDml:                policy.DisallowDml,
	}, nil
}

func convertPolicyType(pType string) (api.PolicyType, error) {
	var policyType api.PolicyType
	switch strings.ToUpper(pType) {
	case v1pb.PolicyType_ROLLOUT_POLICY.String():
		return api.PolicyTypeRollout, nil
	case v1pb.PolicyType_TAG.String():
		return api.PolicyTypeTag, nil
	case v1pb.PolicyType_MASKING.String():
		return api.PolicyTypeMasking, nil
	case v1pb.PolicyType_MASKING_RULE.String():
		return api.PolicyTypeMaskingRule, nil
	case v1pb.PolicyType_MASKING_EXCEPTION.String():
		return api.PolicyTypeMaskingException, nil
	case v1pb.PolicyType_SLOW_QUERY.String():
		return api.PolicyTypeSlowQuery, nil
	case v1pb.PolicyType_DISABLE_COPY_DATA.String():
		return api.PolicyTypeDisableCopyData, nil
	case v1pb.PolicyType_DATA_EXPORT.String():
		return api.PolicyTypeExportData, nil
	case v1pb.PolicyType_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW.String():
		return api.PolicyTypeRestrictIssueCreationForSQLReview, nil
	case v1pb.PolicyType_DATA_SOURCE_QUERY.String():
		return api.PolicyTypeDataSourceQuery, nil
	}
	return policyType, errors.Errorf("invalid policy type %v", pType)
}
