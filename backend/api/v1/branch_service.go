package v1

import (
	"context"
	"fmt"
	"path"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
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
	if err := s.checkBranchPermission(ctx, projectID); err != nil {
		return nil, err
	}
	branch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &projectID, ResourceID: &branchID, LoadFull: true})
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
	if err := s.checkBranchPermission(ctx, projectID); err != nil {
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
	if err := s.checkBranchPermission(ctx, projectID); err != nil {
		return nil, err
	}

	branch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &projectID, ResourceID: &branchID, LoadFull: false})
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
			ProjectID:  projectID,
			ResourceID: branchID,
			Engine:     parentBranch.Engine,
			Base:       parentBranch.Head,
			Head:       parentBranch.Head,
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
		created, err := s.store.CreateBranch(ctx, &store.BranchMessage{
			ProjectID:  projectID,
			ResourceID: branchID,
			Engine:     instance.Engine,
			Base: &storepb.BranchSnapshot{
				Schema:         databaseSchema.GetSchema(),
				Metadata:       databaseSchema.GetMetadata(),
				DatabaseConfig: databaseSchema.GetConfig(),
			},
			Head: &storepb.BranchSnapshot{
				Schema:         databaseSchema.GetSchema(),
				Metadata:       databaseSchema.GetMetadata(),
				DatabaseConfig: databaseSchema.GetConfig(),
			},
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
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid branch name: %v", err))
	}
	if request.UpdateMask == nil || len(request.UpdateMask.Paths) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask is required")
	}
	project, err := s.getProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if err := s.checkBranchPermission(ctx, projectID); err != nil {
		return nil, err
	}

	branch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &projectID, ResourceID: &branchID, LoadFull: true})
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
		updateBranchMessage := &store.UpdateBranchMessage{ProjectID: projectID, ResourceID: branchID, UpdaterID: principalID, UpdateResourceID: &request.Branch.BranchId}
		if err := s.store.UpdateBranch(ctx, updateBranchMessage); err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to update branch, error %v", err))
		}
		// Update the branchID for getting branch in the end.
		branchID = request.Branch.BranchId
	}

	if slices.Contains(request.UpdateMask.Paths, "schema_metadata") {
		sanitizeBranchSchemaMetadata(request.Branch)
		metadata, config := convertV1DatabaseMetadata(request.Branch.GetSchemaMetadata())
		schema, err := getDesignSchema(branch.Engine, string(branch.Head.GetSchema()), metadata)
		if err != nil {
			return nil, err
		}
		headUpdate := &storepb.BranchSnapshot{
			Schema:         []byte(schema),
			Metadata:       metadata,
			DatabaseConfig: config,
		}
		updateBranchMessage := &store.UpdateBranchMessage{ProjectID: projectID, ResourceID: branchID, UpdaterID: principalID}
		updateBranchMessage.Head = headUpdate
		if err := s.store.UpdateBranch(ctx, updateBranchMessage); err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to update branch, error %v", err))
		}
	}

	branch, err = s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &projectID, ResourceID: &branchID, LoadFull: true})
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
	if err := s.checkBranchPermission(ctx, baseProjectID); err != nil {
		return nil, err
	}
	baseBranch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &baseProjectID, ResourceID: &baseBranchID, LoadFull: true})
	if err != nil {
		return nil, err
	}
	if baseBranch == nil {
		return nil, status.Errorf(codes.NotFound, "branch %q not found", baseBranchID)
	}
	if request.Etag != "" && request.Etag != fmt.Sprintf("%d", baseBranch.UpdatedTime.UnixMilli()) {
		return nil, status.Errorf(codes.Aborted, "there is concurrent update to the branch, please refresh and try again.")
	}

	var mergedSchema string
	var mergedMetadata *storepb.DatabaseSchemaMetadata
	// While user specify the merged schema, backend would not parcitipate in the merge process,
	// instead, it would just update the HEAD of the base branch to the merged schema.
	if request.MergedSchema != "" {
		mergedSchema = request.MergedSchema
		metadata, err := transformSchemaStringToDatabaseMetadata(storepb.Engine(baseBranch.Engine), mergedSchema)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert merged schema to metadata, %v", err))
		}
		mergedMetadata = metadata
	} else {
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
		mergedMetadata, err = tryMerge(headBranch.Base.Metadata, headBranch.Head.Metadata, baseBranch.Head.Metadata)
		if err != nil {
			return nil, status.Errorf(codes.Aborted, "cannot auto merge branch due to conflict: %v", err)
		}
		if mergedMetadata == nil {
			return nil, status.Errorf(codes.FailedPrecondition, "failed to merge branch: no change")
		}
		mergedSchema, err = getDesignSchema(storepb.Engine(baseBranch.Engine), string(headBranch.Head.Schema), mergedMetadata)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert merged metadata to schema string, %v", err)
		}
		// TODO(d): handle database config.
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	if err := s.store.UpdateBranch(ctx, &store.UpdateBranchMessage{
		ProjectID:  baseProjectID,
		ResourceID: baseBranchID,
		UpdaterID:  principalID,
		Head: &storepb.BranchSnapshot{
			Schema:   []byte(mergedSchema),
			Metadata: mergedMetadata,
			// TODO(d): handle config.
		}}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed update branch, error %v", err)
	}

	v1Branch, err := s.convertBranchToBranch(ctx, baseProject, baseBranch, v1pb.BranchView_BRANCH_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return v1Branch, nil
}

// RebaseBranch rebases a branch to the target branch.
func (s *BranchService) RebaseBranch(ctx context.Context, request *v1pb.RebaseBranchRequest) (*v1pb.Branch, error) {
	baseProjectID, baseBranchID, err := common.GetProjectAndBranchID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	baseProject, err := s.getProject(ctx, baseProjectID)
	if err != nil {
		return nil, err
	}
	if err := s.checkBranchPermission(ctx, baseProjectID); err != nil {
		return nil, err
	}
	baseBranch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &baseProjectID, ResourceID: &baseBranchID, LoadFull: true})
	if err != nil {
		return nil, err
	}
	if baseBranch == nil {
		return nil, status.Errorf(codes.NotFound, "branch %q not found", baseBranchID)
	}
	if request.Etag != "" && request.Etag != fmt.Sprintf("%d", baseBranch.UpdatedTime.UnixMilli()) {
		return nil, status.Errorf(codes.Aborted, "there is concurrent update to the branch, please refresh and try again.")
	}

	newBaseSchema, err := s.getNewBaseFromRebaseRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	var newHeadSchema string
	var newHeadMetadata *storepb.DatabaseSchemaMetadata
	if request.MergedSchema != "" {
		newHeadSchema = request.MergedSchema
		newHeadMetadata, err = transformSchemaStringToDatabaseMetadata(storepb.Engine(baseBranch.Engine), newBaseSchema)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("failed to convert merged schema to metadata, %v", err))
		}
	} else {
		upstreamMetadata, err := transformSchemaStringToDatabaseMetadata(storepb.Engine(baseBranch.Engine), newBaseSchema)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert upstream schema to metadata, %v", err))
		}
		mergedTarget, err := tryMerge(baseBranch.Base.Metadata, baseBranch.Head.Metadata, upstreamMetadata)
		if err != nil {
			return nil, status.Errorf(codes.Aborted, "cannot auto rebase branch due to conflict: %v", err)
		}
		if mergedTarget == nil {
			return nil, status.Errorf(codes.FailedPrecondition, "failed to rebase branch: no change")
		}
		newHeadSchema, err = getDesignSchema(storepb.Engine(baseBranch.Engine), string(baseBranch.Head.Schema), mergedTarget)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert merged metadata to schema string, %v", err)
		}
		newHeadMetadata, err = transformSchemaStringToDatabaseMetadata(storepb.Engine(baseBranch.Engine), newHeadSchema)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert merged schema to metadata, %v", err)
		}
	}

	newBaseMetadata, err := transformSchemaStringToDatabaseMetadata(storepb.Engine(baseBranch.Engine), newBaseSchema)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert new base schema to metadata, %v", err))
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	if err := s.store.UpdateBranch(ctx, &store.UpdateBranchMessage{
		ProjectID:  baseProjectID,
		ResourceID: baseBranchID,
		UpdaterID:  principalID,
		Base: &storepb.BranchSnapshot{
			Schema:   []byte(newBaseSchema),
			Metadata: newBaseMetadata,
			// TODO(d): handle config.
		},
		Head: &storepb.BranchSnapshot{
			Schema:   []byte(newHeadSchema),
			Metadata: newHeadMetadata,
			// TODO(d): handle config.
		}}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed update branch, error %v", err)
	}

	v1Branch, err := s.convertBranchToBranch(ctx, baseProject, baseBranch, v1pb.BranchView_BRANCH_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return v1Branch, nil
}

func (s *BranchService) getNewBaseFromRebaseRequest(ctx context.Context, request *v1pb.RebaseBranchRequest) (string, error) {
	if request.SourceDatabase != "" {
		instanceID, databaseName, err := common.GetInstanceDatabaseID(request.SourceDatabase)
		if err != nil {
			return "", status.Errorf(codes.InvalidArgument, err.Error())
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return "", status.Errorf(codes.InvalidArgument, err.Error())
		}
		if instance == nil {
			return "", status.Errorf(codes.NotFound, "instance %q not found or had been deleted", instanceID)
		}
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:          &instanceID,
			DatabaseName:        &databaseName,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		})
		if err != nil {
			return "", status.Errorf(codes.Internal, err.Error())
		}
		if database == nil {
			return "", status.Errorf(codes.NotFound, "database %q not found or had been archieve", databaseName)
		}
		databaseSchema, err := s.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return "", status.Errorf(codes.Internal, err.Error())
		}
		if databaseSchema == nil {
			return "", status.Errorf(codes.NotFound, "database schema %q not found", databaseName)
		}
		return string(databaseSchema.GetSchema()), nil
	}

	if request.SourceBranch != "" {
		sourceProjectID, sourceBranchID, err := common.GetProjectAndBranchID(request.SourceBranch)
		if err != nil {
			return "", status.Errorf(codes.InvalidArgument, err.Error())
		}
		sourceBranch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &sourceProjectID, ResourceID: &sourceBranchID, LoadFull: true})
		if err != nil {
			return "", err
		}
		if sourceBranch == nil {
			return "", status.Errorf(codes.NotFound, "branch %q not found", sourceBranchID)
		}
		return string(sourceBranch.Head.Schema), nil
	}

	return "", status.Errorf(codes.InvalidArgument, "either source_database or source_branch should be specified")
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
	branch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &projectID, ResourceID: &branchID, LoadFull: false})
	if err != nil {
		return nil, err
	}
	if branch == nil {
		return nil, status.Errorf(codes.NotFound, "branch %q not found", branchID)
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
	if err := checkDatabaseMetadata(storepb.Engine(request.Engine), storeSourceMetadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid source metadata: %v", err))
	}
	if err := checkDatabaseMetadata(storepb.Engine(request.Engine), storeTargetMetadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid target metadata: %v", err))
	}

	sanitizeCommentForSchemaMetadata(request.SourceMetadata)
	sanitizeCommentForSchemaMetadata(request.TargetMetadata)

	sourceSchema, err := transformDatabaseMetadataToSchemaString(storepb.Engine(request.Engine), storeSourceMetadata)
	if err != nil {
		return nil, err
	}
	targetSchema, err := transformDatabaseMetadataToSchemaString(storepb.Engine(request.Engine), storeTargetMetadata)
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
		if binding.Role == api.Developer || binding.Role == api.Owner {
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

	schemaDesign := &v1pb.Branch{
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
		return schemaDesign, nil
	}

	schemaDesign.Schema = string(branch.Head.Schema)
	schemaDesign.SchemaMetadata = convertStoreDatabaseMetadata(branch.Head.Metadata, branch.Head.DatabaseConfig, v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL, nil /* filter */)
	schemaDesign.BaselineSchema = string(branch.Base.Schema)
	schemaDesign.BaselineSchemaMetadata = convertStoreDatabaseMetadata(branch.Base.Metadata, branch.Base.DatabaseConfig, v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL, nil /* filter */)
	return schemaDesign, nil
}

func sanitizeBranchSchemaMetadata(design *v1pb.Branch) {
	if dbSchema := design.GetBaselineSchemaMetadata(); dbSchema != nil {
		sanitizeCommentForSchemaMetadata(dbSchema)
	}
	if dbSchema := design.GetSchemaMetadata(); dbSchema != nil {
		sanitizeCommentForSchemaMetadata(dbSchema)
	}
}

func sanitizeCommentForSchemaMetadata(dbSchema *v1pb.DatabaseMetadata) {
	for _, schema := range dbSchema.Schemas {
		for _, table := range schema.Tables {
			table.Comment = common.GetCommentFromClassificationAndUserComment(table.Classification, table.UserComment)
			for _, col := range table.Columns {
				col.Comment = common.GetCommentFromClassificationAndUserComment(col.Classification, col.UserComment)
			}
		}
	}
}

func setClassificationAndUserCommentFromComment(dbSchema *storepb.DatabaseSchemaMetadata) {
	for _, schema := range dbSchema.Schemas {
		for _, table := range schema.Tables {
			table.Classification, table.UserComment = common.GetClassificationAndUserComment(table.Comment)
			for _, col := range table.Columns {
				col.Classification, col.UserComment = common.GetClassificationAndUserComment(col.Comment)
			}
		}
	}
}
