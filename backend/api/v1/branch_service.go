package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// NewBranchService implements SchemaDesignServiceServer interface.
type BranchService struct {
	v1pb.UnimplementedSchemaDesignServiceServer
	store          *store.Store
	licenseService enterprise.LicenseService
}

// NewBranchService creates a new SchemaDesignService.
func NewBranchService(store *store.Store, licenseService enterprise.LicenseService) *BranchService {
	return &BranchService{
		store:          store,
		licenseService: licenseService,
	}
}

// GetSchemaDesign gets the schema design.
func (s *BranchService) GetSchemaDesign(ctx context.Context, request *v1pb.GetSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	projectID, branchID, err := common.GetProjectAndBranchID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := s.checkBranchPermission(ctx, projectID); err != nil {
		return nil, err
	}

	project, branch, err := s.getBranch(ctx, projectID, branchID, true /* loadFull */)
	if err != nil {
		return nil, err
	}

	schemaDesign, err := s.convertBranchToSchemaDesign(ctx, project, branch, v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return schemaDesign, nil
}

// ListSchemaDesigns lists schema designs.
func (s *BranchService) ListSchemaDesigns(ctx context.Context, request *v1pb.ListSchemaDesignsRequest) (*v1pb.ListSchemaDesignsResponse, error) {
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
	if request.View == v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_FULL {
		branchFind.LoadFull = true
	}
	branches, err := s.store.ListBranches(ctx, branchFind)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to list sheet: %v", err))
	}

	var schemaDesigns []*v1pb.SchemaDesign
	for _, branch := range branches {
		schemaDesign, err := s.convertBranchToSchemaDesign(ctx, project, branch, request.View)
		if err != nil {
			return nil, err
		}
		schemaDesigns = append(schemaDesigns, schemaDesign)
	}
	response := &v1pb.ListSchemaDesignsResponse{
		SchemaDesigns: schemaDesigns,
	}
	return response, nil
}

// CreateSchemaDesign creates a new schema design.
func (*BranchService) CreateSchemaDesign(_ context.Context, _ *v1pb.CreateSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	return nil, nil
}

// UpdateSchemaDesign updates an existing schema design.
func (*BranchService) UpdateSchemaDesign(_ context.Context, _ *v1pb.UpdateSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	return nil, nil
}

// MergeSchemaDesign merges a personal draft schema design to the target schema design.
func (*BranchService) MergeSchemaDesign(_ context.Context, _ *v1pb.MergeSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	return nil, nil
}

// DeleteSchemaDesign deletes an existing schema design.
func (s *BranchService) DeleteSchemaDesign(ctx context.Context, request *v1pb.DeleteSchemaDesignRequest) (*emptypb.Empty, error) {
	projectID, branchID, err := common.GetProjectAndBranchID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, branch, err := s.getBranch(ctx, projectID, branchID, false /* loadFull */)
	if err != nil {
		return nil, err
	}
	if err := s.checkBranchPermission(ctx, project.ResourceID); err != nil {
		return nil, err
	}

	if err := s.store.DeleteBranch(ctx, project.ResourceID, branch.ResourceID); err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to delete sheet: %v", err))
	}
	return &emptypb.Empty{}, nil
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

func (s *BranchService) getBranch(ctx context.Context, projectID, branchID string, loadFull bool) (*store.ProjectMessage, *store.BranchMessage, error) {
	project, err := s.getProject(ctx, projectID)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, err.Error())
	}
	branch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &projectID, ResourceID: &branchID, LoadFull: loadFull})
	if err != nil {
		return nil, nil, err
	}
	if branch == nil {
		return nil, nil, status.Errorf(codes.NotFound, "branch %q not found", branchID)
	}
	return project, branch, nil
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

func (s *BranchService) convertBranchToSchemaDesign(ctx context.Context, project *store.ProjectMessage, branch *store.BranchMessage, view v1pb.SchemaDesignView) (*v1pb.SchemaDesign, error) {
	creator, err := s.store.GetUserByID(ctx, branch.CreatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get creator: %v", err))
	}
	if creator == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("cannot find the creator: %d", branch.CreatorID))
	}
	updater, err := s.store.GetUserByID(ctx, branch.UpdaterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get updater: %v", err))
	}
	if updater == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("cannot find the updater: %d", branch.UpdaterID))
	}

	var baselineDatabase, baselineBranch string
	schemaDesignType := v1pb.SchemaDesign_MAIN_BRANCH
	if branch.Config != nil {
		baselineDatabase = branch.Config.SourceDatabase
		if branch.Config.SourceBranch != "" {
			schemaDesignType = v1pb.SchemaDesign_PERSONAL_DRAFT
			baselineBranch = branch.Config.SourceBranch
		}
	}

	schemaDesign := &v1pb.SchemaDesign{
		Name:              fmt.Sprintf("%s%s/%s%v", common.ProjectNamePrefix, project.ResourceID, common.BranchPrefix, branch.ResourceID),
		Title:             branch.ResourceID,
		Etag:              fmt.Sprintf("%d", branch.CreatedTime.UnixMilli()),
		BaselineSheetName: baselineBranch,
		Engine:            v1pb.Engine(branch.Engine),
		BaselineDatabase:  baselineDatabase,
		Type:              schemaDesignType,
		Creator:           common.FormatUserEmail(creator.Email),
		Updater:           common.FormatUserEmail(updater.Email),
		CreateTime:        timestamppb.New(branch.CreatedTime),
		UpdateTime:        timestamppb.New(branch.UpdatedTime),
	}

	if view != v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_FULL {
		return schemaDesign, nil
	}

	schemaDesign.Schema = branch.Head.Schema
	schemaDesign.SchemaMetadata = convertDatabaseMetadata(nil /* database */, branch.Head.Metadata, branch.Head.DatabaseConfig, v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL, nil /* filter */)
	schemaDesign.BaselineSchema = branch.Base.Schema
	schemaDesign.BaselineSchemaMetadata = convertDatabaseMetadata(nil /* database */, branch.Base.Metadata, branch.Base.DatabaseConfig, v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL, nil /* filter */)
	return schemaDesign, nil
}
