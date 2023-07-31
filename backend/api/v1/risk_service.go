package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// RiskService implements the risk service.
type RiskService struct {
	v1pb.UnimplementedRiskServiceServer
	store          *store.Store
	licenseService enterpriseAPI.LicenseService
}

// NewRiskService creates a new RiskService.
func NewRiskService(store *store.Store, licenseService enterpriseAPI.LicenseService) *RiskService {
	return &RiskService{
		store:          store,
		licenseService: licenseService,
	}
}

func convertToRisk(risk *store.RiskMessage) (*v1pb.Risk, error) {
	return &v1pb.Risk{
		Name:      fmt.Sprintf("%s%v", common.RiskPrefix, risk.ID),
		Uid:       fmt.Sprintf("%v", risk.ID),
		Source:    convertToSource(risk.Source),
		Title:     risk.Name,
		Level:     risk.Level,
		Condition: risk.Expression,
		Active:    risk.Active,
	}, nil
}

// ListRisks lists risks.
func (s *RiskService) ListRisks(ctx context.Context, _ *v1pb.ListRisksRequest) (*v1pb.ListRisksResponse, error) {
	risks, err := s.store.ListRisks(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.ListRisksResponse{}
	for _, risk := range risks {
		r, err := convertToRisk(risk)
		if err != nil {
			return nil, err
		}
		response.Risks = append(response.Risks, r)
	}
	return response, nil
}

// CreateRisk creates a risk.
func (s *RiskService) CreateRisk(ctx context.Context, request *v1pb.CreateRiskRequest) (*v1pb.Risk, error) {
	if err := s.licenseService.IsFeatureEnabled(api.FeatureCustomApproval); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}
	// Validate the condition.
	if _, err := common.ConvertUnparsedRisk(request.Risk.Condition); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to validate risk expression, error: %v", err)
	}
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)

	risk, err := s.store.CreateRisk(ctx, &store.RiskMessage{
		Source:     convertSource(request.Risk.Source),
		Level:      request.Risk.Level,
		Name:       request.Risk.Title,
		Active:     request.Risk.Active,
		Expression: request.Risk.Condition,
	}, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToRisk(risk)
}

// UpdateRisk updates a risk.
func (s *RiskService) UpdateRisk(ctx context.Context, request *v1pb.UpdateRiskRequest) (*v1pb.Risk, error) {
	if err := s.licenseService.IsFeatureEnabled(api.FeatureCustomApproval); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, err.Error())
	}
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}
	riskID, err := common.GetRiskID(request.Risk.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	risk, err := s.store.GetRisk(ctx, riskID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get risk, error: %v", err)
	}
	if risk == nil {
		return nil, status.Errorf(codes.NotFound, "risk %v not found", request.Risk.Name)
	}
	if risk.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "risk %v has been deleted", request.Risk.Name)
	}

	patch := &store.UpdateRiskMessage{}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Name = &request.Risk.Title
		case "active":
			patch.Active = &request.Risk.Active
		case "level":
			patch.Level = &request.Risk.Level
		case "condition":
			if _, err := common.ConvertUnparsedRisk(request.Risk.Condition); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "failed to validate risk expression, error: %v", err)
			}
			patch.Expression = request.Risk.Condition
		}
	}

	risk, err = s.store.UpdateRisk(ctx, patch, riskID, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertToRisk(risk)
}

// DeleteRisk deletes a risk.
func (s *RiskService) DeleteRisk(ctx context.Context, request *v1pb.DeleteRiskRequest) (*emptypb.Empty, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	riskID, err := common.GetRiskID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	risk, err := s.store.GetRisk(ctx, riskID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get risk, error: %v", err)
	}
	if risk == nil {
		return nil, status.Errorf(codes.NotFound, "risk %v not found", request.Name)
	}
	if risk.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "risk %v has been deleted", request.Name)
	}

	rowStatusArchived := api.Archived
	if _, err := s.store.UpdateRisk(ctx,
		&store.UpdateRiskMessage{
			RowStatus: &rowStatusArchived,
		},
		riskID,
		principalID); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func convertToSource(source store.RiskSource) v1pb.Risk_Source {
	switch source {
	case store.RiskSourceDatabaseCreate:
		return v1pb.Risk_CREATE_DATABASE
	case store.RiskSourceDatabaseSchemaUpdate:
		return v1pb.Risk_DDL
	case store.RiskSourceDatabaseDataUpdate:
		return v1pb.Risk_DML
	case store.RiskRequestQuery:
		return v1pb.Risk_QUERY
	case store.RiskRequestExport:
		return v1pb.Risk_EXPORT
	}
	return v1pb.Risk_SOURCE_UNSPECIFIED
}

func convertSource(source v1pb.Risk_Source) store.RiskSource {
	switch source {
	case v1pb.Risk_CREATE_DATABASE:
		return store.RiskSourceDatabaseCreate
	case v1pb.Risk_DDL:
		return store.RiskSourceDatabaseSchemaUpdate
	case v1pb.Risk_DML:
		return store.RiskSourceDatabaseDataUpdate
	case v1pb.Risk_QUERY:
		return store.RiskRequestQuery
	case v1pb.Risk_EXPORT:
		return store.RiskRequestExport
	}
	return store.RiskSourceUnknown
}
