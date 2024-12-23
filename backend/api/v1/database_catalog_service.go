package v1

import (
	"context"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// DatabaseCatalogService implements the database catalog service.
type DatabaseCatalogService struct {
	v1pb.UnimplementedDatabaseCatalogServiceServer
	store          *store.Store
	licenseService enterprise.LicenseService
}

// NewDatabaseCatalogService creates a new DatabaseCatalogService.
func NewDatabaseCatalogService(store *store.Store, licenseService enterprise.LicenseService) *DatabaseCatalogService {
	return &DatabaseCatalogService{
		store:          store,
		licenseService: licenseService,
	}
}

// GetDatabaseCatalog gets a database catalog.
func (s *DatabaseCatalogService) GetDatabaseCatalog(ctx context.Context, request *v1pb.GetDatabaseCatalogRequest) (*v1pb.DatabaseCatalog, error) {
	return nil, nil
}

// UpdateDatabaseCatalog updates a database catalog.
func (s *DatabaseCatalogService) UpdateDatabaseCatalog(ctx context.Context, request *v1pb.UpdateDatabaseCatalogRequest) (*v1pb.DatabaseCatalog, error) {
	return nil, nil
}
