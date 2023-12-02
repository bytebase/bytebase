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
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
		return nil, status.Errorf(codes.Internal, "failed to list branches, error %v", err)
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
func (s *BranchService) CreateSchemaDesign(ctx context.Context, request *v1pb.CreateSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
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

	// TODO(d): move to resource_id in the request.
	projectID, branchID, err := common.GetProjectAndBranchID(request.SchemaDesign.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
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
	if request.SchemaDesign.ParentBranch != "" {
		parentProjectID, parentBranchID, err := common.GetProjectAndBranchID(request.SchemaDesign.ParentBranch)
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
				SourceBranch:   request.SchemaDesign.ParentBranch,
				SourceDatabase: parentBranch.Config.GetSourceDatabase(),
			},
			CreatorID: principalID,
		})
		if err != nil {
			return nil, err
		}
		createdBranch = created
	} else if request.SchemaDesign.BaselineDatabase != "" {
		instanceID, databaseName, err := common.GetInstanceDatabaseID(request.SchemaDesign.BaselineDatabase)
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
				SourceDatabase: request.SchemaDesign.BaselineDatabase,
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

	schemaDesign, err := s.convertBranchToSchemaDesign(ctx, project, createdBranch, v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return schemaDesign, nil
}

// UpdateSchemaDesign updates an existing schema design.
func (s *BranchService) UpdateSchemaDesign(ctx context.Context, request *v1pb.UpdateSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	projectID, branchID, err := common.GetProjectAndBranchID(request.SchemaDesign.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid schema design name: %v", err))
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
	schemaDesign := request.SchemaDesign

	// TODO(d): handle etag.
	// TODO(d): support branch id update.
	// if slices.Contains(request.UpdateMask.Paths, "title") {
	// }

	headUpdate := branch.Head
	hasHeadUpdate := false
	// TODO(d): this section needs some clarifications for merging branches.
	if slices.Contains(request.UpdateMask.Paths, "schema") && slices.Contains(request.UpdateMask.Paths, "metadata") {
		headUpdate.Schema = []byte(schemaDesign.Schema)
		sanitizeSchemaDesignSchemaMetadata(schemaDesign)
		// TODO(d): update database metadata and config.
		hasHeadUpdate = true
	} else if slices.Contains(request.UpdateMask.Paths, "schema") {
		headUpdate.Schema = []byte(schemaDesign.Schema)
		hasHeadUpdate = true
		// TODO(d): convert schema to metadata.
		// Try to transform the schema string to database metadata to make sure it's valid.
		// if _, err := transformSchemaStringToDatabaseMetadata(schemaDesign.Engine, *sheetUpdate.Statement); err != nil {
		// 	return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to transform schema string to database metadata: %v", err))
		// }
	} else if slices.Contains(request.UpdateMask.Paths, "metadata") {
		sanitizeSchemaDesignSchemaMetadata(schemaDesign)
		// schema, err := getDesignSchema(schemaDesign.Engine, schemaDesign.BaselineSchema, schemaDesign.SchemaMetadata)
		// if err != nil {
		// 	return nil, err
		// }
		// TODO(d): convert metadata to schema.
		hasHeadUpdate = true
	}
	// TODO(d): handle database config as well.

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	updateBranchMessage := &store.UpdateBranchMessage{ProjectID: projectID, ResourceID: branchID, UpdaterID: principalID}
	if hasHeadUpdate {
		updateBranchMessage.Head = headUpdate
	}
	if err := s.store.UpdateBranch(ctx, updateBranchMessage); err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to update branch, error %v", err))
	}

	branch, err = s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &projectID, ResourceID: &branchID, LoadFull: true})
	if err != nil {
		return nil, err
	}
	schemaDesign, err = s.convertBranchToSchemaDesign(ctx, project, branch, v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return schemaDesign, nil
}

// MergeSchemaDesign merges a personal draft schema design to the target schema design.
func (s *BranchService) MergeSchemaDesign(ctx context.Context, request *v1pb.MergeSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	projectID, branchID, err := common.GetProjectAndBranchID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if _, err := s.getProject(ctx, projectID); err != nil {
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

	targetProjectID, targetBranchID, err := common.GetProjectAndBranchID(request.TargetName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	targetProject, err := s.getProject(ctx, targetProjectID)
	if err != nil {
		return nil, err
	}
	if err := s.checkBranchPermission(ctx, targetProjectID); err != nil {
		return nil, err
	}
	targetBranch, err := s.store.GetBranch(ctx, &store.FindBranchMessage{ProjectID: &targetProjectID, ResourceID: &targetBranchID, LoadFull: true})
	if err != nil {
		return nil, err
	}
	if targetBranch == nil {
		return nil, status.Errorf(codes.NotFound, "branch %q not found", targetBranchID)
	}

	// Restrict merging only when the target schema design is not updated.
	// Maybe we can support auto-merging in the future.
	baseMetadata := convertDatabaseMetadata(nil, branch.Base.Metadata, nil, v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL, nil)
	headMetadata := convertDatabaseMetadata(nil, branch.Head.Metadata, nil, v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL, nil)
	targetHeadMetadata := convertDatabaseMetadata(nil, targetBranch.Head.Metadata, nil, v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL, nil)
	mergedTarget, err := tryMerge(baseMetadata, headMetadata, targetHeadMetadata)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, fmt.Sprintf("failed to merge schema design: %v", err))
	}
	if mergedTarget == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "failed to merge schema design: no change")
	}
	mergedTargetSchema, err := getDesignSchema(v1pb.Engine(branch.Engine), string(targetBranch.Head.Schema), mergedTarget)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert merged metadata to schema string, %v", err)
	}
	// TODO(d): handle database config.

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	if err := s.store.UpdateBranch(ctx, &store.UpdateBranchMessage{
		ProjectID:  targetProjectID,
		ResourceID: targetBranchID,
		UpdaterID:  principalID,
		Head: &storepb.BranchSnapshot{
			Schema: []byte(mergedTargetSchema),
			// TODO(d): Metadata: mergedTarget,
			// TODO(d): handle config.
		}}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed update branch, error %v", err)
	}

	targetSchemaDesign, err := s.convertBranchToSchemaDesign(ctx, targetProject, targetBranch, v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return targetSchemaDesign, nil
}

// DeleteSchemaDesign deletes an existing schema design.
func (s *BranchService) DeleteSchemaDesign(ctx context.Context, request *v1pb.DeleteSchemaDesignRequest) (*emptypb.Empty, error) {
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

func (s *BranchService) convertBranchToSchemaDesign(ctx context.Context, project *store.ProjectMessage, branch *store.BranchMessage, view v1pb.SchemaDesignView) (*v1pb.SchemaDesign, error) {
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
	schemaDesignType := v1pb.SchemaDesign_MAIN_BRANCH
	if branch.Config != nil {
		baselineDatabase = branch.Config.SourceDatabase
		if branch.Config.SourceBranch != "" {
			schemaDesignType = v1pb.SchemaDesign_PERSONAL_DRAFT
			baselineBranch = branch.Config.SourceBranch
		}
	}

	schemaDesign := &v1pb.SchemaDesign{
		Name:             fmt.Sprintf("%s%s/%s%v", common.ProjectNamePrefix, project.ResourceID, common.BranchPrefix, branch.ResourceID),
		Title:            branch.ResourceID,
		Etag:             fmt.Sprintf("%d", branch.CreatedTime.UnixMilli()),
		ParentBranch:     baselineBranch,
		Engine:           v1pb.Engine(branch.Engine),
		BaselineDatabase: baselineDatabase,
		Type:             schemaDesignType,
		Creator:          common.FormatUserEmail(creator.Email),
		Updater:          common.FormatUserEmail(updater.Email),
		CreateTime:       timestamppb.New(branch.CreatedTime),
		UpdateTime:       timestamppb.New(branch.UpdatedTime),
	}

	if view != v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_FULL {
		return schemaDesign, nil
	}

	schemaDesign.Schema = string(branch.Head.Schema)
	schemaDesign.SchemaMetadata = convertDatabaseMetadata(nil /* database */, branch.Head.Metadata, branch.Head.DatabaseConfig, v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL, nil /* filter */)
	schemaDesign.BaselineSchema = string(branch.Base.Schema)
	schemaDesign.BaselineSchemaMetadata = convertDatabaseMetadata(nil /* database */, branch.Base.Metadata, branch.Base.DatabaseConfig, v1pb.DatabaseMetadataView_DATABASE_METADATA_VIEW_FULL, nil /* filter */)
	return schemaDesign, nil
}
