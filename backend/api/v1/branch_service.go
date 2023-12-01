package v1

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// NewBranchService implements SchemaDesignServiceServer interface.
type BranchService struct {
	v1pb.UnimplementedSchemaDesignServiceServer
	store          *store.Store
	licenseService enterprise.LicenseService
}

// NewBranchService creates a new SchemaDesignService.
func NewBranchService(store *store.Store, licenseService enterprise.LicenseService) *BranchService {
	return &BranchService{
		store:          store,
		licenseService: licenseService,
	}
}

// GetSchemaDesign gets the schema design.
func (s *BranchService) GetSchemaDesign(ctx context.Context, request *v1pb.GetSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	return nil, nil
}

// ListSchemaDesigns lists schema designs.
func (s *BranchService) ListSchemaDesigns(ctx context.Context, request *v1pb.ListSchemaDesignsRequest) (*v1pb.ListSchemaDesignsResponse, error) {
	return nil, nil
}

// CreateSchemaDesign creates a new schema design.
func (s *BranchService) CreateSchemaDesign(ctx context.Context, request *v1pb.CreateSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	return nil, nil
}

// UpdateSchemaDesign updates an existing schema design.
func (s *BranchService) UpdateSchemaDesign(ctx context.Context, request *v1pb.UpdateSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	return nil, nil
}

// MergeSchemaDesign merges a personal draft schema design to the target schema design.
func (s *BranchService) MergeSchemaDesign(ctx context.Context, request *v1pb.MergeSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	return nil, nil
}

// DeleteSchemaDesign deletes an existing schema design.
func (s *BranchService) DeleteSchemaDesign(ctx context.Context, request *v1pb.DeleteSchemaDesignRequest) (*emptypb.Empty, error) {
	return nil, nil
}
