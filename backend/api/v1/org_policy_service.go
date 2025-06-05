package v1

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var (
	// allowedResourceTypes includes allowed resource types for each policy type.
	allowedResourceTypes = map[storepb.PolicyType][]base.PolicyResourceType{
		storepb.PolicyType_ROLLOUT:                                {base.PolicyResourceTypeEnvironment},
		storepb.PolicyType_TAG:                                    {base.PolicyResourceTypeEnvironment, base.PolicyResourceTypeProject},
		storepb.PolicyType_DISABLE_COPY_DATA:                      {base.PolicyResourceTypeEnvironment, base.PolicyResourceTypeProject},
		storepb.PolicyType_EXPORT_DATA:                            {base.PolicyResourceTypeWorkspace},
		storepb.PolicyType_QUERY_DATA:                             {base.PolicyResourceTypeWorkspace},
		storepb.PolicyType_MASKING_RULE:                           {base.PolicyResourceTypeWorkspace},
		storepb.PolicyType_MASKING_EXCEPTION:                      {base.PolicyResourceTypeProject},
		storepb.PolicyType_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW: {base.PolicyResourceTypeWorkspace, base.PolicyResourceTypeProject},
		storepb.PolicyType_IAM:                                    {base.PolicyResourceTypeWorkspace},
		storepb.PolicyType_DATA_SOURCE_QUERY:                      {base.PolicyResourceTypeEnvironment, base.PolicyResourceTypeProject},
	}
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
	policy, _, err := s.findPolicyMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}

	response, err := s.convertToPolicy(ctx, policy)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return response, nil
}

// ListPolicies lists policies in a specific resource.
func (s *OrgPolicyService) ListPolicies(ctx context.Context, request *v1pb.ListPoliciesRequest) (*v1pb.ListPoliciesResponse, error) {
	resourceType, resource, err := getPolicyResourceTypeAndResource(request.Parent)
	if err != nil {
		return nil, err
	}

	find := &store.FindPolicyMessage{
		ResourceType: &resourceType,
		Resource:     resource,
		ShowAll:      request.ShowDeleted,
	}

	if v := request.PolicyType; v != nil {
		policyType, err := convertV1PBToStorePBPolicyType(*v)
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
		p, err := s.convertToPolicy(ctx, policy)
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
	if request.Policy == nil {
		return nil, status.Errorf(codes.InvalidArgument, "policy must be set")
	}

	// TODO(d): validate policy.
	return s.createPolicyMessage(ctx, request.Parent, request.Policy)
}

// UpdatePolicy updates a policy in a specific resource.
func (s *OrgPolicyService) UpdatePolicy(ctx context.Context, request *v1pb.UpdatePolicyRequest) (*v1pb.Policy, error) {
	if request.Policy == nil {
		return nil, status.Errorf(codes.InvalidArgument, "policy must be set")
	}

	policy, parent, err := s.findPolicyMessage(ctx, request.Policy.Name)
	if err != nil {
		st := status.Convert(err)
		if st.Code() == codes.NotFound && request.AllowMissing {
			return s.createPolicyMessage(ctx, parent, request.Policy)
		}
		return nil, err
	}

	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	patch := &store.UpdatePolicyMessage{
		ResourceType: policy.ResourceType,
		Type:         policy.Type,
		Resource:     policy.Resource,
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

	response, err := s.convertToPolicy(ctx, p)
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
	resourceType, resource, err := getPolicyResourceTypeAndResource(policyParent)
	if err != nil {
		return nil, policyParent, err
	}
	if resource == nil && resourceType != base.PolicyResourceTypeWorkspace {
		return nil, policyParent, status.Errorf(codes.InvalidArgument, "resource for %s must be specific", resourceType)
	}

	// Parse the policy type from the string in the policy name
	v1PolicyType, ok := v1pb.PolicyType_value[strings.ToUpper(tokens[1])]
	if !ok {
		return nil, policyParent, status.Errorf(codes.InvalidArgument, "invalid policy type %v", tokens[1])
	}

	policyType, err := convertV1PBToStorePBPolicyType(v1pb.PolicyType(v1PolicyType))
	if err != nil {
		return nil, policyParent, status.Error(codes.InvalidArgument, err.Error())
	}

	policy, err := s.store.GetPolicyV2(ctx, &store.FindPolicyMessage{
		ResourceType: &resourceType,
		Type:         &policyType,
		Resource:     resource,
	})
	if err != nil {
		return nil, policyParent, status.Error(codes.Internal, err.Error())
	}
	if policy == nil {
		return nil, policyParent, status.Errorf(codes.NotFound, "policy %q not found", policyName)
	}

	return policy, policyParent, nil
}

func getPolicyResourceTypeAndResource(requestName string) (base.PolicyResourceType, *string, error) {
	if requestName == "" {
		return base.PolicyResourceTypeWorkspace, nil, nil
	}

	if strings.HasPrefix(requestName, common.ProjectNamePrefix) {
		projectID, err := common.GetProjectID(requestName)
		if err != nil {
			return base.PolicyResourceTypeUnknown, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if projectID == "-" {
			return base.PolicyResourceTypeProject, nil, nil
		}
		return base.PolicyResourceTypeProject, &requestName, nil
	}

	if strings.HasPrefix(requestName, common.EnvironmentNamePrefix) {
		// environment policy request name should be environments/{environment id}
		environmentID, err := common.GetEnvironmentID(requestName)
		if err != nil {
			return base.PolicyResourceTypeUnknown, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if environmentID == "-" {
			return base.PolicyResourceTypeEnvironment, nil, nil
		}
		return base.PolicyResourceTypeEnvironment, &requestName, nil
	}

	if strings.HasPrefix(requestName, common.InstanceNamePrefix) {
		// instance policy request name should be instances/{instance id}
		instanceID, err := common.GetInstanceID(requestName)
		if err != nil {
			return base.PolicyResourceTypeUnknown, nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if instanceID == "-" {
			return base.PolicyResourceTypeInstance, nil, nil
		}
		return base.PolicyResourceTypeInstance, &requestName, nil
	}

	return base.PolicyResourceTypeUnknown, nil, status.Errorf(codes.InvalidArgument, "unknown request name %s", requestName)
}

func (s *OrgPolicyService) createPolicyMessage(ctx context.Context, parent string, policy *v1pb.Policy) (*v1pb.Policy, error) {
	resourceType, _, err := getPolicyResourceTypeAndResource(parent)
	if err != nil {
		return nil, err
	}

	policyType, err := convertV1PBToStorePBPolicyType(policy.Type)
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
		Resource:          parent,
		Payload:           payloadStr,
		Type:              policyType,
		InheritFromParent: policy.InheritFromParent,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	}

	p, err := s.store.CreatePolicyV2(ctx, create)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	response, err := s.convertToPolicy(ctx, p)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return response, nil
}

func validatePolicyType(policyType storepb.PolicyType, policyResourceType base.PolicyResourceType) error {
	allowedTypes, ok := allowedResourceTypes[policyType]
	if !ok {
		return status.Errorf(codes.InvalidArgument, "unknown policy type %v", policyType)
	}
	for _, rt := range allowedTypes {
		if rt == policyResourceType {
			return nil
		}
	}
	return status.Errorf(codes.InvalidArgument, "policy %v is not allowed in resource %v", policyType, policyResourceType)
}

func validatePolicyPayload(policyType storepb.PolicyType, policy *v1pb.Policy) error {
	switch policyType {
	case storepb.PolicyType_MASKING_RULE:
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
	case storepb.PolicyType_MASKING_EXCEPTION:
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
			if err := s.licenseService.IsFeatureEnabled(base.FeatureRolloutPolicy); err != nil {
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
	case v1pb.PolicyType_DISABLE_COPY_DATA:
		if err := s.licenseService.IsFeatureEnabled(base.FeatureAccessControl); err != nil {
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
		if err := s.licenseService.IsFeatureEnabled(base.FeatureAccessControl); err != nil {
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
	case v1pb.PolicyType_DATA_QUERY:
		if err := s.licenseService.IsFeatureEnabled(base.FeatureAccessControl); err != nil {
			return "", status.Error(codes.PermissionDenied, err.Error())
		}
		payload, err := convertToQueryDataPolicyPayload(policy.GetQueryDataPolicy())
		if err != nil {
			return "", status.Error(codes.InvalidArgument, err.Error())
		}
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_MASKING_RULE:
		if err := s.licenseService.IsFeatureEnabled(base.FeatureSensitiveData); err != nil {
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
		if err := s.licenseService.IsFeatureEnabled(base.FeatureSensitiveData); err != nil {
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
		if err := s.licenseService.IsFeatureEnabled(base.FeatureAccessControl); err != nil {
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

func (s *OrgPolicyService) convertToPolicy(ctx context.Context, policyMessage *store.PolicyMessage) (*v1pb.Policy, error) {
	resourceType := v1pb.PolicyResourceType_RESOURCE_TYPE_UNSPECIFIED
	switch policyMessage.ResourceType {
	case base.PolicyResourceTypeWorkspace:
		resourceType = v1pb.PolicyResourceType_WORKSPACE
	case base.PolicyResourceTypeEnvironment:
		resourceType = v1pb.PolicyResourceType_ENVIRONMENT
	case base.PolicyResourceTypeProject:
		resourceType = v1pb.PolicyResourceType_PROJECT
	case base.PolicyResourceTypeInstance:
		resourceType = v1pb.PolicyResourceType_INSTANCE
	}
	policy := &v1pb.Policy{
		InheritFromParent: policyMessage.InheritFromParent,
		Enforce:           policyMessage.Enforce,
		ResourceType:      resourceType,
	}

	pType := convertStorePBToV1PBPolicyType(policyMessage.Type)
	switch policyMessage.Type {
	case storepb.PolicyType_ROLLOUT:
		payload, err := convertToV1RolloutPolicyPayload(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case storepb.PolicyType_TAG:
		p := &v1pb.TagPolicy{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policyMessage.Payload), p); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal rollout policy payload")
		}
		policy.Policy = &v1pb.Policy_TagPolicy{
			TagPolicy: p,
		}
	case storepb.PolicyType_DISABLE_COPY_DATA:
		payload, err := convertToV1PBDisableCopyDataPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case storepb.PolicyType_EXPORT_DATA:
		payload, err := convertToV1PBExportDataPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case storepb.PolicyType_QUERY_DATA:
		payload, err := convertToV1PBQueryDataPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case storepb.PolicyType_MASKING_RULE:
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
	case storepb.PolicyType_MASKING_EXCEPTION:
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
	case storepb.PolicyType_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW:
		payload, err := convertToV1PBRestrictIssueCreationForSQLReviewPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case storepb.PolicyType_DATA_SOURCE_QUERY:
		payload, err := convertToV1PBDataSourceQueryPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	}

	policy.Type = pType
	policy.Name = fmt.Sprintf("%s%s", common.PolicyNamePrefix, strings.ToLower(pType.String()))
	if policyMessage.Resource != "" {
		policy.Name = fmt.Sprintf("%s/%s", policyMessage.Resource, policy.Name)
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
		Automatic:  policy.Automatic,
		Roles:      policy.Roles,
		IssueRoles: policy.IssueRoles,
	}
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
		return nil, errors.Wrapf(err, "failed to unmarshal export data policy payload")
	}
	return &v1pb.Policy_ExportDataPolicy{
		ExportDataPolicy: &v1pb.ExportDataPolicy{
			Disable: payload.Disable,
		},
	}, nil
}

func convertToV1PBQueryDataPolicy(payloadStr string) (*v1pb.Policy_QueryDataPolicy, error) {
	payload := &storepb.QueryDataPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadStr), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal query data policy payload")
	}
	return &v1pb.Policy_QueryDataPolicy{
		QueryDataPolicy: &v1pb.QueryDataPolicy{
			Timeout: payload.Timeout,
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

func convertToQueryDataPolicyPayload(policy *v1pb.QueryDataPolicy) (*storepb.QueryDataPolicy, error) {
	return &storepb.QueryDataPolicy{
		Timeout: policy.Timeout,
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
			SemanticType: rule.SemanticType,
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
			SemanticType: rule.SemanticType,
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
			Action: convertToStorePBAction(exception.Action),
			Member: member,
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
			Action: convertToV1PBAction(exception.Action),
			Member: memberInBinding,
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

func convertV1PBToStorePBPolicyType(pType v1pb.PolicyType) (storepb.PolicyType, error) {
	switch pType {
	case v1pb.PolicyType_ROLLOUT_POLICY:
		return storepb.PolicyType_ROLLOUT, nil
	case v1pb.PolicyType_TAG:
		return storepb.PolicyType_TAG, nil
	case v1pb.PolicyType_MASKING_RULE:
		return storepb.PolicyType_MASKING_RULE, nil
	case v1pb.PolicyType_MASKING_EXCEPTION:
		return storepb.PolicyType_MASKING_EXCEPTION, nil
	case v1pb.PolicyType_DISABLE_COPY_DATA:
		return storepb.PolicyType_DISABLE_COPY_DATA, nil
	case v1pb.PolicyType_DATA_EXPORT:
		return storepb.PolicyType_EXPORT_DATA, nil
	case v1pb.PolicyType_DATA_QUERY:
		return storepb.PolicyType_QUERY_DATA, nil
	case v1pb.PolicyType_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW:
		return storepb.PolicyType_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW, nil
	case v1pb.PolicyType_DATA_SOURCE_QUERY:
		return storepb.PolicyType_DATA_SOURCE_QUERY, nil
	}
	return storepb.PolicyType_POLICY_TYPE_UNSPECIFIED, errors.Errorf("invalid policy type %v", pType)
}

func convertStorePBToV1PBPolicyType(pType storepb.PolicyType) v1pb.PolicyType {
	switch pType {
	case storepb.PolicyType_ROLLOUT:
		return v1pb.PolicyType_ROLLOUT_POLICY
	case storepb.PolicyType_TAG:
		return v1pb.PolicyType_TAG
	case storepb.PolicyType_MASKING_RULE:
		return v1pb.PolicyType_MASKING_RULE
	case storepb.PolicyType_MASKING_EXCEPTION:
		return v1pb.PolicyType_MASKING_EXCEPTION
	case storepb.PolicyType_DISABLE_COPY_DATA:
		return v1pb.PolicyType_DISABLE_COPY_DATA
	case storepb.PolicyType_EXPORT_DATA:
		return v1pb.PolicyType_DATA_EXPORT
	case storepb.PolicyType_QUERY_DATA:
		return v1pb.PolicyType_DATA_QUERY
	case storepb.PolicyType_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW:
		return v1pb.PolicyType_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW
	case storepb.PolicyType_DATA_SOURCE_QUERY:
		return v1pb.PolicyType_DATA_SOURCE_QUERY
	}
	return v1pb.PolicyType_POLICY_TYPE_UNSPECIFIED
}
