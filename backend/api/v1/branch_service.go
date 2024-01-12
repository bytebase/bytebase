package v1

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"path"
	"slices"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/epiclabs-io/diff3"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
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
}

// NewBranchService creates a new BranchService.
func NewBranchService(store *store.Store, licenseService enterprise.LicenseService) *BranchService {
	return &BranchService{
		store:          store,
		licenseService: licenseService,
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
	if err := s.checkBranchPermission(ctx, project.ResourceID); err != nil {
		return nil, err
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
	if err := s.checkBranchPermission(ctx, project.ResourceID); err != nil {
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
	if err := s.checkBranchPermission(ctx, project.ResourceID); err != nil {
		return nil, err
	}

	branch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &project.ResourceID, ResourceID: &branchID, LoadFull: false})
	if err != nil {
		return nil, err
	}
	if branch != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "branch %q has already existed", branchID)
	}
	// Branch protection check.
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	if err := s.checkProtectionRules(ctx, project, branchID, principalID); err != nil {
		return nil, err
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
		created, err := s.store.CreateBranch(ctx, &store.BranchMessage{
			ProjectID:  project.ResourceID,
			ResourceID: branchID,
			Engine:     parentBranch.Engine,
			Base:       parentBranch.Head,
			Head:       parentBranch.Head,
			BaseSchema: parentBranch.HeadSchema,
			HeadSchema: parentBranch.HeadSchema,
			Config: &storepb.BranchConfig{
				SourceBranch:   request.Branch.ParentBranch,
				SourceDatabase: parentBranch.Config.GetSourceDatabase(),
			},
			CreatorID: principalID,
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
		filteredBaseSchemaMetadata := filterDatabaseMetadata(databaseSchema.GetMetadata())
		created, err := s.store.CreateBranch(ctx, &store.BranchMessage{
			ProjectID:  project.ResourceID,
			ResourceID: branchID,
			Engine:     instance.Engine,
			Base: &storepb.BranchSnapshot{
				Metadata:       filteredBaseSchemaMetadata,
				DatabaseConfig: databaseSchema.GetConfig(),
			},
			Head: &storepb.BranchSnapshot{
				Metadata:       filteredBaseSchemaMetadata,
				DatabaseConfig: databaseSchema.GetConfig(),
			},
			BaseSchema: databaseSchema.GetSchema(),
			HeadSchema: databaseSchema.GetSchema(),
			Config: &storepb.BranchConfig{
				SourceDatabase: request.Branch.BaselineDatabase,
			},
			CreatorID: principalID,
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
	if err := s.checkBranchPermission(ctx, project.ResourceID); err != nil {
		return nil, err
	}

	branch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &project.ResourceID, ResourceID: &branchID, LoadFull: true})
	if err != nil {
		return nil, err
	}
	if branch == nil {
		return nil, status.Errorf(codes.NotFound, "branch %q not found", branchID)
	}
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	if request.Etag != "" && request.Etag != fmt.Sprintf("%d", branch.UpdatedTime.UnixMilli()) {
		return nil, status.Errorf(codes.Aborted, "there is concurrent update to the branch, please refresh and try again.")
	}

	// Handle branch ID update.
	if slices.Contains(request.UpdateMask.Paths, "branch_id") {
		if len(request.UpdateMask.Paths) > 1 {
			return nil, status.Errorf(codes.InvalidArgument, "cannot update branch_id with other types of updates")
		}
		updateBranchMessage := &store.UpdateBranchMessage{ProjectID: project.ResourceID, ResourceID: branchID, UpdaterID: principalID, UpdateResourceID: &request.Branch.BranchId}
		if err := s.store.UpdateBranch(ctx, updateBranchMessage); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update branch, error %v", err)
		}
		// Update the branchID for getting branch in the end.
		branchID = request.Branch.BranchId
	}

	if slices.Contains(request.UpdateMask.Paths, "schema_metadata") {
		metadata, config := convertV1DatabaseMetadata(request.Branch.GetSchemaMetadata())
		sanitizeCommentForSchemaMetadata(metadata)
		filteredMetadata := filterDatabaseMetadata(metadata)

		schema, err := schema.GetDesignSchema(branch.Engine, string(branch.BaseSchema), metadata)
		if err != nil {
			return nil, err
		}
		schemaBytes := []byte(schema)
		headUpdate := &storepb.BranchSnapshot{
			Metadata:       filteredMetadata,
			DatabaseConfig: config,
		}
		updateBranchMessage := &store.UpdateBranchMessage{ProjectID: project.ResourceID, ResourceID: branchID, UpdaterID: principalID}
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

// MergeBranch merges a personal draft branch to the target branch.
func (s *BranchService) MergeBranch(ctx context.Context, request *v1pb.MergeBranchRequest) (*v1pb.Branch, error) {
	baseProjectID, baseBranchID, err := common.GetProjectAndBranchID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	baseProject, err := s.getProject(ctx, baseProjectID)
	if err != nil {
		return nil, err
	}
	if err := s.checkBranchPermission(ctx, baseProject.ResourceID); err != nil {
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

	headProjectID, headBranchID, err := common.GetProjectAndBranchID(request.HeadBranch)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	_, err = s.getProject(ctx, headProjectID)
	if err != nil {
		return nil, err
	}
	if err := s.checkBranchPermission(ctx, headProjectID); err != nil {
		return nil, err
	}
	headBranch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &headProjectID, ResourceID: &headBranchID, LoadFull: true})
	if err != nil {
		return nil, err
	}
	if headBranch == nil {
		return nil, status.Errorf(codes.NotFound, "branch %q not found", headBranchID)
	}

	// Restrict merging only when the head branch is not updated.
	// Maybe we can support auto-merging in the future.
	mergedMetadata, err := tryMerge(headBranch.Base.Metadata, headBranch.Head.Metadata, baseBranch.Head.Metadata)
	if err != nil {
		slog.Info("cannot merge branches", log.BBError(err))
		return nil, status.Errorf(codes.Aborted, "cannot merge branches without conflict, error: %v", err)
	}
	if mergedMetadata == nil {
		return nil, status.Errorf(codes.Internal, "merged metadata should not be nil if there is no error while merging (%+v, %+v, %+v)", headBranch.Base.Metadata, headBranch.Head.Metadata, baseBranch.Head.Metadata)
	}
	mergedSchema, err := schema.GetDesignSchema(storepb.Engine(baseBranch.Engine), string(headBranch.HeadSchema), mergedMetadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert merged metadata to schema string, %v", err)
	}
	// XXX(zp): We only try to merge the schema config while the schema could be merged successfully. Otherwise, users manually merge the
	// metadata in the frontend, and config would be ignored.
	mergedConfig := utils.MergeDatabaseConfig(headBranch.Base.GetDatabaseConfig(), headBranch.Head.GetDatabaseConfig(), baseBranch.Head.GetDatabaseConfig())

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	mergedSchemaBytes := []byte(mergedSchema)

	filteredMergedMetadata := filterDatabaseMetadata(mergedMetadata)
	baseBranchNewHead := &storepb.BranchSnapshot{
		Metadata:       filteredMergedMetadata,
		DatabaseConfig: mergedConfig,
	}
	baseBranchNewHeadSchema := mergedSchemaBytes

	if request.ValidateOnly {
		baseBranch.Head = baseBranchNewHead
		baseBranch.HeadSchema = baseBranchNewHeadSchema
	} else {
		if err := s.store.UpdateBranch(ctx, &store.UpdateBranchMessage{
			ProjectID:  baseProject.ResourceID,
			ResourceID: baseBranchID,
			UpdaterID:  principalID,
			Head:       baseBranchNewHead,
			HeadSchema: &baseBranchNewHeadSchema,
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "failed update branch, error %v", err)
		}
		baseBranch, err = s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &baseProject.ResourceID, ResourceID: &baseBranchID})
		if err != nil {
			return nil, err
		}
	}

	v1Branch, err := s.convertBranchToBranch(ctx, baseProject, baseBranch, v1pb.BranchView_BRANCH_VIEW_FULL)
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
	if err := s.checkBranchPermission(ctx, baseProject.ResourceID); err != nil {
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

	newBaseMetadata, newBaseSchema, newBaseConfig, err := s.getNewBaseFromRebaseRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	filteredNewBaseMetadata := filterDatabaseMetadata(newBaseMetadata)

	var newHeadSchema string
	var newHeadMetadata *storepb.DatabaseSchemaMetadata
	var newHeadConfig *storepb.DatabaseConfig
	if request.MergedSchema != "" {
		newHeadSchema = request.MergedSchema
		newHeadMetadata, err = schema.ParseToMetadata(storepb.Engine(baseBranch.Engine), newHeadSchema)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to convert merged schema to metadata, %v", err)
		}
		newHeadConfig = baseBranch.Head.GetDatabaseConfig()
	} else {
		newHeadMetadata, err = tryMerge(baseBranch.Base.Metadata, baseBranch.Head.Metadata, filteredNewBaseMetadata)
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
			conflictSchemaString := string(sb)
			return &v1pb.RebaseBranchResponse{Result: &v1pb.RebaseBranchResponse_ConflictSchema{ConflictSchema: conflictSchemaString}}, nil
		}
		if newHeadMetadata == nil {
			return nil, status.Errorf(codes.Internal, "merged metadata should not be nil if there is no error while merging (%+v, %+v, %+v)", baseBranch.Base.Metadata, baseBranch.Head.Metadata, filteredNewBaseMetadata)
		}
		// XXX(zp): We only try to merge the schema config while the schema could be merged successfully. Otherwise, users manually merge the
		// metadata in the frontend, and config would be ignored.
		newHeadConfig = utils.MergeDatabaseConfig(baseBranch.Base.GetDatabaseConfig(), baseBranch.Head.GetDatabaseConfig(), newBaseConfig)

		newHeadSchema, err = schema.GetDesignSchema(storepb.Engine(baseBranch.Engine), string(baseBranch.HeadSchema), newHeadMetadata)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert merged metadata to schema string, %v", err)
		}
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	newBaseSchemaBytes := []byte(newBaseSchema)
	newHeadSchemaBytes := []byte(newHeadSchema)
	filteredNewHeadMetadata := filterDatabaseMetadata(newHeadMetadata)
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
			UpdaterID:  principalID,
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

func (s *BranchService) getNewBaseFromRebaseRequest(ctx context.Context, request *v1pb.RebaseBranchRequest) (*storepb.DatabaseSchemaMetadata, string, *storepb.DatabaseConfig, error) {
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
		return databaseSchema.GetMetadata(), string(databaseSchema.GetSchema()), databaseSchema.GetConfig(), nil
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
	if err := s.checkBranchPermission(ctx, project.ResourceID); err != nil {
		return nil, err
	}
	branch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &project.ResourceID, ResourceID: &branchID, LoadFull: false})
	if err != nil {
		return nil, err
	}
	if branch == nil {
		return nil, status.Errorf(codes.NotFound, "branch %q not found", branchID)
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

func (*BranchService) DiffMetadata(_ context.Context, request *v1pb.DiffMetadataRequest) (*v1pb.DiffMetadataResponse, error) {
	switch request.Engine {
	case v1pb.Engine_MYSQL, v1pb.Engine_POSTGRES, v1pb.Engine_TIDB:
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported engine: %v", request.Engine)
	}
	if request.SourceMetadata == nil || request.TargetMetadata == nil {
		return nil, status.Errorf(codes.InvalidArgument, "source_metadata and target_metadata are required")
	}
	storeSourceMetadata, _ := convertV1DatabaseMetadata(request.SourceMetadata)
	storeTargetMetadata, _ := convertV1DatabaseMetadata(request.TargetMetadata)
	if err := checkDatabaseMetadata(storepb.Engine(request.Engine), storeTargetMetadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target metadata: %v", err)
	}
	sanitizeCommentForSchemaMetadata(storeTargetMetadata)

	storeSourceMetadata, storeTargetMetadata = trimDatabaseMetadata(storeSourceMetadata, storeTargetMetadata)
	if err := checkDatabaseMetadataColumnType(storepb.Engine(request.Engine), storeTargetMetadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target metadata: %v", err)
	}

	sourceSchema, err := schema.GetDesignSchema(storepb.Engine(request.Engine), "" /* baseline*/, storeSourceMetadata)
	if err != nil {
		return nil, err
	}
	targetSchema, err := schema.GetDesignSchema(storepb.Engine(request.Engine), "" /* baseline*/, storeTargetMetadata)
	if err != nil {
		return nil, err
	}

	diff, err := base.SchemaDiff(convertEngine(request.Engine), sourceSchema, targetSchema, false /* ignoreCaseSensitive */)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compute diff between source and target schemas, error: %v", err)
	}

	return &v1pb.DiffMetadataResponse{
		Diff: diff,
	}, nil
}

func (s *BranchService) getProject(ctx context.Context, projectID string) (*store.ProjectMessage, error) {
	var project *store.ProjectMessage
	projectUID, isNumber := isNumber(projectID)
	if isNumber {
		v, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			UID: &projectUID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		project = v
	} else {
		v, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID: &projectID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		project = v
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", projectID)
	}
	return project, nil
}

func (s *BranchService) checkBranchPermission(ctx context.Context, projectID string) error {
	role, ok := ctx.Value(common.RoleContextKey).(api.Role)
	if !ok {
		return status.Errorf(codes.Internal, "role not found")
	}
	if isOwnerOrDBA(role) {
		return nil
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return status.Errorf(codes.Internal, "principal ID not found")
	}
	policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &projectID})
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	for _, binding := range policy.Bindings {
		if binding.Role == api.ProjectDeveloper || binding.Role == api.ProjectOwner {
			for _, member := range binding.Members {
				if member.ID == principalID || member.Email == api.AllUsers {
					return nil
				}
			}
		}
	}
	return status.Errorf(codes.PermissionDenied, "permission denied")
}

func (s *BranchService) checkProtectionRules(ctx context.Context, project *store.ProjectMessage, branchID string, currentPrincipalID int) error {
	if project.Setting == nil {
		return nil
	}
	user, err := s.store.GetUserByID(ctx, currentPrincipalID)
	if err != nil {
		return err
	}
	policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return err
	}
	// Skip protection check for workspace owner and DBA.
	if isOwnerOrDBA(user.Role) {
		return nil
	}

	for _, rule := range project.Setting.ProtectionRules {
		if rule.Target != storepb.ProtectionRule_BRANCH {
			continue
		}
		ok, err := path.Match(rule.NameFilter, branchID)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		pass := false
		for _, binding := range policy.Bindings {
			matchUser := false
			for _, member := range binding.Members {
				if member.Email == user.Email {
					matchUser = true
					break
				}
			}
			if matchUser {
				for _, role := range rule.CreateAllowedRoles {
					// Convert role format.
					if role == convertToProjectRole(binding.Role) {
						pass = true
						break
					}
				}
			}
			if pass {
				break
			}
		}
		if !pass {
			return status.Errorf(codes.InvalidArgument, "not allowed to create branch by project protection rules")
		}
	}
	return nil
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
	v1Branch.SchemaMetadata = convertStoreDatabaseMetadata(branch.Head.Metadata, branch.Head.DatabaseConfig, v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL, nil /* filter */)
	v1Branch.BaselineSchema = string(branch.BaseSchema)
	v1Branch.BaselineSchemaMetadata = convertStoreDatabaseMetadata(branch.Base.Metadata, branch.Base.DatabaseConfig, v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL, nil /* filter */)
	return v1Branch, nil
}

func sanitizeCommentForSchemaMetadata(dbSchema *storepb.DatabaseSchemaMetadata) {
	for _, schema := range dbSchema.Schemas {
		for _, table := range schema.Tables {
			table.Comment = common.GetCommentFromClassificationAndUserComment(table.Classification, table.UserComment)
			for _, col := range table.Columns {
				col.Comment = common.GetCommentFromClassificationAndUserComment(col.Classification, col.UserComment)
			}
		}
	}
}

// filterDatabaseMetadata filter out the objects/attributes we do not support.
// TODO: While supporting new objects/attributes, we should update this function.
func filterDatabaseMetadata(metadata *storepb.DatabaseSchemaMetadata) *storepb.DatabaseSchemaMetadata {
	filteredDatabase := &storepb.DatabaseSchemaMetadata{
		Name: metadata.Name,
	}
	for _, schema := range metadata.Schemas {
		filteredSchema := &storepb.SchemaMetadata{
			Name: schema.Name,
		}
		for _, table := range schema.Tables {
			filteredTable := &storepb.TableMetadata{
				Name:           table.Name,
				Classification: table.Classification,
				Comment:        table.Comment,
				UserComment:    table.UserComment,
				Collation:      table.Collation,
				Engine:         table.Engine,
			}
			for _, column := range table.Columns {
				filteredColumn := &storepb.ColumnMetadata{
					Name:           column.Name,
					Comment:        column.Comment,
					UserComment:    column.UserComment,
					Classification: column.Classification,
					Type:           column.Type,
					DefaultValue:   column.DefaultValue,
					Nullable:       column.Nullable,
					Position:       column.Position,
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
			filteredSchema.Tables = append(filteredSchema.Tables, filteredTable)
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

			if !equalTable(table, tt.GetProto()) {
				trimSchema.Tables = append(trimSchema.Tables, table)
				continue
			}
		}
		if len(trimSchema.Tables) > 0 {
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

			if !equalTable(table, tt.GetProto()) {
				trimSchema.Tables = append(trimSchema.Tables, table)
				continue
			}
		}
		if len(trimSchema.Tables) > 0 {
			t.Schemas = append(t.Schemas, trimSchema)
		}
	}

	return s, t
}

func equalTable(s, t *storepb.TableMetadata) bool {
	if len(s.GetColumns()) != len(t.GetColumns()) {
		return false
	}
	if len(s.Indexes) != len(t.Indexes) {
		return false
	}
	if s.GetComment() != t.GetComment() {
		return false
	}
	if s.GetUserComment() != t.GetUserComment() {
		return false
	}
	if s.GetClassification() != t.GetClassification() {
		return false
	}
	for i := 0; i < len(s.GetColumns()); i++ {
		sc, tc := s.GetColumns()[i], t.GetColumns()[i]
		if sc.Name != tc.Name {
			return false
		}
		if sc.Comment != tc.Comment {
			return false
		}
		if sc.UserComment != tc.UserComment {
			return false
		}
		if sc.Classification != tc.Classification {
			return false
		}
		if sc.Type != tc.Type {
			return false
		}
		if sc.Nullable != tc.Nullable {
			return false
		}
		if sc.GetDefault().GetValue() != tc.GetDefault().GetValue() {
			return false
		}
		if sc.GetDefaultExpression() != tc.GetDefaultExpression() {
			return false
		}
		if sc.GetDefaultNull() != tc.GetDefaultNull() {
			return false
		}
	}
	for i := 0; i < len(s.GetIndexes()); i++ {
		si, ti := s.GetIndexes()[i], t.GetIndexes()[i]
		if si.GetName() != ti.GetName() {
			return false
		}
		if si.GetDefinition() != ti.GetDefinition() {
			return false
		}
		if si.GetPrimary() != ti.GetPrimary() {
			return false
		}
		if si.GetUnique() != ti.GetUnique() {
			return false
		}
		if si.GetType() != ti.GetType() {
			return false
		}
		if si.GetVisible() != ti.GetVisible() {
			return false
		}
		if si.GetComment() != ti.GetComment() {
			return false
		}
		if !slices.Equal(si.GetExpressions(), ti.GetExpressions()) {
			return false
		}
	}
	return true
}
