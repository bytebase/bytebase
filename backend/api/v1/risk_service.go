package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// RiskService implements the risk service.
type RiskService struct {
	v1pb.UnimplementedRiskServiceServer
	store *store.Store
}

// NewRiskService creates a new RiskService.
func NewRiskService(store *store.Store) *RiskService {
	return &RiskService{
		store: store,
	}
}

// ListRisks lists risks.
func (*RiskService) ListRisks(context.Context, *v1pb.ListRisksRequest) (*v1pb.ListRisksResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListRisks not implemented")
}

// CreateRisk creates a risk.
func (*RiskService) CreateRisk(context.Context, *v1pb.CreateRiskRequest) (*v1pb.Risk, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateRisk not implemented")
}

// UpdateRisk updates a risk.
func (*RiskService) UpdateRisk(context.Context, *v1pb.UpdateRiskRequest) (*v1pb.Risk, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateRisk not implemented")
}

// DeleteRisk deletes a risk.
func (*RiskService) DeleteRisk(context.Context, *v1pb.DeleteRiskRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteRisk not implemented")
}
