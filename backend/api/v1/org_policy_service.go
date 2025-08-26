package v1

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
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
		storepb.Policy_ROLLOUT:                                {storepb.Policy_ENVIRONMENT},
		storepb.Policy_TAG:                                    {storepb.Policy_ENVIRONMENT, storepb.Policy_PROJECT},
		storepb.Policy_DISABLE_COPY_DATA:                      {storepb.Policy_ENVIRONMENT, storepb.Policy_PROJECT},
		storepb.Policy_EXPORT_DATA:                            {storepb.Policy_WORKSPACE},
		storepb.Policy_QUERY_DATA:                             {storepb.Policy_WORKSPACE},
		storepb.Policy_MASKING_RULE:                           {storepb.Policy_WORKSPACE},
		storepb.Policy_MASKING_EXCEPTION:                      {storepb.Policy_PROJECT},
		storepb.Policy_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW: {storepb.Policy_WORKSPACE, storepb.Policy_PROJECT},
		storepb.Policy_IAM:                                    {storepb.Policy_WORKSPACE},
		storepb.Policy_DATA_SOURCE_QUERY:                      {storepb.Policy_ENVIRONMENT, storepb.Policy_PROJECT},
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
	policy, _, err := s.findPolicyMessage(ctx, req.Msg.Name)
	if err != nil {
		return nil, err
	}

	response, err := s.convertToPolicy(ctx, policy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(response), nil
}

// ListPolicies lists policies in a specific resource.
func (s *OrgPolicyService) ListPolicies(ctx context.Context, req *connect.Request[v1pb.ListPoliciesRequest]) (*connect.Response[v1pb.ListPoliciesResponse], error) {
	resourceType, resource, err := getPolicyResourceTypeAndResource(req.Msg.Parent)
	if err != nil {
		return nil, err
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

	policies, err := s.store.ListPoliciesV2(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1pb.ListPoliciesResponse{}
	for _, policy := range policies {
		p, err := s.convertToPolicy(ctx, policy)
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
			"masking_exception_policy",
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
			payloadStr, err := s.convertPolicyPayloadToString(ctx, req.Msg.Policy)
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

	p, err := s.store.UpdatePolicyV2(ctx, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response, err := s.convertToPolicy(ctx, p)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(response), nil
}

func pathMatchType(path string, policyType storepb.Policy_Type) bool {
	switch policyType {
	case storepb.Policy_ROLLOUT:
		return path == "rollout_policy"
	case storepb.Policy_DISABLE_COPY_DATA:
		return path == "disable_copy_data_policy"
	case storepb.Policy_MASKING_RULE:
		return path == "masking_rule_policy"
	case storepb.Policy_MASKING_EXCEPTION:
		return path == "masking_exception_policy"
	case storepb.Policy_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW:
		return path == "restrict_issue_creation_for_sql_review_policy"
	case storepb.Policy_TAG:
		return path == "tag_policy"
	case storepb.Policy_DATA_SOURCE_QUERY:
		return path == "data_source_query_policy"
	case storepb.Policy_EXPORT_DATA:
		return path == "export_data_policy"
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

	if err := s.store.DeletePolicyV2(ctx, policy); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
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
	resourceType, resource, err := getPolicyResourceTypeAndResource(policyParent)
	if err != nil {
		return nil, policyParent, err
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

	policy, err := s.store.GetPolicyV2(ctx, &store.FindPolicyMessage{
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

func getPolicyResourceTypeAndResource(requestName string) (storepb.Policy_Resource, *string, error) {
	if requestName == "" {
		return storepb.Policy_WORKSPACE, nil, nil
	}

	if strings.HasPrefix(requestName, common.ProjectNamePrefix) {
		projectID, err := common.GetProjectID(requestName)
		if err != nil {
			return storepb.Policy_RESOURCE_UNSPECIFIED, nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if projectID == "-" {
			return storepb.Policy_PROJECT, nil, nil
		}
		return storepb.Policy_PROJECT, &requestName, nil
	}

	if strings.HasPrefix(requestName, common.EnvironmentNamePrefix) {
		// environment policy request name should be environments/{environment id}
		environmentID, err := common.GetEnvironmentID(requestName)
		if err != nil {
			return storepb.Policy_RESOURCE_UNSPECIFIED, nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if environmentID == "-" {
			return storepb.Policy_ENVIRONMENT, nil, nil
		}
		return storepb.Policy_ENVIRONMENT, &requestName, nil
	}

	return storepb.Policy_RESOURCE_UNSPECIFIED, nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unknown request name %s", requestName))
}

func (s *OrgPolicyService) createPolicyMessage(ctx context.Context, parent string, policy *v1pb.Policy) (*v1pb.Policy, error) {
	resourceType, _, err := getPolicyResourceTypeAndResource(parent)
	if err != nil {
		return nil, err
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
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response, err := s.convertToPolicy(ctx, p)
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
	case storepb.Policy_MASKING_EXCEPTION:
		maskingExceptionPolicy, ok := policy.Policy.(*v1pb.Policy_MaskingExceptionPolicy)
		if !ok {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unmatched policy type %v and policy %v", policyType, policy.Policy))
		}
		if maskingExceptionPolicy.MaskingExceptionPolicy == nil {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("masking exception policy must be set"))
		}
		for _, exception := range maskingExceptionPolicy.MaskingExceptionPolicy.MaskingExceptions {
			if exception.Action == v1pb.MaskingExceptionPolicy_MaskingException_ACTION_UNSPECIFIED {
				return connect.NewError(connect.CodeInvalidArgument, errors.New("masking exception must have action set"))
			}
			if _, err := common.ValidateMaskingExceptionCELExpr(exception.Condition); err != nil {
				return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid masking exception expression: %v", err))
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
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_RESTRICT_COPYING_DATA); err != nil {
			return "", connect.NewError(connect.CodePermissionDenied, err)
		}
		payload := convertToDisableCopyDataPolicyPayload(policy.GetDisableCopyDataPolicy())
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_DATA_EXPORT:
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_QUERY_POLICY); err != nil {
			return "", connect.NewError(connect.CodePermissionDenied, err)
		}
		payload := convertToExportDataPolicyPayload(policy.GetExportDataPolicy())
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_DATA_QUERY:
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
	case v1pb.PolicyType_MASKING_EXCEPTION:
		if err := s.licenseService.IsFeatureEnabled(v1pb.PlanFeature_FEATURE_DATA_MASKING); err != nil {
			return "", connect.NewError(connect.CodePermissionDenied, err)
		}
		payload, err := s.convertToStorePBMaskingExceptionPolicyPayload(ctx, policy.GetMaskingExceptionPolicy())
		if err != nil {
			return "", connect.NewError(connect.CodeInvalidArgument, err)
		}
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal masking exception policy")
		}
		return string(payloadBytes), nil
	case v1pb.PolicyType_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW:
		payload := convertToRestrictIssueCreationForSQLReviewPayload(policy.GetRestrictIssueCreationForSqlReviewPolicy())
		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return "", errors.Wrap(err, "failed to marshal restrict issue creation for SQL review policy")
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

func (s *OrgPolicyService) convertToPolicy(ctx context.Context, policyMessage *store.PolicyMessage) (*v1pb.Policy, error) {
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
	case storepb.Policy_DISABLE_COPY_DATA:
		payload, err := convertToV1PBDisableCopyDataPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
	case storepb.Policy_EXPORT_DATA:
		payload, err := convertToV1PBExportDataPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
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
	case storepb.Policy_MASKING_EXCEPTION:
		maskingRulePolicy := &storepb.MaskingExceptionPolicy{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policyMessage.Payload), maskingRulePolicy); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal masking exception policy")
		}
		payload := s.convertToV1PBMaskingExceptionPolicyPayload(ctx, maskingRulePolicy)
		policy.Policy = &v1pb.Policy_MaskingExceptionPolicy{
			MaskingExceptionPolicy: payload,
		}
	case storepb.Policy_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW:
		payload, err := convertToV1PBRestrictIssueCreationForSQLReviewPolicy(policyMessage.Payload)
		if err != nil {
			return nil, err
		}
		policy.Policy = payload
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
		default:
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
	default:
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
	default:
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

func convertToDisableCopyDataPolicyPayload(policy *v1pb.DisableCopyDataPolicy) *storepb.DisableCopyDataPolicy {
	return &storepb.DisableCopyDataPolicy{
		Active: policy.Active,
	}
}

func convertToExportDataPolicyPayload(policy *v1pb.ExportDataPolicy) *storepb.ExportDataPolicy {
	return &storepb.ExportDataPolicy{
		Disable: policy.Disable,
	}
}

func convertToQueryDataPolicyPayload(policy *v1pb.QueryDataPolicy) *storepb.QueryDataPolicy {
	return &storepb.QueryDataPolicy{
		Timeout: policy.Timeout,
	}
}

func convertToStorePBMskingRulePolicy(policy *v1pb.MaskingRulePolicy) *storepb.MaskingRulePolicy {
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
	}
}

func convertToV1PBMaskingRulePolicy(policy *storepb.MaskingRulePolicy) *v1pb.MaskingRulePolicy {
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
	}
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

func (s *OrgPolicyService) convertToV1PBMaskingExceptionPolicyPayload(ctx context.Context, policy *storepb.MaskingExceptionPolicy) *v1pb.MaskingExceptionPolicy {
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
	}
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

func convertToRestrictIssueCreationForSQLReviewPayload(policy *v1pb.RestrictIssueCreationForSQLReviewPolicy) *storepb.RestrictIssueCreationForSQLReviewPolicy {
	return &storepb.RestrictIssueCreationForSQLReviewPolicy{
		Disallow: policy.Disallow,
	}
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

func convertToDataSourceQueryPayload(policy *v1pb.DataSourceQueryPolicy) *storepb.DataSourceQueryPolicy {
	return &storepb.DataSourceQueryPolicy{
		AdminDataSourceRestriction: storepb.DataSourceQueryPolicy_Restriction(policy.AdminDataSourceRestriction),
		DisallowDdl:                policy.DisallowDdl,
		DisallowDml:                policy.DisallowDml,
	}
}

func convertV1PBToStorePBPolicyType(pType v1pb.PolicyType) (storepb.Policy_Type, error) {
	switch pType {
	case v1pb.PolicyType_ROLLOUT_POLICY:
		return storepb.Policy_ROLLOUT, nil
	case v1pb.PolicyType_TAG:
		return storepb.Policy_TAG, nil
	case v1pb.PolicyType_MASKING_RULE:
		return storepb.Policy_MASKING_RULE, nil
	case v1pb.PolicyType_MASKING_EXCEPTION:
		return storepb.Policy_MASKING_EXCEPTION, nil
	case v1pb.PolicyType_DISABLE_COPY_DATA:
		return storepb.Policy_DISABLE_COPY_DATA, nil
	case v1pb.PolicyType_DATA_EXPORT:
		return storepb.Policy_EXPORT_DATA, nil
	case v1pb.PolicyType_DATA_QUERY:
		return storepb.Policy_QUERY_DATA, nil
	case v1pb.PolicyType_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW:
		return storepb.Policy_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW, nil
	case v1pb.PolicyType_DATA_SOURCE_QUERY:
		return storepb.Policy_DATA_SOURCE_QUERY, nil
	default:
	}
	return storepb.Policy_TYPE_UNSPECIFIED, errors.Errorf("invalid policy type %v", pType)
}

func convertStorePBToV1PBPolicyType(pType storepb.Policy_Type) v1pb.PolicyType {
	switch pType {
	case storepb.Policy_ROLLOUT:
		return v1pb.PolicyType_ROLLOUT_POLICY
	case storepb.Policy_TAG:
		return v1pb.PolicyType_TAG
	case storepb.Policy_MASKING_RULE:
		return v1pb.PolicyType_MASKING_RULE
	case storepb.Policy_MASKING_EXCEPTION:
		return v1pb.PolicyType_MASKING_EXCEPTION
	case storepb.Policy_DISABLE_COPY_DATA:
		return v1pb.PolicyType_DISABLE_COPY_DATA
	case storepb.Policy_EXPORT_DATA:
		return v1pb.PolicyType_DATA_EXPORT
	case storepb.Policy_QUERY_DATA:
		return v1pb.PolicyType_DATA_QUERY
	case storepb.Policy_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW:
		return v1pb.PolicyType_RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW
	case storepb.Policy_DATA_SOURCE_QUERY:
		return v1pb.PolicyType_DATA_SOURCE_QUERY
	default:
	}
	return v1pb.PolicyType_POLICY_TYPE_UNSPECIFIED
}
