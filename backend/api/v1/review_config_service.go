package v1

import (
	"context"
	"regexp"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/store"
)

// ReviewConfigService implements the review config service.
type ReviewConfigService struct {
	v1connect.UnimplementedReviewConfigServiceHandler
	store *store.Store
}

// NewReviewConfigService creates a new ReviewConfigService.
func NewReviewConfigService(store *store.Store) *ReviewConfigService {
	return &ReviewConfigService{
		store: store,
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
			if err := validateSQLReviewRules(req.Msg.ReviewConfig.Rules); err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			ruleList, err := ConvertToSQLReviewRules(req.Msg.ReviewConfig.Rules)
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
	policies, err := s.store.ListPolicies(ctx, &store.FindPolicyMessage{
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
		if _, err := s.store.UpdatePolicy(ctx, &store.UpdatePolicyMessage{
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
	ruleList, err := ConvertToSQLReviewRules(reviewConfig.Rules)
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
	tagPolicies, err := s.store.ListPolicies(ctx, &store.FindPolicyMessage{
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
		Rules:   ConvertToV1PBSQLReviewRules(reviewConfigMessage.Payload.SqlReviewRules),
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
			project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
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
		if rule.Level == v1pb.SQLReviewRule_LEVEL_UNSPECIFIED {
			return errors.Errorf("invalid rule level: LEVEL_UNSPECIFIED is not allowed for rule %q", rule.Type)
		}
		if rule.Type == v1pb.SQLReviewRule_TYPE_UNSPECIFIED {
			return errors.Errorf("invalid rule type: TYPE_UNSPECIFIED is not allowed")
		}
		if err := validateSQLReviewRule(rule); err != nil {
			return err
		}
	}
	return nil
}

// validateSQLReviewRule validates a single SQL review rule's payload.
func validateSQLReviewRule(rule *v1pb.SQLReviewRule) error {
	ruleType := storepb.SQLReviewRule_Type(rule.Type)

	switch ruleType {
	// Naming rules with regex validation
	case storepb.SQLReviewRule_NAMING_TABLE, storepb.SQLReviewRule_NAMING_COLUMN, storepb.SQLReviewRule_NAMING_COLUMN_AUTO_INCREMENT:
		payload := rule.GetNamingPayload()
		if payload == nil {
			return errors.Errorf("rule %s requires naming payload", ruleType)
		}
		// At least one of format or maxLength must be set
		if payload.Format == "" && payload.MaxLength <= 0 {
			return errors.Errorf("naming rule must specify either format or max_length for rule %s", ruleType)
		}
		// If format is set, validate it compiles
		if payload.Format != "" {
			if _, err := regexp.Compile(payload.Format); err != nil {
				return errors.Wrapf(err, "invalid naming rule format pattern %q for rule %s", payload.Format, ruleType)
			}
		}
		// If maxLength is set, validate it's positive (maxLength == 0 means not set)
		if payload.MaxLength < 0 {
			return errors.Errorf("naming rule max_length cannot be negative for rule %s, got %d", ruleType, payload.MaxLength)
		}

	// Naming rules with template token validation
	case storepb.SQLReviewRule_NAMING_INDEX_FK, storepb.SQLReviewRule_NAMING_INDEX_IDX, storepb.SQLReviewRule_NAMING_INDEX_UK, storepb.SQLReviewRule_NAMING_INDEX_PK, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION:
		payload := rule.GetNamingPayload()
		if payload == nil {
			return errors.Errorf("rule %s requires naming payload", ruleType)
		}
		// At least one of format or maxLength must be set
		if payload.Format == "" && payload.MaxLength <= 0 {
			return errors.Errorf("naming rule must specify either format or max_length for rule %s", ruleType)
		}
		// If format is set, validate template tokens
		if payload.Format != "" {
			tokens, _ := advisor.ParseTemplateTokens(payload.Format)
			for _, token := range tokens {
				if _, ok := advisor.TemplateNamingTokens[ruleType][token]; !ok {
					return errors.Errorf("invalid template token %s for rule %s", token, ruleType)
				}
			}
		}
		// If maxLength is set, validate it's positive (maxLength == 0 means not set)
		if payload.MaxLength < 0 {
			return errors.Errorf("naming rule max_length cannot be negative for rule %s, got %d", ruleType, payload.MaxLength)
		}

	// Number payload rules
	case storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT,
		storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT,
		storepb.SQLReviewRule_STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT,
		storepb.SQLReviewRule_STATEMENT_MAXIMUM_LIMIT_VALUE,
		storepb.SQLReviewRule_STATEMENT_MAXIMUM_JOIN_TABLE_COUNT,
		storepb.SQLReviewRule_STATEMENT_MAXIMUM_STATEMENTS_IN_TRANSACTION,
		storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH,
		storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH,
		storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_INITIAL_VALUE,
		storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT,
		storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT,
		storepb.SQLReviewRule_TABLE_TEXT_FIELDS_TOTAL_LENGTH,
		storepb.SQLReviewRule_TABLE_LIMIT_SIZE,
		storepb.SQLReviewRule_SYSTEM_COMMENT_LENGTH,
		storepb.SQLReviewRule_ADVICE_ONLINE_MIGRATION:
		payload := rule.GetNumberPayload()
		if payload == nil {
			return errors.Errorf("rule %s requires number payload", ruleType)
		}
		if payload.Number <= 0 {
			return errors.Errorf("number payload must be positive for rule %s, got %d", ruleType, payload.Number)
		}

	// String payload rules
	case storepb.SQLReviewRule_STATEMENT_QUERY_MINIMUM_PLAN_LEVEL:
		payload := rule.GetStringPayload()
		if payload == nil {
			return errors.Errorf("rule %s requires string payload", ruleType)
		}
		validLevels := map[string]bool{
			"ALL": true, "INDEX": true, "RANGE": true,
			"REF": true, "EQ_REF": true, "CONST": true,
		}
		upperValue := strings.ToUpper(payload.Value)
		if !validLevels[upperValue] {
			return errors.Errorf("invalid plan level %q for rule %s, must be one of: ALL, INDEX, RANGE, REF, EQ_REF, CONST", payload.Value, ruleType)
		}

	// String array payload rules that require non-empty arrays
	case storepb.SQLReviewRule_COLUMN_REQUIRED,
		storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST,
		storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST,
		storepb.SQLReviewRule_INDEX_TYPE_ALLOW_LIST,
		storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST,
		storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST,
		storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOWED_LIST,
		storepb.SQLReviewRule_TABLE_DISALLOW_DDL,
		storepb.SQLReviewRule_TABLE_DISALLOW_DML:
		payload := rule.GetStringArrayPayload()
		if payload == nil {
			return errors.Errorf("rule %s requires string array payload", ruleType)
		}
		if len(payload.List) == 0 {
			return errors.Errorf("string array payload cannot be empty for rule %s", ruleType)
		}

	// Comment convention payload rules
	case storepb.SQLReviewRule_COLUMN_COMMENT, storepb.SQLReviewRule_TABLE_COMMENT:
		payload := rule.GetCommentConventionPayload()
		if payload == nil {
			return errors.Errorf("rule %s requires comment convention payload", ruleType)
		}
		if payload.MaxLength <= 0 {
			return errors.Errorf("comment convention max_length must be positive for rule %s, got %d", ruleType, payload.MaxLength)
		}

	// Naming case payload rules
	case storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE:
		payload := rule.GetNamingCasePayload()
		if payload == nil {
			return errors.Errorf("rule %s requires naming case payload", ruleType)
		}
		// Upper field is boolean, no value validation needed

	// Rules that explicitly should NOT have payloads
	case storepb.SQLReviewRule_NAMING_FULLY_QUALIFIED,
		storepb.SQLReviewRule_STATEMENT_MAX_EXECUTION_TIME,
		storepb.SQLReviewRule_COLUMN_CURRENT_TIME_COUNT_LIMIT,
		storepb.SQLReviewRule_ENGINE_MYSQL_USE_INNODB,
		storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD,
		storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD,
		storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT,
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE,
		storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_ON_DEL_CASCADE,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_RM_TBL_CASCADE,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_LIMIT,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_ORDER_BY,
		storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE,
		storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN,
		storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND,
		storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_COLUMN_WITH_DEFAULT,
		storepb.SQLReviewRule_STATEMENT_ADD_CHECK_NOT_VALID,
		storepb.SQLReviewRule_STATEMENT_ADD_FOREIGN_KEY_NOT_VALID,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_NOT_NULL,
		storepb.SQLReviewRule_STATEMENT_SELECT_FULL_TABLE_SCAN,
		storepb.SQLReviewRule_STATEMENT_CREATE_SPECIFY_SCHEMA,
		storepb.SQLReviewRule_STATEMENT_CHECK_SET_ROLE_VARIABLE,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_USING_FILESORT,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_USING_TEMPORARY,
		storepb.SQLReviewRule_STATEMENT_WHERE_NO_EQUAL_NULL,
		storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS,
		storepb.SQLReviewRule_STATEMENT_JOIN_STRICT_COLUMN_ATTRS,
		storepb.SQLReviewRule_STATEMENT_NON_TRANSACTIONAL,
		storepb.SQLReviewRule_STATEMENT_ADD_COLUMN_WITHOUT_POSITION,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_OFFLINE_DDL,
		storepb.SQLReviewRule_STATEMENT_DISALLOW_CROSS_DB_QUERIES,
		storepb.SQLReviewRule_STATEMENT_REQUIRE_ALGORITHM_OPTION,
		storepb.SQLReviewRule_STATEMENT_REQUIRE_LOCK_OPTION,
		storepb.SQLReviewRule_STATEMENT_OBJECT_OWNER_CHECK,
		storepb.SQLReviewRule_TABLE_REQUIRE_PK,
		storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY,
		storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION,
		storepb.SQLReviewRule_TABLE_DISALLOW_TRIGGER,
		storepb.SQLReviewRule_TABLE_NO_DUPLICATE_INDEX,
		storepb.SQLReviewRule_TABLE_DISALLOW_SET_CHARSET,
		storepb.SQLReviewRule_TABLE_REQUIRE_CHARSET,
		storepb.SQLReviewRule_TABLE_REQUIRE_COLLATION,
		storepb.SQLReviewRule_COLUMN_NO_NULL,
		storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE,
		storepb.SQLReviewRule_COLUMN_SET_DEFAULT_FOR_NOT_NULL,
		storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE,
		storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGING_ORDER,
		storepb.SQLReviewRule_COLUMN_DISALLOW_DROP,
		storepb.SQLReviewRule_COLUMN_DISALLOW_DROP_IN_INDEX,
		storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_INTEGER,
		storepb.SQLReviewRule_COLUMN_DISALLOW_SET_CHARSET,
		storepb.SQLReviewRule_COLUMN_AUTO_INCREMENT_MUST_UNSIGNED,
		storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT,
		storepb.SQLReviewRule_COLUMN_DEFAULT_DISALLOW_VOLATILE,
		storepb.SQLReviewRule_COLUMN_ADD_NOT_NULL_REQUIRE_DEFAULT,
		storepb.SQLReviewRule_COLUMN_REQUIRE_CHARSET,
		storepb.SQLReviewRule_COLUMN_REQUIRE_COLLATION,
		storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY,
		storepb.SQLReviewRule_DATABASE_DROP_EMPTY_DATABASE,
		storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN,
		storepb.SQLReviewRule_INDEX_PK_TYPE_LIMIT,
		storepb.SQLReviewRule_INDEX_TYPE_NO_BLOB,
		storepb.SQLReviewRule_INDEX_CREATE_CONCURRENTLY,
		storepb.SQLReviewRule_INDEX_NOT_REDUNDANT,
		storepb.SQLReviewRule_SYSTEM_PROCEDURE_DISALLOW_CREATE,
		storepb.SQLReviewRule_SYSTEM_EVENT_DISALLOW_CREATE,
		storepb.SQLReviewRule_SYSTEM_VIEW_DISALLOW_CREATE,
		storepb.SQLReviewRule_SYSTEM_FUNCTION_DISALLOW_CREATE,
		storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK:
		// These rules should not have any payload
		if rule.Payload != nil {
			return errors.Errorf("rule %s should not have a payload", ruleType)
		}

	default:
		// Unknown rule type - should not happen if proto is in sync
		return errors.Errorf("unknown rule type %s", ruleType)
	}

	return nil
}
