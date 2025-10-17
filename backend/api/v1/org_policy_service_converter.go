package v1

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func convertToV1PBSQLReviewRules(ruleList []*storepb.SQLReviewRule) []*v1pb.SQLReviewRule {
	var rules []*v1pb.SQLReviewRule
	for _, rule := range ruleList {
		level := v1pb.SQLReviewRuleLevel_LEVEL_UNSPECIFIED
		switch rule.Level {
		case storepb.SQLReviewRuleLevel_ERROR:
			level = v1pb.SQLReviewRuleLevel_ERROR
		case storepb.SQLReviewRuleLevel_WARNING:
			level = v1pb.SQLReviewRuleLevel_WARNING
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
		Automatic: policy.Automatic,
		Roles:     policy.Roles,
		Checkers:  convertToStorePBCheckers(policy.Checkers),
	}
}

func convertToStorePBCheckers(checkers *v1pb.RolloutPolicy_Checkers) *storepb.RolloutPolicy_Checkers {
	if checkers == nil {
		return nil
	}
	result := &storepb.RolloutPolicy_Checkers{
		RequiredIssueApproval: checkers.RequiredIssueApproval,
	}
	if checkers.RequiredStatusChecks != nil {
		result.RequiredStatusChecks = &storepb.RolloutPolicy_Checkers_RequiredStatusChecks{
			PlanCheckEnforcement: storepb.RolloutPolicy_Checkers_PlanCheckEnforcement(checkers.RequiredStatusChecks.PlanCheckEnforcement),
		}
	}
	return result
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
	case storepb.Policy_MASKING_EXCEPTION:
		return v1pb.PolicyType_MASKING_EXCEPTION
	case storepb.Policy_QUERY_DATA:
		return v1pb.PolicyType_DATA_QUERY
	case storepb.Policy_DATA_SOURCE_QUERY:
		return v1pb.PolicyType_DATA_SOURCE_QUERY
	default:
	}
	return v1pb.PolicyType_POLICY_TYPE_UNSPECIFIED
}
