package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

// SensitiveDataService implements the v1.SensitiveDataService interface.
type SensitiveDataService struct {
	v1pb.UnimplementedSensitiveDataServiceServer

	store *store.Store
}

// NewSensitiveDataService creates a new SensitiveDataService.
func NewSensitiveDataService(store *store.Store) *SensitiveDataService {
	return &SensitiveDataService{
		store: store,
	}
}

// ListSensitiveDataRules implements v1.SensitiveDataServiceServer.ListSensitiveDataRules.
func (s *SensitiveDataService) ListSensitiveDataRules(ctx context.Context, req *v1pb.ListSensitiveDataRulesRequest) (*v1pb.ListSensitiveDataRulesResponse, error) {
	find := &store.FindSensitiveDataRuleMessage{}
	if req.Level != storepb.SensitiveDataLevel_SENSITIVE_DATA_LEVEL_UNSPECIFIED {
		find.Level = &req.Level
	}
	if req.Enabled {
		find.Enabled = &req.Enabled
	}

	rules, err := s.store.ListSensitiveDataRules(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list sensitive data rules: %v", err)
	}

	return &v1pb.ListSensitiveDataRulesResponse{
		Rules:        rules,
		NextPageToken: "", // TODO: Implement pagination
		TotalSize:     int32(len(rules)),
	}, nil
}

// GetSensitiveDataRule implements v1.SensitiveDataServiceServer.GetSensitiveDataRule.
func (s *SensitiveDataService) GetSensitiveDataRule(ctx context.Context, req *v1pb.GetSensitiveDataRuleRequest) (*v1pb.GetSensitiveDataRuleResponse, error) {
	rule, err := s.store.GetSensitiveDataRule(ctx, req.RuleId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get sensitive data rule: %v", err)
	}
	if rule == nil {
		return nil, status.Errorf(codes.NotFound, "sensitive data rule not found")
	}

	return &v1pb.GetSensitiveDataRuleResponse{
		Rule: rule,
	}, nil
}

// CreateSensitiveDataRule implements v1.SensitiveDataServiceServer.CreateSensitiveDataRule.
func (s *SensitiveDataService) CreateSensitiveDataRule(ctx context.Context, req *v1pb.CreateSensitiveDataRuleRequest) (*v1pb.CreateSensitiveDataRuleResponse, error) {
	if req.Rule == nil {
		return nil, status.Errorf(codes.InvalidArgument, "rule is required")
	}

	// Set default values
	if req.Rule.Enabled == false {
		req.Rule.Enabled = true
	}

	rule, err := s.store.CreateSensitiveDataRule(ctx, req.Rule)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create sensitive data rule: %v", err)
	}

	return &v1pb.CreateSensitiveDataRuleResponse{
		Rule: rule,
	}, nil
}

// UpdateSensitiveDataRule implements v1.SensitiveDataServiceServer.UpdateSensitiveDataRule.
func (s *SensitiveDataService) UpdateSensitiveDataRule(ctx context.Context, req *v1pb.UpdateSensitiveDataRuleRequest) (*v1pb.UpdateSensitiveDataRuleResponse, error) {
	if req.Rule == nil {
		return nil, status.Errorf(codes.InvalidArgument, "rule is required")
	}

	// Ensure the rule ID matches
	req.Rule.Id = req.RuleId

	rule, err := s.store.UpdateSensitiveDataRule(ctx, req.Rule)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update sensitive data rule: %v", err)
	}

	return &v1pb.UpdateSensitiveDataRuleResponse{
		Rule: rule,
	}, nil
}

// DeleteSensitiveDataRule implements v1.SensitiveDataServiceServer.DeleteSensitiveDataRule.
func (s *SensitiveDataService) DeleteSensitiveDataRule(ctx context.Context, req *v1pb.DeleteSensitiveDataRuleRequest) (*emptypb.Empty, error) {
	err := s.store.DeleteSensitiveDataRule(ctx, req.RuleId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete sensitive data rule: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// ListSensitiveDataMappings implements v1.SensitiveDataServiceServer.ListSensitiveDataMappings.
func (s *SensitiveDataService) ListSensitiveDataMappings(ctx context.Context, req *v1pb.ListSensitiveDataMappingsRequest) (*v1pb.ListSensitiveDataMappingsResponse, error) {
	find := &store.FindSensitiveDataMappingMessage{}
	if req.InstanceId != 0 {
		find.InstanceID = &req.InstanceId
	}
	if req.DatabaseName != "" {
		find.DatabaseName = &req.DatabaseName
	}
	if req.TableName != "" {
		find.TableName = &req.TableName
	}
	if req.Level != storepb.SensitiveDataLevel_SENSITIVE_DATA_LEVEL_UNSPECIFIED {
		find.Level = &req.Level
	}

	mappings, err := s.store.ListSensitiveDataMappings(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list sensitive data mappings: %v", err)
	}

	return &v1pb.ListSensitiveDataMappingsResponse{
		Mappings:     mappings,
		NextPageToken: "", // TODO: Implement pagination
		TotalSize:     int32(len(mappings)),
	}, nil
}

// GetSensitiveDataMapping implements v1.SensitiveDataServiceServer.GetSensitiveDataMapping.
func (s *SensitiveDataService) GetSensitiveDataMapping(ctx context.Context, req *v1pb.GetSensitiveDataMappingRequest) (*v1pb.GetSensitiveDataMappingResponse, error) {
	mapping, err := s.store.GetSensitiveDataMapping(ctx, req.MappingId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get sensitive data mapping: %v", err)
	}
	if mapping == nil {
		return nil, status.Errorf(codes.NotFound, "sensitive data mapping not found")
	}

	return &v1pb.GetSensitiveDataMappingResponse{
		Mapping: mapping,
	}, nil
}

// CreateSensitiveDataMapping implements v1.SensitiveDataServiceServer.CreateSensitiveDataMapping.
func (s *SensitiveDataService) CreateSensitiveDataMapping(ctx context.Context, req *v1pb.CreateSensitiveDataMappingRequest) (*v1pb.CreateSensitiveDataMappingResponse, error) {
	if req.Mapping == nil {
		return nil, status.Errorf(codes.InvalidArgument, "mapping is required")
	}

	mapping, err := s.store.CreateSensitiveDataMapping(ctx, req.Mapping)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create sensitive data mapping: %v", err)
	}

	return &v1pb.CreateSensitiveDataMappingResponse{
		Mapping: mapping,
	}, nil
}

// UpdateSensitiveDataMapping implements v1.SensitiveDataServiceServer.UpdateSensitiveDataMapping.
func (s *SensitiveDataService) UpdateSensitiveDataMapping(ctx context.Context, req *v1pb.UpdateSensitiveDataMappingRequest) (*v1pb.UpdateSensitiveDataMappingResponse, error) {
	if req.Mapping == nil {
		return nil, status.Errorf(codes.InvalidArgument, "mapping is required")
	}

	// Ensure the mapping ID matches
	req.Mapping.Id = req.MappingId

	mapping, err := s.store.UpdateSensitiveDataMapping(ctx, req.Mapping)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update sensitive data mapping: %v", err)
	}

	return &v1pb.UpdateSensitiveDataMappingResponse{
		Mapping: mapping,
	}, nil
}

// DeleteSensitiveDataMapping implements v1.SensitiveDataServiceServer.DeleteSensitiveDataMapping.
func (s *SensitiveDataService) DeleteSensitiveDataMapping(ctx context.Context, req *v1pb.DeleteSensitiveDataMappingRequest) (*emptypb.Empty, error) {
	err := s.store.DeleteSensitiveDataMapping(ctx, req.MappingId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete sensitive data mapping: %v", err)
	}

	return &emptypb.Empty{}, nil
}
