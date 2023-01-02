package v1

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/api"
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
func (s *DatabaseService) ListDatabases(ctx context.Context, request *v1pb.ListDatabasesRequest) (*v1pb.ListDatabasesResponse, error) {
	environmentID, instanceID, err := getEnvironmentAndInstanceID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	find := &store.FindDatabaseMessage{}
	if environmentID != "-" {
		find.EnvironmentID = &environmentID
	}
	if instanceID != "-" {
		find.InstanceID = &instanceID
	}
	databases, err := s.store.ListDatabases(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.ListDatabasesResponse{}
	for _, database := range databases {
		response.Databases = append(response.Databases, convertDatabase(database))
	}
	return response, nil
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

func convertDatabase(database *store.DatabaseMessage) *v1pb.Database {
	syncState := v1pb.State_STATE_UNSPECIFIED
	switch database.SyncState {
	case api.OK:
		syncState = v1pb.State_ACTIVE
	case api.NotFound:
		syncState = v1pb.State_DELETED
	}
	return &v1pb.Database{
		Name:               fmt.Sprintf("environments/%s/instances/%s/databases/%s", database.EnvironmentID, database.InstanceID, database.DatabaseName),
		SyncState:          syncState,
		SuccessfulSyncTime: timestamppb.New(time.Unix(database.SuccessfulSyncTimeTs, 0)),
		Project:            fmt.Sprintf("%s%s", projectNamePrefix, database.ProjectID),
		CharacterSet:       database.CharacterSet,
		Collation:          database.Collation,
		SchemaVersion:      database.SchemaVersion,
		Labels:             database.Labels,
	}
}
