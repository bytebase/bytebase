package v1

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
	"github.com/bytebase/bytebase/backend/component/iam"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// WorksheetService implements the worksheet service.
type WorksheetService struct {
	v1connect.UnimplementedWorksheetServiceHandler
	store      *store.Store
	iamManager *iam.Manager
}

// NewWorksheetService creates a new WorksheetService.
func NewWorksheetService(store *store.Store, iamManager *iam.Manager) *WorksheetService {
	return &WorksheetService{
		store:      store,
		iamManager: iamManager,
	}
}

// CreateWorksheet creates a new worksheet.
func (s *WorksheetService) CreateWorksheet(
	ctx context.Context,
	req *connect.Request[v1pb.CreateWorksheetRequest],
) (*connect.Response[v1pb.Worksheet], error) {
	request := req.Msg
	if request.Worksheet == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("worksheet must be set"))
	}
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	projectResourceID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		Workspace:  common.GetWorkspaceIDFromContext(ctx),
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get project with resource id %q, err: %v", projectResourceID, err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project with resource id %q not found", projectResourceID))
	}
	if project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project with resource id %q had deleted", projectResourceID))
	}

	var database *store.DatabaseMessage
	if request.Worksheet.Database != "" {
		instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Worksheet.Database)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", request.Worksheet.Database))
		}
		db, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
			Workspace:    common.GetWorkspaceIDFromContext(ctx),
			InstanceID:   &instanceID,
			DatabaseName: &databaseName,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
		}
		if db == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", request.Worksheet.Database))
		}
		// Verify the database belongs to the specified project
		if db.ProjectID != projectResourceID {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found in project %q", request.Worksheet.Database, projectResourceID))
		}
		database = db
	}
	storeWorksheetCreate, err := convertToStoreWorksheetMessage(project, database, user.Email, request.Worksheet)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to convert worksheet: %v", err))
	}
	worksheet, err := s.store.CreateWorkSheet(ctx, storeWorksheetCreate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create worksheet: %v", err))
	}
	v1pbWorksheet := convertToAPIWorksheetMessage(worksheet)
	return connect.NewResponse(v1pbWorksheet), nil
}

// GetWorksheet returns the requested worksheet, cutoff the content if the content is too long and the `raw` flag in request is false.
func (s *WorksheetService) GetWorksheet(
	ctx context.Context,
	req *connect.Request[v1pb.GetWorksheetRequest],
) (*connect.Response[v1pb.Worksheet], error) {
	projectID, worksheetID, err := common.GetProjectIDWorksheetID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	worksheet, err := s.findWorksheet(ctx, projectID, worksheetID, true /* loadFull */)
	if err != nil {
		return nil, err
	}

	ok, err := s.canReadWorksheet(ctx, worksheet)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access with error: %v", err))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("cannot access worksheet %s", worksheet.Title))
	}

	v1pbWorksheet := convertToAPIWorksheetMessage(worksheet)
	return connect.NewResponse(v1pbWorksheet), nil
}

// ListWorksheets returns a list of worksheets.
func (s *WorksheetService) ListWorksheets(
	ctx context.Context,
	req *connect.Request[v1pb.ListWorksheetsRequest],
) (*connect.Response[v1pb.ListWorksheetsResponse], error) {
	request := req.Msg
	if request.PageSize < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("page size must be non-negative: %d", request.PageSize))
	}

	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	projectIDs := []string{projectID}
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	if projectID == "-" {
		projectIDs = nil
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   request.PageToken,
		limit:   int(request.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	filterQ, err := store.GetListWorksheetFilter(request.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	worksheetList, err := s.store.ListWorkSheets(ctx, &store.FindWorkSheetMessage{
		ProjectIDs:     projectIDs,
		Workspace:      workspaceID,
		PrincipalEmail: user.Email,
		FilterQ:        filterQ,
		Limit:          &limitPlusOne,
		Offset:         &offset.offset,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list worksheets: %v", err))
	}

	var nextPageToken string
	if len(worksheetList) == limitPlusOne {
		worksheetList = worksheetList[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get next page token"))
		}
	}

	v1pbWorksheets := make([]*v1pb.Worksheet, 0, len(worksheetList))
	for _, worksheet := range worksheetList {
		v1pbWorksheets = append(v1pbWorksheets, convertToAPIWorksheetMessage(worksheet))
	}
	return connect.NewResponse(&v1pb.ListWorksheetsResponse{
		Worksheets:    v1pbWorksheets,
		NextPageToken: nextPageToken,
	}), nil
}

// ListWorksheetFolders returns the caller's worksheet folders.
func (s *WorksheetService) ListWorksheetFolders(
	ctx context.Context,
	req *connect.Request[v1pb.ListWorksheetFoldersRequest],
) (*connect.Response[v1pb.ListWorksheetFoldersResponse], error) {
	request := req.Msg
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if projectID == "-" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(`ListWorksheetFolders does not support parent "projects/-"`))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	find := &store.FindWorkSheetMessage{
		ProjectIDs:     []string{projectID},
		PrincipalEmail: user.Email,
	}
	organizerList, err := s.store.ListWorksheetOrganizers(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list worksheet folders: %v", err))
	}

	type worksheetFolder struct {
		folders  []string
		category v1pb.WorksheetFolder_Category
	}
	folderByKey := make(map[string]worksheetFolder)
	for _, organizer := range organizerList {
		var category v1pb.WorksheetFolder_Category
		if organizer.WorksheetCreator == user.Email {
			category = v1pb.WorksheetFolder_MINE
		} else {
			category = v1pb.WorksheetFolder_SHARED
		}
		for i := range organizer.Payload.Folders {
			folders := append([]string(nil), organizer.Payload.Folders[:i+1]...)
			key := fmt.Sprintf("%d\x00%s", category, strings.Join(folders, "\x00"))
			folderByKey[key] = worksheetFolder{
				folders:  folders,
				category: category,
			}
		}
	}

	keys := make([]string, 0, len(folderByKey))
	for key := range folderByKey {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	response := &v1pb.ListWorksheetFoldersResponse{
		Folders: make([]*v1pb.WorksheetFolder, 0, len(keys)),
	}
	for _, key := range keys {
		response.Folders = append(response.Folders, &v1pb.WorksheetFolder{
			Folders:  folderByKey[key].folders,
			Category: folderByKey[key].category,
		})
	}
	return connect.NewResponse(response), nil
}

// SearchWorksheets returns a list of worksheets based on the search filters.
func (s *WorksheetService) SearchWorksheets(
	ctx context.Context,
	req *connect.Request[v1pb.SearchWorksheetsRequest],
) (*connect.Response[v1pb.SearchWorksheetsResponse], error) {
	request := req.Msg
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if projectID == "-" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(`SearchWorksheets does not support parent "projects/-"`))
	}
	if request.PageSize < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("page size cannot be negative"))
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   request.PageToken,
		limit:   int(request.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	worksheetFind := &store.FindWorkSheetMessage{
		ProjectIDs:     []string{projectID},
		PrincipalEmail: user.Email,
		Limit:          &limitPlusOne,
		Offset:         &offset.offset,
	}

	filterQ, err := store.GetListSheetFilter(ctx, s.store, user.Email, request.Filter, true /* allowTitleContains */)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	worksheetFind.FilterQ = filterQ

	worksheetList, err := s.store.ListWorkSheets(ctx, worksheetFind)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list worksheets: %v", err))
	}
	nextPageToken := ""
	if len(worksheetList) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate next page token: %v", err))
		}
		worksheetList = worksheetList[:offset.limit]
	}

	var v1pbWorksheets []*v1pb.Worksheet
	for _, worksheet := range worksheetList {
		ok, err := s.canReadWorksheet(ctx, worksheet)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access with error: %v", err))
		}
		if !ok {
			slog.Warn("cannot access worksheet", slog.String("name", worksheet.Title))
			continue
		}
		v1pbWorksheet := convertToAPIWorksheetMessage(worksheet)
		v1pbWorksheets = append(v1pbWorksheets, v1pbWorksheet)
	}
	return connect.NewResponse(&v1pb.SearchWorksheetsResponse{
		Worksheets:    v1pbWorksheets,
		NextPageToken: nextPageToken,
	}), nil
}

// UpdateWorksheet updates a worksheet.
func (s *WorksheetService) UpdateWorksheet(
	ctx context.Context,
	req *connect.Request[v1pb.UpdateWorksheetRequest],
) (*connect.Response[v1pb.Worksheet], error) {
	request := req.Msg
	if request.Worksheet == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("worksheet cannot be empty"))
	}
	if request.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update mask cannot be empty"))
	}
	if request.Worksheet.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("worksheet name cannot be empty"))
	}

	projectID, worksheetID, err := common.GetProjectIDWorksheetID(request.Worksheet.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	worksheet, err := s.findWorksheet(ctx, projectID, worksheetID, false /* loadFull */)
	if err != nil {
		return nil, err
	}
	ok, err = s.canWriteWorksheet(ctx, worksheet)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access with error: %v", err))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("cannot write worksheet %s", worksheet.Title))
	}

	worksheetPatch := &store.PatchWorkSheetMessage{
		ResourceID: worksheet.ResourceID,
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			worksheetPatch.Title = &request.Worksheet.Title
		case "content":
			statement := string(request.Worksheet.Content)
			worksheetPatch.Statement = &statement
		case "visibility":
			visibility, err := convertToStoreWorksheetVisibility(request.Worksheet.Visibility)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid visibility %q", request.Worksheet.Visibility))
			}
			stringVisibility := string(visibility)
			worksheetPatch.Visibility = &stringVisibility
		case "database":
			if request.Worksheet.Database != "" {
				instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Worksheet.Database)
				if err != nil {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", request.Worksheet.Database))
				}
				database, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
					Workspace:    common.GetWorkspaceIDFromContext(ctx),
					InstanceID:   &instanceID,
					DatabaseName: &databaseName,
				})
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
				}
				if database == nil {
					return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %v not found", request.Worksheet.Database))
				}
				// Verify the database belongs to the worksheet's project.
				if database.ProjectID != worksheet.ProjectID {
					return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found in project %q", request.Worksheet.Database, worksheet.ProjectID))
				}
				worksheetPatch.InstanceID, worksheetPatch.DatabaseName = &database.InstanceID, &database.DatabaseName
			} else {
				emptyStr := ""
				worksheetPatch.InstanceID = &emptyStr
				worksheetPatch.DatabaseName = &emptyStr
			}
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update mask path %q", path))
		}
	}
	if err := s.store.PatchWorkSheet(ctx, worksheetPatch); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to update worksheet: %v", err))
	}

	worksheet, err = s.store.GetWorkSheet(ctx, &store.FindWorkSheetMessage{
		ProjectIDs:     []string{worksheet.ProjectID},
		ResourceID:     &worksheetID,
		PrincipalEmail: user.Email,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get worksheet: %v", err))
	}
	if worksheet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("worksheet %q not found", request.Worksheet.Name))
	}
	v1pbWorksheet := convertToAPIWorksheetMessage(worksheet)
	return connect.NewResponse(v1pbWorksheet), nil
}

// DeleteWorksheet deletes a worksheet.
func (s *WorksheetService) DeleteWorksheet(
	ctx context.Context,
	req *connect.Request[v1pb.DeleteWorksheetRequest],
) (*connect.Response[emptypb.Empty], error) {
	projectID, worksheetID, err := common.GetProjectIDWorksheetID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	worksheet, err := s.findWorksheet(ctx, projectID, worksheetID, false /* loadFull */)
	if err != nil {
		return nil, err
	}
	ok, err := s.canWriteWorksheet(ctx, worksheet)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access with error: %v", err))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("cannot write worksheet %s", worksheet.Title))
	}

	if err := s.store.DeleteWorkSheet(ctx, worksheet.ResourceID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to delete worksheet: %v", err))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// UpdateWorksheetOrganizer upsert the worksheet organizer.
func (s *WorksheetService) UpdateWorksheetOrganizer(
	ctx context.Context,
	req *connect.Request[v1pb.UpdateWorksheetOrganizerRequest],
) (*connect.Response[v1pb.WorksheetOrganizer], error) {
	request := req.Msg
	projectID, worksheetID, err := common.GetProjectIDWorksheetID(request.Organizer.Worksheet)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	worksheet, err := s.findWorksheet(ctx, projectID, worksheetID, false /* loadFull */)
	if err != nil {
		return nil, err
	}

	ok, err := s.canReadWorksheet(ctx, worksheet)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access with error: %v", err))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("cannot access worksheet %s", worksheet.Title))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	worksheetOrganizerUpsert, err := s.store.GetWorksheetOrganizer(ctx, worksheet.ResourceID, user.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to found worksheet organizer with error: %v", err))
	}

	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "starred":
			worksheetOrganizerUpsert.Payload.Starred = request.Organizer.Starred
		case "folders":
			worksheetOrganizerUpsert.Payload.Folders = request.Organizer.Folders
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update mask path %q", path))
		}
	}

	organizer, err := s.store.UpsertWorksheetOrganizer(ctx, worksheetOrganizerUpsert)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to upsert organizer for worksheet %s with error: %v", request.Organizer.Worksheet, err))
	}

	return connect.NewResponse(&v1pb.WorksheetOrganizer{
		Worksheet: request.Organizer.Worksheet,
		Starred:   organizer.Payload.Starred,
		Folders:   organizer.Payload.Folders,
	}), nil
}

func (s *WorksheetService) BatchUpdateWorksheetOrganizer(
	ctx context.Context,
	req *connect.Request[v1pb.BatchUpdateWorksheetOrganizerRequest],
) (*connect.Response[v1pb.BatchUpdateWorksheetOrganizerResponse], error) {
	request := req.Msg
	if request.Organizer == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("organizer cannot be empty"))
	}
	if request.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update mask cannot be empty"))
	}
	if len(request.UpdateMask.Paths) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update mask paths cannot be empty"))
	}

	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if projectID == "-" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("projects/- is not supported for batch update worksheet organizers"))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	filterQ, err := store.GetListSheetFilter(ctx, s.store, user.Email, request.Filter, false /* allowTitleContains */)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	worksheetList, err := s.store.ListWorkSheets(ctx, &store.FindWorkSheetMessage{
		ProjectIDs:     []string{projectID},
		PrincipalEmail: user.Email,
		FilterQ:        filterQ,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list worksheets: %v", err))
	}

	var worksheetIDs []string
	for _, worksheet := range worksheetList {
		ok, err := s.canReadWorksheet(ctx, worksheet)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access with error: %v", err))
		}
		if !ok {
			slog.Warn("cannot access worksheet", slog.String("name", worksheet.Title))
			continue
		}
		worksheetIDs = append(worksheetIDs, worksheet.ResourceID)
	}

	patch := &store.BatchUpdateWorksheetOrganizerPatch{
		WorksheetResourceIDs: worksheetIDs,
		Principal:            user.Email,
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "starred":
			starred := request.Organizer.Starred
			patch.Starred = &starred
		case "folders":
			folders := request.Organizer.Folders
			patch.Folders = &folders
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update mask path %q", path))
		}
	}

	organizers, err := s.store.BatchUpdateWorksheetOrganizer(ctx, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to batch update worksheet organizers: %v", err))
	}
	response := &v1pb.BatchUpdateWorksheetOrganizerResponse{
		UpdatedCount: int32(len(organizers)),
	}
	for _, organizer := range organizers {
		response.WorksheetOrganizers = append(response.WorksheetOrganizers, &v1pb.WorksheetOrganizer{
			Worksheet: fmt.Sprintf("%s/worksheets/%s", request.Parent, organizer.WorksheetResourceID),
			Starred:   organizer.Payload.Starred,
			Folders:   organizer.Payload.Folders,
		})
	}
	return connect.NewResponse(response), nil
}

func (s *WorksheetService) findWorksheet(ctx context.Context, projectID string, worksheetID string, loadFull bool) (*store.WorkSheetMessage, error) {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	worksheet, err := s.store.GetWorkSheet(ctx, &store.FindWorkSheetMessage{
		ProjectIDs:     []string{projectID},
		ResourceID:     &worksheetID,
		LoadFull:       loadFull,
		PrincipalEmail: user.Email,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get worksheet: %v", err))
	}
	if worksheet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot find the worksheet"))
	}
	return worksheet, nil
}

// canWriteWorksheet check if the principal can write the worksheet.
// worksheet is writable when the user has bb.worksheets.manage permission on the workspace, or.
// PRIVATE: the creator.
// PROJECT_WRITE: all members with bb.projects.get permission in the project.
func (s *WorksheetService) canWriteWorksheet(ctx context.Context, worksheet *store.WorkSheetMessage) (bool, error) {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return false, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	// Worksheet creator and workspace "bb.worksheets.manage" can always write.
	if worksheet.Creator == user.Email {
		return true, nil
	}
	ok, err := s.iamManager.CheckPermission(ctx, permission.WorksheetsManage, user, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		return false, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
	}
	if ok {
		return true, nil
	}

	switch worksheet.Visibility {
	case store.PrivateWorkSheet:
		return false, nil
	case store.ProjectReadWorkSheet:
		// For READ visibility, check the "bb.worksheets.manage" permission in the project.
		return s.checkWorksheetPermission(ctx, worksheet.ProjectID, user, permission.WorksheetsManage)
	case store.ProjectWriteWorkSheet:
		// For READ visibility, needs "bb.worksheets.get" permission in the project.
		return s.checkWorksheetPermission(ctx, worksheet.ProjectID, user, permission.WorksheetsGet)
	default:
		return false, nil
	}
}

// canReadWorksheet check if the principal can read the worksheet.
// worksheet is readable when the user has bb.worksheets.get permission on the workspace, or.
// PRIVATE: the creator only.
// PROJECT_WRITE: all members with bb.projects.get permission in the project.
// PROJECT_READ: all members with bb.projects.get permission in the project.
func (s *WorksheetService) canReadWorksheet(ctx context.Context, worksheet *store.WorkSheetMessage) (bool, error) {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return false, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	// Worksheet creator and workspace bb.worksheets.get can always read.
	if worksheet.Creator == user.Email {
		return true, nil
	}
	ok, err := s.iamManager.CheckPermission(ctx, permission.WorksheetsManage, user, common.GetWorkspaceIDFromContext(ctx))
	if err != nil {
		return false, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
	}
	if ok {
		return true, nil
	}

	switch worksheet.Visibility {
	case store.PrivateWorkSheet:
		return false, nil
	case store.ProjectReadWorkSheet, store.ProjectWriteWorkSheet:
		// Check the "bb.worksheets.get" permission in the project.
		return s.checkWorksheetPermission(ctx, worksheet.ProjectID, user, permission.WorksheetsGet)
	default:
		return false, nil
	}
}

func (s *WorksheetService) checkWorksheetPermission(
	ctx context.Context,
	projectID string,
	user *store.UserMessage,
	permission permission.Permission,
) (bool, error) {
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		Workspace:  workspaceID,
		ResourceID: &projectID,
	})
	if err != nil {
		return false, err
	}
	ok, err := s.iamManager.CheckPermission(ctx, permission, user, workspaceID, project.ResourceID)
	if err != nil {
		return false, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
	}
	return ok, nil
}

func convertToAPIWorksheetMessage(worksheet *store.WorkSheetMessage) *v1pb.Worksheet {
	databaseParent := ""
	if worksheet.InstanceID != nil && worksheet.DatabaseName != nil {
		databaseParent = common.FormatDatabase(*worksheet.InstanceID, *worksheet.DatabaseName)
	}

	visibility := v1pb.Worksheet_VISIBILITY_UNSPECIFIED
	switch worksheet.Visibility {
	case store.ProjectReadWorkSheet:
		visibility = v1pb.Worksheet_PROJECT_READ
	case store.ProjectWriteWorkSheet:
		visibility = v1pb.Worksheet_PROJECT_WRITE
	case store.PrivateWorkSheet:
		visibility = v1pb.Worksheet_PRIVATE
	default:
		// Keep VISIBILITY_UNSPECIFIED
	}
	return &v1pb.Worksheet{
		Name:        common.FormatWorksheet(worksheet.ProjectID, worksheet.ResourceID),
		Project:     common.FormatProject(worksheet.ProjectID),
		Database:    databaseParent,
		Title:       worksheet.Title,
		Creator:     fmt.Sprintf("users/%s", worksheet.Creator),
		CreateTime:  timestamppb.New(worksheet.CreatedAt),
		UpdateTime:  timestamppb.New(worksheet.UpdatedAt),
		Content:     []byte(worksheet.Statement),
		ContentSize: worksheet.Size,
		Visibility:  visibility,
		Starred:     worksheet.Starred,
		Folders:     worksheet.Folders,
	}
}

func convertToStoreWorksheetMessage(project *store.ProjectMessage, database *store.DatabaseMessage, creator string, worksheet *v1pb.Worksheet) (*store.WorkSheetMessage, error) {
	visibility, err := convertToStoreWorksheetVisibility(worksheet.Visibility)
	if err != nil {
		return nil, err
	}

	worksheetMessage := &store.WorkSheetMessage{
		ProjectID:  project.ResourceID,
		Creator:    creator,
		Title:      worksheet.Title,
		Statement:  string(worksheet.Content),
		Visibility: visibility,
	}
	if database != nil {
		worksheetMessage.InstanceID = &database.InstanceID
		worksheetMessage.DatabaseName = &database.DatabaseName
	}

	return worksheetMessage, nil
}

func convertToStoreWorksheetVisibility(visibility v1pb.Worksheet_Visibility) (store.WorkSheetVisibility, error) {
	switch visibility {
	case v1pb.Worksheet_VISIBILITY_UNSPECIFIED:
		return store.WorkSheetVisibility(""), connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid visibility %q", visibility))
	case v1pb.Worksheet_PROJECT_READ:
		return store.ProjectReadWorkSheet, nil
	case v1pb.Worksheet_PROJECT_WRITE:
		return store.ProjectWriteWorkSheet, nil
	case v1pb.Worksheet_PRIVATE:
		return store.PrivateWorkSheet, nil
	default:
		return store.WorkSheetVisibility(""), connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid visibility %q", visibility))
	}
}
