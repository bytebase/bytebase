package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/pkg/errors"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/runner/backuprun"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

const (
	filterKeyProject   = "project"
	filterKeyStartTime = "start_time"

	// Support order by count, latest_log_time, average_query_time, maximum_query_time,
	// average_rows_sent, maximum_rows_sent, average_rows_examined, maximum_rows_examined for now.
	orderByKeyCount               = "count"
	orderByKeyLatestLogTime       = "latest_log_time"
	orderByKeyAverageQueryTime    = "average_query_time"
	orderByKeyMaximumQueryTime    = "maximum_query_time"
	orderByKeyAverageRowsSent     = "average_rows_sent"
	orderByKeyMaximumRowsSent     = "maximum_rows_sent"
	orderByKeyAverageRowsExamined = "average_rows_examined"
	orderByKeyMaximumRowsExamined = "maximum_rows_examined"
)

// DatabaseService implements the database service.
type DatabaseService struct {
	v1pb.UnimplementedDatabaseServiceServer
	store          *store.Store
	BackupRunner   *backuprun.Runner
	licenseService enterpriseAPI.LicenseService
}

// NewDatabaseService creates a new DatabaseService.
func NewDatabaseService(store *store.Store, br *backuprun.Runner, licenseService enterpriseAPI.LicenseService) *DatabaseService {
	return &DatabaseService{
		store:          store,
		BackupRunner:   br,
		licenseService: licenseService,
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

	var project *store.ProjectMessage
	patch := &store.UpdateDatabaseMessage{}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "project":
			projectID, err := getProjectID(request.Database.Project)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			project, err = s.store.GetProjectV2(ctx, &store.FindProjectMessage{
				ResourceID:  &projectID,
				ShowDeleted: true,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, err.Error())
			}
			if project == nil {
				return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
			}
			if project.Deleted {
				return nil, status.Errorf(codes.FailedPrecondition, "project %q is deleted", projectID)
			}
			patch.ProjectID = &project.ResourceID
		case "labels":
			patch.Labels = &request.Database.Labels
		}
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	updatedDatabase, err := s.store.UpdateDatabase(ctx, patch, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project != nil {
		if err := s.createTransferProjectActivity(ctx, project, principalID, database); err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}

	return convertToDatabase(updatedDatabase), nil
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
	if project.Deleted {
		return nil, status.Errorf(codes.FailedPrecondition, "project %q is deleted", projectID)
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	updatedDatabases, err := s.store.BatchUpdateDatabaseProject(ctx, databases, project.ResourceID, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if err := s.createTransferProjectActivity(ctx, project, principalID, databases...); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	response := &v1pb.BatchUpdateDatabasesResponse{}
	for _, database := range updatedDatabases {
		response.Databases = append(response.Databases, convertToDatabase(database))
	}
	return response, nil
}

// GetDatabaseMetadata gets the metadata of a database.
func (s *DatabaseService) GetDatabaseMetadata(ctx context.Context, request *v1pb.GetDatabaseMetadataRequest) (*v1pb.DatabaseMetadata, error) {
	environmentID, instanceID, databaseName, err := trimSuffixAndGetEnvironmentInstanceDatabaseID(request.Name, metadataSuffix)
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
	environmentID, instanceID, databaseName, err := trimSuffixAndGetEnvironmentInstanceDatabaseID(request.Name, schemaSuffix)
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

// GetBackupSetting gets the backup setting of a database.
func (s *DatabaseService) GetBackupSetting(ctx context.Context, request *v1pb.GetBackupSettingRequest) (*v1pb.BackupSetting, error) {
	environmentID, instanceID, databaseName, err := trimSuffixAndGetEnvironmentInstanceDatabaseID(request.Name, backupSettingSuffix)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
		ResourceID: &environmentID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if environment == nil {
		return nil, status.Errorf(codes.NotFound, "environment %q not found", environmentID)
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		EnvironmentID: &environmentID,
		ResourceID:    &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
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
	backupSetting, err := s.store.GetBackupSettingV2(ctx, database.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if backupSetting == nil {
		// If the backup setting is not found, return the default backup setting.
		return getDefaultBackupSetting(environment.ResourceID, instance.ResourceID, database.DatabaseName), nil
	}
	return convertToBackupSetting(backupSetting, environment.ResourceID, instance.ResourceID, database.DatabaseName)
}

// UpdateBackupSetting updates the backup setting of a database.
func (s *DatabaseService) UpdateBackupSetting(ctx context.Context, request *v1pb.UpdateBackupSettingRequest) (*v1pb.BackupSetting, error) {
	environmentID, instanceID, databaseName, err := trimSuffixAndGetEnvironmentInstanceDatabaseID(request.Setting.Name, backupSettingSuffix)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
		ResourceID: &environmentID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if environment == nil {
		return nil, status.Errorf(codes.NotFound, "environment %q not found", environmentID)
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		EnvironmentID: &environmentID,
		ResourceID:    &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
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
	backupSetting, err := validateAndConvertToStoreBackupSetting(request.Setting, database.UID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	backupSetting, err = s.store.UpsertBackupSettingV2(ctx, principalID, backupSetting)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToBackupSetting(backupSetting, environment.ResourceID, instance.ResourceID, database.DatabaseName)
}

// ListBackup lists the backups of a database.
func (s *DatabaseService) ListBackup(ctx context.Context, request *v1pb.ListBackupRequest) (*v1pb.ListBackupResponse, error) {
	environmentID, instanceID, databaseName, err := getEnvironmentInstanceDatabaseID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
		ResourceID: &environmentID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if environment == nil {
		return nil, status.Errorf(codes.NotFound, "environment %q not found", environmentID)
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		EnvironmentID: &environmentID,
		ResourceID:    &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
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

	rowStatus := api.Normal
	existedBackupList, err := s.store.ListBackupV2(ctx, &store.FindBackupMessage{
		DatabaseUID: &database.UID,
		RowStatus:   &rowStatus,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	var backupList []*v1pb.Backup
	for _, existedBackup := range existedBackupList {
		backupList = append(backupList, convertToBackup(existedBackup, environment.ResourceID, instance.ResourceID, database.DatabaseName))
	}
	return &v1pb.ListBackupResponse{
		Backups: backupList,
	}, nil
}

// CreateBackup creates a backup of a database.
func (s *DatabaseService) CreateBackup(ctx context.Context, request *v1pb.CreateBackupRequest) (*v1pb.Backup, error) {
	environmentID, instanceID, databaseName, backupName, err := getEnvironmentIDInstanceDatabaseIDBackupName(request.Backup.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
		ResourceID: &environmentID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if environment == nil {
		return nil, status.Errorf(codes.NotFound, "environment %q not found", environmentID)
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		EnvironmentID: &environmentID,
		ResourceID:    &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
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

	existedBackupList, err := s.store.ListBackupV2(ctx, &store.FindBackupMessage{
		DatabaseUID: &database.UID,
		Name:        &backupName,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if len(existedBackupList) > 0 {
		return nil, status.Errorf(codes.AlreadyExists, "backup %q in database %q already exists", backupName, databaseName)
	}

	creatorID := ctx.Value(common.PrincipalIDContextKey).(int)
	backup, err := s.BackupRunner.ScheduleBackupTask(ctx, database, backupName, api.BackupTypeManual, creatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToBackup(backup, environmentID, instanceID, databaseName), nil
}

// ListSecrets lists the secrets of a database.
func (s *DatabaseService) ListSecrets(ctx context.Context, request *v1pb.ListSecretsRequest) (*v1pb.ListSecretsResponse, error) {
	environmentID, instanceID, databaseName, err := getEnvironmentInstanceDatabaseID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
		ResourceID: &environmentID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if environment == nil {
		return nil, status.Errorf(codes.NotFound, "environment %q not found", environmentID)
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		EnvironmentID: &environmentID,
		ResourceID:    &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
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
	return &v1pb.ListSecretsResponse{
		Secrets: stripeAndConvertToServiceSecrets(database.Secrets, database.EnvironmentID, database.InstanceID, database.DatabaseName),
	}, nil
}

// UpdateSecret updates a secret of a database.
func (s *DatabaseService) UpdateSecret(ctx context.Context, request *v1pb.UpdateSecretRequest) (*v1pb.Secret, error) {
	if !s.licenseService.IsFeatureEnabled(api.FeatureEncryptedSecrets) {
		return nil, status.Errorf(codes.PermissionDenied, api.FeatureEncryptedSecrets.AccessErrorMessage())
	}

	if request.Secret == nil {
		return nil, status.Errorf(codes.InvalidArgument, "secret is required")
	}

	environmentID, instanceID, databaseName, updateSecretName, err := getEnvironmentInstanceDatabaseIDSecretName(request.Secret.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
		ResourceID: &environmentID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if environment == nil {
		return nil, status.Errorf(codes.NotFound, "environment %q not found", environmentID)
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		EnvironmentID: &environmentID,
		ResourceID:    &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
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

	// We retrieve the secret from the database, convert secrets to map, upsert the new secret and store it back.
	// But if two processes are doing this at the same time, the second one will override the first one.
	// It is not a big deal for now because it's not a common case and users can find that secret he set is not existed or not correct.
	secretsMap := make(map[string]*storepb.SecretItem)
	if database.Secrets != nil {
		for _, secret := range database.Secrets.Items {
			secretsMap[secret.Name] = secret
		}
	}

	var newSecret storepb.SecretItem
	if _, ok := secretsMap[updateSecretName]; !ok {
		// If the secret is not existed and allow_missing is false, we will not create it.
		if !request.AllowMissing {
			return nil, status.Errorf(codes.NotFound, "secret %q not found", updateSecretName)
		}
		newSecret.Name = updateSecretName
		newSecret.Value = request.Secret.Value
		newSecret.Description = request.Secret.Description
	} else {
		oldSecret := secretsMap[updateSecretName]
		newSecret.Name = oldSecret.Name
		newSecret.Value = oldSecret.Value
		newSecret.Description = oldSecret.Description
		for _, path := range request.UpdateMask.Paths {
			switch path {
			case "value":
				newSecret.Value = request.Secret.Value
			case "name":
				// We don't allow users to update the name of a secret.
				return nil, status.Errorf(codes.InvalidArgument, "name of a secret is not allowed to be updated")
			case "description":
				newSecret.Description = request.Secret.Description
			}
		}
	}
	if err := isSecretValid(&newSecret); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	secretsMap[updateSecretName] = &newSecret
	// Flatten the map to a slice.
	var secretItems []*storepb.SecretItem
	for _, secret := range secretsMap {
		secretItems = append(secretItems, secret)
	}
	var updateDatabaseMessage store.UpdateDatabaseMessage
	updateDatabaseMessage.Secrets = &storepb.Secrets{
		Items: secretItems,
	}
	updateDatabaseMessage.EnvironmentID = database.EnvironmentID
	updateDatabaseMessage.InstanceID = database.InstanceID
	updateDatabaseMessage.DatabaseName = database.DatabaseName
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	updatedDatabase, err := s.store.UpdateDatabase(ctx, &updateDatabaseMessage, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	// Get the secret from the updated database.
	for _, secret := range updatedDatabase.Secrets.Items {
		if secret.Name == updateSecretName {
			return stripeAndConvertToServiceSecret(secret, updatedDatabase.EnvironmentID, updatedDatabase.InstanceID, updatedDatabase.DatabaseName), nil
		}
	}
	return &v1pb.Secret{}, nil
}

// DeleteSecret deletes a secret of a database.
func (s *DatabaseService) DeleteSecret(ctx context.Context, request *v1pb.DeleteSecretRequest) (*emptypb.Empty, error) {
	if !s.licenseService.IsFeatureEnabled(api.FeatureEncryptedSecrets) {
		return nil, status.Errorf(codes.PermissionDenied, api.FeatureEncryptedSecrets.AccessErrorMessage())
	}

	environmentID, instanceID, databaseName, secretName, err := getEnvironmentInstanceDatabaseIDSecretName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{
		ResourceID: &environmentID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if environment == nil {
		return nil, status.Errorf(codes.NotFound, "environment %q not found", environmentID)
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		EnvironmentID: &environmentID,
		ResourceID:    &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
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

	// We retrieve the secret from the database, convert secrets to map, upsert the new secret and store it back.
	// But if two processes are doing this at the same time, the second one will override the first one.
	// It is not a big deal for now because it's not a common case and users can find that secret he set is not existed or not correct.
	secretsMap := make(map[string]*storepb.SecretItem)
	if database.Secrets != nil {
		for _, secret := range database.Secrets.Items {
			secretsMap[secret.Name] = secret
		}
	}
	delete(secretsMap, secretName)

	// Flatten the map to a slice.
	var secretItems []*storepb.SecretItem
	for _, secret := range secretsMap {
		secretItems = append(secretItems, secret)
	}
	var updateDatabaseMessage store.UpdateDatabaseMessage
	updateDatabaseMessage.Secrets = &storepb.Secrets{
		Items: secretItems,
	}
	updateDatabaseMessage.EnvironmentID = database.EnvironmentID
	updateDatabaseMessage.InstanceID = database.InstanceID
	updateDatabaseMessage.DatabaseName = database.DatabaseName
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	if _, err := s.store.UpdateDatabase(ctx, &updateDatabaseMessage, principalID); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

type totalValue struct {
	totalQueryTime time.Duration
	totalCount     int64
}

// ListSlowQueries lists the slow queries.
func (s *DatabaseService) ListSlowQueries(ctx context.Context, request *v1pb.ListSlowQueriesRequest) (*v1pb.ListSlowQueriesResponse, error) {
	findDatabase := &store.FindDatabaseMessage{}
	environmentID, instanceID, databaseName, err := getEnvironmentInstanceDatabaseID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if environmentID != "-" {
		findDatabase.EnvironmentID = &environmentID
	}
	if instanceID != "-" {
		findDatabase.InstanceID = &instanceID
	}
	if databaseName != "-" {
		findDatabase.DatabaseName = &databaseName
	}

	filters, err := parseFilter(request.Filter)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	var startLogDate, endLogDate *time.Time
	for _, expr := range filters {
		switch expr.key {
		case filterKeyProject:
			reg := regexp.MustCompile(`^projects/(.+)`)
			match := reg.FindStringSubmatch(expr.value)
			if len(match) != 2 {
				return nil, status.Errorf(codes.InvalidArgument, "invalid project filter %q", expr.value)
			}
			findDatabase.ProjectID = &match[1]
		case filterKeyStartTime:
			switch expr.comparator {
			case comparatorTypeGreater:
				if startLogDate != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid filter %q", request.Filter)
				}
				t, err := time.Parse(time.RFC3339, expr.value)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q", expr.value)
				}
				t = t.AddDate(0, 0, 1).UTC()
				startLogDate = &t
			case comparatorTypeGreaterEqual:
				if startLogDate != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid filter %q", request.Filter)
				}
				t, err := time.Parse(time.RFC3339, expr.value)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q", expr.value)
				}
				t = t.UTC()
				startLogDate = &t
			case comparatorTypeLess:
				if endLogDate != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid filter %q", request.Filter)
				}
				t, err := time.Parse(time.RFC3339, expr.value)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q", expr.value)
				}
				t = t.UTC()
				endLogDate = &t
			case comparatorTypeLessEqual:
				if endLogDate != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid filter %q", request.Filter)
				}
				t, err := time.Parse(time.RFC3339, expr.value)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q", expr.value)
				}
				t = t.AddDate(0, 0, 1).UTC()
				endLogDate = &t
			default:
				return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q %q %q", expr.key, expr.comparator, expr.value)
			}
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid filter key %q", expr.key)
		}
	}

	orderByKeys, err := parseOrderBy(request.OrderBy)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := validSlowQueryOrderByKey(orderByKeys); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	databases, err := s.store.ListDatabases(ctx, findDatabase)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find database list %q", err.Error())
	}

	var canAccessDBs []*store.DatabaseMessage

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	user, err := s.store.GetUserByID(ctx, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user %q", err.Error())
	}
	switch user.Role {
	case api.Owner, api.DBA:
		canAccessDBs = databases
	case api.Developer:
		for _, database := range databases {
			policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &database.ProjectID})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to find project policy %q", err.Error())
			}
			if isProjectOwnerOrDeveloper(principalID, policy) {
				canAccessDBs = append(canAccessDBs, database)
			}
		}
	default:
		return nil, status.Errorf(codes.PermissionDenied, "unknown role %q", user.Role)
	}

	result := &v1pb.ListSlowQueriesResponse{}
	instanceMap := make(map[string]*totalValue)

	for _, database := range canAccessDBs {
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
			ResourceID: &database.InstanceID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to find instance %q", err.Error())
		}
		if instance == nil {
			return nil, status.Errorf(codes.NotFound, "instance %q not found", database.InstanceID)
		}
		listSlowQuery := &store.ListSlowQueryMessage{
			InstanceUID:  &instance.UID,
			DatabaseUID:  &database.UID,
			StartLogDate: startLogDate,
			EndLogDate:   endLogDate,
		}
		logs, err := s.store.ListSlowQuery(ctx, listSlowQuery)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to find slow query %q", err.Error())
		}

		for _, log := range logs {
			result.SlowQueryLogs = append(result.SlowQueryLogs, convertToSlowQueryLog(database.EnvironmentID, database.InstanceID, database.DatabaseName, database.ProjectID, log))
			if value, exists := instanceMap[database.InstanceID]; exists {
				value.totalQueryTime += log.Statistics.AverageQueryTime.AsDuration() * time.Duration(log.Statistics.Count)
				value.totalCount += log.Statistics.Count
			} else {
				instanceMap[database.InstanceID] = &totalValue{
					totalQueryTime: log.Statistics.AverageQueryTime.AsDuration() * time.Duration(log.Statistics.Count),
					totalCount:     log.Statistics.Count,
				}
			}
		}
	}

	for _, log := range result.SlowQueryLogs {
		_, instanceID, _, err := getEnvironmentInstanceDatabaseID(log.Resource)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get instance id %q", err.Error())
		}
		totalQueryTime := log.Statistics.AverageQueryTime.AsDuration() * time.Duration(log.Statistics.Count)
		log.Statistics.QueryTimePercent = float64(totalQueryTime) / float64(instanceMap[instanceID].totalQueryTime)
		log.Statistics.CountPercent = float64(log.Statistics.Count) / float64(instanceMap[instanceID].totalCount)
	}

	result, err = sortSlowQueryLogResponse(result, orderByKeys)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to sort slow query logs %q", err.Error())
	}

	return result, nil
}

func sortSlowQueryLogResponse(response *v1pb.ListSlowQueriesResponse, orderByKeys []orderByKey) (*v1pb.ListSlowQueriesResponse, error) {
	if len(orderByKeys) == 0 {
		orderByKeys = []orderByKey{
			{
				key:      orderByKeyAverageQueryTime,
				isAscend: false,
			},
		}
	}

	if err := validSlowQueryOrderByKey(orderByKeys); err != nil {
		return nil, err
	}

	sort.Slice(response.SlowQueryLogs, func(i, j int) bool {
		for _, key := range orderByKeys {
			switch key.key {
			case orderByKeyCount:
				lCount := response.SlowQueryLogs[i].Statistics.Count
				rCount := response.SlowQueryLogs[j].Statistics.Count
				if lCount != rCount {
					if key.isAscend {
						return lCount < rCount
					}
					return lCount > rCount
				}
			case orderByKeyLatestLogTime:
				lTime := response.SlowQueryLogs[i].Statistics.LatestLogTime.AsTime()
				rTime := response.SlowQueryLogs[j].Statistics.LatestLogTime.AsTime()
				if !lTime.Equal(rTime) {
					if key.isAscend {
						return lTime.Before(rTime)
					}
					return lTime.After(rTime)
				}
			case orderByKeyAverageQueryTime:
				lTime := response.SlowQueryLogs[i].Statistics.AverageQueryTime.AsDuration()
				rTime := response.SlowQueryLogs[j].Statistics.AverageQueryTime.AsDuration()
				if lTime != rTime {
					if key.isAscend {
						return lTime < rTime
					}
					return lTime > rTime
				}
			case orderByKeyMaximumQueryTime:
				lDuration := response.SlowQueryLogs[i].Statistics.MaximumQueryTime.AsDuration()
				rDuration := response.SlowQueryLogs[j].Statistics.MaximumQueryTime.AsDuration()
				if lDuration != rDuration {
					if key.isAscend {
						return lDuration < rDuration
					}
					return lDuration > rDuration
				}
			case orderByKeyAverageRowsSent:
				lSent := response.SlowQueryLogs[i].Statistics.AverageRowsSent
				rSent := response.SlowQueryLogs[j].Statistics.AverageRowsSent
				if lSent != rSent {
					if key.isAscend {
						return lSent < rSent
					}
					return lSent > rSent
				}
			case orderByKeyMaximumRowsSent:
				lSent := response.SlowQueryLogs[i].Statistics.MaximumRowsSent
				rSent := response.SlowQueryLogs[j].Statistics.MaximumRowsSent
				if lSent != rSent {
					if key.isAscend {
						return lSent < rSent
					}
					return lSent > rSent
				}
			case orderByKeyAverageRowsExamined:
				lExamined := response.SlowQueryLogs[i].Statistics.AverageRowsExamined
				rExamined := response.SlowQueryLogs[j].Statistics.AverageRowsExamined
				if lExamined != rExamined {
					if key.isAscend {
						return lExamined < rExamined
					}
					return lExamined > rExamined
				}
			case orderByKeyMaximumRowsExamined:
				lExamined := response.SlowQueryLogs[i].Statistics.MaximumRowsExamined
				rExamined := response.SlowQueryLogs[j].Statistics.MaximumRowsExamined
				if lExamined != rExamined {
					if key.isAscend {
						return lExamined < rExamined
					}
					return lExamined > rExamined
				}
			}
		}
		return false
	})

	return response, nil
}

func validSlowQueryOrderByKey(keys []orderByKey) error {
	for _, key := range keys {
		switch key.key {
		// Support order by count, latest_log_time, average_query_time, maximum_query_time,
		// average_rows_sent, maximum_rows_sent, average_rows_examined, maximum_rows_examined for now.
		case orderByKeyCount, orderByKeyLatestLogTime, orderByKeyAverageQueryTime, orderByKeyMaximumQueryTime,
			orderByKeyAverageRowsSent, orderByKeyMaximumRowsSent, orderByKeyAverageRowsExamined, orderByKeyMaximumRowsExamined:
		default:
			return errors.Errorf("invalid order_by key %q", key.key)
		}
	}
	return nil
}

func convertToSlowQueryLog(environmentID string, instanceID string, databaseName string, projectID string, log *v1pb.SlowQueryLog) *v1pb.SlowQueryLog {
	return &v1pb.SlowQueryLog{
		Resource:   fmt.Sprintf("%s%s/%s%s/%s%s", environmentNamePrefix, environmentID, instanceNamePrefix, instanceID, databaseIDPrefix, databaseName),
		Project:    fmt.Sprintf("%s%s", projectNamePrefix, projectID),
		Statistics: log.Statistics,
	}
}

func convertToBackup(backup *store.BackupMessage, enviromentID string, instanceID string, databaseName string) *v1pb.Backup {
	createTime := timestamppb.New(time.Unix(backup.CreatedTs, 0))
	updateTime := timestamppb.New(time.Unix(backup.UpdatedTs, 0))
	backupState := v1pb.Backup_BACKUP_STATE_UNSPECIFIED
	switch backup.Status {
	case api.BackupStatusPendingCreate:
		backupState = v1pb.Backup_PENDING_CREATE
	case api.BackupStatusDone:
		backupState = v1pb.Backup_DONE
	case api.BackupStatusFailed:
		backupState = v1pb.Backup_FAILED
	}
	backupType := v1pb.Backup_BACKUP_TYPE_UNSPECIFIED
	switch backup.BackupType {
	case api.BackupTypeManual:
		backupType = v1pb.Backup_MANUAL
	case api.BackupTypeAutomatic:
		backupType = v1pb.Backup_AUTOMATIC
	case api.BackupTypePITR:
		backupType = v1pb.Backup_PITR
	}
	return &v1pb.Backup{
		Name:       fmt.Sprintf("%s%s/%s%s/%s%s/%s", environmentNamePrefix, enviromentID, instanceNamePrefix, instanceID, databaseIDPrefix, databaseName, backup.Name),
		CreateTime: createTime,
		UpdateTime: updateTime,
		State:      backupState,
		BackupType: backupType,
		Comment:    backup.Comment,
	}
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
			var dependentColumnList []*v1pb.DependentColumn
			for _, dependentColumn := range view.DependentColumns {
				dependentColumnList = append(dependentColumnList, &v1pb.DependentColumn{
					Schema: dependentColumn.Schema,
					Table:  dependentColumn.Table,
					Column: dependentColumn.Column,
				})
			}

			s.Views = append(s.Views, &v1pb.ViewMetadata{
				Name:             view.Name,
				Definition:       view.Definition,
				Comment:          view.Comment,
				DependentColumns: dependentColumnList,
			})
		}
		for _, function := range schema.Functions {
			s.Functions = append(s.Functions, &v1pb.FunctionMetadata{
				Name:       function.Name,
				Definition: function.Definition,
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

func (s *DatabaseService) createTransferProjectActivity(ctx context.Context, newProject *store.ProjectMessage, updaterID int, databases ...*store.DatabaseMessage) error {
	var creates []*api.ActivityCreate
	for _, database := range databases {
		oldProject, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
		if err != nil {
			return err
		}
		bytes, err := json.Marshal(api.ActivityProjectDatabaseTransferPayload{
			DatabaseID:   database.UID,
			DatabaseName: database.DatabaseName,
		})
		if err != nil {
			return err
		}
		creates = append(creates,
			&api.ActivityCreate{
				CreatorID:   updaterID,
				ContainerID: oldProject.UID,
				Type:        api.ActivityProjectDatabaseTransfer,
				Level:       api.ActivityInfo,
				Comment:     fmt.Sprintf("Transferred out database %q to project %q.", database.DatabaseName, newProject.Title),
				Payload:     string(bytes),
			},
			&api.ActivityCreate{
				CreatorID:   updaterID,
				ContainerID: newProject.UID,
				Type:        api.ActivityProjectDatabaseTransfer,
				Level:       api.ActivityInfo,
				Comment:     fmt.Sprintf("Transferred in database %q from project %q.", database.DatabaseName, oldProject.Title),
				Payload:     string(bytes),
			},
		)
	}
	if _, err := s.store.BatchCreateActivity(ctx, creates); err != nil {
		log.Warn("failed to create activities for database project updates", zap.Error(err))
	}
	return nil
}

func getDefaultBackupSetting(environmentID, instanceID, databaseName string) *v1pb.BackupSetting {
	sevenDays, err := convertPeriodTsToDuration(int(time.Duration(7 * 24 * time.Hour).Seconds()))
	if err != nil {
		log.Warn("failed to convert period ts to duration", zap.Error(err))
	}
	return &v1pb.BackupSetting{
		Name:                 fmt.Sprintf("%s%s/%s%s/%s%s/%s", environmentNamePrefix, environmentID, instanceNamePrefix, instanceID, databaseIDPrefix, databaseName, backupSettingSuffix),
		BackupRetainDuration: sevenDays,
		CronSchedule:         "", /* Disable automatic backup */
		HookUrl:              "",
	}
}

func convertToBackupSetting(backupSetting *store.BackupSettingMessage, environmentID, instanceID, databaseName string) (*v1pb.BackupSetting, error) {
	period, err := convertPeriodTsToDuration(backupSetting.RetentionPeriodTs)
	if err != nil {
		return nil, err
	}
	cronSchedule := ""
	if backupSetting.Enabled {
		cronSchedule = buildSimpleCron(backupSetting.HourOfDay, backupSetting.DayOfWeek)
	}
	return &v1pb.BackupSetting{
		Name:                 fmt.Sprintf("%s%s%s%s%s%s%s", environmentNamePrefix, environmentID, instanceNamePrefix, instanceID, databaseIDPrefix, databaseName, backupSettingSuffix),
		BackupRetainDuration: period,
		CronSchedule:         cronSchedule,
		HookUrl:              backupSetting.HookURL,
	}, nil
}

func validateAndConvertToStoreBackupSetting(backupSetting *v1pb.BackupSetting, databaseUID int) (*store.BackupSettingMessage, error) {
	enable := backupSetting.CronSchedule != ""
	hourOfDay := 0
	dayOfWeek := -1
	var err error
	if enable {
		hourOfDay, dayOfWeek, err = parseSimpleCron(backupSetting.CronSchedule)
		if err != nil {
			return nil, err
		}
	}
	periodTs, err := convertDurationToPeriodTs(backupSetting.BackupRetainDuration)
	if err != nil {
		return nil, err
	}
	return &store.BackupSettingMessage{
		DatabaseUID:       databaseUID,
		Enabled:           enable,
		HourOfDay:         hourOfDay,
		DayOfWeek:         dayOfWeek,
		RetentionPeriodTs: periodTs,
		HookURL:           backupSetting.HookUrl,
	}, nil
}

// parseSimpleCron parses a simple cron expression(only support hour of day and day of week), and returns them as int.
func parseSimpleCron(cron string) (int, int, error) {
	fields := strings.Fields(cron)
	if len(fields) != 5 {
		return 0, 0, errors.New("invalid cron expression")
	}
	hourOfDay, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, errors.New("invalid cron hour field")
	}
	if hourOfDay < 0 || hourOfDay > 23 {
		return 0, 0, errors.New("invalid cron hour range")
	}
	weekDay := fields[4]
	// "*" means any day of week.
	if weekDay == "*" {
		return hourOfDay, -1, nil
	}
	dayOfWeek, err := strconv.Atoi(weekDay)
	if err != nil {
		return 0, 0, err
	}
	if dayOfWeek < 0 || dayOfWeek > 6 {
		return 0, 0, errors.New("invalid cron day of week range")
	}
	return hourOfDay, dayOfWeek, nil
}

func buildSimpleCron(hourOfDay int, dayOfWeek int) string {
	if dayOfWeek == -1 {
		return fmt.Sprintf("0 %d * * *", hourOfDay)
	}
	return fmt.Sprintf("0 %d * * %d", hourOfDay, dayOfWeek)
}

func convertDurationToPeriodTs(duration *durationpb.Duration) (int, error) {
	if err := duration.CheckValid(); err != nil {
		return 0, errors.Wrap(err, "invalid duration")
	}
	// Round up to days
	return int(duration.AsDuration().Round(time.Hour * 24).Seconds()), nil
}

func convertPeriodTsToDuration(periodTs int) (*durationpb.Duration, error) {
	if periodTs < 0 {
		return nil, errors.New("invalid period")
	}
	return durationpb.New(time.Duration(periodTs) * time.Second), nil
}

// isProjectOwnerOrDeveloper returns whether a principal is a project owner or developer in the project.
func isProjectOwnerOrDeveloper(principalID int, projectPolicy *store.IAMPolicyMessage) bool {
	for _, binding := range projectPolicy.Bindings {
		if binding.Role != api.Owner && binding.Role != api.Developer {
			continue
		}
		for _, member := range binding.Members {
			if member.ID == principalID {
				return true
			}
		}
	}
	return false
}

func stripeAndConvertToServiceSecrets(secrets *storepb.Secrets, environmentID, instanceID, databaseName string) []*v1pb.Secret {
	var serviceSecrets []*v1pb.Secret
	if secrets == nil || len(secrets.Items) == 0 {
		return serviceSecrets
	}
	for _, secret := range secrets.Items {
		serviceSecrets = append(serviceSecrets, stripeAndConvertToServiceSecret(secret, environmentID, instanceID, databaseName))
	}
	return serviceSecrets
}

func stripeAndConvertToServiceSecret(secretEntry *storepb.SecretItem, environmentID, instanceID, databaseName string) *v1pb.Secret {
	return &v1pb.Secret{
		Name:        fmt.Sprintf("%s%s/%s%s/%s%s/%s%s", environmentNamePrefix, environmentID, instanceNamePrefix, instanceID, databaseIDPrefix, databaseName, secretNamePrefix, secretEntry.Name),
		Value:       "", /* stripped */
		Description: secretEntry.Description,
	}
}

func isSecretValid(secret *storepb.SecretItem) error {
	// Names can not be empty.
	if secret.Name == "" {
		return errors.Errorf("invalid secret name: %s, name can not be empty", secret.Name)
	}
	// Values can not be empty.
	if secret.Value == "" {
		return errors.Errorf("the value of secret: %s can not be empty", secret.Name)
	}

	// Names must not start with the 'BYTEBASE_' prefix.
	bytebaseCaseInsensitivePrefixRegexp := regexp.MustCompile(`(?i)^BYTEBASE_`)
	if bytebaseCaseInsensitivePrefixRegexp.MatchString(secret.Name) {
		return errors.Errorf("invalid secret name: %s, name must not start with the 'BYTEBASE_' prefix", secret.Name)
	}
	// Names must not start with a number.
	if unicode.IsDigit(rune(secret.Name[0])) {
		return errors.Errorf("invalid secret name: %s, name must not start with a number", secret.Name)
	}

	// Names can only contain alphanumeric characters ([A-Z], [0-9]) or underscores (_). Spaces are not allowed.
	for _, c := range secret.Name {
		if !isUpperCaseLetter(c) && !unicode.IsDigit(c) && c != '_' {
			return errors.Errorf("invalid secret name: %s, expect [A-Z], [0-9], '_', but meet: %v", secret.Name, c)
		}
	}
	return nil
}

func isUpperCaseLetter(c rune) bool {
	return 'A' <= c && c <= 'Z'
}
