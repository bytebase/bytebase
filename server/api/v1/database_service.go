package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/store"
)

// DatabaseService implements the database service.
type DatabaseService struct {
	v1pb.UnimplementedDatabaseServiceServer
	store *store.Store
}

// NewDatabaseService creates a new DatabaseService.
func NewDatabaseService(store *store.Store) *DatabaseService {
	return &DatabaseService{
		store: store,
	}
}

// GetDatabase gets a database.
func (*DatabaseService) GetDatabase(_ context.Context, _ *v1pb.GetDatabaseRequest) (*v1pb.Database, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDatabase not implemented")
}

// ListDatabases lists all databases.
func (*DatabaseService) ListDatabases(_ context.Context, _ *v1pb.ListDatabasesRequest) (*v1pb.ListDatabasesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListDatabases not implemented")
}

// UpdateDatabase updates a database.
func (*DatabaseService) UpdateDatabase(_ context.Context, _ *v1pb.UpdateDatabaseRequest) (*v1pb.Database, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateDatabase not implemented")
}

// BatchUpdateDatabases updates a database in batch.
func (*DatabaseService) BatchUpdateDatabases(_ context.Context, _ *v1pb.BatchUpdateDatabasesRequest) (*v1pb.BatchUpdateDatabasesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BatchUpdateDatabases not implemented")
}

// GetDatabaseMetadata gets the metadata of a database.
func (*DatabaseService) GetDatabaseMetadata(_ context.Context, _ *v1pb.GetDatabaseMetadataRequest) (*v1pb.DatabaseMetadata, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDatabaseMetadata not implemented")
}

// GetDatabaseSchema gets the schema of a database.
func (*DatabaseService) GetDatabaseSchema(_ context.Context, _ *v1pb.GetDatabaseSchemaRequest) (*v1pb.DatabaseSchema, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDatabaseSchema not implemented")
}
