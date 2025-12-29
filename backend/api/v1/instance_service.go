package v1

import (
	"context"
	"log/slog"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"

	"github.com/bytebase/bytebase/backend/component/sampleinstance"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
)

// InstanceService implements the instance service.
type InstanceService struct {
	v1connect.UnimplementedInstanceServiceHandler
	store                 *store.Store
	profile               *config.Profile
	licenseService        *enterprise.LicenseService
	dbFactory             *dbfactory.DBFactory
	schemaSyncer          *schemasync.Syncer
	sampleInstanceManager *sampleinstance.Manager
}

// NewInstanceService creates a new InstanceService.
func NewInstanceService(store *store.Store, profile *config.Profile, licenseService *enterprise.LicenseService, dbFactory *dbfactory.DBFactory, schemaSyncer *schemasync.Syncer, sampleInstanceManager *sampleinstance.Manager) *InstanceService {
	return &InstanceService{
		store:                 store,
		profile:               profile,
		licenseService:        licenseService,
		dbFactory:             dbFactory,
		schemaSyncer:          schemaSyncer,
		sampleInstanceManager: sampleInstanceManager,
	}
}

// GetInstance gets an instance.
func (s *InstanceService) GetInstance(ctx context.Context, req *connect.Request[v1pb.GetInstanceRequest]) (*connect.Response[v1pb.Instance], error) {
	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	result := convertInstanceMessage(instance)
	return connect.NewResponse(result), nil
}

// ListInstances lists all instances.
func (s *InstanceService) ListInstances(ctx context.Context, req *connect.Request[v1pb.ListInstancesRequest]) (*connect.Response[v1pb.ListInstancesResponse], error) {
	offset, err := parseLimitAndOffset(&pageSize{
		token:   req.Msg.PageToken,
		limit:   int(req.Msg.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	find := &store.FindInstanceMessage{
		ShowDeleted: req.Msg.ShowDeleted,
		Limit:       &limitPlusOne,
		Offset:      &offset.offset,
	}
	filterQ, err := store.GetListInstanceFilter(req.Msg.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.FilterQ = filterQ

	orderByKeys, err := store.GetInstanceOrders(req.Msg.OrderBy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.OrderByKeys = orderByKeys

	instances, err := s.store.ListInstances(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	nextPageToken := ""
	if len(instances) == limitPlusOne {
		instances = instances[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to marshal next page token"))
		}
	}

	response := &v1pb.ListInstancesResponse{
		NextPageToken: nextPageToken,
	}
	for _, instance := range instances {
		ins := convertInstanceMessage(instance)
		response.Instances = append(response.Instances, ins)
	}
	return connect.NewResponse(response), nil
}

// ListInstanceDatabase list all databases in the instance.
func (s *InstanceService) ListInstanceDatabase(ctx context.Context, req *connect.Request[v1pb.ListInstanceDatabaseRequest]) (*connect.Response[v1pb.ListInstanceDatabaseResponse], error) {
	var instanceMessage *store.InstanceMessage

	if req.Msg.Instance != nil {
		instanceID, err := common.GetInstanceID(req.Msg.Name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		if instanceMessage, err = convertInstanceToInstanceMessage(instanceID, req.Msg.Instance); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	} else {
		instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		instanceMessage = instance
	}

	instanceMeta, err := s.schemaSyncer.GetInstanceMeta(ctx, instanceMessage)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	response := &v1pb.ListInstanceDatabaseResponse{}
	for _, database := range instanceMeta.Databases {
		response.Databases = append(response.Databases, database.Name)
	}
	return connect.NewResponse(response), nil
}

// CreateInstance creates an instance.
func (s *InstanceService) CreateInstance(ctx context.Context, req *connect.Request[v1pb.CreateInstanceRequest]) (*connect.Response[v1pb.Instance], error) {
	if req.Msg.Instance == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("instance must be set"))
	}
	if !isValidResourceID(req.Msg.InstanceId) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid instance ID %v", req.Msg.InstanceId))
	}

	if err := s.instanceCountGuard(ctx); err != nil {
		return nil, err
	}

	if err := validateLabels(req.Msg.Instance.Labels); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	instanceMessage, err := convertInstanceToInstanceMessage(req.Msg.InstanceId, req.Msg.Instance)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Test connection.
	if req.Msg.ValidateOnly {
		for _, ds := range instanceMessage.Metadata.GetDataSources() {
			err := func() error {
				driver, err := s.dbFactory.GetDataSourceDriver(
					ctx, instanceMessage, ds,
					db.ConnectionContext{
						ReadOnly: ds.GetType() == storepb.DataSourceType_READ_ONLY,
					},
				)
				if err != nil {
					return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database driver"))
				}
				defer driver.Close(ctx)
				if err := driver.Ping(ctx); err != nil {
					return connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid datasource %s", ds.GetType()))
				}
				return nil
			}()
			if err != nil {
				return nil, err
			}
		}

		result := convertInstanceMessage(instanceMessage)
		return connect.NewResponse(result), nil
	}

	activatedInstanceLimit := s.licenseService.GetActivatedInstanceLimit(ctx)
	if instanceMessage.Metadata.GetActivation() {
		count, err := s.store.GetActivatedInstanceCount(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if count >= activatedInstanceLimit {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.Errorf(instanceExceededError, activatedInstanceLimit))
		}
	}

	if err := s.checkInstanceDataSources(instanceMessage, instanceMessage.Metadata.GetDataSources()); err != nil {
		return nil, err
	}

	instance, err := s.store.CreateInstance(ctx, instanceMessage)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, nil /* database */, db.ConnectionContext{})
	if err == nil {
		defer driver.Close(ctx)
		updatedInstance, _, _, err := s.schemaSyncer.SyncInstance(ctx, instance)
		if err != nil {
			slog.Warn("Failed to sync instance",
				slog.String("instance", instance.ResourceID),
				log.BBError(err))
		} else {
			instance = updatedInstance
		}
		// Sync all databases in the instance asynchronously.
		s.schemaSyncer.SyncAllDatabases(ctx, instance)
	}

	result := convertInstanceMessage(instance)
	return connect.NewResponse(result), nil
}

func (s *InstanceService) checkInstanceDataSources(instance *store.InstanceMessage, dataSources []*storepb.DataSource) error {
	dsIDMap := map[string]bool{}
	for _, ds := range dataSources {
		if err := s.checkDataSource(instance, ds); err != nil {
			return err
		}
		if dsIDMap[ds.GetId()] {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`duplicate data source id "%s"`, ds.GetId()))
		}
		dsIDMap[ds.GetId()] = true
	}

	return nil
}

const instanceExceededError = "activation instance count has reached the limit (%v)"

func (s *InstanceService) checkDataSource(instance *store.InstanceMessage, dataSource *storepb.DataSource) error {
	if dataSource.GetId() == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("data source id is required"))
	}

	// Validate IAM credential restrictions in SaaS mode
	if err := s.validateIAMCredentialForSaaS(dataSource); err != nil {
		return err
	}

	if err := s.licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_EXTERNAL_SECRET_MANAGER, instance); err != nil {
		missingFeatureError := connect.NewError(connect.CodePermissionDenied, err)
		if dataSource.GetExternalSecret() != nil {
			return missingFeatureError
		}
		return nil
	}

	// Validate extra connection parameters for MySQL-based engines
	if err := validateExtraConnectionParameters(instance.Metadata.GetEngine(), dataSource.GetExtraConnectionParameters()); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	return nil
}

func (s *InstanceService) validateIAMCredentialForSaaS(dataSource *storepb.DataSource) error {
	if !s.profile.SaaS {
		return nil
	}

	// Check if using IAM authentication
	iamAuthTypes := map[storepb.DataSource_AuthenticationType]bool{
		storepb.DataSource_GOOGLE_CLOUD_SQL_IAM: true,
		storepb.DataSource_AWS_RDS_IAM:          true,
		storepb.DataSource_AZURE_IAM:            true,
	}

	if !iamAuthTypes[dataSource.GetAuthenticationType()] {
		return nil
	}

	// Check if using default credentials (iam_extension is not set)
	if dataSource.GetIamExtension() == nil {
		return connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("default credentials are not allowed in SaaS mode for security. Please provide specific credentials"),
		)
	}

	return nil
}

// UpdateInstance updates an instance.
func (s *InstanceService) UpdateInstance(ctx context.Context, req *connect.Request[v1pb.UpdateInstanceRequest]) (*connect.Response[v1pb.Instance], error) {
	if req.Msg.Instance == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("instance must be set"))
	}
	if req.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask must be set"))
	}

	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Instance.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") && req.Msg.AllowMissing {
			// When allow_missing is true and instance doesn't exist, create a new one
			instanceID, ierr := common.GetInstanceID(req.Msg.Instance.Name)
			if ierr != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, ierr)
			}

			return s.CreateInstance(ctx, connect.NewRequest(&v1pb.CreateInstanceRequest{
				InstanceId: instanceID,
				Instance:   req.Msg.Instance,
			}))
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if instance.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q has been deleted", req.Msg.Instance.Name))
	}

	metadata := proto.CloneOf(instance.Metadata)
	patch := &store.UpdateInstanceMessage{
		ResourceID: &instance.ResourceID,
		Metadata:   metadata,
	}
	updateActivation := false
	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Metadata.Title = req.Msg.Instance.Title
		case "environment":
			if req.Msg.Instance.Environment == nil || *req.Msg.Instance.Environment == "" {
				// Clear the environment if null or empty string is provided
				emptyStr := ""
				patch.EnvironmentID = &emptyStr
			} else {
				envID, err := common.GetEnvironmentID(*req.Msg.Instance.Environment)
				if err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, err)
				}
				environment, err := s.store.GetEnvironmentByID(ctx, envID)
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, err)
				}
				if environment == nil {
					return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("environment %q not found", envID))
				}
				patch.EnvironmentID = &envID
			}
		case "external_link":
			patch.Metadata.ExternalLink = req.Msg.Instance.ExternalLink
		case "data_sources":
			dataSources, err := convertV1DataSources(req.Msg.Instance.DataSources)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			if err := s.checkInstanceDataSources(instance, dataSources); err != nil {
				return nil, err
			}
			patch.Metadata.DataSources = dataSources
		case "activation":
			if !instance.Metadata.GetActivation() && req.Msg.Instance.Activation {
				updateActivation = true
			}
			patch.Metadata.Activation = req.Msg.Instance.Activation
		case "sync_interval":
			if err := s.licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_CUSTOM_INSTANCE_SYNC_TIME, instance); err != nil {
				return nil, connect.NewError(connect.CodePermissionDenied, err)
			}
			patch.Metadata.SyncInterval = req.Msg.Instance.SyncInterval
		case "sync_databases":
			patch.Metadata.SyncDatabases = req.Msg.Instance.SyncDatabases
		case "labels":
			if err := validateLabels(req.Msg.Instance.Labels); err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			patch.Metadata.Labels = req.Msg.Instance.Labels
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`unsupported update_mask "%s"`, path))
		}
	}

	activatedInstanceLimit := s.licenseService.GetActivatedInstanceLimit(ctx)
	if updateActivation {
		count, err := s.store.GetActivatedInstanceCount(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if count >= activatedInstanceLimit {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.Errorf(instanceExceededError, activatedInstanceLimit))
		}
	}

	ins, err := s.store.UpdateInstance(ctx, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result := convertInstanceMessage(ins)
	return connect.NewResponse(result), nil
}

// DeleteInstance deletes an instance.
func (s *InstanceService) DeleteInstance(ctx context.Context, req *connect.Request[v1pb.DeleteInstanceRequest]) (*connect.Response[emptypb.Empty], error) {
	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}

	// Handle purge (hard delete) of soft-deleted instance
	if req.Msg.Purge {
		// Following AIP-165, purge only works on already soft-deleted instances
		if !instance.Deleted {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("instance %q must be soft-deleted before it can be purged", req.Msg.Name))
		}

		// Permanently delete the instance and all related resources
		if err := s.store.DeleteInstance(ctx, instance.ResourceID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to purge instance"))
		}

		return connect.NewResponse(&emptypb.Empty{}), nil
	}

	// Regular soft delete flow
	if instance.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q has been deleted", req.Msg.Name))
	}

	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID})
	if err != nil {
		return nil, err
	}
	if req.Msg.Force {
		if len(databases) > 0 {
			defaultProjectID := common.DefaultProjectID
			if _, err := s.store.BatchUpdateDatabases(ctx, databases, &store.BatchUpdateDatabases{ProjectID: &defaultProjectID}); err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		}
	} else {
		var databaseNames []string
		for _, database := range databases {
			if database.ProjectID != common.DefaultProjectID {
				databaseNames = append(databaseNames, database.DatabaseName)
			}
		}
		if len(databaseNames) > 0 {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("all databases should be transferred to the unassigned project before deleting the instance"))
		}
	}

	metadata := proto.CloneOf(instance.Metadata)
	metadata.Activation = false
	if _, err := s.store.UpdateInstance(ctx, &store.UpdateInstanceMessage{
		ResourceID: &instance.ResourceID,
		Deleted:    &deletePatch,
		Metadata:   metadata,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Handle sample instance deletion if applicable
	if err := s.sampleInstanceManager.HandleInstanceDeletion(ctx, instance.ResourceID); err != nil {
		slog.Warn("failed to handle sample instance deletion", log.BBError(err), slog.String("instance", instance.ResourceID))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// UndeleteInstance undeletes an instance.
func (s *InstanceService) UndeleteInstance(ctx context.Context, req *connect.Request[v1pb.UndeleteInstanceRequest]) (*connect.Response[v1pb.Instance], error) {
	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	if !instance.Deleted {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("instance %q is active", req.Msg.Name))
	}
	if err := s.instanceCountGuard(ctx); err != nil {
		return nil, err
	}

	ins, err := s.store.UpdateInstance(ctx, &store.UpdateInstanceMessage{
		ResourceID: &instance.ResourceID,
		Deleted:    &undeletePatch,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Handle sample instance undelete (restart) if applicable
	if err := s.sampleInstanceManager.HandleInstanceCreation(ctx, ins.ResourceID); err != nil {
		slog.Warn("failed to handle sample instance undelete", log.BBError(err), slog.String("instance", ins.ResourceID))
	}

	result := convertInstanceMessage(ins)
	return connect.NewResponse(result), nil
}

// SyncInstance syncs the instance.
func (s *InstanceService) SyncInstance(ctx context.Context, req *connect.Request[v1pb.SyncInstanceRequest]) (*connect.Response[v1pb.SyncInstanceResponse], error) {
	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q has been deleted", req.Msg.Name))
	}

	updatedInstance, allDatabases, newDatabases, err := s.schemaSyncer.SyncInstance(ctx, instance)
	if err != nil {
		return nil, err
	}
	if req.Msg.EnableFullSync {
		// Sync all databases in the instance asynchronously.
		s.schemaSyncer.SyncAllDatabases(ctx, updatedInstance)
	} else {
		s.schemaSyncer.SyncDatabasesAsync(newDatabases)
	}

	response := &v1pb.SyncInstanceResponse{}
	for _, database := range allDatabases {
		response.Databases = append(response.Databases, database.Name)
	}
	return connect.NewResponse(response), nil
}

// BatchSyncInstances syncs multiple instances.
func (s *InstanceService) BatchSyncInstances(ctx context.Context, req *connect.Request[v1pb.BatchSyncInstancesRequest]) (*connect.Response[v1pb.BatchSyncInstancesResponse], error) {
	for _, r := range req.Msg.Requests {
		instance, err := getInstanceMessage(ctx, s.store, r.Name)
		if err != nil {
			return nil, err
		}
		if instance.Deleted {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q has been deleted", r.Name))
		}

		updatedInstance, _, newDatabases, err := s.schemaSyncer.SyncInstance(ctx, instance)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to sync instance"))
		}
		if r.EnableFullSync {
			// Sync all databases in the instance asynchronously.
			s.schemaSyncer.SyncAllDatabases(ctx, updatedInstance)
		} else {
			s.schemaSyncer.SyncDatabasesAsync(newDatabases)
		}
	}

	return connect.NewResponse(&v1pb.BatchSyncInstancesResponse{}), nil
}

// BatchUpdateInstances update multiple instances.
func (s *InstanceService) BatchUpdateInstances(ctx context.Context, req *connect.Request[v1pb.BatchUpdateInstancesRequest]) (*connect.Response[v1pb.BatchUpdateInstancesResponse], error) {
	response := &v1pb.BatchUpdateInstancesResponse{}
	for _, updateReq := range req.Msg.GetRequests() {
		updated, err := s.UpdateInstance(ctx, connect.NewRequest(updateReq))
		if err != nil {
			return nil, err
		}
		response.Instances = append(response.Instances, updated.Msg)
	}
	return connect.NewResponse(response), nil
}

// AddDataSource adds a data source to an instance.
func (s *InstanceService) AddDataSource(ctx context.Context, req *connect.Request[v1pb.AddDataSourceRequest]) (*connect.Response[v1pb.Instance], error) {
	if req.Msg.DataSource == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("data sources is required"))
	}
	// We only support add RO type datasouce to instance now, see more details in instance_service.proto.
	if req.Msg.DataSource.Type != v1pb.DataSourceType_READ_ONLY {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("only support adding read-only data source"))
	}

	dataSource, err := convertV1DataSource(req.Msg.DataSource)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("failed to convert data source"))
	}

	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q has been deleted", req.Msg.Name))
	}
	for _, ds := range instance.Metadata.GetDataSources() {
		if ds.GetId() == req.Msg.DataSource.Id {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("data source already exists with the same name"))
		}
	}
	if err := s.checkDataSource(instance, dataSource); err != nil {
		return nil, err
	}

	// Test connection.
	if req.Msg.ValidateOnly {
		err := func() error {
			driver, err := s.dbFactory.GetDataSourceDriver(
				ctx, instance, dataSource,
				db.ConnectionContext{
					ReadOnly: dataSource.GetType() == storepb.DataSourceType_READ_ONLY,
				},
			)
			if err != nil {
				return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database driver"))
			}
			defer driver.Close(ctx)
			if err := driver.Ping(ctx); err != nil {
				return connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid datasource %s", dataSource.GetType()))
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
		result := convertInstanceMessage(instance)
		return connect.NewResponse(result), nil
	}

	if dataSource.GetType() != storepb.DataSourceType_READ_ONLY {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("only read-only data source can be added"))
	}
	if err := s.licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_INSTANCE_READ_ONLY_CONNECTION, instance); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	metadata := proto.CloneOf(instance.Metadata)
	metadata.DataSources = append(metadata.DataSources, dataSource)
	instance, err = s.store.UpdateInstance(ctx, &store.UpdateInstanceMessage{
		ResourceID: &instance.ResourceID,
		Metadata:   metadata,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := convertInstanceMessage(instance)
	return connect.NewResponse(result), nil
}

// UpdateDataSource updates a data source of an instance.
func (s *InstanceService) UpdateDataSource(ctx context.Context, req *connect.Request[v1pb.UpdateDataSourceRequest]) (*connect.Response[v1pb.Instance], error) {
	if req.Msg.DataSource == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("datasource is required"))
	}
	if req.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask must be set"))
	}

	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q has been deleted", req.Msg.Name))
	}
	metadata := proto.CloneOf(instance.Metadata)
	var dataSource *storepb.DataSource
	for _, ds := range metadata.GetDataSources() {
		if ds.GetId() == req.Msg.DataSource.Id {
			dataSource = ds
			break
		}
	}
	if dataSource == nil {
		if req.Msg.AllowMissing {
			return s.AddDataSource(ctx, connect.NewRequest(&v1pb.AddDataSourceRequest{
				Name:       req.Msg.Name,
				DataSource: req.Msg.DataSource,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf(`cannot found data source "%s"`, req.Msg.DataSource.Id))
	}

	if dataSource.GetType() == storepb.DataSourceType_READ_ONLY {
		if err := s.licenseService.IsFeatureEnabledForInstance(v1pb.PlanFeature_FEATURE_INSTANCE_READ_ONLY_CONNECTION, instance); err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
	}

	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "username":
			dataSource.Username = req.Msg.DataSource.Username
		case "password":
			dataSource.Password = req.Msg.DataSource.Password
		case "ssl_ca":
			dataSource.SslCa = req.Msg.DataSource.SslCa
		case "ssl_cert":
			dataSource.SslCert = req.Msg.DataSource.SslCert
		case "ssl_key":
			dataSource.SslKey = req.Msg.DataSource.SslKey
		case "host":
			dataSource.Host = req.Msg.DataSource.Host
		case "port":
			dataSource.Port = req.Msg.DataSource.Port
		case "database":
			dataSource.Database = req.Msg.DataSource.Database
		case "srv":
			dataSource.Srv = req.Msg.DataSource.Srv
		case "authentication_database":
			dataSource.AuthenticationDatabase = req.Msg.DataSource.AuthenticationDatabase
		case "sid":
			dataSource.Sid = req.Msg.DataSource.Sid
		case "service_name":
			dataSource.ServiceName = req.Msg.DataSource.ServiceName
		case "ssh_host":
			dataSource.SshHost = req.Msg.DataSource.SshHost
		case "ssh_port":
			dataSource.SshPort = req.Msg.DataSource.SshPort
		case "ssh_user":
			dataSource.SshUser = req.Msg.DataSource.SshUser
		case "ssh_password":
			dataSource.SshPassword = req.Msg.DataSource.SshPassword
		case "ssh_private_key":
			dataSource.SshPrivateKey = req.Msg.DataSource.SshPrivateKey
		case "authentication_private_key":
			dataSource.AuthenticationPrivateKey = req.Msg.DataSource.AuthenticationPrivateKey
		case "authentication_private_key_passphrase":
			dataSource.AuthenticationPrivateKeyPassphrase = req.Msg.DataSource.AuthenticationPrivateKeyPassphrase
		case "external_secret":
			externalSecret, err := convertV1DataSourceExternalSecret(req.Msg.DataSource.ExternalSecret)
			if err != nil {
				return nil, err
			}
			dataSource.ExternalSecret = externalSecret
		case "sasl_config":
			dataSource.SaslConfig = convertV1DataSourceSaslConfig(req.Msg.DataSource.SaslConfig)
		case "authentication_type":
			dataSource.AuthenticationType = convertV1AuthenticationType(req.Msg.DataSource.AuthenticationType)
		case "additional_addresses":
			dataSource.AdditionalAddresses = convertAdditionalAddresses(req.Msg.DataSource.AdditionalAddresses)
		case "replica_set":
			dataSource.ReplicaSet = req.Msg.DataSource.ReplicaSet
		case "direct_connection":
			dataSource.DirectConnection = req.Msg.DataSource.DirectConnection
		case "region":
			dataSource.Region = req.Msg.DataSource.Region
		case "warehouse_id":
			dataSource.WarehouseId = req.Msg.DataSource.WarehouseId
		case "use_ssl":
			dataSource.UseSsl = req.Msg.DataSource.UseSsl
		case "verify_tls_certificate":
			dataSource.VerifyTlsCertificate = req.Msg.DataSource.VerifyTlsCertificate
		case "redis_type":
			dataSource.RedisType = convertV1RedisType(req.Msg.DataSource.RedisType)
		case "master_name":
			dataSource.MasterName = req.Msg.DataSource.MasterName
		case "master_username":
			dataSource.MasterUsername = req.Msg.DataSource.MasterUsername
		case "master_password":
			dataSource.MasterPassword = req.Msg.DataSource.MasterPassword
		case "extra_connection_parameters":
			dataSource.ExtraConnectionParameters = req.Msg.DataSource.ExtraConnectionParameters
		case "azure_credential", "aws_credential", "gcp_credential":
			switch req.Msg.DataSource.AuthenticationType {
			case v1pb.DataSource_AZURE_IAM:
				if azureCredential := req.Msg.DataSource.GetAzureCredential(); azureCredential != nil {
					dataSource.IamExtension = &storepb.DataSource_AzureCredential_{
						AzureCredential: &storepb.DataSource_AzureCredential{
							TenantId:     azureCredential.TenantId,
							ClientId:     azureCredential.ClientId,
							ClientSecret: azureCredential.ClientSecret,
						},
					}
				} else {
					dataSource.IamExtension = nil
				}
			case v1pb.DataSource_AWS_RDS_IAM:
				if awsCredential := req.Msg.DataSource.GetAwsCredential(); awsCredential != nil {
					dataSource.IamExtension = &storepb.DataSource_AwsCredential{
						AwsCredential: &storepb.DataSource_AWSCredential{
							AccessKeyId:     awsCredential.AccessKeyId,
							SecretAccessKey: awsCredential.SecretAccessKey,
							SessionToken:    awsCredential.SessionToken,
							RoleArn:         awsCredential.RoleArn,
							ExternalId:      awsCredential.ExternalId,
						},
					}
				} else {
					dataSource.IamExtension = nil
				}
			case v1pb.DataSource_GOOGLE_CLOUD_SQL_IAM:
				if gcpCredential := req.Msg.DataSource.GetGcpCredential(); gcpCredential != nil {
					dataSource.IamExtension = &storepb.DataSource_GcpCredential{
						GcpCredential: &storepb.DataSource_GCPCredential{
							Content: gcpCredential.Content,
						},
					}
				} else {
					dataSource.IamExtension = nil
				}
			default:
			}
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`unsupported update_mask "%s"`, path))
		}
	}

	if err := s.checkDataSource(instance, dataSource); err != nil {
		return nil, err
	}

	// Test connection.
	if req.Msg.ValidateOnly {
		err := func() error {
			driver, err := s.dbFactory.GetDataSourceDriver(
				ctx, instance, dataSource,
				db.ConnectionContext{ReadOnly: dataSource.GetType() == storepb.DataSourceType_READ_ONLY},
			)
			if err != nil {
				return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database driver"))
			}
			defer driver.Close(ctx)
			if err := driver.Ping(ctx); err != nil {
				return connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid datasource %s", dataSource.GetType()))
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
		result := convertInstanceMessage(instance)
		return connect.NewResponse(result), nil
	}

	instance, err = s.store.UpdateInstance(ctx, &store.UpdateInstanceMessage{
		ResourceID: &instance.ResourceID,
		Metadata:   metadata,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result := convertInstanceMessage(instance)
	return connect.NewResponse(result), nil
}

// RemoveDataSource removes a data source to an instance.
func (s *InstanceService) RemoveDataSource(ctx context.Context, req *connect.Request[v1pb.RemoveDataSourceRequest]) (*connect.Response[v1pb.Instance], error) {
	if req.Msg.DataSource == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("data sources is required"))
	}

	instance, err := getInstanceMessage(ctx, s.store, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	if instance.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q has been deleted", req.Msg.Name))
	}

	metadata := proto.CloneOf(instance.Metadata)
	var updatedDataSources []*storepb.DataSource
	var dataSource *storepb.DataSource
	for _, ds := range instance.Metadata.GetDataSources() {
		if ds.GetId() == req.Msg.DataSource.Id {
			dataSource = ds
		} else {
			updatedDataSources = append(updatedDataSources, ds)
		}
	}
	if dataSource == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("data source not found"))
	}

	// We only support remove RO type datasource to instance now, see more details in instance_service.proto.
	if dataSource.GetType() != storepb.DataSourceType_READ_ONLY {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("only support remove read-only data source"))
	}

	metadata.DataSources = updatedDataSources
	instance, err = s.store.UpdateInstance(ctx, &store.UpdateInstanceMessage{
		ResourceID: &instance.ResourceID,
		Metadata:   metadata,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := convertInstanceMessage(instance)
	return connect.NewResponse(result), nil
}

func getInstanceMessage(ctx context.Context, stores *store.Store, name string) (*store.InstanceMessage, error) {
	instanceID, err := common.GetInstanceID(name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	find := &store.FindInstanceMessage{
		ResourceID: &instanceID,
	}
	instance, err := stores.GetInstance(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if instance == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", name))
	}

	return instance, nil
}

// buildInstanceName builds the instance name with the given instance ID.
func buildInstanceName(instanceID string) string {
	var b strings.Builder
	b.Grow(len(common.InstanceNamePrefix) + len(instanceID))
	_, _ = b.WriteString(common.InstanceNamePrefix)
	_, _ = b.WriteString(instanceID)
	return b.String()
}

// buildEnvironmentName builds the environment name with the given environment ID.
func buildEnvironmentName(environmentID *string) *string {
	if environmentID == nil || *environmentID == "" {
		return nil
	}
	var b strings.Builder
	b.Grow(len("environments/") + len(*environmentID))
	_, _ = b.WriteString("environments/")
	_, _ = b.WriteString(*environmentID)
	result := b.String()
	return &result
}

func (s *InstanceService) instanceCountGuard(ctx context.Context) error {
	instanceLimit := s.licenseService.GetInstanceLimit(ctx)

	count, err := s.store.CountActiveInstances(ctx)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if count >= instanceLimit {
		return connect.NewError(connect.CodeResourceExhausted, errors.Errorf("reached the maximum instance count %d", instanceLimit))
	}

	return nil
}

// validateExtraConnectionParameters validates extra connection parameters for security risks.
func validateExtraConnectionParameters(engine storepb.Engine, params map[string]string) error {
	// Validate MySQL-based engines
	switch engine {
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		for key := range params {
			normalizedKey := strings.ToLower(strings.TrimSpace(key))
			if normalizedKey == "allowallfiles" {
				// Disables file allowlist for LOAD DATA LOCAL INFILE and allows all files (might be insecure)
				return errors.Errorf("connection parameter %q is not allowed for security reasons. This parameter can allow a malicious database server to read arbitrary files from the client", key)
			}
		}
	default:
		// No validation needed for other engines
	}
	return nil
}
