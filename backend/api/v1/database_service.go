package v1

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	return convertToDatabase(database), nil
}

// ListDatabases lists all databases.
func (s *DatabaseService) ListDatabases(ctx context.Context, request *v1pb.ListDatabasesRequest) (*v1pb.ListDatabasesResponse, error) {
	environmentID, instanceID, err := getEnvironmentInstanceID(request.Parent)
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
		response.Databases = append(response.Databases, convertToDatabase(database))
	}
	return response, nil
}

// UpdateDatabase updates a database.
func (s *DatabaseService) UpdateDatabase(ctx context.Context, request *v1pb.UpdateDatabaseRequest) (*v1pb.Database, error) {
	if request.Database == nil {
		return nil, status.Errorf(codes.InvalidArgument, "database must be set")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	environmentID, instanceID, databaseName, err := getEnvironmentInstanceDatabaseID(request.Database.Name)
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
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}

	patch := &store.UpdateDatabaseMessage{}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "database.project":
			projectID, err := getProjectID(request.Database.Project)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
				ResourceID:  &projectID,
				ShowDeleted: true,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, err.Error())
			}
			if project == nil {
				return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
			}

			patch.ProjectID = &project.ResourceID
		case "database.labels":
			patch.Labels = &request.Database.Labels
		}
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	database, err = s.store.UpdateDatabase(ctx, patch, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToDatabase(database), nil
}

// BatchUpdateDatabases updates a database in batch.
func (s *DatabaseService) BatchUpdateDatabases(ctx context.Context, request *v1pb.BatchUpdateDatabasesRequest) (*v1pb.BatchUpdateDatabasesResponse, error) {
	var databases []*store.DatabaseMessage
	projectURI := ""
	for _, req := range request.Requests {
		if req.Database == nil {
			return nil, status.Errorf(codes.InvalidArgument, "database must be set")
		}
		if req.UpdateMask == nil {
			return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
		}
		environmentID, instanceID, databaseName, err := getEnvironmentInstanceDatabaseID(req.Database.Name)
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
			return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
		}
		if projectURI != "" && projectURI != req.Database.Project {
			return nil, status.Errorf(codes.InvalidArgument, "database should use the same project")
		}
		projectURI = req.Database.Project
		databases = append(databases, database)
	}
	projectID, err := getProjectID(projectURI)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID:  &projectID,
		ShowDeleted: true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	databases, err = s.store.BatchUpdateDatabaseProject(ctx, databases, project.ResourceID, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	response := &v1pb.BatchUpdateDatabasesResponse{}
	for _, database := range databases {
		response.Databases = append(response.Databases, convertToDatabase(database))
	}
	return response, nil
}

// GetDatabaseMetadata gets the metadata of a database.
func (s *DatabaseService) GetDatabaseMetadata(ctx context.Context, request *v1pb.GetDatabaseMetadataRequest) (*v1pb.DatabaseMetadata, error) {
	name, err := trimSuffix(request.Name, "/metadata")
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	environmentID, instanceID, databaseName, err := getEnvironmentInstanceDatabaseID(name)
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
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertDatabaseMetadata(dbSchema.Metadata), nil
}

// GetDatabaseSchema gets the schema of a database.
func (s *DatabaseService) GetDatabaseSchema(ctx context.Context, request *v1pb.GetDatabaseSchemaRequest) (*v1pb.DatabaseSchema, error) {
	name, err := trimSuffix(request.Name, "/schema")
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	environmentID, instanceID, databaseName, err := getEnvironmentInstanceDatabaseID(name)
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
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if dbSchema == nil {
		return nil, status.Errorf(codes.NotFound, "database schema %q not found", databaseName)
	}
	return &v1pb.DatabaseSchema{Schema: string(dbSchema.Schema)}, nil
}

func convertToDatabase(database *store.DatabaseMessage) *v1pb.Database {
	syncState := v1pb.State_STATE_UNSPECIFIED
	switch database.SyncState {
	case api.OK:
		syncState = v1pb.State_ACTIVE
	case api.NotFound:
		syncState = v1pb.State_DELETED
	}
	return &v1pb.Database{
		Name:               fmt.Sprintf("environments/%s/instances/%s/databases/%s", database.EnvironmentID, database.InstanceID, database.DatabaseName),
		Uid:                fmt.Sprintf("%d", database.UID),
		SyncState:          syncState,
		SuccessfulSyncTime: timestamppb.New(time.Unix(database.SuccessfulSyncTimeTs, 0)),
		Project:            fmt.Sprintf("%s%s", projectNamePrefix, database.ProjectID),
		SchemaVersion:      database.SchemaVersion,
		Labels:             database.Labels,
	}
}

func convertDatabaseMetadata(metadata *storepb.DatabaseMetadata) *v1pb.DatabaseMetadata {
	m := &v1pb.DatabaseMetadata{
		Name:         metadata.Name,
		CharacterSet: metadata.CharacterSet,
		Collation:    metadata.Collation,
	}
	for _, schema := range metadata.Schemas {
		s := &v1pb.SchemaMetadata{
			Name: schema.Name,
		}
		for _, table := range schema.Tables {
			t := &v1pb.TableMetadata{
				Name:          table.Name,
				Engine:        table.Engine,
				Collation:     table.Collation,
				RowCount:      table.RowCount,
				DataSize:      table.DataSize,
				IndexSize:     table.IndexSize,
				DataFree:      table.DataFree,
				CreateOptions: table.CreateOptions,
				Comment:       table.Comment,
			}
			for _, column := range table.Columns {
				t.Columns = append(t.Columns, &v1pb.ColumnMetadata{
					Name:         column.Name,
					Position:     column.Position,
					Default:      column.Default,
					Nullable:     column.Nullable,
					Type:         column.Type,
					CharacterSet: column.CharacterSet,
					Collation:    column.Collation,
					Comment:      column.Comment,
				})
			}
			for _, index := range table.Indexes {
				t.Indexes = append(t.Indexes, &v1pb.IndexMetadata{
					Name:        index.Name,
					Expressions: index.Expressions,
					Type:        index.Type,
					Unique:      index.Unique,
					Primary:     index.Primary,
					Visible:     index.Visible,
					Comment:     index.Comment,
				})
			}
			for _, foreignKey := range table.ForeignKeys {
				t.ForeignKeys = append(t.ForeignKeys, &v1pb.ForeignKeyMetadata{
					Name:              foreignKey.Name,
					Columns:           foreignKey.Columns,
					ReferencedSchema:  foreignKey.ReferencedSchema,
					ReferencedTable:   foreignKey.ReferencedTable,
					ReferencedColumns: foreignKey.ReferencedColumns,
					OnDelete:          foreignKey.OnDelete,
					OnUpdate:          foreignKey.OnUpdate,
					MatchType:         foreignKey.MatchType,
				})
			}
			s.Tables = append(s.Tables, t)
		}
		for _, view := range schema.Views {
			s.Views = append(s.Views, &v1pb.ViewMetadata{
				Name:       view.Name,
				Definition: view.Definition,
				Comment:    view.Comment,
			})
		}
		m.Schemas = append(m.Schemas, s)
	}
	for _, extension := range metadata.Extensions {
		m.Extensions = append(m.Extensions, &v1pb.ExtensionMetadata{
			Name:        extension.Name,
			Schema:      extension.Schema,
			Version:     extension.Version,
			Description: extension.Description,
		})
	}
	return m
}
