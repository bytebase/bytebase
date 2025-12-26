package v1

import (
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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

func convertToV1PBRolloutPolicy(policy *storepb.RolloutPolicy) *v1pb.RolloutPolicy {
	return &v1pb.RolloutPolicy{
		Automatic: policy.Automatic,
		Roles:     policy.Roles,
	}
}

func convertToV1PBQueryDataPolicy(payloadStr string) (*v1pb.Policy_QueryDataPolicy, error) {
	payload := &storepb.QueryDataPolicy{}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(payloadStr), payload); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal query data policy payload")
	}
	return &v1pb.Policy_QueryDataPolicy{
		QueryDataPolicy: &v1pb.QueryDataPolicy{
			Timeout:           payload.Timeout,
			DisableExport:     payload.DisableExport,
			MaximumResultSize: payload.MaximumResultSize,
			MaximumResultRows: payload.MaximumResultRows,
			DisableCopyData:   payload.DisableCopyData,
		},
	}, nil
}

func convertToQueryDataPolicyPayload(policy *v1pb.QueryDataPolicy) *storepb.QueryDataPolicy {
	return &storepb.QueryDataPolicy{
		Timeout:           policy.Timeout,
		DisableExport:     policy.DisableExport,
		MaximumResultSize: policy.MaximumResultSize,
		MaximumResultRows: policy.MaximumResultRows,
		DisableCopyData:   policy.DisableCopyData,
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
	case v1pb.PolicyType_MASKING_EXEMPTION:
		return storepb.Policy_MASKING_EXEMPTION, nil
	case v1pb.PolicyType_DATA_QUERY:
		return storepb.Policy_QUERY_DATA, nil
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
	case storepb.Policy_MASKING_EXEMPTION:
		return v1pb.PolicyType_MASKING_EXEMPTION
	case storepb.Policy_QUERY_DATA:
		return v1pb.PolicyType_DATA_QUERY
	case storepb.Policy_DATA_SOURCE_QUERY:
		return v1pb.PolicyType_DATA_SOURCE_QUERY
	default:
	}
	return v1pb.PolicyType_POLICY_TYPE_UNSPECIFIED
}
