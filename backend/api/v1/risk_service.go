package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// RiskService implements the risk service.
type RiskService struct {
	v1pb.UnimplementedRiskServiceServer
	store          *store.Store
	licenseService enterprise.LicenseService
}

// NewRiskService creates a new RiskService.
func NewRiskService(store *store.Store, licenseService enterprise.LicenseService) *RiskService {
	return &RiskService{
		store:          store,
		licenseService: licenseService,
	}
}

func convertToRisk(risk *store.RiskMessage) (*v1pb.Risk, error) {
	return &v1pb.Risk{
		Name:      fmt.Sprintf("%s%v", common.RiskPrefix, risk.ID),
		Source:    ConvertToV1Source(risk.Source),
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
		return nil, status.Error(codes.Internal, err.Error())
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

// GetRisk gets the risk.
func (s *RiskService) GetRisk(ctx context.Context, request *v1pb.GetRiskRequest) (*v1pb.Risk, error) {
	risk, err := s.getRiskByName(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	return convertToRisk(risk)
}

// CreateRisk creates a risk.
func (s *RiskService) CreateRisk(ctx context.Context, request *v1pb.CreateRiskRequest) (*v1pb.Risk, error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanLimitConfig_RISK_ASSESSMENT); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	// Validate the condition.
	if _, err := common.ConvertUnparsedRisk(request.Risk.Condition); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to validate risk expression, error: %v", err)
	}

	risk, err := s.store.CreateRisk(ctx, &store.RiskMessage{
		Source:     convertToSource(request.Risk.Source),
		Level:      request.Risk.Level,
		Name:       request.Risk.Title,
		Active:     request.Risk.Active,
		Expression: request.Risk.Condition,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return convertToRisk(risk)
}

// UpdateRisk updates a risk.
func (s *RiskService) UpdateRisk(ctx context.Context, request *v1pb.UpdateRiskRequest) (*v1pb.Risk, error) {
	if err := s.licenseService.IsFeatureEnabled(v1pb.PlanLimitConfig_RISK_ASSESSMENT); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}
	risk, err := s.getRiskByName(ctx, request.Risk.Name)
	if err != nil {
		return nil, err
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
		case "source":
			source := convertToSource(request.Risk.Source)
			if risk.Source != source {
				patch.Source = &source
			}
		}
	}

	risk, err = s.store.UpdateRisk(ctx, patch, risk.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return convertToRisk(risk)
}

// DeleteRisk deletes a risk.
func (s *RiskService) DeleteRisk(ctx context.Context, request *v1pb.DeleteRiskRequest) (*emptypb.Empty, error) {
	riskID, err := common.GetRiskID(request.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := s.store.DeleteRisk(ctx, riskID); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

func (s *RiskService) getRiskByName(ctx context.Context, name string) (*store.RiskMessage, error) {
	riskID, err := common.GetRiskID(name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	risk, err := s.store.GetRisk(ctx, riskID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get risk, error: %v", err)
	}
	if risk == nil {
		return nil, status.Errorf(codes.NotFound, "risk %v not found", name)
	}
	return risk, nil
}

func ConvertToV1Source(source store.RiskSource) v1pb.Risk_Source {
	switch source {
	case store.RiskSourceDatabaseCreate:
		return v1pb.Risk_CREATE_DATABASE
	case store.RiskSourceDatabaseSchemaUpdate:
		return v1pb.Risk_DDL
	case store.RiskSourceDatabaseDataUpdate:
		return v1pb.Risk_DML
	case store.RiskSourceDatabaseDataExport:
		return v1pb.Risk_DATA_EXPORT
	case store.RiskRequestRole:
		return v1pb.Risk_REQUEST_ROLE
	}
	return v1pb.Risk_SOURCE_UNSPECIFIED
}

func convertToSource(source v1pb.Risk_Source) store.RiskSource {
	switch source {
	case v1pb.Risk_CREATE_DATABASE:
		return store.RiskSourceDatabaseCreate
	case v1pb.Risk_DDL:
		return store.RiskSourceDatabaseSchemaUpdate
	case v1pb.Risk_DML:
		return store.RiskSourceDatabaseDataUpdate
	case v1pb.Risk_DATA_EXPORT:
		return store.RiskSourceDatabaseDataExport
	case v1pb.Risk_REQUEST_ROLE:
		return store.RiskRequestRole
	}
	return store.RiskSourceUnknown
}
