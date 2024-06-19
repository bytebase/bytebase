package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ReviewConfigService implements the review config service.
type ReviewConfigService struct {
	v1pb.UnimplementedReviewConfigServiceServer
	store          *store.Store
	licenseService enterprise.LicenseService
}

// NewReviewConfigService creates a new ReviewConfigService.
func NewReviewConfigService(store *store.Store, licenseService enterprise.LicenseService) *ReviewConfigService {
	return &ReviewConfigService{
		store:          store,
		licenseService: licenseService,
	}
}

// CreateReviewConfig creates a new review config.
func (s *ReviewConfigService) CreateReviewConfig(ctx context.Context, request *v1pb.CreateReviewConfigRequest) (*v1pb.ReviewConfig, error) {
	if err := s.licenseService.IsFeatureEnabled(api.FeatureSQLReview); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}
	if err := validateSQLReviewRules(request.ReviewConfig.Rules); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	reviewConfigMessage, err := convertToReviewConfigMessage(request.ReviewConfig)
	if err != nil {
		return nil, err
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}

	reviewConfigMessage.CreatorUID = principalID
	created, err := s.store.CreateReviewConfig(ctx, reviewConfigMessage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return s.convertToV1ReviewConfig(ctx, created)
}

// ListReviewConfigs lists the review configs.
func (s *ReviewConfigService) ListReviewConfigs(ctx context.Context, _ *v1pb.ListReviewConfigsRequest) (*v1pb.ListReviewConfigsResponse, error) {
	messages, err := s.store.ListReviewConfigs(ctx, &store.FindReviewConfigMessage{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	response := &v1pb.ListReviewConfigsResponse{}
	for _, message := range messages {
		sqlReview, err := s.convertToV1ReviewConfig(ctx, message)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		response.ReviewConfigs = append(response.ReviewConfigs, sqlReview)
	}
	return response, nil
}

// GetReviewConfig gets the review config.
func (s *ReviewConfigService) GetReviewConfig(ctx context.Context, request *v1pb.GetReviewConfigRequest) (*v1pb.ReviewConfig, error) {
	id, err := common.GetReviewConfigID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	message, err := s.store.GetReviewConfig(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return s.convertToV1ReviewConfig(ctx, message)
}

// UpdateReviewConfig updates the review config.
func (s *ReviewConfigService) UpdateReviewConfig(ctx context.Context, request *v1pb.UpdateReviewConfigRequest) (*v1pb.ReviewConfig, error) {
	id, err := common.GetReviewConfigID(request.ReviewConfig.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}

	patch := &store.PatchReviewConfigMessage{
		ID:        id,
		UpdaterID: principalID,
	}

	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Name = &request.ReviewConfig.Title
		case "payload":
			ruleList, err := convertToSQLReviewRules(request.ReviewConfig.Rules)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to convert rules, error %v", err)
			}
			patch.Payload = &storepb.ReviewConfigPayload{
				SqlReviewRules: ruleList,
			}
		case "enabled":
			patch.Enforce = &request.ReviewConfig.Enabled
		default:
			return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid update mask path %q", path))
		}
	}

	message, err := s.store.UpdateReviewConfig(ctx, patch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return s.convertToV1ReviewConfig(ctx, message)
}

// DeleteReviewConfig deletes the review config.
func (s *ReviewConfigService) DeleteReviewConfig(ctx context.Context, request *v1pb.DeleteReviewConfigRequest) (*emptypb.Empty, error) {
	id, err := common.GetReviewConfigID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	if err := s.store.DeleteReviewConfig(ctx, id); err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to delete review config: %v", err))
	}

	return &emptypb.Empty{}, nil
}

func convertToReviewConfigMessage(reviewConfig *v1pb.ReviewConfig) (*store.ReviewConfigMessage, error) {
	ruleList, err := convertToSQLReviewRules(reviewConfig.Rules)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert rules, error %v", err)
	}

	id, err := common.GetReviewConfigID(reviewConfig.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid config name %s, error %v", reviewConfig.Name, err)
	}

	if !isValidResourceID(id) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid config id %v", reviewConfig.Name)
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
	creator, err := s.store.GetUserByID(ctx, reviewConfigMessage.CreatorUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get creator, error %v", err)
	}
	if creator == nil {
		return nil, status.Errorf(codes.NotFound, "creator %d not found", reviewConfigMessage.CreatorUID)
	}

	resourceType := api.PolicyResourceTypeEnvironment
	policyType := api.PolicyTypeTag
	tagPolicies, err := s.store.ListPoliciesV2(ctx, &store.FindPolicyMessage{
		ResourceType: &resourceType,
		Type:         &policyType,
		ShowDeleted:  false,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list tag policy, error %v", err)
	}

	config := &v1pb.ReviewConfig{
		Name:       common.FormatReviewConfig(reviewConfigMessage.ID),
		Creator:    common.FormatUserEmail(creator.Email),
		CreateTime: timestamppb.New(reviewConfigMessage.CreatedTime),
		UpdateTime: timestamppb.New(reviewConfigMessage.CreatedTime),
		Title:      reviewConfigMessage.Name,
		Enabled:    reviewConfigMessage.Enforce,
		Rules:      convertToV1PBSQLReviewRules(reviewConfigMessage.Payload.SqlReviewRules),
	}

	for _, policy := range tagPolicies {
		p := &v1pb.TagPolicy{}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), p); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal tag policy, error %v", err)
		}
		if p.Tags[string(api.ReservedTagReviewConfig)] != config.Name {
			continue
		}
		environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
			UID:         &policy.ResourceUID,
			ShowDeleted: false,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get environment %d with error: %v", policy.ResourceUID, err)
		}
		if environment == nil {
			continue
		}
		config.Resources = append(config.Resources, common.FormatEnvironment(environment.ResourceID))
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
		}
	}
	return nil
}
