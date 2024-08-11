package v1

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/epiclabs-io/diff3"
	"github.com/google/go-cmp/cmp"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// NewBranchService implements BranchServiceServer interface.
type BranchService struct {
	v1pb.UnimplementedBranchServiceServer
	store          *store.Store
	licenseService enterprise.LicenseService
	profile        *config.Profile
	iamManager     *iam.Manager
}

// NewBranchService creates a new BranchService.
func NewBranchService(store *store.Store, licenseService enterprise.LicenseService, profile *config.Profile, iamManager *iam.Manager) *BranchService {
	return &BranchService{
		store:          store,
		licenseService: licenseService,
		profile:        profile,
		iamManager:     iamManager,
	}
}

// GetBranch gets the branch.
func (s *BranchService) GetBranch(ctx context.Context, request *v1pb.GetBranchRequest) (*v1pb.Branch, error) {
	projectID, branchID, err := common.GetProjectAndBranchID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.getProject(ctx, projectID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	branch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &project.ResourceID, ResourceID: &branchID, LoadFull: true})
	if err != nil {
		return nil, err
	}
	if branch == nil {
		return nil, status.Errorf(codes.NotFound, "branch %q not found", branchID)
	}

	branchV1, err := s.convertBranchToBranch(ctx, project, branch, v1pb.BranchView_BRANCH_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return branchV1, nil
}

// ListBranches lists branches.
func (s *BranchService) ListBranches(ctx context.Context, request *v1pb.ListBranchesRequest) (*v1pb.ListBranchesResponse, error) {
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.getProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	branchFind := &store.FindBranchMessage{
		ProjectID: &project.ResourceID,
	}
	if request.View == v1pb.BranchView_BRANCH_VIEW_FULL {
		branchFind.LoadFull = true
	}
	branches, err := s.store.ListBranches(ctx, branchFind)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list branches, error %v", err)
	}

	var v1Branches []*v1pb.Branch
	for _, branch := range branches {
		v1Branch, err := s.convertBranchToBranch(ctx, project, branch, request.View)
		if err != nil {
			return nil, err
		}
		v1Branches = append(v1Branches, v1Branch)
	}
	response := &v1pb.ListBranchesResponse{
		Branches: v1Branches,
	}
	return response, nil
}

// CreateBranch creates a new branch.
func (s *BranchService) CreateBranch(ctx context.Context, request *v1pb.CreateBranchRequest) (*v1pb.Branch, error) {
	branchID := request.BranchId
	// TODO(d): regex check.
	if branchID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "branch ID is empty")
	}
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.getProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	branch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &project.ResourceID, ResourceID: &branchID, LoadFull: false})
	if err != nil {
		return nil, err
	}
	if branch != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "branch %q has already existed", branchID)
	}
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	// Main branch IAM admin check.
	if request.GetBranch().GetParentBranch() == "" {
		ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionBranchesAdmin, user, project.ResourceID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "only users with %s permission can create a main branch", iam.PermissionBranchesAdmin)
		}
	}

	var createdBranch *store.BranchMessage
	if request.Branch.ParentBranch != "" {
		parentProjectID, parentBranchID, err := common.GetProjectAndBranchID(request.Branch.ParentBranch)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		parentBranch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &parentProjectID, ResourceID: &parentBranchID, LoadFull: true})
		if err != nil {
			return nil, err
		}
		if parentBranch == nil {
			return nil, status.Errorf(codes.NotFound, "parent branch %q not found", parentBranchID)
		}
		parentBranchHeadConfig := parentBranch.Head.GetDatabaseConfig()
		created, err := s.store.CreateBranch(ctx, &store.BranchMessage{
			ProjectID:  project.ResourceID,
			ResourceID: branchID,
			Engine:     parentBranch.Engine,
			Base: &storepb.BranchSnapshot{
				Metadata:       parentBranch.Head.Metadata,
				DatabaseConfig: parentBranchHeadConfig,
			},
			Head: &storepb.BranchSnapshot{
				Metadata:       parentBranch.Head.Metadata,
				DatabaseConfig: parentBranchHeadConfig,
			},
			BaseSchema: parentBranch.HeadSchema,
			HeadSchema: parentBranch.HeadSchema,
			Config: &storepb.BranchConfig{
				SourceBranch:   request.Branch.ParentBranch,
				SourceDatabase: parentBranch.Config.GetSourceDatabase(),
			},
			CreatorID: user.ID,
		})
		if err != nil {
			return nil, err
		}
		createdBranch = created
	} else if request.Branch.BaselineDatabase != "" {
		instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Branch.BaselineDatabase)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:          &instanceID,
			DatabaseName:        &databaseName,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		if database == nil {
			return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
		}
		databaseSchema, err := s.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		if databaseSchema == nil {
			return nil, status.Errorf(codes.NotFound, "database schema %q not found", databaseName)
		}
		filteredBaseSchemaMetadata := filterDatabaseMetadataByEngine(databaseSchema.GetMetadata(), instance.Engine)
		defaultSchema := extractDefaultSchemaForOracleBranch(storepb.Engine(instance.Engine), filteredBaseSchemaMetadata)
		baseSchema, err := schema.GetDesignSchema(instance.Engine, defaultSchema, "" /* baseline */, filteredBaseSchemaMetadata)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create branch: %v", err)
		}

		classificationConfig, err := s.store.GetDataClassificationConfigByID(ctx, project.DataClassificationConfigID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, `failed to get classification config by id "%s" with error: %v`, project.DataClassificationConfigID, err)
		}

		config := databaseSchema.GetConfig()
		sanitizeCommentForSchemaMetadata(filteredBaseSchemaMetadata, model.NewDatabaseConfig(config), classificationConfig.ClassificationFromConfig)
		initBranchLastUpdateInfoConfig(filteredBaseSchemaMetadata, config, "" /* User ID */, common.FormatBranchResourceID(project.ResourceID, branchID))
		created, err := s.store.CreateBranch(ctx, &store.BranchMessage{
			ProjectID:  project.ResourceID,
			ResourceID: branchID,
			Engine:     instance.Engine,
			Base: &storepb.BranchSnapshot{
				Metadata:       filteredBaseSchemaMetadata,
				DatabaseConfig: config,
			},
			Head: &storepb.BranchSnapshot{
				Metadata:       filteredBaseSchemaMetadata,
				DatabaseConfig: config,
			},
			BaseSchema: []byte(baseSchema),
			HeadSchema: []byte(baseSchema),
			Config: &storepb.BranchConfig{
				SourceDatabase: request.Branch.BaselineDatabase,
			},
			CreatorID: user.ID,
		})
		if err != nil {
			return nil, err
		}
		createdBranch = created
	} else {
		return nil, status.Errorf(codes.InvalidArgument, "either baseline database or parent branch must be specified")
	}

	v1Branch, err := s.convertBranchToBranch(ctx, project, createdBranch, v1pb.BranchView_BRANCH_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return v1Branch, nil
}

// UpdateBranch updates an existing branch.
func (s *BranchService) UpdateBranch(ctx context.Context, request *v1pb.UpdateBranchRequest) (*v1pb.Branch, error) {
	projectID, branchID, err := common.GetProjectAndBranchID(request.Branch.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid branch name: %v", err)
	}
	if request.UpdateMask == nil || len(request.UpdateMask.Paths) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask is required")
	}
	project, err := s.getProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	branch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &project.ResourceID, ResourceID: &branchID, LoadFull: true})
	if err != nil {
		return nil, err
	}
	if branch == nil {
		return nil, status.Errorf(codes.NotFound, "branch %q not found", branchID)
	}
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	ok, err = func() (bool, error) {
		if branch.CreatorID == user.ID {
			return true, nil
		}
		return s.iamManager.CheckPermission(ctx, iam.PermissionBranchesUpdate, user, project.ResourceID)
	}()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check permission, error: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied to update branch")
	}

	if request.Etag != "" && request.Etag != fmt.Sprintf("%d", branch.UpdatedTime.UnixMilli()) {
		return nil, status.Errorf(codes.Aborted, "there is concurrent update to the branch, please refresh and try again.")
	}

	// Handle branch ID update.
	if slices.Contains(request.UpdateMask.Paths, "branch_id") {
		if len(request.UpdateMask.Paths) > 1 {
			return nil, status.Errorf(codes.InvalidArgument, "cannot update branch_id with other types of updates")
		}
		updateBranchMessage := &store.UpdateBranchMessage{ProjectID: project.ResourceID, ResourceID: branchID, UpdaterID: user.ID, UpdateResourceID: &request.Branch.BranchId}
		if err := s.store.UpdateBranch(ctx, updateBranchMessage); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update branch, error %v", err)
		}
		// Update the branchID for getting branch in the end.
		branchID = request.Branch.BranchId
	}

	if slices.Contains(request.UpdateMask.Paths, "schema_metadata") {
		metadata, config, err := convertV1DatabaseMetadata(ctx, request.Branch.GetSchemaMetadata(), s.store)
		if err != nil {
			return nil, err
		}

		classificationConfig, err := s.store.GetDataClassificationConfigByID(ctx, project.DataClassificationConfigID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, `failed to get classification config by id "%s" with error: %v`, project.DataClassificationConfigID, err)
		}
		sanitizeCommentForSchemaMetadata(metadata, model.NewDatabaseConfig(config), classificationConfig.ClassificationFromConfig)

		reconcileMetadata(metadata, branch.Engine)
		filteredMetadata := filterDatabaseMetadataByEngine(metadata, branch.Engine)
		updateConfigBranchUpdateInfoForUpdate(branch.Head.Metadata, filteredMetadata, config, common.FormatUserUID(user.ID), common.FormatBranchResourceID(projectID, branchID))
		defaultSchema := extractDefaultSchemaForOracleBranch(storepb.Engine(branch.Engine), filteredMetadata)
		schema, err := schema.GetDesignSchema(branch.Engine, defaultSchema, "", filteredMetadata)
		if err != nil {
			return nil, err
		}
		schemaBytes := []byte(schema)
		headUpdate := &storepb.BranchSnapshot{
			Metadata:       filteredMetadata,
			DatabaseConfig: config,
		}
		updateBranchMessage := &store.UpdateBranchMessage{ProjectID: project.ResourceID, ResourceID: branchID, UpdaterID: user.ID}
		updateBranchMessage.Head = headUpdate
		updateBranchMessage.HeadSchema = &schemaBytes
		if err := s.store.UpdateBranch(ctx, updateBranchMessage); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update branch, error %v", err)
		}
	}

	branch, err = s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &project.ResourceID, ResourceID: &branchID, LoadFull: true})
	if err != nil {
		return nil, err
	}
	v1Branch, err := s.convertBranchToBranch(ctx, project, branch, v1pb.BranchView_BRANCH_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return v1Branch, nil
}

// We assume that the database name is the same as the schema name,
// and that an Oracle database with only one schema is managed based on the schema.
func extractDefaultSchemaForOracleBranch(engine storepb.Engine, metadata *storepb.DatabaseSchemaMetadata) string {
	if engine != storepb.Engine_ORACLE {
		return ""
	}
	defaultSchema := metadata.Name
	for _, schema := range metadata.Schemas {
		if schema.Name != defaultSchema {
			return ""
		}
	}
	return defaultSchema
}

// MergeBranch merges a personal draft branch to the target branch.
func (s *BranchService) MergeBranch(ctx context.Context, request *v1pb.MergeBranchRequest) (*v1pb.Branch, error) {
	projectID, baseBranchID, err := common.GetProjectAndBranchID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.getProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	baseBranch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &project.ResourceID, ResourceID: &baseBranchID, LoadFull: true})
	if err != nil {
		return nil, err
	}
	if baseBranch == nil {
		return nil, status.Errorf(codes.NotFound, "branch %q not found", baseBranchID)
	}
	if request.Etag != "" && request.Etag != fmt.Sprintf("%d", baseBranch.UpdatedTime.UnixMilli()) {
		return nil, status.Errorf(codes.Aborted, "there is concurrent update to the branch, please refresh and try again.")
	}

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}

	headProjectID, headBranchID, err := common.GetProjectAndBranchID(request.HeadBranch)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	_, err = s.getProject(ctx, headProjectID)
	if err != nil {
		return nil, err
	}
	ok, err = s.iamManager.CheckPermission(ctx, iam.PermissionBranchesGet, user, headProjectID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check permission, error: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied to get head branch")
	}
	headBranch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &headProjectID, ResourceID: &headBranchID, LoadFull: true})
	if err != nil {
		return nil, err
	}
	if headBranch == nil {
		return nil, status.Errorf(codes.NotFound, "branch %q not found", headBranchID)
	}

	ok, err = func() (bool, error) {
		if baseBranch.CreatorID == user.ID {
			return true, nil
		}
		if headBranch.CreatorID == user.ID {
			return true, nil
		}
		return s.iamManager.CheckPermission(ctx, iam.PermissionBranchesUpdate, user, project.ResourceID)
	}()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check permission, error: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied to merge branch")
	}

	// Restrict merging only when the head branch is not updated.
	// Maybe we can support auto-merging in the future.

	classificationConfig, err := s.store.GetDataClassificationConfigByID(ctx, project.DataClassificationConfigID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, `failed to get classification config by id "%s" with error: %v`, project.DataClassificationConfigID, err)
	}

	// The first crazy night in 2024.
	sanitizeCommentForSchemaMetadata(headBranch.Base.Metadata, model.NewDatabaseConfig(headBranch.Base.DatabaseConfig), classificationConfig.ClassificationFromConfig)
	sanitizeCommentForSchemaMetadata(headBranch.Head.Metadata, model.NewDatabaseConfig(headBranch.Head.DatabaseConfig), classificationConfig.ClassificationFromConfig)
	sanitizeCommentForSchemaMetadata(baseBranch.Head.Metadata, model.NewDatabaseConfig(baseBranch.Head.DatabaseConfig), classificationConfig.ClassificationFromConfig)

	adHead, err := tryMerge(headBranch.Base.Metadata, headBranch.Head.Metadata, baseBranch.Head.Metadata, baseBranch.Engine)
	if err != nil {
		slog.Info("cannot merge branches", log.BBError(err))
		return nil, status.Errorf(codes.Aborted, "cannot merge branches without conflict, error: %v", err)
	}
	mergedMetadata, err := tryMerge(baseBranch.Head.Metadata, adHead, baseBranch.Head.Metadata, baseBranch.Engine)
	if err != nil {
		slog.Info("cannot merge branches", log.BBError(err))
		return nil, status.Errorf(codes.Aborted, "cannot merge branches without conflict, error: %v", err)
	}
	if mergedMetadata == nil {
		return nil, status.Errorf(codes.Internal, "merged metadata should not be nil if there is no error while merging (%+v, %+v, %+v)", headBranch.Base.Metadata, headBranch.Head.Metadata, baseBranch.Head.Metadata)
	}
	defaultSchema := extractDefaultSchemaForOracleBranch(storepb.Engine(baseBranch.Engine), mergedMetadata)
	mergedSchema, err := schema.GetDesignSchema(storepb.Engine(baseBranch.Engine), defaultSchema, "", mergedMetadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert merged metadata to schema string, %v", err)
	}
	// XXX(zp): We only try to merge the schema config while the schema could be merged successfully. Otherwise, users manually merge the
	// metadata in the frontend, and config would be ignored.
	mergedConfig := utils.MergeDatabaseConfig(baseBranch.Head.GetDatabaseConfig(), headBranch.Base.GetDatabaseConfig(), headBranch.Head.GetDatabaseConfig())

	mergedSchemaBytes := []byte(mergedSchema)

	reconcileMetadata(mergedMetadata, baseBranch.Engine)
	filteredMergedMetadata := filterDatabaseMetadataByEngine(mergedMetadata, baseBranch.Engine)
	updateDatabaseConfigLastModifierForMerge(baseBranch.Head.Metadata, filteredMergedMetadata, baseBranch.Head.DatabaseConfig, mergedConfig)
	baseBranchNewHead := &storepb.BranchSnapshot{
		Metadata:       filteredMergedMetadata,
		DatabaseConfig: baseBranch.Head.DatabaseConfig,
	}
	baseBranchNewHeadSchema := mergedSchemaBytes

	if request.ValidateOnly {
		baseBranch.Head = baseBranchNewHead
		baseBranch.HeadSchema = baseBranchNewHeadSchema
	} else {
		if err := s.store.UpdateBranch(ctx, &store.UpdateBranchMessage{
			ProjectID:  project.ResourceID,
			ResourceID: baseBranchID,
			UpdaterID:  user.ID,
			Head:       baseBranchNewHead,
			HeadSchema: &baseBranchNewHeadSchema,
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "failed update branch, error %v", err)
		}
		baseBranch, err = s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &project.ResourceID, ResourceID: &baseBranchID, LoadFull: true})
		if err != nil {
			return nil, err
		}
	}

	v1Branch, err := s.convertBranchToBranch(ctx, project, baseBranch, v1pb.BranchView_BRANCH_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return v1Branch, nil
}

// RebaseBranch rebases a branch to the target branch.
func (s *BranchService) RebaseBranch(ctx context.Context, request *v1pb.RebaseBranchRequest) (*v1pb.RebaseBranchResponse, error) {
	baseProjectID, baseBranchID, err := common.GetProjectAndBranchID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	baseProject, err := s.getProject(ctx, baseProjectID)
	if err != nil {
		return nil, err
	}
	baseBranch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &baseProject.ResourceID, ResourceID: &baseBranchID, LoadFull: true})
	if err != nil {
		return nil, err
	}
	if baseBranch == nil {
		return nil, status.Errorf(codes.NotFound, "branch %q not found", baseBranchID)
	}
	if request.Etag != "" && request.Etag != fmt.Sprintf("%d", baseBranch.UpdatedTime.UnixMilli()) {
		return nil, status.Errorf(codes.Aborted, "there is concurrent update to the branch, please refresh and try again.")
	}

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	ok, err = func() (bool, error) {
		if baseBranch.CreatorID == user.ID {
			return true, nil
		}
		return s.iamManager.CheckPermission(ctx, iam.PermissionBranchesUpdate, user, baseProject.ResourceID)
	}()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check permission, error: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied to rebase branch")
	}
	// Main branch IAM admin check.
	if baseBranch.Config.GetSourceBranch() == "" {
		ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionBranchesAdmin, user, baseProject.ResourceID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "only users with %s permission can rebase a main branch", iam.PermissionBranchesAdmin)
		}
	}

	filteredNewBaseMetadata, newBaseSchema, newBaseConfig, err := s.getFilteredNewBaseFromRebaseRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	var newHeadMetadata *storepb.DatabaseSchemaMetadata
	var newHeadConfig *storepb.DatabaseConfig
	if request.MergedSchema != "" {
		newHeadMetadata, err = schema.ParseToMetadata(storepb.Engine(baseBranch.Engine), "" /* defaultSchemaName */, request.MergedSchema)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to convert merged schema to metadata, %v", err)
		}
		newHeadConfig = baseBranch.Head.GetDatabaseConfig()
		updateDatabaseConfigLastModifierForMerge(baseBranch.Head.Metadata, newHeadMetadata, baseBranch.Head.DatabaseConfig, newHeadConfig)
		newHeadConfig = baseBranch.Head.GetDatabaseConfig()
		// String-based rebase operation do not include the structural information, such as classification, so we need to sanitize the user comment,
		// trim the classification in user comment if the classification is not from the config.
		modelNewHeadConfig := model.NewDatabaseConfig(newHeadConfig)

		classificationConfig, err := s.store.GetDataClassificationConfigByID(ctx, baseProject.DataClassificationConfigID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, `failed to get classification config by id "%s" with error: %v`, baseProject.DataClassificationConfigID, err)
		}

		trimClassificationIDFromCommentIfNeeded(newHeadMetadata, classificationConfig.ClassificationFromConfig)
		sanitizeCommentForSchemaMetadata(newHeadMetadata, modelNewHeadConfig, classificationConfig.ClassificationFromConfig)
	} else {
		newHeadMetadata, err = tryMerge(baseBranch.Base.Metadata, baseBranch.Head.Metadata, filteredNewBaseMetadata, baseBranch.Engine)
		if err != nil {
			slog.Info("cannot rebase branches", log.BBError(err))
			conflictSchema, err := diff3.Merge(
				strings.NewReader(newBaseSchema),
				bytes.NewReader(baseBranch.BaseSchema),
				bytes.NewReader(baseBranch.HeadSchema),
				true,
				"HEAD",
				baseBranch.ResourceID,
			)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to compute conflict schema, %v", err)
			}
			sb, err := io.ReadAll(conflictSchema.Result)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to read conflict schema, %v", err)
			}
			if strings.HasSuffix(newBaseSchema, "\n") && bytes.HasSuffix(baseBranch.BaseSchema, []byte("\n")) && bytes.HasSuffix(baseBranch.HeadSchema, []byte("\n")) {
				sb = append(sb, []byte("\n")...)
			}
			conflictSchemaString := string(sb)
			return &v1pb.RebaseBranchResponse{Result: &v1pb.RebaseBranchResponse_ConflictSchema{ConflictSchema: conflictSchemaString}}, nil
		}
		if newHeadMetadata == nil {
			return nil, status.Errorf(codes.Internal, "merged metadata should not be nil if there is no error while merging (%+v, %+v, %+v)", baseBranch.Base.Metadata, baseBranch.Head.Metadata, filteredNewBaseMetadata)
		}
		// XXX(zp): We only try to merge the schema config while the schema could be merged successfully. Otherwise, users manually merge the
		// metadata in the frontend, and config would be ignored.
		newHeadConfig = utils.MergeDatabaseConfig(newBaseConfig, baseBranch.Base.GetDatabaseConfig(), baseBranch.Head.GetDatabaseConfig())
	}

	reconcileMetadata(newHeadMetadata, baseBranch.Engine)
	filteredNewHeadMetadata := filterDatabaseMetadataByEngine(newHeadMetadata, baseBranch.Engine)
	defaultSchema := extractDefaultSchemaForOracleBranch(storepb.Engine(baseBranch.Engine), newHeadMetadata)
	newHeadSchema, err := schema.GetDesignSchema(storepb.Engine(baseBranch.Engine), defaultSchema, "", filteredNewHeadMetadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert new head metadata to schema string, %v", err)
	}

	newBaseSchemaBytes := []byte(newBaseSchema)
	newHeadSchemaBytes := []byte(newHeadSchema)
	updateDatabaseConfigLastModifierForMerge(filteredNewHeadMetadata, filteredNewBaseMetadata, newHeadConfig, newBaseConfig)
	if request.ValidateOnly {
		baseBranch.Base = &storepb.BranchSnapshot{
			Metadata:       filteredNewBaseMetadata,
			DatabaseConfig: newBaseConfig,
		}
		baseBranch.BaseSchema = newBaseSchemaBytes
		baseBranch.Head = &storepb.BranchSnapshot{
			Metadata:       filteredNewHeadMetadata,
			DatabaseConfig: newHeadConfig,
		}
		baseBranch.HeadSchema = newHeadSchemaBytes
	} else {
		if err := s.store.UpdateBranch(ctx, &store.UpdateBranchMessage{
			ProjectID:  baseProject.ResourceID,
			ResourceID: baseBranchID,
			UpdaterID:  user.ID,
			Base: &storepb.BranchSnapshot{
				Metadata:       filteredNewBaseMetadata,
				DatabaseConfig: newBaseConfig,
			},
			BaseSchema: &newBaseSchemaBytes,
			Head: &storepb.BranchSnapshot{
				Metadata: filteredNewHeadMetadata,
				// TODO(d): handle config.
				DatabaseConfig: newHeadConfig,
			},
			HeadSchema: &newHeadSchemaBytes,
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "failed update branch, error %v", err)
		}
		baseBranch, err = s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &baseProject.ResourceID, ResourceID: &baseBranchID, LoadFull: true})
		if err != nil {
			return nil, err
		}
	}
	v1Branch, err := s.convertBranchToBranch(ctx, baseProject, baseBranch, v1pb.BranchView_BRANCH_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return &v1pb.RebaseBranchResponse{Result: &v1pb.RebaseBranchResponse_Branch{Branch: v1Branch}}, nil
}

func (s *BranchService) getFilteredNewBaseFromRebaseRequest(ctx context.Context, request *v1pb.RebaseBranchRequest) (*storepb.DatabaseSchemaMetadata, string, *storepb.DatabaseConfig, error) {
	if request.SourceDatabase != "" {
		instanceID, databaseName, err := common.GetInstanceDatabaseID(request.SourceDatabase)
		if err != nil {
			return nil, "", nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, "", nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		if instance == nil {
			return nil, "", nil, status.Errorf(codes.NotFound, "instance %q not found or had been deleted", instanceID)
		}
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:          &instanceID,
			DatabaseName:        &databaseName,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		})
		if err != nil {
			return nil, "", nil, status.Errorf(codes.Internal, err.Error())
		}
		if database == nil {
			return nil, "", nil, status.Errorf(codes.NotFound, "database %q not found or had been archive", databaseName)
		}
		databaseSchema, err := s.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return nil, "", nil, status.Errorf(codes.Internal, err.Error())
		}
		if databaseSchema == nil {
			return nil, "", nil, status.Errorf(codes.NotFound, "database schema %q not found", databaseName)
		}
		databaseMetadata := databaseSchema.GetMetadata()
		filteredNewBaseMetadata := filterDatabaseMetadataByEngine(databaseMetadata, instance.Engine)
		defaultStoreSourceSchema := extractDefaultSchemaForOracleBranch(instance.Engine, filteredNewBaseMetadata)
		sourceSchema, err := schema.GetDesignSchema(instance.Engine, defaultStoreSourceSchema, "" /* baseline*/, filteredNewBaseMetadata)
		if err != nil {
			return nil, "", nil, status.Errorf(codes.Internal, err.Error())
		}

		alignedDatabaseConfig := databaseSchema.GetConfig()
		alignDatabaseConfig(filteredNewBaseMetadata, alignedDatabaseConfig)
		return filteredNewBaseMetadata, sourceSchema, alignedDatabaseConfig, nil
	}

	if request.SourceBranch != "" {
		sourceProjectID, sourceBranchID, err := common.GetProjectAndBranchID(request.SourceBranch)
		if err != nil {
			return nil, "", nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		sourceBranch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &sourceProjectID, ResourceID: &sourceBranchID, LoadFull: true})
		if err != nil {
			return nil, "", nil, err
		}
		if sourceBranch == nil {
			return nil, "", nil, status.Errorf(codes.NotFound, "branch %q not found", sourceBranchID)
		}
		return sourceBranch.Head.GetMetadata(), string(sourceBranch.HeadSchema), sourceBranch.Head.GetDatabaseConfig(), nil
	}

	return nil, "", nil, status.Errorf(codes.InvalidArgument, "either source_database or source_branch should be specified")
}

// DeleteBranch deletes an existing branch.
func (s *BranchService) DeleteBranch(ctx context.Context, request *v1pb.DeleteBranchRequest) (*emptypb.Empty, error) {
	projectID, branchID, err := common.GetProjectAndBranchID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.getProject(ctx, projectID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	branch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &project.ResourceID, ResourceID: &branchID, LoadFull: false})
	if err != nil {
		return nil, err
	}
	if branch == nil {
		return nil, status.Errorf(codes.NotFound, "branch %q not found", branchID)
	}

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	ok, err = func() (bool, error) {
		if branch.CreatorID == user.ID {
			return true, nil
		}
		return s.iamManager.CheckPermission(ctx, iam.PermissionBranchesDelete, user, project.ResourceID)
	}()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check permission, error: %v", err)
	}
	if !ok {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied to delete branch")
	}
	// Main branch IAM admin check.
	if branch.Config.GetSourceBranch() == "" {
		ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionBranchesAdmin, user, project.ResourceID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "only users with %s permission can delete a main branch", iam.PermissionBranchesAdmin)
		}
	}

	if !request.Force {
		childBranches, err := s.store.ListBranches(ctx, &store.FindBranchMessage{
			ProjectID:              &project.ResourceID,
			LoadFull:               false,
			ParentBranchResourceID: &request.Name,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list child branches, error %v", err)
		}
		if len(childBranches) > 0 {
			return nil, status.Errorf(codes.FailedPrecondition, "branch %q has child branches, please delete them first", branchID)
		}
	}

	if err := s.store.DeleteBranch(ctx, project.ResourceID, branch.ResourceID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete branch, error %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (*BranchService) DiffMetadata(ctx context.Context, request *v1pb.DiffMetadataRequest) (*v1pb.DiffMetadataResponse, error) {
	switch request.Engine {
	case v1pb.Engine_MYSQL, v1pb.Engine_POSTGRES, v1pb.Engine_TIDB, v1pb.Engine_ORACLE:
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported engine: %v", request.Engine)
	}
	if request.SourceMetadata == nil || request.TargetMetadata == nil {
		return nil, status.Errorf(codes.InvalidArgument, "source_metadata and target_metadata are required")
	}
	storeSourceMetadata, sourceConfig, err := convertV1DatabaseMetadata(ctx, request.SourceMetadata, nil /* optionalStores */)
	if err != nil {
		return nil, err
	}
	sanitizeCommentForSchemaMetadata(storeSourceMetadata, model.NewDatabaseConfig(sourceConfig), request.ClassificationFromConfig)

	storeTargetMetadata, targetConfig, err := convertV1DatabaseMetadata(ctx, request.TargetMetadata, nil /* optionalStores */)
	if err != nil {
		return nil, err
	}
	if err := checkDatabaseMetadata(storepb.Engine(request.Engine), storeTargetMetadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target metadata: %v", err)
	}
	sanitizeCommentForSchemaMetadata(storeTargetMetadata, model.NewDatabaseConfig(targetConfig), request.ClassificationFromConfig)

	storeSourceMetadata, storeTargetMetadata = trimDatabaseMetadata(storeSourceMetadata, storeTargetMetadata)
	if err := checkDatabaseMetadataColumnType(storepb.Engine(request.Engine), storeTargetMetadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target metadata: %v", err)
	}

	defaultStoreSourceSchema := extractDefaultSchemaForOracleBranch(storepb.Engine(request.Engine), storeSourceMetadata)
	sourceSchema, err := schema.GetDesignSchema(storepb.Engine(request.Engine), defaultStoreSourceSchema, "" /* baseline*/, storeSourceMetadata)
	if err != nil {
		return nil, err
	}
	defaultStoreTargetSchema := extractDefaultSchemaForOracleBranch(storepb.Engine(request.Engine), storeTargetMetadata)
	targetSchema, err := schema.GetDesignSchema(storepb.Engine(request.Engine), defaultStoreTargetSchema, "" /* baseline*/, storeTargetMetadata)
	if err != nil {
		return nil, err
	}

	diff, err := base.SchemaDiff(convertEngine(request.Engine), base.DiffContext{
		IgnoreCaseSensitive: false,
		StrictMode:          true,
	}, sourceSchema, targetSchema)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compute diff between source and target schemas, error: %v", err)
	}

	return &v1pb.DiffMetadataResponse{
		Diff: diff,
	}, nil
}

func (s *BranchService) getProject(ctx context.Context, projectID string) (*store.ProjectMessage, error) {
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", projectID)
	}
	return project, nil
}

func (s *BranchService) convertBranchToBranch(ctx context.Context, project *store.ProjectMessage, branch *store.BranchMessage, view v1pb.BranchView) (*v1pb.Branch, error) {
	creator, err := s.store.GetUserByID(ctx, branch.CreatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get creator, error %v", err)
	}
	if creator == nil {
		return nil, status.Errorf(codes.NotFound, "creator %d not found", branch.CreatorID)
	}
	updater, err := s.store.GetUserByID(ctx, branch.UpdaterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get updater, error %v", err)
	}
	if updater == nil {
		return nil, status.Errorf(codes.NotFound, "updater %d not found", branch.UpdaterID)
	}

	var baselineDatabase, baselineBranch string
	if branch.Config != nil {
		baselineDatabase = branch.Config.SourceDatabase
		if branch.Config.SourceBranch != "" {
			baselineBranch = branch.Config.SourceBranch
		}
	}

	v1Branch := &v1pb.Branch{
		Name:             fmt.Sprintf("%s%s/%s%v", common.ProjectNamePrefix, project.ResourceID, common.BranchPrefix, branch.ResourceID),
		BranchId:         branch.ResourceID,
		Etag:             fmt.Sprintf("%d", branch.UpdatedTime.UnixMilli()),
		ParentBranch:     baselineBranch,
		Engine:           v1pb.Engine(branch.Engine),
		BaselineDatabase: baselineDatabase,
		Creator:          common.FormatUserEmail(creator.Email),
		Updater:          common.FormatUserEmail(updater.Email),
		CreateTime:       timestamppb.New(branch.CreatedTime),
		UpdateTime:       timestamppb.New(branch.UpdatedTime),
	}

	if view != v1pb.BranchView_BRANCH_VIEW_FULL {
		return v1Branch, nil
	}

	v1Branch.Schema = string(branch.HeadSchema)
	sm, err := convertStoreDatabaseMetadata(ctx, branch.Head.Metadata, branch.Head.DatabaseConfig, nil /* filter */, s.store)
	if err != nil {
		return nil, err
	}
	v1Branch.SchemaMetadata = sm
	v1Branch.BaselineSchema = string(branch.BaseSchema)
	bsm, err := convertStoreDatabaseMetadata(ctx, branch.Base.Metadata, branch.Base.DatabaseConfig, nil /* filter */, s.store)
	if err != nil {
		return nil, err
	}
	v1Branch.BaselineSchemaMetadata = bsm
	return v1Branch, nil
}

func sanitizeCommentForSchemaMetadata(dbSchema *storepb.DatabaseSchemaMetadata, dbModelConfig *model.DatabaseConfig, classificationFromConfig bool) {
	for _, schema := range dbSchema.Schemas {
		schemaConfig := dbModelConfig.CreateOrGetSchemaConfig(schema.Name)
		for _, table := range schema.Tables {
			tableConfig := schemaConfig.CreateOrGetTableConfig(table.Name)
			classificationID := ""
			if !classificationFromConfig {
				classificationID = tableConfig.ClassificationID
			}
			table.Comment = common.GetCommentFromClassificationAndUserComment(classificationID, table.UserComment)
			for _, col := range table.Columns {
				columnConfig := tableConfig.CreateOrGetColumnConfig(col.Name)
				classificationID := ""
				if !classificationFromConfig {
					classificationID = columnConfig.ClassificationId
				}
				col.Comment = common.GetCommentFromClassificationAndUserComment(classificationID, col.UserComment)
			}
		}
	}
}

// filterDatabaseMetadata filter out the objects/attributes we do not support.
// TODO: While supporting new objects/attributes, we should update this function.
func filterDatabaseMetadataByEngine(metadata *storepb.DatabaseSchemaMetadata, engine storepb.Engine) *storepb.DatabaseSchemaMetadata {
	filteredDatabase := &storepb.DatabaseSchemaMetadata{
		Name: metadata.Name,
	}
	for _, schema := range metadata.Schemas {
		filteredSchema := &storepb.SchemaMetadata{
			Name: schema.Name,
		}
		for _, table := range schema.Tables {
			filteredTable := &storepb.TableMetadata{
				Name:        table.Name,
				Comment:     table.Comment,
				UserComment: table.UserComment,
				// For Display only.
				Collation: table.Collation,
				Engine:    table.Engine,
			}
			for _, column := range table.Columns {
				filteredColumn := &storepb.ColumnMetadata{
					Name:         column.Name,
					OnUpdate:     column.OnUpdate,
					Comment:      column.Comment,
					UserComment:  column.UserComment,
					Type:         column.Type,
					DefaultValue: column.DefaultValue,
					Nullable:     column.Nullable,
					Position:     column.Position,
				}
				filteredTable.Columns = append(filteredTable.Columns, filteredColumn)
			}
			for _, index := range table.Indexes {
				filteredIndex := &storepb.IndexMetadata{
					Name:        index.Name,
					Definition:  index.Definition,
					Primary:     index.Primary,
					Unique:      index.Unique,
					Comment:     index.Comment,
					Expressions: index.Expressions,
					KeyLength:   index.KeyLength,
				}
				filteredTable.Indexes = append(filteredTable.Indexes, filteredIndex)
			}
			for _, fk := range table.ForeignKeys {
				filteredFK := &storepb.ForeignKeyMetadata{
					Name:              fk.Name,
					Columns:           fk.Columns,
					ReferencedTable:   fk.ReferencedTable,
					ReferencedColumns: fk.ReferencedColumns,
					ReferencedSchema:  fk.ReferencedSchema,
				}
				filteredTable.ForeignKeys = append(filteredTable.ForeignKeys, filteredFK)
			}
			if engine == storepb.Engine_MYSQL {
				filteredTable.Partitions = table.Partitions
			}
			filteredSchema.Tables = append(filteredSchema.Tables, filteredTable)
		}

		if engine == storepb.Engine_MYSQL {
			for _, function := range schema.Functions {
				filteredFunction := &storepb.FunctionMetadata{
					Name:       function.Name,
					Definition: function.Definition,
				}
				filteredSchema.Functions = append(filteredSchema.Functions, filteredFunction)
			}
			for _, procedure := range schema.Procedures {
				filteredProcedure := &storepb.ProcedureMetadata{
					Name:       procedure.Name,
					Definition: procedure.Definition,
				}
				filteredSchema.Procedures = append(filteredSchema.Procedures, filteredProcedure)
			}
			for _, view := range schema.Views {
				filteredView := &storepb.ViewMetadata{
					Name:       view.Name,
					Comment:    view.Comment,
					Definition: view.Definition,
				}
				filteredSchema.Views = append(filteredSchema.Views, filteredView)
			}
		}
		filteredDatabase.Schemas = append(filteredDatabase.Schemas, filteredSchema)
	}

	return filteredDatabase
}

func trimDatabaseMetadata(sourceMetadata *storepb.DatabaseSchemaMetadata, targetMetadata *storepb.DatabaseSchemaMetadata) (*storepb.DatabaseSchemaMetadata, *storepb.DatabaseSchemaMetadata) {
	// TODO(d): handle indexes, etc.
	sourceModel, targetModel := model.NewDatabaseMetadata(sourceMetadata), model.NewDatabaseMetadata(targetMetadata)
	s, t := &storepb.DatabaseSchemaMetadata{}, &storepb.DatabaseSchemaMetadata{}
	for _, schema := range sourceMetadata.GetSchemas() {
		ts := targetModel.GetSchema(schema.GetName())
		if ts == nil {
			s.Schemas = append(s.Schemas, schema)
			continue
		}
		trimSchema := &storepb.SchemaMetadata{Name: schema.GetName()}
		for _, table := range schema.GetTables() {
			tt := ts.GetTable(table.GetName())
			if tt == nil {
				trimSchema.Tables = append(trimSchema.Tables, table)
				continue
			}

			if !common.EqualTable(table, tt.GetProto()) {
				trimSchema.Tables = append(trimSchema.Tables, table)
				continue
			}
		}
		for _, view := range schema.GetViews() {
			tv := ts.GetView(view.GetName())
			if tv == nil {
				trimSchema.Views = append(trimSchema.Views, view)
				continue
			}
			if view.GetComment() != tv.GetProto().GetComment() {
				trimSchema.Views = append(trimSchema.Views, view)
				continue
			}
			if view.GetDefinition() != tv.Definition {
				trimSchema.Views = append(trimSchema.Views, view)
				continue
			}
		}
		for _, function := range schema.GetFunctions() {
			tf := ts.GetFunction(function.GetName())
			if tf == nil {
				trimSchema.Functions = append(trimSchema.Functions, function)
				continue
			}
			if function.GetDefinition() != tf.Definition {
				trimSchema.Functions = append(trimSchema.Functions, function)
				continue
			}
		}
		for _, procedure := range schema.GetProcedures() {
			tp := ts.GetProcedure(procedure.GetName())
			if tp == nil {
				trimSchema.Procedures = append(trimSchema.Procedures, procedure)
				continue
			}
			if procedure.GetDefinition() != tp.Definition {
				trimSchema.Procedures = append(trimSchema.Procedures, procedure)
				continue
			}
		}
		if len(trimSchema.Tables) > 0 || len(trimSchema.Views) > 0 || len(trimSchema.Functions) > 0 || len(trimSchema.Procedures) > 0 {
			s.Schemas = append(s.Schemas, trimSchema)
		}
	}

	for _, schema := range targetMetadata.GetSchemas() {
		ts := sourceModel.GetSchema(schema.GetName())
		if ts == nil {
			t.Schemas = append(t.Schemas, schema)
			continue
		}
		trimSchema := &storepb.SchemaMetadata{Name: schema.GetName()}
		for _, table := range schema.GetTables() {
			tt := ts.GetTable(table.GetName())
			if tt == nil {
				trimSchema.Tables = append(trimSchema.Tables, table)
				continue
			}

			if !common.EqualTable(table, tt.GetProto()) {
				trimSchema.Tables = append(trimSchema.Tables, table)
				continue
			}
		}
		for _, view := range schema.GetViews() {
			tv := ts.GetView(view.GetName())
			if tv == nil {
				trimSchema.Views = append(trimSchema.Views, view)
				continue
			}
			if view.GetDefinition() != tv.Definition {
				trimSchema.Views = append(trimSchema.Views, view)
				continue
			}
		}
		for _, function := range schema.GetFunctions() {
			tf := ts.GetFunction(function.GetName())
			if tf == nil {
				trimSchema.Functions = append(trimSchema.Functions, function)
				continue
			}
			if function.GetDefinition() != tf.Definition {
				trimSchema.Functions = append(trimSchema.Functions, function)
				continue
			}
		}
		for _, procedure := range schema.GetProcedures() {
			tp := ts.GetProcedure(procedure.GetName())
			if tp == nil {
				trimSchema.Procedures = append(trimSchema.Procedures, procedure)
				continue
			}
			if procedure.GetDefinition() != tp.Definition {
				trimSchema.Procedures = append(trimSchema.Procedures, procedure)
				continue
			}
		}
		if len(trimSchema.Tables) > 0 || len(trimSchema.Views) > 0 || len(trimSchema.Functions) > 0 || len(trimSchema.Procedures) > 0 {
			t.Schemas = append(t.Schemas, trimSchema)
		}
	}

	return s, t
}

func reconcileMetadata(metadata *storepb.DatabaseSchemaMetadata, engine storepb.Engine) {
	for _, schema := range metadata.Schemas {
		for _, table := range schema.Tables {
			if engine == storepb.Engine_MYSQL {
				reconcileMySQLPartitionMetadata(table.Partitions, "")
			}
			if engine == storepb.Engine_MYSQL || engine == storepb.Engine_TIDB {
				for _, column := range table.Columns {
					// If the column can take NULL as a value, the column is defined with an explicit DEFAULT NULL clause.
					if column.Nullable && column.DefaultValue == nil {
						column.DefaultValue = &storepb.ColumnMetadata_DefaultNull{
							DefaultNull: true,
						}
					}
				}
			}
		}
		for _, view := range schema.Views {
			if engine == storepb.Engine_MYSQL || engine == storepb.Engine_TIDB {
				view.Definition = formatViewDef(view.Definition)
			}
		}
	}
}

func reconcileMySQLPartitionMetadata(partitions []*storepb.TablePartitionMetadata, parentName string) {
	if len(partitions) == 0 {
		return
	}
	defaultParentGenerator := mysql.NewPartitionDefaultNameGenerator(parentName)
	defaultParentNames := make([]string, len(partitions))
	for i := range partitions {
		defaultParentNames[i] = defaultParentGenerator.Next()
	}

	if len(partitions) > 0 && partitions[0].UseDefault != "" {
		useDefault, err := strconv.Atoi(partitions[0].UseDefault)
		if err != nil {
			slog.Warn("failed to parse use default", log.BBError(err))
			return
		}
		if useDefault != 0 && useDefault != len(partitions) {
			for i := range partitions {
				partitions[i].UseDefault = strconv.Itoa(len(partitions))
			}
		}
	}
	names := make([]string, len(partitions))
	for i := range partitions {
		names[i] = partitions[i].Name
	}
	if !slices.Equal(names, defaultParentNames) {
		for i := range partitions {
			partitions[i].UseDefault = ""
		}
	}

	for _, partition := range partitions {
		reconcileMySQLPartitionMetadata(partition.Subpartitions, partition.Name)
	}
}

// updateConfigBranchUpdateInfoForUpdate compare the proto of old and new metadata, and update the config branch update info.
// NOTE: this function would not delete the config of deleted objects, and it's safe because the next time adding the object
// back will trigger the update of the config branch update info.
func updateConfigBranchUpdateInfoForUpdate(old *storepb.DatabaseSchemaMetadata, new *storepb.DatabaseSchemaMetadata, config *storepb.DatabaseConfig, formattedUserUID string, formattedBranchResourceID string) {
	time := timestamppb.Now()

	oldModel := model.NewDatabaseMetadata(old)

	newSchemaConfigMap := buildMap(config.SchemaConfigs, func(s *storepb.SchemaConfig) string {
		return s.Name
	})
	var newSchemaConfigs []*storepb.SchemaConfig
	for _, schema := range new.Schemas {
		newSchemaConfig, ok := newSchemaConfigMap[schema.Name]
		if !ok {
			newSchemaConfigs = append(newSchemaConfigs, initSchemaConfig(schema, formattedUserUID, formattedBranchResourceID, time))
			continue
		}
		oldSchema := oldModel.GetSchema(schema.Name)
		if oldSchema == nil {
			for _, tableConfig := range newSchemaConfig.TableConfigs {
				tableConfig.Updater = formattedUserUID
				tableConfig.UpdateTime = time
				tableConfig.SourceBranch = formattedBranchResourceID
			}
			for _, viewConfig := range newSchemaConfig.ViewConfigs {
				viewConfig.Updater = formattedUserUID
				viewConfig.UpdateTime = time
				viewConfig.SourceBranch = formattedBranchResourceID
			}
			for _, functionConfig := range newSchemaConfig.FunctionConfigs {
				functionConfig.Updater = formattedUserUID
				functionConfig.UpdateTime = time
				functionConfig.SourceBranch = formattedBranchResourceID
			}
			for _, procedureConfig := range newSchemaConfig.ProcedureConfigs {
				procedureConfig.Updater = formattedUserUID
				procedureConfig.UpdateTime = time
				procedureConfig.SourceBranch = formattedBranchResourceID
			}
			continue
		}

		var newTableConfig []*storepb.TableConfig
		tableConfigMap := buildMap(newSchemaConfig.TableConfigs, func(t *storepb.TableConfig) string {
			return t.Name
		})
		for _, table := range schema.Tables {
			tableConfig, ok := tableConfigMap[table.Name]
			if !ok {
				newTableConfig = append(newTableConfig, initTableConfig(table, formattedUserUID, formattedBranchResourceID, time))
				continue
			}
			oldTable := oldSchema.GetTable(table.Name)
			if oldTable == nil {
				// If users delete the table first, and then add it back, we should update the config branch update info.
				tableConfig.Updater = formattedUserUID
				tableConfig.UpdateTime = time
				tableConfig.SourceBranch = formattedBranchResourceID
				continue
			}
			if !cmp.Equal(table, oldTable.GetProto(), protocmp.Transform()) {
				tableConfig.Updater = formattedUserUID
				tableConfig.UpdateTime = time
				tableConfig.SourceBranch = formattedBranchResourceID
			}
		}

		var newViewConfig []*storepb.ViewConfig
		viewConfigMap := buildMap(newSchemaConfig.ViewConfigs, func(v *storepb.ViewConfig) string {
			return v.Name
		})
		for _, view := range schema.Views {
			viewConfig, ok := viewConfigMap[view.Name]
			if !ok {
				newViewConfig = append(newViewConfig, initViewConfig(view, formattedUserUID, formattedBranchResourceID, time))
				continue
			}
			oldView := oldSchema.GetView(view.Name)
			if oldView == nil {
				// If users delete the view first, and then add it back, we should update the config branch update info.
				viewConfig.Updater = formattedUserUID
				viewConfig.UpdateTime = time
				viewConfig.SourceBranch = formattedBranchResourceID
				continue
			}
			if !cmp.Equal(view, oldView.GetProto(), protocmp.Transform()) {
				viewConfig.Updater = formattedUserUID
				viewConfig.UpdateTime = time
				viewConfig.SourceBranch = formattedBranchResourceID
			}
		}

		var newFunctionConfig []*storepb.FunctionConfig
		functionConfigMap := buildMap(newSchemaConfig.FunctionConfigs, func(f *storepb.FunctionConfig) string {
			return f.Name
		})
		for _, function := range schema.Functions {
			functionConfig, ok := functionConfigMap[function.Name]
			if !ok {
				newFunctionConfig = append(newFunctionConfig, initFunctionConfig(function, formattedUserUID, formattedBranchResourceID, time))
				continue
			}
			oldFunction := oldSchema.GetFunction(function.Name)
			if oldFunction == nil {
				// If users delete the function first, and then add it back, we should update the config branch update info.
				functionConfig.Updater = formattedUserUID
				functionConfig.UpdateTime = time
				functionConfig.SourceBranch = formattedBranchResourceID
				continue
			}
			if !cmp.Equal(function, oldFunction.GetProto(), protocmp.Transform()) {
				functionConfig.Updater = formattedUserUID
				functionConfig.UpdateTime = time
				functionConfig.SourceBranch = formattedBranchResourceID
			}
		}

		var newProcedureConfig []*storepb.ProcedureConfig
		procedureConfigMap := buildMap(newSchemaConfig.ProcedureConfigs, func(p *storepb.ProcedureConfig) string {
			return p.Name
		})
		for _, procedure := range schema.Procedures {
			procedureConfig, ok := procedureConfigMap[procedure.Name]
			if !ok {
				newProcedureConfig = append(newProcedureConfig, initProcedureConfig(procedure, formattedUserUID, formattedBranchResourceID, time))
				continue
			}
			oldProcedure := oldSchema.GetProcedure(procedure.Name)
			if oldProcedure == nil {
				// If users delete the procedure first, and then add it back, we should update the config branch update info.
				procedureConfig.Updater = formattedUserUID
				procedureConfig.UpdateTime = time
				procedureConfig.SourceBranch = formattedBranchResourceID
				continue
			}
			if !cmp.Equal(procedure, oldProcedure.GetProto(), protocmp.Transform()) {
				procedureConfig.Updater = formattedUserUID
				procedureConfig.UpdateTime = time
				procedureConfig.SourceBranch = formattedBranchResourceID
			}
		}

		newSchemaConfig.TableConfigs = append(newSchemaConfig.TableConfigs, newTableConfig...)
		newSchemaConfig.ViewConfigs = append(newSchemaConfig.ViewConfigs, newViewConfig...)
		newSchemaConfig.FunctionConfigs = append(newSchemaConfig.FunctionConfigs, newFunctionConfig...)
		newSchemaConfig.ProcedureConfigs = append(newSchemaConfig.ProcedureConfigs, newProcedureConfig...)
	}
	config.SchemaConfigs = append(config.SchemaConfigs, newSchemaConfigs...)
}

func initSchemaConfig(schema *storepb.SchemaMetadata, formattedUserUID string, branchResourceID string, time *timestamppb.Timestamp) *storepb.SchemaConfig {
	s := &storepb.SchemaConfig{
		Name: schema.Name,
	}

	for _, table := range schema.Tables {
		s.TableConfigs = append(s.TableConfigs, initTableConfig(table, formattedUserUID, branchResourceID, time))
	}

	for _, view := range schema.Views {
		s.ViewConfigs = append(s.ViewConfigs, initViewConfig(view, formattedUserUID, branchResourceID, time))
	}

	for _, function := range schema.Functions {
		s.FunctionConfigs = append(s.FunctionConfigs, initFunctionConfig(function, formattedUserUID, branchResourceID, time))
	}

	for _, procedure := range schema.Procedures {
		s.ProcedureConfigs = append(s.ProcedureConfigs, initProcedureConfig(procedure, formattedUserUID, branchResourceID, time))
	}

	return s
}

func initTableConfig(table *storepb.TableMetadata, formattedUserEmail string, branchResourceID string, time *timestamppb.Timestamp) *storepb.TableConfig {
	return &storepb.TableConfig{
		Name:         table.Name,
		Updater:      formattedUserEmail,
		SourceBranch: branchResourceID,
		UpdateTime:   time,
	}
}

func initViewConfig(view *storepb.ViewMetadata, formattedUserEmail string, branchResourceID string, time *timestamppb.Timestamp) *storepb.ViewConfig {
	return &storepb.ViewConfig{
		Name:         view.Name,
		Updater:      formattedUserEmail,
		SourceBranch: branchResourceID,
		UpdateTime:   time,
	}
}

func initFunctionConfig(function *storepb.FunctionMetadata, formattedUserEmail string, branchResourceID string, time *timestamppb.Timestamp) *storepb.FunctionConfig {
	return &storepb.FunctionConfig{
		Name:         function.Name,
		Updater:      formattedUserEmail,
		SourceBranch: branchResourceID,
		UpdateTime:   time,
	}
}

func initProcedureConfig(procedure *storepb.ProcedureMetadata, formattedUserEmail string, branchResourceID string, time *timestamppb.Timestamp) *storepb.ProcedureConfig {
	return &storepb.ProcedureConfig{
		Name:         procedure.Name,
		Updater:      formattedUserEmail,
		SourceBranch: branchResourceID,
		UpdateTime:   time,
	}
}

func buildMap[T any](objects []T, getUniqueIdentifier func(T) string) map[string]T {
	m := make(map[string]T)
	for _, obj := range objects {
		m[getUniqueIdentifier(obj)] = obj
	}
	return m
}

func initBranchLastUpdateInfoConfig(metadata *storepb.DatabaseSchemaMetadata, config *storepb.DatabaseConfig, formattedUserUID, formattedBranchResourceID string) {
	time := timestamppb.Now()
	schemaConfigMap := buildMap(config.SchemaConfigs, func(s *storepb.SchemaConfig) string {
		return s.Name
	})
	for _, schema := range metadata.Schemas {
		schemaConfig, ok := schemaConfigMap[schema.Name]
		if !ok {
			config.SchemaConfigs = append(config.SchemaConfigs, initSchemaConfig(schema, formattedUserUID, formattedBranchResourceID, time))
			continue
		}
		tableConfigMap := buildMap(schemaConfig.TableConfigs, func(t *storepb.TableConfig) string {
			return t.Name
		})
		for _, table := range schema.Tables {
			tableConfig, ok := tableConfigMap[table.Name]
			if !ok {
				schemaConfig.TableConfigs = append(schemaConfig.TableConfigs, initTableConfig(table, formattedUserUID, formattedBranchResourceID, time))
			} else {
				tableConfig.Updater = formattedUserUID
				tableConfig.UpdateTime = time
				tableConfig.SourceBranch = formattedBranchResourceID
			}
		}
		viewConfigMap := buildMap(schemaConfig.ViewConfigs, func(v *storepb.ViewConfig) string {
			return v.Name
		})
		for _, view := range schema.Views {
			viewConfig, ok := viewConfigMap[view.Name]
			if !ok {
				schemaConfig.ViewConfigs = append(schemaConfig.ViewConfigs, initViewConfig(view, formattedUserUID, formattedBranchResourceID, time))
			} else {
				viewConfig.Updater = formattedUserUID
				viewConfig.UpdateTime = time
				viewConfig.SourceBranch = formattedBranchResourceID
			}
		}
		functionConfigMap := buildMap(schemaConfig.FunctionConfigs, func(f *storepb.FunctionConfig) string {
			return f.Name
		})
		for _, function := range schema.Functions {
			functionConfig, ok := functionConfigMap[function.Name]
			if !ok {
				schemaConfig.FunctionConfigs = append(schemaConfig.FunctionConfigs, initFunctionConfig(function, formattedUserUID, formattedBranchResourceID, time))
			} else {
				functionConfig.Updater = formattedUserUID
				functionConfig.UpdateTime = time
				functionConfig.SourceBranch = formattedBranchResourceID
			}
		}
		procedureConfigMap := buildMap(schemaConfig.ProcedureConfigs, func(p *storepb.ProcedureConfig) string {
			return p.Name
		})
		for _, procedure := range schema.Procedures {
			procedureConfig, ok := procedureConfigMap[procedure.Name]
			if !ok {
				schemaConfig.ProcedureConfigs = append(schemaConfig.ProcedureConfigs, initProcedureConfig(procedure, formattedUserUID, formattedBranchResourceID, time))
			} else {
				procedureConfig.Updater = formattedUserUID
				procedureConfig.UpdateTime = time
				procedureConfig.SourceBranch = formattedBranchResourceID
			}
		}
	}
}

func trimClassificationIDFromCommentIfNeeded(dbSchema *storepb.DatabaseSchemaMetadata, classificationFromConfig bool) {
	if classificationFromConfig {
		return
	}
	for _, schema := range dbSchema.Schemas {
		for _, table := range schema.Tables {
			if !classificationFromConfig {
				_, table.UserComment = common.GetClassificationAndUserComment(table.UserComment)
			}
			for _, col := range table.Columns {
				if !classificationFromConfig {
					_, col.UserComment = common.GetClassificationAndUserComment(col.UserComment)
				}
			}
		}
	}
}

func updateDatabaseConfigLastModifierForMerge(baseMetadata *storepb.DatabaseSchemaMetadata, headMetadata *storepb.DatabaseSchemaMetadata, baseConfig *storepb.DatabaseConfig, headConfig *storepb.DatabaseConfig) {
	now := timestamppb.Now()
	baseModel := model.NewDatabaseMetadata(baseMetadata)
	baseSchemaConfigMap := buildMap(baseConfig.SchemaConfigs, func(s *storepb.SchemaConfig) string {
		return s.GetName()
	})
	headSchemaConfigMap := buildMap(headConfig.SchemaConfigs, func(s *storepb.SchemaConfig) string {
		return s.GetName()
	})

	var newBaseSchemaConfigs []*storepb.SchemaConfig
	for _, headSchema := range headMetadata.Schemas {
		headSchemaConfig := headSchemaConfigMap[headSchema.Name]
		if headSchemaConfig == nil {
			headSchemaConfig = initSchemaConfig(headSchema, "", "", now)
		}
		baseSchema := baseModel.GetSchema(headSchema.Name)
		if baseSchema == nil {
			//nolint
			newBaseSchemaConfig := proto.Clone(headSchemaConfig).(*storepb.SchemaConfig)
			newBaseSchemaConfigs = append(newBaseSchemaConfigs, newBaseSchemaConfig)
			continue
		}

		// Modified schema, set the last updater as head in base.
		baseSchemaConfig := baseSchemaConfigMap[headSchema.Name]
		// Rebase database, no schema config.
		if baseSchemaConfig == nil {
			baseSchemaConfig = initSchemaConfig(baseSchema.GetProto(), "", "", now)
		}

		// Tables
		baseTableConfigMap := buildMap(baseSchemaConfig.TableConfigs, func(s *storepb.TableConfig) string {
			return s.GetName()
		})
		headTableConfigMap := buildMap(headSchemaConfig.TableConfigs, func(s *storepb.TableConfig) string {
			return s.GetName()
		})
		var newBaseTableConfigs []*storepb.TableConfig
		for _, headTable := range headSchema.Tables {
			headTableConfig := headTableConfigMap[headTable.Name]
			if headTableConfig == nil {
				headTableConfig = initTableConfig(headTable, "", "", now)
			}
			baseTable := baseSchema.GetTable(headTable.Name)
			// New table, reset the source branch and last modifier because we do not remove the config while deleting the object.
			if baseTable == nil {
				//nolint
				newBaseTableConfig := proto.Clone(headTableConfig).(*storepb.TableConfig)
				newBaseTableConfig.SourceBranch = headTableConfig.SourceBranch
				newBaseTableConfig.Updater = headTableConfig.Updater
				newBaseTableConfig.UpdateTime = now
				newBaseTableConfigs = append(newBaseTableConfigs, newBaseTableConfig)
				continue
			}
			// Modified table, set the last updater as head in base.
			baseTableConfig := baseTableConfigMap[headTable.Name]
			// Rebase database, no schema config.
			if baseTableConfig == nil {
				baseTableConfig = initTableConfig(baseTable.GetProto(), "", "", now)
			}
			if !cmp.Equal(headTable, baseTable.GetProto(), protocmp.Transform()) {
				baseTableConfig.SourceBranch = headTableConfig.SourceBranch
				baseTableConfig.Updater = headTableConfig.Updater
				baseTableConfig.UpdateTime = now
			}
		}
		baseSchemaConfig.TableConfigs = append(baseSchemaConfig.TableConfigs, newBaseTableConfigs...)

		// Functions
		baseFunctionConfigMap := buildMap(baseSchemaConfig.FunctionConfigs, func(s *storepb.FunctionConfig) string {
			return s.GetName()
		})
		headFunctionConfigMap := buildMap(headSchemaConfig.FunctionConfigs, func(s *storepb.FunctionConfig) string {
			return s.GetName()
		})
		var newBaseFunctionConfigs []*storepb.FunctionConfig
		for _, headFunction := range headSchema.Functions {
			headFunctionConfig := headFunctionConfigMap[headFunction.Name]
			if headFunctionConfig == nil {
				headFunctionConfig = initFunctionConfig(headFunction, "", "", now)
			}
			baseFunction := baseSchema.GetFunction(headFunction.Name)
			// New function, reset the source branch and last modifier because we do not remove the config while deleting the object.
			if baseFunction == nil {
				//nolint
				newBaseFunctionConfig := proto.Clone(headFunctionConfig).(*storepb.FunctionConfig)
				newBaseFunctionConfig.SourceBranch = headFunctionConfig.SourceBranch
				newBaseFunctionConfig.Updater = headFunctionConfig.Updater
				newBaseFunctionConfig.UpdateTime = now
				newBaseFunctionConfigs = append(newBaseFunctionConfigs, newBaseFunctionConfig)
				continue
			}
			// Modified function, set the last updater as head in base.
			baseFunctionConfig := baseFunctionConfigMap[headFunction.Name]
			// Rebase database, no schema config.
			if baseFunctionConfig == nil {
				baseFunctionConfig = initFunctionConfig(baseFunction.GetProto(), "", "", now)
			}
			if !cmp.Equal(headFunction, baseFunction.GetProto(), protocmp.Transform()) {
				baseFunctionConfig.SourceBranch = headFunctionConfig.SourceBranch
				baseFunctionConfig.Updater = headFunctionConfig.Updater
				baseFunctionConfig.UpdateTime = now
			}
		}
		baseSchemaConfig.FunctionConfigs = append(baseSchemaConfig.FunctionConfigs, newBaseFunctionConfigs...)

		// Views
		baseViewConfigMap := buildMap(baseSchemaConfig.ViewConfigs, func(s *storepb.ViewConfig) string {
			return s.GetName()
		})
		headViewConfigMap := buildMap(headSchemaConfig.ViewConfigs, func(s *storepb.ViewConfig) string {
			return s.GetName()
		})
		var newBaseViewConfigs []*storepb.ViewConfig
		for _, headView := range headSchema.Views {
			headViewConfig := headViewConfigMap[headView.Name]
			if headViewConfig == nil {
				headViewConfig = initViewConfig(headView, "", "", now)
			}
			baseView := baseSchema.GetView(headView.Name)
			// New view, reset the source branch and last modifier because we do not remove the config while deleting the object.
			if baseView == nil {
				//nolint
				newBaseViewConfig := proto.Clone(headViewConfig).(*storepb.ViewConfig)
				newBaseViewConfig.SourceBranch = headViewConfig.SourceBranch
				newBaseViewConfig.Updater = headViewConfig.Updater
				newBaseViewConfig.UpdateTime = now
				newBaseViewConfigs = append(newBaseViewConfigs, newBaseViewConfig)
				continue
			}
			// Modified view, set the last updater as head in base.
			baseViewConfig := baseViewConfigMap[headView.Name]
			// Rebase database, no schema config.
			if baseViewConfig == nil {
				baseViewConfig = initViewConfig(baseView.GetProto(), "", "", now)
			}
			if !cmp.Equal(headView, baseView.GetProto(), protocmp.Transform()) {
				baseViewConfig.SourceBranch = headViewConfig.SourceBranch
				baseViewConfig.Updater = headViewConfig.Updater
				baseViewConfig.UpdateTime = now
			}
		}
		baseSchemaConfig.ViewConfigs = append(baseSchemaConfig.ViewConfigs, newBaseViewConfigs...)

		// Procedures
		baseProcedureConfigMap := buildMap(baseSchemaConfig.ProcedureConfigs, func(s *storepb.ProcedureConfig) string {
			return s.GetName()
		})
		headProcedureConfigMap := buildMap(headSchemaConfig.ProcedureConfigs, func(s *storepb.ProcedureConfig) string {
			return s.GetName()
		})
		var newBaseProcedureConfigs []*storepb.ProcedureConfig
		for _, headProcedure := range headSchema.Procedures {
			headProcedureConfig := headProcedureConfigMap[headProcedure.Name]
			if headProcedureConfig == nil {
				headProcedureConfig = initProcedureConfig(headProcedure, "", "", now)
			}
			baseProcedure := baseSchema.GetProcedure(headProcedure.Name)
			// New procedure, reset the source branch and last modifier because we do not remove the config while deleting the object.
			if baseProcedure == nil {
				//nolint
				newBaseProcedureConfig := proto.Clone(headProcedureConfig).(*storepb.ProcedureConfig)
				newBaseProcedureConfig.SourceBranch = headProcedureConfig.SourceBranch
				newBaseProcedureConfig.Updater = headProcedureConfig.Updater
				newBaseProcedureConfig.UpdateTime = now
				newBaseProcedureConfigs = append(newBaseProcedureConfigs, newBaseProcedureConfig)
				continue
			}
			// Modified procedure, set the last updater as head in base.
			baseProcedureConfig := baseProcedureConfigMap[headProcedure.Name]
			if !cmp.Equal(headProcedure, baseProcedure.GetProto(), protocmp.Transform()) {
				baseProcedureConfig.SourceBranch = headProcedureConfig.SourceBranch
				baseProcedureConfig.Updater = headProcedureConfig.Updater
				baseProcedureConfig.UpdateTime = now
			}
		}
		baseSchemaConfig.ProcedureConfigs = append(baseSchemaConfig.ProcedureConfigs, newBaseProcedureConfigs...)
	}

	baseConfig.SchemaConfigs = append(baseConfig.SchemaConfigs, newBaseSchemaConfigs...)
}

func alignDatabaseConfig(metadata *storepb.DatabaseSchemaMetadata, config *storepb.DatabaseConfig) {
	if config == nil {
		config = &storepb.DatabaseConfig{
			Name: metadata.Name,
		}
	}
	now := timestamppb.Now()
	dbModel := model.NewDatabaseMetadata(metadata)
	schemaConfigMap := buildMap(config.SchemaConfigs, func(s *storepb.SchemaConfig) string {
		return s.Name
	})
	var newSchemaConfigs []*storepb.SchemaConfig
	for _, schemaName := range dbModel.ListSchemaNames() {
		schemaModel := dbModel.GetSchema(schemaName)
		schemaConfig, ok := schemaConfigMap[schemaName]
		if !ok {
			newSchemaConfigs = append(newSchemaConfigs, initSchemaConfig(schemaModel.GetProto(), "", "", now))
			continue
		}

		var newTableConfigs []*storepb.TableConfig
		tableConfigMap := buildMap(schemaConfig.TableConfigs, func(s *storepb.TableConfig) string {
			return s.Name
		})
		for _, tableName := range schemaModel.ListTableNames() {
			tableModel := schemaModel.GetTable(tableName)
			if _, ok := tableConfigMap[tableName]; !ok {
				newTableConfigs = append(newTableConfigs, initTableConfig(tableModel.GetProto(), "", "", now))
				continue
			}
		}
		schemaConfig.TableConfigs = append(schemaConfig.TableConfigs, newTableConfigs...)

		var newViewConfigs []*storepb.ViewConfig
		viewConfigMap := buildMap(schemaConfig.ViewConfigs, func(s *storepb.ViewConfig) string {
			return s.Name
		})
		for _, viewName := range schemaModel.ListViewNames() {
			viewModel := schemaModel.GetView(viewName)
			if _, ok := viewConfigMap[viewName]; !ok {
				newViewConfigs = append(newViewConfigs, initViewConfig(viewModel.GetProto(), "", "", now))
				continue
			}
		}
		schemaConfig.ViewConfigs = append(schemaConfig.ViewConfigs, newViewConfigs...)

		var newProcedureConfigs []*storepb.ProcedureConfig
		procedureConfigMap := buildMap(schemaConfig.ProcedureConfigs, func(s *storepb.ProcedureConfig) string {
			return s.Name
		})
		for _, procedureName := range schemaModel.ListProcedureNames() {
			procedureModel := schemaModel.GetProcedure(procedureName)
			if _, ok := procedureConfigMap[procedureName]; !ok {
				newProcedureConfigs = append(newProcedureConfigs, initProcedureConfig(procedureModel.GetProto(), "", "", now))
				continue
			}
		}
		schemaConfig.ProcedureConfigs = append(schemaConfig.ProcedureConfigs, newProcedureConfigs...)

		var newFunctionConfigs []*storepb.FunctionConfig
		functionConfigMap := buildMap(schemaConfig.FunctionConfigs, func(s *storepb.FunctionConfig) string {
			return s.Name
		})
		for _, functionName := range schemaModel.ListFunctionNames() {
			functionModel := schemaModel.GetFunction(functionName)
			if _, ok := functionConfigMap[functionName]; !ok {
				newFunctionConfigs = append(newFunctionConfigs, initFunctionConfig(functionModel.GetProto(), "", "", now))
				continue
			}
		}
		schemaConfig.FunctionConfigs = append(schemaConfig.FunctionConfigs, newFunctionConfigs...)
	}
	config.SchemaConfigs = append(config.SchemaConfigs, newSchemaConfigs...)
}

func formatViewDef(def string) string {
	return strings.TrimRight(def, "; \n\r\t")
}
