package v1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/store"
)

const databaseNamePrefix = "databases/"

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
func (s *DatabaseService) GetDatabase(ctx context.Context, request *v1pb.GetDatabaseRequest) (*v1pb.Database, error) {
	environmentID, instanceID, databaseName, err := getEnvironmentInstanceDatabaseID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		EnvironmentID: &environmentID,
		InstanceID:    &instanceID,
		DatabaseName:  &databaseName,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.InvalidArgument, "database %q not found", databaseName)
	}
	return convertDatabase(database), nil
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
	if request.Filter != "" {
		projectFilter, err := getFilter(request.Filter, "project")
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		projectID, err := getProjectID(projectFilter)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid project %q in the filter", projectFilter)
		}
		find.ProjectID = &projectID
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

func getEnvironmentInstanceDatabaseID(name string) (string, string, string, error) {
	// the instance request should be environments/{environment-id}/instances/{instance-id}/databases/{database}
	sections := strings.Split(name, "/")
	if len(sections) != 6 {
		return "", "", "", errors.Errorf("invalid request %q", name)
	}

	if fmt.Sprintf("%s/", sections[0]) != environmentNamePrefix {
		return "", "", "", errors.Errorf("invalid request %q", name)
	}
	if fmt.Sprintf("%s/", sections[2]) != instanceNamePrefix {
		return "", "", "", errors.Errorf("invalid request %q", name)
	}
	if fmt.Sprintf("%s/", sections[4]) != databaseNamePrefix {
		return "", "", "", errors.Errorf("invalid request %q", name)
	}

	if sections[1] == "" || sections[3] == "" || sections[5] == "" {
		return "", "", "", errors.Errorf("invalid request %q", name)
	}
	return sections[1], sections[3], sections[5], nil
}
