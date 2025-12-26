package v1

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

var (
	// allowedResourceTypes includes allowed resource types for each policy type.
	allowedResourceTypes = map[storepb.Policy_Type][]storepb.Policy_Resource{
		storepb.Policy_ROLLOUT:           {storepb.Policy_ENVIRONMENT},
		storepb.Policy_TAG:               {storepb.Policy_ENVIRONMENT, storepb.Policy_PROJECT},
		storepb.Policy_QUERY_DATA:        {storepb.Policy_WORKSPACE, storepb.Policy_ENVIRONMENT, storepb.Policy_PROJECT},
		storepb.Policy_MASKING_RULE:      {storepb.Policy_WORKSPACE},
		storepb.Policy_MASKING_EXEMPTION: {storepb.Policy_PROJECT},
		storepb.Policy_IAM:               {storepb.Policy_WORKSPACE},
		storepb.Policy_DATA_SOURCE_QUERY: {storepb.Policy_ENVIRONMENT, storepb.Policy_PROJECT},
	}
)

// OrgPolicyService implements the workspace policy service.
type OrgPolicyService struct {
	v1connect.UnimplementedOrgPolicyServiceHandler
	store          *store.Store
	licenseService *enterprise.LicenseService
}

// NewOrgPolicyService creates a new OrgPolicyService.
func NewOrgPolicyService(store *store.Store, licenseService *enterprise.LicenseService) *OrgPolicyService {
	return &OrgPolicyService{
		store:          store,
		licenseService: licenseService,
	}
}

// GetPolicy gets a policy in a specific resource.
func (s *OrgPolicyService) GetPolicy(ctx context.Context, req *connect.Request[v1pb.GetPolicyRequest]) (*connect.Response[v1pb.Policy], error) {
	policy, parent, err := s.findPolicyMessage(ctx, req.Msg.Name)
	if err != nil {
		connectErr := connect.CodeOf(err)
		// For ROLLOUT_POLICY, return default policy if not found
		if connectErr == connect.CodeNotFound {
			policyType, extractErr := extractPolicyTypeFromName(req.Msg.Name)
			if extractErr == nil && policyType == storepb.Policy_ROLLOUT {
				defaultPolicy, defaultErr := s.getDefaultRolloutPolicy(parent, req.Msg.Name)
				if defaultErr != nil {
					return nil, defaultErr
				}
				return connect.NewResponse(defaultPolicy), nil
			}
		}
		return nil, err
	}

	response, err := convertToPolicy(policy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(response), nil
}

// ListPolicies lists policies in a specific resource.
func (s *OrgPolicyService) ListPolicies(ctx context.Context, req *connect.Request[v1pb.ListPoliciesRequest]) (*connect.Response[v1pb.ListPoliciesResponse], error) {
	resourceType, resource, err := common.GetPolicyResourceTypeAndResource(req.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	find := &store.FindPolicyMessage{
		ResourceType: &resourceType,
		Resource:     resource,
		ShowAll:      req.Msg.ShowDeleted,
	}

	if v := req.Msg.PolicyType; v != nil {
		policyType, err := convertV1PBToStorePBPolicyType(*v)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		find.Type = &policyType
	}

	policies, err := s.store.ListPolicies(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1pb.ListPoliciesResponse{}
	for _, policy := range policies {
		p, err := convertToPolicy(policy)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if p.Type == v1pb.PolicyType_POLICY_TYPE_UNSPECIFIED {
			// skip unknown type policy and environment tier policy
			continue
		}
		response.Policies = append(response.Policies, p)
	}
	return connect.NewResponse(response), nil
}

// CreatePolicy creates a policy in a specific resource.
func (s *OrgPolicyService) CreatePolicy(ctx context.Context, req *connect.Request[v1pb.CreatePolicyRequest]) (*connect.Response[v1pb.Policy], error) {
	if req.Msg.Policy == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("policy must be set"))
	}

	if err := s.checkPolicyFeatureGuard(req.Msg.Policy.Type); err != nil {
		return nil, err
	}

	// TODO(d): validate policy.
	response, err := s.createPolicyMessage(ctx, req.Msg.Parent, req.Msg.Policy)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

// UpdatePolicy updates a policy in a specific resource.
func (s *OrgPolicyService) UpdatePolicy(ctx context.Context, req *connect.Request[v1pb.UpdatePolicyRequest]) (*connect.Response[v1pb.Policy], error) {
	if req.Msg.Policy == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("policy must be set"))
	}

	if err := s.checkPolicyFeatureGuard(req.Msg.Policy.Type); err != nil {
		return nil, err
	}

	policy, parent, err := s.findPolicyMessage(ctx, req.Msg.Policy.Name)
	if err != nil {
		connectErr := connect.CodeOf(err)
		if connectErr == connect.CodeNotFound && req.Msg.AllowMissing {
			response, err := s.createPolicyMessage(ctx, parent, req.Msg.Policy)
			if err != nil {
				return nil, err
			}
			return connect.NewResponse(response), nil
		}
		return nil, err
	}

	if req.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask must be set"))
	}

	patch := &store.UpdatePolicyMessage{
		ResourceType: policy.ResourceType,
		Type:         policy.Type,
		Resource:     policy.Resource,
	}
	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "inherit_from_parent":
			patch.InheritFromParent = &req.Msg.Policy.InheritFromParent
		case
			"rollout_policy",
			"disable_copy_data_policy",
			"masking_rule_policy",
			"masking_exemption_policy",
			"restrict_issue_creation_for_sql_review_policy",
			"tag_policy",
			"data_source_query_policy",
			"export_data_policy",
			"query_data_policy":
			if !pathMatchType(path, policy.Type) {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid path %s for policy type %s", path, policy.Type.String()))
			}
			if err := validatePolicyPayload(policy.Type, req.Msg.Policy); err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "invalid policy"))
			}
			payloadStr, err := s.convertPolicyPayloadToString(req.Msg.Policy)
			if err != nil {
				return nil, err
			}
			patch.Payload = &payloadStr
		case "enforce":
			patch.Enforce = &req.Msg.Policy.Enforce
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected path %s", path))
		}
	}

	p, err := s.store.UpdatePolicy(ctx, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response, err := convertToPolicy(p)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(response), nil
}

func pathMatchType(path string, policyType storepb.Policy_Type) bool {
	switch policyType {
	case storepb.Policy_ROLLOUT:
		return path == "rollout_policy"
	case storepb.Policy_MASKING_RULE:
		return path == "masking_rule_policy"
	case storepb.Policy_MASKING_EXEMPTION:
		return path == "masking_exemption_policy"
	case storepb.Policy_TAG:
		return path == "tag_policy"
	case storepb.Policy_DATA_SOURCE_QUERY:
		return path == "data_source_query_policy"
	case storepb.Policy_QUERY_DATA:
		return path == "query_data_policy"
	default:
		return false
	}
}

// DeletePolicy deletes a policy for a specific resource.
func (s *OrgPolicyService) DeletePolicy(ctx context.Context, req *connect.Request[v1pb.DeletePolicyRequest]) (*connect.Response[emptypb.Empty], error) {
	policy, _, err := s.findPolicyMessage(ctx, req.Msg.Name)
	if err != nil {
		return nil, err
	}

	if err := s.store.DeletePolicy(ctx, policy); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// extractPolicyTypeFromName extracts the policy type from a policy name.
func extractPolicyTypeFromName(policyName string) (storepb.Policy_Type, error) {
	tokens := strings.Split(policyName, common.PolicyNamePrefix)
	if len(tokens) != 2 {
		return storepb.Policy_TYPE_UNSPECIFIED, errors.Errorf("invalid policy name %s", policyName)
	}

	v1PolicyType, ok := v1pb.PolicyType_value[strings.ToUpper(tokens[1])]
	if !ok {
		return storepb.Policy_TYPE_UNSPECIFIED, errors.Errorf("invalid policy type %v", tokens[1])
	}

	return convertV1PBToStorePBPolicyType(v1pb.PolicyType(v1PolicyType))
}

// getDefaultRolloutPolicy returns the default rollout policy when no custom policy exists.
// Uses the shared store.GetDefaultRolloutPolicy to ensure consistency across API and store layers.
func (*OrgPolicyService) getDefaultRolloutPolicy(parent string, policyName string) (*v1pb.Policy, error) {
	resourceType, _, err := common.GetPolicyResourceTypeAndResource(parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	v1ResourceType := v1pb.PolicyResourceType_RESOURCE_TYPE_UNSPECIFIED
	switch resourceType {
	case storepb.Policy_WORKSPACE:
		v1ResourceType = v1pb.PolicyResourceType_WORKSPACE
	case storepb.Policy_ENVIRONMENT:
		v1ResourceType = v1pb.PolicyResourceType_ENVIRONMENT
	case storepb.Policy_PROJECT:
		v1ResourceType = v1pb.PolicyResourceType_PROJECT
	default:
		// Keep the default RESOURCE_TYPE_UNSPECIFIED
	}

	// Get the default rollout policy from the shared store function
	defaultStorePBPolicy := store.GetDefaultRolloutPolicy()

	return &v1pb.Policy{
		Name:              policyName,
		ResourceType:      v1ResourceType,
		Type:              v1pb.PolicyType_ROLLOUT_POLICY,
		InheritFromParent: false,
		Enforce:           true,
		Policy: &v1pb.Policy_RolloutPolicy{
			RolloutPolicy: convertToV1PBRolloutPolicy(defaultStorePBPolicy),
		},
	}, nil
}

// findPolicyMessage finds the policy and the parent name by the policy name.
func (s *OrgPolicyService) findPolicyMessage(ctx context.Context, policyName string) (*store.PolicyMessage, string, error) {
	tokens := strings.Split(policyName, common.PolicyNamePrefix)
	if len(tokens) != 2 {
		return nil, "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid request %s", policyName))
	}

	policyParent := tokens[0]
	if strings.HasSuffix(policyParent, "/") {
		policyParent = policyParent[:(len(policyParent) - 1)]
	}
	resourceType, resource, err := common.GetPolicyResourceTypeAndResource(policyParent)
	if err != nil {
		return nil, policyParent, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if resource == nil && resourceType != storepb.Policy_WORKSPACE {
		return nil, policyParent, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("resource for %s must be specific", resourceType))
	}

	// Parse the policy type from the string in the policy name
	v1PolicyType, ok := v1pb.PolicyType_value[strings.ToUpper(tokens[1])]
	if !ok {
		return nil, policyParent, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid policy type %v", tokens[1]))
	}

	policyType, err := convertV1PBToStorePBPolicyType(v1pb.PolicyType(v1PolicyType))
	if err != nil {
		return nil, policyParent, connect.NewError(connect.CodeInvalidArgument, err)
	}

	policy, err := s.store.GetPolicy(ctx, &store.FindPolicyMessage{
		ResourceType: &resourceType,
		Type:         &policyType,
		Resource:     resource,
	})
	if err != nil {
		return nil, policyParent, connect.NewError(connect.CodeInternal, err)
	}
	if policy == nil {
		return nil, policyParent, connect.NewError(connect.CodeNotFound, errors.Errorf("policy %q not found", policyName))
	}

	return policy, policyParent, nil
}

func (s *OrgPolicyService) createPolicyMessage(ctx context.Context, parent string, policy *v1pb.Policy) (*v1pb.Policy, error) {
	resourceType, _, err := common.GetPolicyResourceTypeAndResource(parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	policyType, err := convertV1PBToStorePBPolicyType(policy.Type)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := validatePolicyType(policyType, resourceType); err != nil {
		return nil, err
	}

	if err := validatePolicyPayload(policyType, policy); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "invalid policy"))
	}

	payloadStr, err := s.convertPolicyPayloadToString(policy)
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

	p, err := s.store.CreatePolicy(ctx, create)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response, err := convertToPolicy(p)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return response, nil
}

func validatePolicyType(policyType storepb.Policy_Type, policyResourceType storepb.Policy_Resource) error {
	allowedTypes, ok := allowedResourceTypes[policyType]
	if !ok {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unknown policy type %v", policyType))
	}
	for _, rt := range allowedTypes {
		if rt == policyResourceType {
			return nil
		}
	}
	return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("policy %v is not allowed in resource %v", policyType, policyResourceType))
}

func (s *OrgPolicyService) checkPolicyFeatureGuard(policyType v1pb.PolicyType) error {
	if policyType == v1pb.PolicyType_DATA_QUERY || policyType == v1pb.PolicyType_DATA_SOURCE_QUERY {
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_QUERY_POLICY); err != nil {
			return connect.NewError(connect.CodePermissionDenied, err)
		}
	}
	return nil
}

func validatePolicyPayload(policyType storepb.Policy_Type, policy *v1pb.Policy) error {
	switch policyType {
	case storepb.Policy_MASKING_RULE:
		maskingRulePolicy, ok := policy.Policy.(*v1pb.Policy_MaskingRulePolicy)
		if !ok {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unmatched policy type %v and policy %v", policyType, policy.Policy))
		}
		if maskingRulePolicy.MaskingRulePolicy == nil {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("masking rule policy must be set"))
		}
		for _, rule := range maskingRulePolicy.MaskingRulePolicy.Rules {
			if rule.Id == "" {
				return connect.NewError(connect.CodeInvalidArgument, errors.New("masking rule must have ID set"))
			}
			if _, err := common.ValidateMaskingRuleCELExpr(rule.Condition.Expression); err != nil {
				return connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "invalid masking rule expression"))
			}
		}
	case storepb.Policy_MASKING_EXEMPTION:
		maskingExemptionPolicy, ok := policy.Policy.(*v1pb.Policy_MaskingExemptionPolicy)
		if !ok {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unmatched policy type %v and policy %v", policyType, policy.Policy))
		}
		if maskingExemptionPolicy.MaskingExemptionPolicy == nil {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("masking exception policy must be set"))
		}
		for _, exemption := range maskingExemptionPolicy.MaskingExemptionPolicy.Exemptions {
			if _, err := common.ValidateMaskingExemptionCELExpr(exemption.Condition); err != nil {
				return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid masking exemption expression: %v", err))
			}
			for _, member := range exemption.Members {
				if err := validateMember(member); err != nil {
					return err
				}
			}
		}
	default:
	}
	return nil
}

func (s *OrgPolicyService) convertPolicyPayloadToString(policy *v1pb.Policy) (string, error) {
	switch policy.Type {
	case v1pb.PolicyType_ROLLOUT_POLICY:
		rolloutPolicy := convertToStorePBRolloutPolicy(policy.GetRolloutPolicy())
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
	case v1pb.PolicyType_DATA_QUERY:
		// Check license for both query policy and restrict copying data features
		if policy.GetQueryDataPolicy() != nil && policy.GetQueryDataPolicy().DisableCopyData {
			if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_RESTRICT_COPYING_DATA); err != nil {
				return "", connect.NewError(connect.CodePermissionDenied, err)
			}
		}
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_QUERY_POLICY); err != nil {
			return "", connect.NewError(connect.CodePermissionDenied, err)
		}
		payload := convertToQueryDataPolicyPayload(policy.GetQueryDataPolicy())
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_MASKING_RULE:
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DATA_MASKING); err != nil {
			return "", connect.NewError(connect.CodePermissionDenied, err)
		}
		payload := convertToStorePBMskingRulePolicy(policy.GetMaskingRulePolicy())
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal masking rule policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_MASKING_EXEMPTION:
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DATA_MASKING); err != nil {
			return "", connect.NewError(connect.CodePermissionDenied, err)
		}
		payload, err := convertToStorePBMaskingExemptionPolicyPayload(policy.GetMaskingExemptionPolicy())
		if err != nil {
			return "", connect.NewError(connect.CodeInvalidArgument, err)
		}
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal masking exemption policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_DATA_SOURCE_QUERY:
		payload := convertToDataSourceQueryPayload(policy.GetDataSourceQueryPolicy())
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal data source query policy")
		}
		return string(payloadBytes), nil
	default:
	}

	return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid policy %v", policy.Type))
}

func convertToPolicy(policyMessage *store.PolicyMessage) (*v1pb.Policy, error) {
	resourceType := v1pb.PolicyResourceType_RESOURCE_TYPE_UNSPECIFIED
	switch policyMessage.ResourceType {
	case storepb.Policy_WORKSPACE:
		resourceType = v1pb.PolicyResourceType_WORKSPACE
	case storepb.Policy_ENVIRONMENT:
		resourceType = v1pb.PolicyResourceType_ENVIRONMENT
	case storepb.Policy_PROJECT:
		resourceType = v1pb.PolicyResourceType_PROJECT
	default:
	}
	policy := &v1pb.Policy{
		InheritFromParent: policyMessage.InheritFromParent,
		Enforce:           policyMessage.Enforce,
		ResourceType:      resourceType,
	}

	pType := convertStorePBToV1PBPolicyType(policyMessage.Type)
	switch policyMessage.Type {
	case storepb.Policy_ROLLOUT:
		payload, err := convertToV1RolloutPolicyPayload(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case storepb.Policy_TAG:
		p := &v1pb.TagPolicy{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policyMessage.Payload), p); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal tag policy payload")
		}
		policy.Policy = &v1pb.Policy_TagPolicy{
			TagPolicy: p,
		}
	case storepb.Policy_QUERY_DATA:
		payload, err := convertToV1PBQueryDataPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case storepb.Policy_MASKING_RULE:
		maskingRulePolicy := &storepb.MaskingRulePolicy{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policyMessage.Payload), maskingRulePolicy); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal masking rule policy")
		}
		payload := convertToV1PBMaskingRulePolicy(maskingRulePolicy)
		policy.Policy = &v1pb.Policy_MaskingRulePolicy{
			MaskingRulePolicy: payload,
		}
	case storepb.Policy_MASKING_EXEMPTION:
		maskingExemptionPolicy := &storepb.MaskingExemptionPolicy{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policyMessage.Payload), maskingExemptionPolicy); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal masking exemption policy")
		}
		payload := convertToV1PBMaskingExemptionPolicyPayload(maskingExemptionPolicy)
		policy.Policy = &v1pb.Policy_MaskingExemptionPolicy{
			MaskingExemptionPolicy: payload,
		}
	case storepb.Policy_DATA_SOURCE_QUERY:
		payload, err := convertToV1PBDataSourceQueryPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	default:
	}

	policy.Type = pType
	policy.Name = fmt.Sprintf("%s%s", common.PolicyNamePrefix, strings.ToLower(pType.String()))
	if policyMessage.Resource != "" {
		policy.Name = fmt.Sprintf("%s/%s", policyMessage.Resource, policy.Name)
	}

	return policy, nil
}
