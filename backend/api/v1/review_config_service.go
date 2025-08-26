package v1

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/store"
)

// ReviewConfigService implements the review config service.
type ReviewConfigService struct {
	v1connect.UnimplementedReviewConfigServiceHandler
	store          *store.Store
	licenseService *enterprise.LicenseService
}

// NewReviewConfigService creates a new ReviewConfigService.
func NewReviewConfigService(store *store.Store, licenseService *enterprise.LicenseService) *ReviewConfigService {
	return &ReviewConfigService{
		store:          store,
		licenseService: licenseService,
	}
}

// CreateReviewConfig creates a new review config.
func (s *ReviewConfigService) CreateReviewConfig(ctx context.Context, req *connect.Request[v1pb.CreateReviewConfigRequest]) (*connect.Response[v1pb.ReviewConfig], error) {
	if err := validateSQLReviewRules(req.Msg.ReviewConfig.Rules); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	reviewConfigMessage, err := convertToReviewConfigMessage(req.Msg.ReviewConfig)
	if err != nil {
		return nil, err
	}

	created, err := s.store.CreateReviewConfig(ctx, reviewConfigMessage)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result, err := s.convertToV1ReviewConfig(ctx, created)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

// ListReviewConfigs lists the review configs.
func (s *ReviewConfigService) ListReviewConfigs(ctx context.Context, _ *connect.Request[v1pb.ListReviewConfigsRequest]) (*connect.Response[v1pb.ListReviewConfigsResponse], error) {
	messages, err := s.store.ListReviewConfigs(ctx, &store.FindReviewConfigMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1pb.ListReviewConfigsResponse{}
	for _, message := range messages {
		sqlReview, err := s.convertToV1ReviewConfig(ctx, message)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		response.ReviewConfigs = append(response.ReviewConfigs, sqlReview)
	}
	return connect.NewResponse(response), nil
}

// GetReviewConfig gets the review config.
func (s *ReviewConfigService) GetReviewConfig(ctx context.Context, req *connect.Request[v1pb.GetReviewConfigRequest]) (*connect.Response[v1pb.ReviewConfig], error) {
	id, err := common.GetReviewConfigID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	message, err := s.store.GetReviewConfig(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if message == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot found review config %s", req.Msg.Name))
	}
	result, err := s.convertToV1ReviewConfig(ctx, message)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

// UpdateReviewConfig updates the review config.
func (s *ReviewConfigService) UpdateReviewConfig(ctx context.Context, req *connect.Request[v1pb.UpdateReviewConfigRequest]) (*connect.Response[v1pb.ReviewConfig], error) {
	id, err := common.GetReviewConfigID(req.Msg.ReviewConfig.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	existed, err := s.store.GetReviewConfig(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get review config %q", id))
	}
	if existed == nil {
		if req.Msg.AllowMissing {
			return s.CreateReviewConfig(ctx, connect.NewRequest(&v1pb.CreateReviewConfigRequest{
				ReviewConfig: req.Msg.ReviewConfig,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("review config %q not found", id))
	}

	patch := &store.PatchReviewConfigMessage{
		ID: id,
	}

	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Name = &req.Msg.ReviewConfig.Title
		case "rules":
			ruleList, err := convertToSQLReviewRules(req.Msg.ReviewConfig.Rules)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to convert rules"))
			}
			patch.Payload = &storepb.ReviewConfigPayload{
				SqlReviewRules: ruleList,
			}
		case "enabled":
			patch.Enforce = &req.Msg.ReviewConfig.Enabled
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update mask path %q", path))
		}
	}

	message, err := s.store.UpdateReviewConfig(ctx, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result, err := s.convertToV1ReviewConfig(ctx, message)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

// DeleteReviewConfig deletes the review config.
func (s *ReviewConfigService) DeleteReviewConfig(ctx context.Context, req *connect.Request[v1pb.DeleteReviewConfigRequest]) (*connect.Response[emptypb.Empty], error) {
	id, err := common.GetReviewConfigID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := s.store.DeleteReviewConfig(ctx, id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to delete review config"))
	}

	policyType := storepb.Policy_TAG
	policies, err := s.store.ListPoliciesV2(ctx, &store.FindPolicyMessage{
		Type: &policyType,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to list tag policy"))
	}
	for _, policy := range policies {
		payload := &storepb.TagPolicy{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), payload); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to unmarshal rollout policy payload"))
		}
		reviewConfigName, ok := payload.Tags[common.ReservedTagReviewConfig]
		if !ok {
			continue
		}
		if reviewConfigName != req.Msg.Name {
			continue
		}
		delete(payload.Tags, common.ReservedTagReviewConfig)

		payloadBytes, err := protojson.Marshal(payload)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to marshal tag policy"))
		}
		patch := string(payloadBytes)
		if _, err := s.store.UpdatePolicyV2(ctx, &store.UpdatePolicyMessage{
			ResourceType: policy.ResourceType,
			Resource:     policy.Resource,
			Payload:      &patch,
		}); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to update tag policy"))
		}
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func convertToReviewConfigMessage(reviewConfig *v1pb.ReviewConfig) (*store.ReviewConfigMessage, error) {
	ruleList, err := convertToSQLReviewRules(reviewConfig.Rules)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to convert rules"))
	}

	id, err := common.GetReviewConfigID(reviewConfig.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid config name %s", reviewConfig.Name))
	}

	if !isValidResourceID(id) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid config id %v", reviewConfig.Name))
	}

	return &store.ReviewConfigMessage{
		ID:      id,
		Name:    reviewConfig.Title,
		Enforce: reviewConfig.Enabled,
		Payload: &storepb.ReviewConfigPayload{
			SqlReviewRules: ruleList,
		},
	}, nil
}

func (s *ReviewConfigService) convertToV1ReviewConfig(ctx context.Context, reviewConfigMessage *store.ReviewConfigMessage) (*v1pb.ReviewConfig, error) {
	policyType := storepb.Policy_TAG
	tagPolicies, err := s.store.ListPoliciesV2(ctx, &store.FindPolicyMessage{
		Type:    &policyType,
		ShowAll: false,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to list tag policy"))
	}

	config := &v1pb.ReviewConfig{
		Name:    common.FormatReviewConfig(reviewConfigMessage.ID),
		Title:   reviewConfigMessage.Name,
		Enabled: reviewConfigMessage.Enforce,
		Rules:   convertToV1PBSQLReviewRules(reviewConfigMessage.Payload.SqlReviewRules),
	}

	for _, policy := range tagPolicies {
		p := &v1pb.TagPolicy{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), p); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to unmarshal tag policy"))
		}
		if p.Tags[common.ReservedTagReviewConfig] != config.Name {
			continue
		}

		switch policy.ResourceType {
		case storepb.Policy_ENVIRONMENT:
			environmentID, err := common.GetEnvironmentID(policy.Resource)
			if err != nil {
				return nil, err
			}
			environment, err := s.store.GetEnvironmentByID(ctx, environmentID)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get environment %s", environmentID))
			}
			if environment == nil {
				continue
			}
			config.Resources = append(config.Resources, common.FormatEnvironment(environment.Id))
		case storepb.Policy_PROJECT:
			projectID, err := common.GetProjectID(policy.Resource)
			if err != nil {
				return nil, err
			}
			project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
				ResourceID:  &projectID,
				ShowDeleted: false,
			})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project %s", projectID))
			}
			if project == nil {
				continue
			}
			config.Resources = append(config.Resources, common.FormatProject(project.ResourceID))
		default:
		}
	}

	return config, nil
}

// validateSQLReviewRules validates the SQL review rule.
func validateSQLReviewRules(rules []*v1pb.SQLReviewRule) error {
	if len(rules) == 0 {
		return errors.Errorf("invalid payload, rule list cannot be empty")
	}
	for _, rule := range rules {
		ruleType := advisor.SQLReviewRuleType(rule.Type)
		// TODO(rebelice): add other SQL review rule validation.
		switch ruleType {
		case advisor.SchemaRuleTableNaming, advisor.SchemaRuleColumnNaming, advisor.SchemaRuleAutoIncrementColumnNaming:
			if _, _, err := advisor.UnmarshalNamingRulePayloadAsRegexp(rule.Payload); err != nil {
				return err
			}
		case advisor.SchemaRuleFKNaming, advisor.SchemaRuleIDXNaming, advisor.SchemaRuleUKNaming:
			if _, _, _, err := advisor.UnmarshalNamingRulePayloadAsTemplate(ruleType, rule.Payload); err != nil {
				return err
			}
		case advisor.SchemaRuleRequiredColumn:
			if _, err := advisor.UnmarshalRequiredColumnList(rule.Payload); err != nil {
				return err
			}
		case advisor.SchemaRuleColumnCommentConvention, advisor.SchemaRuleTableCommentConvention:
			if _, err := advisor.UnmarshalCommentConventionRulePayload(rule.Payload); err != nil {
				return err
			}
		case advisor.SchemaRuleIndexKeyNumberLimit, advisor.SchemaRuleStatementInsertRowLimit, advisor.SchemaRuleIndexTotalNumberLimit,
			advisor.SchemaRuleColumnMaximumCharacterLength, advisor.SchemaRuleColumnMaximumVarcharLength, advisor.SchemaRuleColumnAutoIncrementInitialValue, advisor.SchemaRuleStatementAffectedRowLimit:
			if _, err := advisor.UnmarshalNumberTypeRulePayload(rule.Payload); err != nil {
				return err
			}
		case advisor.SchemaRuleColumnTypeDisallowList, advisor.SchemaRuleCharsetAllowlist, advisor.SchemaRuleCollationAllowlist, advisor.SchemaRuleIndexPrimaryKeyTypeAllowlist:
			if _, err := advisor.UnmarshalStringArrayTypeRulePayload(rule.Payload); err != nil {
				return err
			}
		case advisor.SchemaRuleIdentifierCase:
			if _, err := advisor.UnmarshalNamingCaseRulePayload(rule.Payload); err != nil {
				return err
			}
		default:
		}
	}
	return nil
}
