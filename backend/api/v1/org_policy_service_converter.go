package v1

import (
	"slices"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// ConvertToV1PBSQLReviewRules converts store SQL review rules to v1 API format.
func ConvertToV1PBSQLReviewRules(ruleList []*storepb.SQLReviewRule) []*v1pb.SQLReviewRule {
	var rules []*v1pb.SQLReviewRule
	for _, rule := range ruleList {
		level := v1pb.SQLReviewRule_LEVEL_UNSPECIFIED
		switch rule.Level {
		case storepb.SQLReviewRule_ERROR:
			level = v1pb.SQLReviewRule_ERROR
		case storepb.SQLReviewRule_WARNING:
			level = v1pb.SQLReviewRule_WARNING
		default:
		}

		v1Rule := &v1pb.SQLReviewRule{
			Level:  level,
			Type:   v1pb.SQLReviewRule_Type(rule.Type),
			Engine: convertToEngine(rule.Engine),
		}

		// Convert typed payload from store to v1 API
		switch payload := rule.Payload.(type) {
		case *storepb.SQLReviewRule_NamingPayload:
			v1Rule.Payload = &v1pb.SQLReviewRule_NamingPayload{
				NamingPayload: &v1pb.SQLReviewRule_NamingRulePayload{
					MaxLength: payload.NamingPayload.MaxLength,
					Format:    payload.NamingPayload.Format,
				},
			}
		case *storepb.SQLReviewRule_NumberPayload:
			v1Rule.Payload = &v1pb.SQLReviewRule_NumberPayload{
				NumberPayload: &v1pb.SQLReviewRule_NumberRulePayload{
					Number: payload.NumberPayload.Number,
				},
			}
		case *storepb.SQLReviewRule_StringArrayPayload:
			v1Rule.Payload = &v1pb.SQLReviewRule_StringArrayPayload{
				StringArrayPayload: &v1pb.SQLReviewRule_StringArrayRulePayload{
					List: payload.StringArrayPayload.List,
				},
			}
		case *storepb.SQLReviewRule_CommentConventionPayload:
			v1Rule.Payload = &v1pb.SQLReviewRule_CommentConventionPayload{
				CommentConventionPayload: &v1pb.SQLReviewRule_CommentConventionRulePayload{
					Required:  payload.CommentConventionPayload.Required,
					MaxLength: payload.CommentConventionPayload.MaxLength,
				},
			}
		case *storepb.SQLReviewRule_NamingCasePayload:
			v1Rule.Payload = &v1pb.SQLReviewRule_NamingCasePayload{
				NamingCasePayload: &v1pb.SQLReviewRule_NamingCaseRulePayload{
					Upper: payload.NamingCasePayload.Upper,
				},
			}
		case *storepb.SQLReviewRule_StringPayload:
			v1Rule.Payload = &v1pb.SQLReviewRule_StringPayload{
				StringPayload: &v1pb.SQLReviewRule_StringRulePayload{
					Value: payload.StringPayload.Value,
				},
			}
		default:
		}

		rules = append(rules, v1Rule)
	}

	return rules
}

// ConvertToSQLReviewRules converts v1 API SQL review rules to store format.
func ConvertToSQLReviewRules(rules []*v1pb.SQLReviewRule) ([]*storepb.SQLReviewRule, error) {
	var ruleList []*storepb.SQLReviewRule
	for _, rule := range rules {
		var level storepb.SQLReviewRule_Level
		switch rule.Level {
		case v1pb.SQLReviewRule_ERROR:
			level = storepb.SQLReviewRule_ERROR
		case v1pb.SQLReviewRule_WARNING:
			level = storepb.SQLReviewRule_WARNING
		default:
			return nil, errors.Errorf("invalid rule level %v", rule.Level)
		}

		storeRule := &storepb.SQLReviewRule{
			Type:   storepb.SQLReviewRule_Type(rule.Type),
			Level:  level,
			Engine: convertEngine(rule.Engine),
		}

		// Convert typed payload from v1 API to store
		switch payload := rule.Payload.(type) {
		case *v1pb.SQLReviewRule_NamingPayload:
			storeRule.Payload = &storepb.SQLReviewRule_NamingPayload{
				NamingPayload: &storepb.SQLReviewRule_NamingRulePayload{
					MaxLength: payload.NamingPayload.MaxLength,
					Format:    payload.NamingPayload.Format,
				},
			}
		case *v1pb.SQLReviewRule_NumberPayload:
			storeRule.Payload = &storepb.SQLReviewRule_NumberPayload{
				NumberPayload: &storepb.SQLReviewRule_NumberRulePayload{
					Number: payload.NumberPayload.Number,
				},
			}
		case *v1pb.SQLReviewRule_StringArrayPayload:
			storeRule.Payload = &storepb.SQLReviewRule_StringArrayPayload{
				StringArrayPayload: &storepb.SQLReviewRule_StringArrayRulePayload{
					List: payload.StringArrayPayload.List,
				},
			}
		case *v1pb.SQLReviewRule_CommentConventionPayload:
			storeRule.Payload = &storepb.SQLReviewRule_CommentConventionPayload{
				CommentConventionPayload: &storepb.SQLReviewRule_CommentConventionRulePayload{
					Required:  payload.CommentConventionPayload.Required,
					MaxLength: payload.CommentConventionPayload.MaxLength,
				},
			}
		case *v1pb.SQLReviewRule_NamingCasePayload:
			storeRule.Payload = &storepb.SQLReviewRule_NamingCasePayload{
				NamingCasePayload: &storepb.SQLReviewRule_NamingCaseRulePayload{
					Upper: payload.NamingCasePayload.Upper,
				},
			}
		case *v1pb.SQLReviewRule_StringPayload:
			storeRule.Payload = &storepb.SQLReviewRule_StringPayload{
				StringPayload: &storepb.SQLReviewRule_StringRulePayload{
					Value: payload.StringPayload.Value,
				},
			}
		default:
		}

		ruleList = append(ruleList, storeRule)
	}

	return ruleList, nil
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
		Automatic: policy.Automatic,
		Roles:     policy.Roles,
	}
}

// convertToV1PBQueryDataPolicy returns the stored policy, with the default
// export destination policy on a workspace policy that has none.
func convertToV1PBQueryDataPolicy(payloadStr string, resourceType storepb.Policy_Resource) (*v1pb.Policy_QueryDataPolicy, error) {
	payload := &storepb.QueryDataPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadStr), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal query data policy payload")
	}
	policy := &v1pb.QueryDataPolicy{
		MaximumResultRows:    payload.MaximumResultRows,
		DisableCopyData:      payload.DisableCopyData,
		AllowAdminDataSource: payload.AllowAdminDataSource,
	}
	if resourceType == storepb.Policy_WORKSPACE {
		var allowed []v1pb.ExportDestination
		for _, destination := range store.GetAllowedExportDestinations(payload) {
			allowed = append(allowed, convertToV1ExportDestination(destination))
		}
		policy.ExportDestinationPolicy = &v1pb.QueryDataPolicy_ExportDestinationPolicy{AllowedDestinations: allowed}
	}
	return &v1pb.Policy_QueryDataPolicy{QueryDataPolicy: policy}, nil
}

// convertToQueryDataPolicyPayload rejects disable_export=true so stale configs learn the
// setting is gone; false is accepted and dropped.
func convertToQueryDataPolicyPayload(policy *v1pb.QueryDataPolicy) (*storepb.QueryDataPolicy, error) {
	if policy.GetDisableExport() { //nolint:staticcheck // The deprecated field is read only to reject it.
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("disable_export is removed. Export requires the bb.sql.export permission; to require approval for export, assign roles without it.")) //nolint:revive // The message is an API contract.
	}
	payload := &storepb.QueryDataPolicy{
		MaximumResultRows:    policy.GetMaximumResultRows(),
		DisableCopyData:      policy.GetDisableCopyData(),
		AllowAdminDataSource: policy.GetAllowAdminDataSource(),
	}
	if exportDestinationPolicy := policy.GetExportDestinationPolicy(); exportDestinationPolicy != nil {
		allowed, err := convertToStoreAllowedExportDestinations(exportDestinationPolicy.GetAllowedDestinations())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		payload.ExportDestinationPolicy = &storepb.QueryDataPolicy_ExportDestinationPolicy{AllowedDestinations: allowed}
	}
	return payload, nil
}

// convertToStoreAllowedExportDestinations accepts an empty list, which turns
// off export for everyone, including holders of bb.sql.export or an export grant.
func convertToStoreAllowedExportDestinations(destinations []v1pb.ExportDestination) ([]storepb.ExportDestination, error) {
	var allowed []storepb.ExportDestination
	for _, destination := range destinations {
		storeDestination, err := convertToStoreExportDestination(destination)
		if err != nil {
			return nil, err
		}
		if slices.Contains(allowed, storeDestination) {
			return nil, errors.Errorf("export_destination_policy.allowed_destinations lists %s more than once", destination)
		}
		allowed = append(allowed, storeDestination)
	}
	return allowed, nil
}

func convertToStoreExportDestination(destination v1pb.ExportDestination) (storepb.ExportDestination, error) {
	switch destination {
	case v1pb.ExportDestination_DOWNLOAD:
		return storepb.ExportDestination_DOWNLOAD, nil
	case v1pb.ExportDestination_SHAREPOINT:
		return storepb.ExportDestination_SHAREPOINT, nil
	default:
		return storepb.ExportDestination_EXPORT_DESTINATION_UNSPECIFIED, errors.Errorf("unsupported export destination %v", destination)
	}
}

func convertToV1ExportDestination(destination storepb.ExportDestination) v1pb.ExportDestination {
	switch destination {
	case storepb.ExportDestination_DOWNLOAD:
		return v1pb.ExportDestination_DOWNLOAD
	case storepb.ExportDestination_SHAREPOINT:
		return v1pb.ExportDestination_SHAREPOINT
	default:
		return v1pb.ExportDestination_EXPORT_DESTINATION_UNSPECIFIED
	}
}

// keepStoredExportDestinationPolicy carries the stored export destination
// policy into an updated query data policy that omits it, so a client that does
// not know the field cannot clear it.
func keepStoredExportDestinationPolicy(payloadStr, storedPayloadStr string) (string, error) {
	payload := &storepb.QueryDataPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadStr), payload); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal query data policy payload")
	}
	if payload.ExportDestinationPolicy != nil {
		return payloadStr, nil
	}
	stored := &storepb.QueryDataPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(storedPayloadStr), stored); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal stored query data policy payload")
	}
	if stored.ExportDestinationPolicy == nil {
		return payloadStr, nil
	}
	payload.ExportDestinationPolicy = stored.ExportDestinationPolicy
	payloadBytes, err := protojson.Marshal(payload)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal query data policy payload")
	}
	return string(payloadBytes), nil
}

func convertToV1PBReviewRulePolicy(payloadStr string) (*v1pb.Policy_ReviewRulePolicy, error) {
	payload := &storepb.ReviewRulePolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadStr), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal review rule policy payload")
	}
	rules := make([]v1pb.ReviewRuleType, 0, len(payload.Rules))
	for _, rule := range payload.Rules {
		rules = append(rules, v1pb.ReviewRuleType(rule))
	}
	return &v1pb.Policy_ReviewRulePolicy{
		ReviewRulePolicy: &v1pb.ReviewRulePolicy{Rules: rules},
	}, nil
}

// convertToStorePBReviewRulePolicy casts each rule across the two mirrored
// enums; TestReviewRuleTypeEnumsMirror holds their numbering together.
func convertToStorePBReviewRulePolicy(policy *v1pb.ReviewRulePolicy) *storepb.ReviewRulePolicy {
	rules := make([]storepb.ReviewRuleType, 0, len(policy.GetRules()))
	for _, rule := range policy.GetRules() {
		rules = append(rules, storepb.ReviewRuleType(rule))
	}
	return &storepb.ReviewRulePolicy{Rules: rules}
}

func convertToV1PBReviewAIPolicy(payloadStr string) (*v1pb.Policy_ReviewAiPolicy, error) {
	payload := &storepb.ReviewAIPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadStr), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal AI review policy payload")
	}
	return &v1pb.Policy_ReviewAiPolicy{
		ReviewAiPolicy: &v1pb.ReviewAIPolicy{Content: payload.Content},
	}, nil
}

func convertToStorePBReviewAIPolicy(policy *v1pb.ReviewAIPolicy) *storepb.ReviewAIPolicy {
	return &storepb.ReviewAIPolicy{Content: policy.GetContent()}
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

func convertToStorePBMaskingExemptionPolicyPayload(policy *v1pb.MaskingExemptionPolicy) (*storepb.MaskingExemptionPolicy, error) {
	var exemptions []*storepb.MaskingExemptionPolicy_Exemption
	for _, exemption := range policy.Exemptions {
		var members []string
		for _, v1Member := range exemption.Members {
			member, err := convertToStoreIamPolicyMember(v1Member)
			if err != nil {
				return nil, err
			}
			members = append(members, member)
		}
		exemptions = append(exemptions, &storepb.MaskingExemptionPolicy_Exemption{
			Members: members,
			Condition: &expr.Expr{
				Title:       exemption.Condition.Title,
				Expression:  exemption.Condition.Expression,
				Description: exemption.Condition.Description,
				Location:    exemption.Condition.Location,
			},
		})
	}

	return &storepb.MaskingExemptionPolicy{
		Exemptions: exemptions,
	}, nil
}

func convertToV1PBMaskingExemptionPolicyPayload(policy *storepb.MaskingExemptionPolicy) *v1pb.MaskingExemptionPolicy {
	var exemptions []*v1pb.MaskingExemptionPolicy_Exemption
	for _, exemption := range policy.Exemptions {
		var members []string
		for _, storeMember := range exemption.Members {
			memberInBinding := convertToV1MemberInBinding(storeMember)
			if memberInBinding == "" {
				continue
			}
			members = append(members, memberInBinding)
		}

		if len(members) == 0 {
			continue
		}

		exemptions = append(exemptions, &v1pb.MaskingExemptionPolicy_Exemption{
			Members: members,
			Condition: &expr.Expr{
				Title:       exemption.Condition.Title,
				Expression:  exemption.Condition.Expression,
				Description: exemption.Condition.Description,
				Location:    exemption.Condition.Location,
			},
		})
	}

	return &v1pb.MaskingExemptionPolicy{
		Exemptions: exemptions,
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
	case v1pb.PolicyType_MASKING_EXEMPTION:
		return storepb.Policy_MASKING_EXEMPTION, nil
	case v1pb.PolicyType_DATA_QUERY:
		return storepb.Policy_QUERY_DATA, nil
	case v1pb.PolicyType_REVIEW_RULE:
		return storepb.Policy_REVIEW_RULE, nil
	case v1pb.PolicyType_REVIEW_AI:
		return storepb.Policy_REVIEW_AI, nil
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
	case storepb.Policy_MASKING_EXEMPTION:
		return v1pb.PolicyType_MASKING_EXEMPTION
	case storepb.Policy_QUERY_DATA:
		return v1pb.PolicyType_DATA_QUERY
	case storepb.Policy_REVIEW_RULE:
		return v1pb.PolicyType_REVIEW_RULE
	case storepb.Policy_REVIEW_AI:
		return v1pb.PolicyType_REVIEW_AI
	default:
	}
	return v1pb.PolicyType_POLICY_TYPE_UNSPECIFIED
}
