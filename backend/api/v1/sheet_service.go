package v1

import (
	"context"
	"fmt"
	"slices"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// SheetService implements the sheet service.
type SheetService struct {
	v1pb.UnimplementedSheetServiceServer
	store          *store.Store
	licenseService enterprise.LicenseService
	iamManager     *iam.Manager
	profile        *config.Profile
}

// NewSheetService creates a new SheetService.
func NewSheetService(store *store.Store, licenseService enterprise.LicenseService, iamManager *iam.Manager, profile *config.Profile) *SheetService {
	return &SheetService{
		store:          store,
		licenseService: licenseService,
		iamManager:     iamManager,
		profile:        profile,
	}
}

// CreateSheet creates a new sheet.
func (s *SheetService) CreateSheet(ctx context.Context, request *v1pb.CreateSheetRequest) (*v1pb.Sheet, error) {
	if request.Sheet == nil {
		return nil, status.Errorf(codes.InvalidArgument, "sheet must be set")
	}
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}

	projectResourceID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get project with resource id %q, err: %s", projectResourceID, err.Error()))
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q not found", projectResourceID))
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q had deleted", projectResourceID))
	}

	var databaseUID *int
	if request.Sheet.Database != "" {
		instanceResourceID, databaseName, err := common.GetInstanceDatabaseID(request.Sheet.Database)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
			ResourceID: &instanceResourceID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get instance with resource id %q, err: %s", instanceResourceID, err.Error()))
		}
		if instance == nil {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("instance with resource id %q not found", instanceResourceID))
		}

		find := &store.FindDatabaseMessage{
			ProjectID:  &projectResourceID,
			InstanceID: &instanceResourceID,
		}
		// It's chaos. We return /instance/{resource id}/databases/{uid} database in find sheet request,
		// but the frontend use both /instance/{resource id}/databases/{uid} and /instance/{resource id}/databases/{name}, sometimes the name will convert to int id incorrectly.
		// For database v1 api, we should only use the /instance/{resource id}/databases/{name}
		// We need to remove legacy code after the migration.
		dbUID, isNumber := isNumber(databaseName)
		if instanceResourceID == "-" && isNumber {
			find.UID = &dbUID
		} else {
			find.DatabaseName = &databaseName
			find.IgnoreCaseSensitive = store.IgnoreDatabaseAndTableCaseSensitive(instance)
		}

		database, err := s.store.GetDatabaseV2(ctx, find)
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get database with name %q, err: %s", databaseName, err.Error()))
		}
		if database == nil {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("database with name %q not found in project %q instance %q", databaseName, projectResourceID, instanceResourceID))
		}
		databaseUID = &database.UID
	}
	storeSheetCreate, err := convertToStoreSheetMessage(project.UID, databaseUID, principalID, request.Sheet)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("failed to convert sheet: %v", err))
	}
	sheet, err := s.store.CreateSheet(ctx, storeSheetCreate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to create sheet: %v", err))
	}
	v1pbSheet, err := s.convertToAPISheetMessage(ctx, sheet)
	if err != nil {
		return nil, err
	}
	return v1pbSheet, nil
}

// GetSheet returns the requested sheet, cutoff the content if the content is too long and the `raw` flag in request is false.
func (s *SheetService) GetSheet(ctx context.Context, request *v1pb.GetSheetRequest) (*v1pb.Sheet, error) {
	projectResourceID, sheetUID, err := common.GetProjectResourceIDSheetUID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if sheetUID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid sheet id %d, must be positive integer", sheetUID))
	}

	find := &store.FindSheetMessage{
		UID:      &sheetUID,
		LoadFull: request.Raw,
	}

	// this allows get the sheet only by the id: /projects/-/sheets/{sheet uid}.
	// so that we can easily get the sheet from the issue.
	// we can remove this after migrate the issue to v1 API.
	if projectResourceID != "-" {
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID: &projectResourceID,
		})
		if err != nil {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %s not found", projectResourceID))
		}
		if project == nil {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %s not found", projectResourceID))
		}
		if project.Deleted {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q had deleted", projectResourceID))
		}
		find.ProjectUID = &project.UID
	}

	sheet, err := s.findSheet(ctx, find)
	if err != nil {
		return nil, err
	}

	// For issue sheets, check the bb.issues.get permission.
	if s.profile.DevelopmentIAM && sheet.Source == store.SheetFromBytebaseArtifact {
		user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
		if !ok {
			return nil, status.Errorf(codes.Internal, "user not found")
		}
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			UID: &sheet.ProjectUID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get project %q", sheet.ProjectUID)
		}
		if project == nil {
			return nil, status.Errorf(codes.NotFound, "project %q not found", sheet.ProjectUID)
		}
		ok, err = s.iamManager.CheckPermission(ctx, iam.PermissionIssuesGet, user, project.ResourceID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check permission: %v", err)
		}
		if !ok {
			return nil, status.Errorf(codes.PermissionDenied, "permission denied to get sheet")
		}
	} else {
		canAccess, err := s.canReadSheet(ctx, sheet)
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to check access with error: %v", err))
		}
		if !canAccess {
			return nil, status.Errorf(codes.PermissionDenied, "cannot access sheet %s", sheet.Title)
		}
	}

	v1pbSheet, err := s.convertToAPISheetMessage(ctx, sheet)
	if err != nil {
		return nil, err
	}
	return v1pbSheet, nil
}

// UpdateSheet updates a sheet.
func (s *SheetService) UpdateSheet(ctx context.Context, request *v1pb.UpdateSheetRequest) (*v1pb.Sheet, error) {
	if request.Sheet == nil {
		return nil, status.Errorf(codes.InvalidArgument, "sheet cannot be empty")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update mask cannot be empty")
	}
	if request.Sheet.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "sheet name cannot be empty")
	}

	projectResourceID, sheetUID, err := common.GetProjectResourceIDSheetUID(request.Sheet.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if sheetUID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid sheet id %d, must be positive integer", sheetUID))
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %s not found", projectResourceID))
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %s not found", projectResourceID))
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q had deleted", projectResourceID))
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{
		UID:        &sheetUID,
		ProjectUID: &project.UID,
	}, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get sheet: %v", err))
	}
	if sheet == nil {
		return nil, status.Errorf(codes.NotFound, "sheet %q not found", request.Sheet.Name)
	}
	canAccess, err := s.canWriteSheet(ctx, sheet)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to check access with error: %v", err))
	}
	if !canAccess {
		return nil, status.Errorf(codes.PermissionDenied, "cannot write sheet %s", sheet.Title)
	}

	sheetPatch := &store.PatchSheetMessage{
		UID:       sheet.UID,
		UpdaterID: principalID,
	}

	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			sheetPatch.Title = &request.Sheet.Title
		case "content":
			statement := string(request.Sheet.Content)
			sheetPatch.Statement = &statement
		case "visibility":
			visibility, err := convertToStoreSheetVisibility(request.Sheet.Visibility)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid visibility %q", request.Sheet.Visibility))
			}
			stringVisibility := string(visibility)
			sheetPatch.Visibility = &stringVisibility
		default:
			return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid update mask path %q", path))
		}
	}
	storeSheet, err := s.store.PatchSheet(ctx, sheetPatch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to update sheet: %v", err))
	}
	v1pbSheet, err := s.convertToAPISheetMessage(ctx, storeSheet)
	if err != nil {
		return nil, err
	}

	return v1pbSheet, nil
}

// DeleteSheet deletes a sheet.
func (s *SheetService) DeleteSheet(ctx context.Context, request *v1pb.DeleteSheetRequest) (*emptypb.Empty, error) {
	projectResourceID, sheetUID, err := common.GetProjectResourceIDSheetUID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get project with resource id %q, err: %s", projectResourceID, err.Error()))
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q not found", projectResourceID))
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q had deleted", projectResourceID))
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{
		UID:        &sheetUID,
		ProjectUID: &project.UID,
	}, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get sheet: %v", err))
	}
	if sheet == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("sheet with id %d not found", sheetUID))
	}
	canAccess, err := s.canWriteSheet(ctx, sheet)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to check access with error: %v", err))
	}
	if !canAccess {
		return nil, status.Errorf(codes.PermissionDenied, "cannot write sheet %s", sheet.Title)
	}

	if err := s.store.DeleteSheet(ctx, sheetUID); err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to delete sheet: %v", err))
	}

	return &emptypb.Empty{}, nil
}

func (s *SheetService) findSheet(ctx context.Context, find *store.FindSheetMessage) (*store.SheetMessage, error) {
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	sheet, err := s.store.GetSheet(ctx, find, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get sheet: %v", err))
	}
	if sheet == nil {
		return nil, status.Errorf(codes.NotFound, "cannot find the sheet")
	}
	return sheet, nil
}

// canWriteSheet check if the principal can write the sheet.
// sheet if writable when:
// PRIVATE: the creator only.
// PROJECT: the creator or project role can manage sheet, workspace Owner and DBA.
// PUBLIC: the creator only.
func (s *SheetService) canWriteSheet(ctx context.Context, sheet *store.SheetMessage) (bool, error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return false, status.Errorf(codes.Internal, "user not found")
	}

	if sheet.CreatorID == user.ID {
		return true, nil
	}

	if sheet.Visibility == store.ProjectSheet {
		projectRoles, err := s.findProjectRoles(ctx, sheet.ProjectUID, user)
		if err != nil {
			return false, err
		}
		if len(projectRoles) == 0 {
			return false, nil
		}
		return projectRoles[api.ProjectOwner], nil
	}

	return false, nil
}

// canReadSheet check if the principal can read the sheet.
// sheet is readable when:
// PRIVATE: the creator only.
// PROJECT: the creator and members in the project.
// PUBLIC: everyone in the workspace.
func (s *SheetService) canReadSheet(ctx context.Context, sheet *store.SheetMessage) (bool, error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return false, status.Errorf(codes.Internal, "user not found")
	}

	switch sheet.Visibility {
	case store.PrivateSheet:
		return sheet.CreatorID == user.ID, nil
	case store.PublicSheet:
		return true, nil
	case store.ProjectSheet:
		if slices.Contains(user.Roles, api.WorkspaceAdmin) || slices.Contains(user.Roles, api.WorkspaceDBA) {
			return true, nil
		}
		projectRoles, err := s.findProjectRoles(ctx, sheet.ProjectUID, user)
		if err != nil {
			return false, err
		}
		return len(projectRoles) > 0, nil
	}
	return false, nil
}

func (s *SheetService) findProjectRoles(ctx context.Context, projectUID int, user *store.UserMessage) (map[api.Role]bool, error) {
	policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &projectUID})
	if err != nil {
		return nil, err
	}
	return utils.GetUserRolesMap(user, policy)
}

func (s *SheetService) convertToAPISheetMessage(ctx context.Context, sheet *store.SheetMessage) (*v1pb.Sheet, error) {
	databaseParent := ""
	if sheet.DatabaseUID != nil {
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			UID: sheet.DatabaseUID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get database: %v", err))
		}
		if database == nil {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("database with id %d not found", *sheet.DatabaseUID))
		}
		databaseParent = fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName)
	}

	visibility := v1pb.Sheet_VISIBILITY_UNSPECIFIED
	switch sheet.Visibility {
	case store.PublicSheet:
		visibility = v1pb.Sheet_VISIBILITY_PUBLIC
	case store.ProjectSheet:
		visibility = v1pb.Sheet_VISIBILITY_PROJECT
	case store.PrivateSheet:
		visibility = v1pb.Sheet_VISIBILITY_PRIVATE
	}

	source := v1pb.Sheet_SOURCE_UNSPECIFIED
	switch sheet.Source {
	case store.SheetFromBytebase:
		source = v1pb.Sheet_SOURCE_BYTEBASE
	case store.SheetFromBytebaseArtifact:
		source = v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT
	}

	tp := v1pb.Sheet_TYPE_UNSPECIFIED
	switch sheet.Type {
	case store.SheetForSQL:
		tp = v1pb.Sheet_TYPE_SQL
	default:
	}

	creator, err := s.store.GetUserByID(ctx, sheet.CreatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get creator: %v", err))
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		UID: &sheet.ProjectUID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get project: %v", err))
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with id %d not found", sheet.ProjectUID))
	}
	var v1SheetPayload *v1pb.SheetPayload
	var v1PushEvent *v1pb.PushEvent
	if sheet.Payload != nil {
		payload := sheet.Payload
		if payload.VcsPayload != nil && payload.VcsPayload.PushEvent != nil {
			v1PushEvent = convertToPushEvent(payload.VcsPayload.PushEvent)
		}
		if payload.DatabaseConfig != nil && payload.BaselineDatabaseConfig != nil {
			v1SheetPayload = &v1pb.SheetPayload{
				DatabaseConfig:         convertStoreDatabaseConfig(payload.DatabaseConfig, nil /* filter */),
				BaselineDatabaseConfig: convertStoreDatabaseConfig(payload.BaselineDatabaseConfig, nil /* filter */),
			}
		}
	}

	return &v1pb.Sheet{
		Name:        fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, project.ResourceID, common.SheetIDPrefix, sheet.UID),
		Database:    databaseParent,
		Title:       sheet.Title,
		Creator:     fmt.Sprintf("users/%s", creator.Email),
		CreateTime:  timestamppb.New(sheet.CreatedTime),
		UpdateTime:  timestamppb.New(sheet.UpdatedTime),
		Content:     []byte(sheet.Statement),
		ContentSize: sheet.Size,
		Visibility:  visibility,
		Source:      source,
		Type:        tp,
		Starred:     sheet.Starred,
		PushEvent:   v1PushEvent,
		Payload:     v1SheetPayload,
	}, nil
}

func convertToStoreSheetMessage(projectUID int, databaseUID *int, creatorID int, sheet *v1pb.Sheet) (*store.SheetMessage, error) {
	visibility, err := convertToStoreSheetVisibility(sheet.Visibility)
	if err != nil {
		return nil, err
	}
	var source store.SheetSource
	switch sheet.Source {
	case v1pb.Sheet_SOURCE_UNSPECIFIED:
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid source %q", sheet.Source))
	case v1pb.Sheet_SOURCE_BYTEBASE:
		source = store.SheetFromBytebase
	case v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT:
		source = store.SheetFromBytebaseArtifact
	default:
		source = store.SheetFromBytebaseArtifact
	}
	var tp store.SheetType
	switch sheet.Type {
	case v1pb.Sheet_TYPE_UNSPECIFIED:
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid type %q", sheet.Type))
	case v1pb.Sheet_TYPE_SQL:
		tp = store.SheetForSQL
	default:
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid type %q", sheet.Type))
	}

	sheetMessage := &store.SheetMessage{
		ProjectUID:  projectUID,
		DatabaseUID: databaseUID,
		CreatorID:   creatorID,
		Title:       sheet.Title,
		Statement:   string(sheet.Content),
		Visibility:  visibility,
		Source:      source,
		Type:        tp,
	}
	if sheet.Payload != nil {
		sheetMessage.Payload = &storepb.SheetPayload{
			DatabaseConfig:         convertV1DatabaseConfig(sheet.Payload.DatabaseConfig),
			BaselineDatabaseConfig: convertV1DatabaseConfig(sheet.Payload.BaselineDatabaseConfig),
		}
	}

	return sheetMessage, nil
}

func convertToStoreSheetVisibility(visibility v1pb.Sheet_Visibility) (store.SheetVisibility, error) {
	switch visibility {
	case v1pb.Sheet_VISIBILITY_UNSPECIFIED:
		return store.SheetVisibility(""), status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid visibility %q", visibility))
	case v1pb.Sheet_VISIBILITY_PUBLIC:
		return store.PublicSheet, nil
	case v1pb.Sheet_VISIBILITY_PROJECT:
		return store.ProjectSheet, nil
	case v1pb.Sheet_VISIBILITY_PRIVATE:
		return store.PrivateSheet, nil
	default:
		return store.SheetVisibility(""), status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid visibility %q", visibility))
	}
}
