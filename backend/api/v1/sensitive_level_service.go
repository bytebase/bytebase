// Copyright 2024 Bytebase Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// SensitiveLevelService implements the sensitive level service.
type SensitiveLevelService struct {
	v1connect.UnimplementedSensitiveLevelServiceHandler
	store          *store.Store
	stateCfg       *state.State
	licenseService *enterprise.LicenseService
	profile        *config.Profile
	iamManager     *iam.Manager
}

// NewSensitiveLevelService creates a new SensitiveLevelService.
func NewSensitiveLevelService(
	store *store.Store,
	stateCfg *state.State,
	licenseService *enterprise.LicenseService,
	profile *config.Profile,
	iamManager *iam.Manager,
) *SensitiveLevelService {
	return &SensitiveLevelService{
		store:          store,
		stateCfg:       stateCfg,
		licenseService: licenseService,
		profile:        profile,
	iamManager:     iamManager,
	}
}

// CreateSensitiveLevel creates a new sensitive data level configuration.
func (s *SensitiveLevelService) CreateSensitiveLevel(ctx context.Context, req *connect.Request[v1pb.CreateSensitiveLevelRequest]) (*connect.Response[v1pb.SensitiveLevel], error) {
	// Validate request
	if req.Msg.SensitiveLevel == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("sensitive_level is required"))
	}

	if req.Msg.SensitiveLevel.DisplayName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("display_name is required"))
	}

	if req.Msg.SensitiveLevel.Level == v1pb.SensitivityLevel_SENSITIVITY_LEVEL_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("level is required"))
	}

	if req.Msg.SensitiveLevel.TableName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("table_name is required"))
	}

	if req.Msg.SensitiveLevel.SchemaName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("schema_name is required"))
	}

	if req.Msg.SensitiveLevel.InstanceId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("instance_id is required"))
	}

	// Check if instance exists
	instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{ResourceID: &req.Msg.SensitiveLevel.InstanceId})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get instance: %v", err))
	}

	if instance == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %s not found", req.Msg.SensitiveLevel.InstanceId))
	}

	// Create sensitive level
	sensitiveLevel := &store.SensitiveLevelMessage{
		DisplayName:  req.Msg.SensitiveLevel.DisplayName,
		Description:  req.Msg.SensitiveLevel.Description,
		Level:        int32(req.Msg.SensitiveLevel.Level),
		TableName:    req.Msg.SensitiveLevel.TableName,
		SchemaName:   req.Msg.SensitiveLevel.SchemaName,
		InstanceId:   req.Msg.SensitiveLevel.InstanceId,
		FieldRules:   convertFieldRulesToStore(req.Msg.SensitiveLevel.FieldRules),
	}

	createdSensitiveLevel, err := s.store.CreateSensitiveLevel(ctx, sensitiveLevel)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create sensitive level: %v", err))
	}

	// Convert to v1pb.SensitiveLevel
	result := convertToSensitiveLevel(ctx, createdSensitiveLevel)

	return connect.NewResponse(result), nil
}

// GetSensitiveLevel gets a sensitive data level configuration.
func (s *SensitiveLevelService) GetSensitiveLevel(ctx context.Context, req *connect.Request[v1pb.GetSensitiveLevelRequest]) (*connect.Response[v1pb.SensitiveLevel], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	// Extract sensitive level ID from name
	parts := strings.Split(req.Msg.Name, "/")
	if len(parts) != 2 || parts[0] != "sensitiveLevels" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid name format: %s", req.Msg.Name))
	}

	id := parts[1]

	// Get sensitive level
	sensitiveLevel, err := s.store.GetSensitiveLevel(ctx, &store.FindSensitiveLevelMessage{ID: &id})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get sensitive level: %v", err))
	}

	if sensitiveLevel == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("sensitive level %s not found", id))
	}

	// Convert to v1pb.SensitiveLevel
	result := convertToSensitiveLevel(ctx, sensitiveLevel)

	return connect.NewResponse(result), nil
}

// ListSensitiveLevels lists all sensitive data level configurations.
func (s *SensitiveLevelService) ListSensitiveLevels(ctx context.Context, req *connect.Request[v1pb.ListSensitiveLevelsRequest]) (*connect.Response[v1pb.ListSensitiveLevelsResponse], error) {
	// Build find message
	find := &store.FindSensitiveLevelMessage{}

	if req.Msg.InstanceId != "" {
		find.InstanceID = &req.Msg.InstanceId
	}

	if req.Msg.SchemaName != "" {
		find.SchemaName = &req.Msg.SchemaName
	}

	if req.Msg.TableName != "" {
		find.TableName = &req.Msg.TableName
	}

	if req.Msg.Level != v1pb.SensitivityLevel_SENSITIVITY_LEVEL_UNSPECIFIED {
		level := int32(req.Msg.Level)
		find.Level = &level
	}

	// Set pagination
	limit := int(req.Msg.PageSize)
	if limit == 0 {
		limit = 100
	}
	find.Limit = &limit

	// Get sensitive levels
	sensitiveLevels, err := s.store.ListSensitiveLevels(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list sensitive levels: %v", err))
	}

	// Convert to v1pb.SensitiveLevel
	result := &v1pb.ListSensitiveLevelsResponse{}
	for _, sl := range sensitiveLevels {
		result.SensitiveLevels = append(result.SensitiveLevels, convertToSensitiveLevel(ctx, sl))
	}

	// Set next page token if needed
	if len(sensitiveLevels) == limit {
		result.NextPageToken = fmt.Sprintf("%d", limit)
	}

	return connect.NewResponse(result), nil
}

// UpdateSensitiveLevel updates a sensitive data level configuration.
func (s *SensitiveLevelService) UpdateSensitiveLevel(ctx context.Context, req *connect.Request[v1pb.UpdateSensitiveLevelRequest]) (*connect.Response[v1pb.SensitiveLevel], error) {
	if req.Msg.SensitiveLevel == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("sensitive_level is required"))
	}

	if req.Msg.SensitiveLevel.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	// Extract sensitive level ID from name
	parts := strings.Split(req.Msg.SensitiveLevel.Name, "/")
	if len(parts) != 2 || parts[0] != "sensitiveLevels" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid name format: %s", req.Msg.SensitiveLevel.Name))
	}

	id := parts[1]

	// Get existing sensitive level
	existing, err := s.store.GetSensitiveLevel(ctx, &store.FindSensitiveLevelMessage{ID: &id})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get sensitive level: %v", err))
	}

	if existing == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("sensitive level %s not found", id))
	}

	// Build update message
	update := &store.UpdateSensitiveLevelMessage{}

	// Apply field mask
	for _, field := range req.Msg.UpdateMask.Paths {
		switch field {
		case "display_name":
			update.DisplayName = &req.Msg.SensitiveLevel.DisplayName
		case "description":
			update.Description = &req.Msg.SensitiveLevel.Description
		case "level":
			level := int32(req.Msg.SensitiveLevel.Level)
			update.Level = &level
		case "table_name":
			update.TableName = &req.Msg.SensitiveLevel.TableName
		case "schema_name":
			update.SchemaName = &req.Msg.SensitiveLevel.SchemaName
		case "instance_id":
			update.InstanceID = &req.Msg.SensitiveLevel.InstanceId
		case "field_rules":
			update.FieldRules = convertFieldRulesToStore(req.Msg.SensitiveLevel.FieldRules)
		}
	}

	// Update sensitive level
	updatedSensitiveLevel, err := s.store.UpdateSensitiveLevel(ctx, id, update)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to update sensitive level: %v", err))
	}

	// Convert to v1pb.SensitiveLevel
	result := convertToSensitiveLevel(ctx, updatedSensitiveLevel)

	return connect.NewResponse(result), nil
}

// DeleteSensitiveLevel deletes a sensitive data level configuration.
func (s *SensitiveLevelService) DeleteSensitiveLevel(ctx context.Context, req *connect.Request[v1pb.DeleteSensitiveLevelRequest]) (*connect.Response[v1pb.Empty], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	// Extract sensitive level ID from name
	parts := strings.Split(req.Msg.Name, "/")
	if len(parts) != 2 || parts[0] != "sensitiveLevels" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid name format: %s", req.Msg.Name))
	}

	id := parts[1]

	// Delete sensitive level
	err := s.store.DeleteSensitiveLevel(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to delete sensitive level: %v", err))
	}

	return connect.NewResponse(&v1pb.Empty{}), nil
}

// convertToSensitiveLevel converts a store.SensitiveLevelMessage to v1pb.SensitiveLevel.
func convertToSensitiveLevel(ctx context.Context, sl *store.SensitiveLevelMessage) *v1pb.SensitiveLevel {
	if sl == nil {
		return nil
	}

	return &v1pb.SensitiveLevel{
		Name:        fmt.Sprintf("sensitiveLevels/%s", sl.ID),
		DisplayName: sl.DisplayName,
		Description: sl.Description,
		Level:       v1pb.SensitivityLevel(sl.Level),
		TableName:   sl.TableName,
		SchemaName:  sl.SchemaName,
		InstanceId:  sl.InstanceId,
		FieldRules:  convertFieldRulesFromStore(sl.FieldRules),
		CreateTime:  utils.TimestampFromTime(sl.CreatedAt),
		UpdateTime:  utils.TimestampFromTime(sl.UpdatedAt),
	}
}

// convertFieldRulesFromStore converts store.FieldRuleMessage to v1pb.FieldMatchingRule.
func convertFieldRulesFromStore(rules []*store.FieldRuleMessage) []*v1pb.FieldMatchingRule {
	result := make([]*v1pb.FieldMatchingRule, 0, len(rules))
	for _, rule := range rules {
		result = append(result, &v1pb.FieldMatchingRule{
			Type:        v1pb.MatchingRuleType(rule.Type),
			Pattern:     rule.Pattern,
			Description: rule.Description,
		})
	}
	return result
}

// convertFieldRulesToStore converts v1pb.FieldMatchingRule to store.FieldRuleMessage.
func convertFieldRulesToStore(rules []*v1pb.FieldMatchingRule) []*store.FieldRuleMessage {
	result := make([]*store.FieldRuleMessage, 0, len(rules))
	for _, rule := range rules {
		result = append(result, &store.FieldRuleMessage{
			Type:        int32(rule.Type),
			Pattern:     rule.Pattern,
			Description: rule.Description,
		})
	}
	return result
}