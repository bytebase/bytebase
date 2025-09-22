package v1

import (
	"context"
	"fmt"
	"strings"

	"log/slog"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
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

	projectResourceID, err := common.GetProjectID(request.Worksheet.Project)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
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
		db, err := getDatabaseMessage(ctx, s.store, request.Worksheet.Database)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		// Verify the database belongs to the specified project
		if db.ProjectID != projectResourceID {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found in project %q", request.Worksheet.Database, projectResourceID))
		}
		if db == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", request.Worksheet.Database))
		}
		database = db
	}
	storeWorksheetCreate, err := convertToStoreWorksheetMessage(project, database, user.ID, request.Worksheet)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to convert worksheet: %v", err))
	}
	worksheet, err := s.store.CreateWorkSheet(ctx, storeWorksheetCreate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create worksheet: %v", err))
	}
	v1pbWorksheet, err := s.convertToAPIWorksheetMessage(ctx, worksheet)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(v1pbWorksheet), nil
}

// GetWorksheet returns the requested worksheet, cutoff the content if the content is too long and the `raw` flag in request is false.
func (s *WorksheetService) GetWorksheet(
	ctx context.Context,
	req *connect.Request[v1pb.GetWorksheetRequest],
) (*connect.Response[v1pb.Worksheet], error) {
	request := req.Msg
	worksheetUID, err := common.GetWorksheetUID(request.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if worksheetUID <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid worksheet id %d, must be positive integer", worksheetUID))
	}

	find := &store.FindWorkSheetMessage{
		UID:      &worksheetUID,
		LoadFull: true,
	}
	worksheet, err := s.findWorksheet(ctx, find)
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

	v1pbWorksheet, err := s.convertToAPIWorksheetMessage(ctx, worksheet)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(v1pbWorksheet), nil
}

func (s *WorksheetService) getListSheetFilter(ctx context.Context, callerID int, filter string) (*store.ListResourceFilter, error) {
	if filter == "" {
		return nil, nil
	}
	e, err := cel.NewEnv()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create cel env"))
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String()))
	}

	var getFilter func(expr celast.Expr) (string, error)
	var positionalArgs []any

	getUserID := func(name string) (int, error) {
		creatorEmail := strings.TrimPrefix(name, "users/")
		if creatorEmail == "" {
			return 0, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid empty creator identifier"))
		}
		user, err := s.store.GetUserByEmail(ctx, creatorEmail)
		if err != nil {
			return 0, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get user: %v", err))
		}
		if user == nil {
			return 0, connect.NewError(connect.CodeNotFound, errors.Errorf("user with email %s not found", creatorEmail))
		}
		return user.ID, nil
	}

	parseToSQL := func(variable, value any) (string, error) {
		switch variable {
		case "creator":
			userID, err := getUserID(value.(string))
			if err != nil {
				return "", err
			}
			positionalArgs = append(positionalArgs, userID)
			return fmt.Sprintf("worksheet.creator_id = $%d", len(positionalArgs)), nil
		case "starred":
			if starred, ok := value.(bool); ok {
				positionalArgs = append(positionalArgs, callerID)
				return fmt.Sprintf("worksheet.id IN (SELECT worksheet_id FROM worksheet_organizer WHERE principal_id = $%d AND starred = %v)", len(positionalArgs), starred), nil
			}
			return "TRUE", nil
		case "visibility":
			visibility, err := convertToStoreWorksheetVisibility(v1pb.Worksheet_Visibility(v1pb.Worksheet_Visibility_value[value.(string)]))
			if err != nil {
				return "", err
			}
			positionalArgs = append(positionalArgs, visibility)
			return fmt.Sprintf("worksheet.visibility = $%d", len(positionalArgs)), nil
		default:
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %q", variable))
		}
	}

	getFilter = func(expr celast.Expr) (string, error) {
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalOr:
				return getSubConditionFromExpr(expr, getFilter, "OR")
			case celoperators.LogicalAnd:
				return getSubConditionFromExpr(expr, getFilter, "AND")
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				return parseToSQL(variable, value)
			case celoperators.NotEquals:
				variable, value := getVariableAndValueFromExpr(expr)
				if variable != "creator" {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`only "creator" support "!=" operator`))
				}
				userID, err := getUserID(value.(string))
				if err != nil {
					return "", err
				}
				positionalArgs = append(positionalArgs, userID)
				return fmt.Sprintf("worksheet.creator_id != $%d", len(positionalArgs)), nil
			case celoperators.In:
				variable, value := getVariableAndValueFromExpr(expr)
				if variable != "visibility" {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`only "visibility" support "visibility in [xx]" filter`))
				}
				rawList, ok := value.([]any)
				if !ok {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid visibility value %q", value))
				}
				if len(rawList) == 0 {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("empty visibility filter"))
				}
				visibilityList := []string{}
				for _, raw := range rawList {
					visibility, err := convertToStoreWorksheetVisibility(v1pb.Worksheet_Visibility(v1pb.Worksheet_Visibility_value[raw.(string)]))
					if err != nil {
						return "", err
					}
					visibilityList = append(visibilityList, fmt.Sprintf(`'%s'`, visibility))
				}
				return fmt.Sprintf(`worksheet.visibility IN (%s)`, strings.Join(visibilityList, ",")), nil
			default:
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected function %v", functionName))
			}
		default:
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected expr kind %v", expr.Kind()))
		}
	}

	where, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return nil, err
	}

	return &store.ListResourceFilter{
		Args:  positionalArgs,
		Where: "(" + where + ")",
	}, nil
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

	worksheetFind := &store.FindWorkSheetMessage{}
	// TODO(zp): It is difficult to find all the worksheets visible to a principal atomically
	// without adding a new store layer method, which has two parts:
	// 1. creator = principal && visibility in (PROJECT, PUBLIC, PRIVATE)
	// 2. creator ! = principal && visibility in (PROJECT, PUBLIC)
	// So we don't allow empty filter for now.
	if request.Filter == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("filter should not be empty"))
	}

	filter, err := s.getListSheetFilter(ctx, user.ID, request.Filter)
	if err != nil {
		return nil, err
	}
	worksheetFind.Filter = filter

	worksheetList, err := s.store.ListWorkSheets(ctx, worksheetFind, user.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list worksheets: %v", err))
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
		v1pbWorksheet, err := s.convertToAPIWorksheetMessage(ctx, worksheet)
		if err != nil {
			var connectErr *connect.Error
			if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeNotFound {
				slog.Debug("failed to found resource for worksheet", log.BBError(err), slog.Int("id", worksheet.UID), slog.String("project", worksheet.ProjectID))
				continue
			}
			return nil, err
		}
		v1pbWorksheets = append(v1pbWorksheets, v1pbWorksheet)
	}
	return connect.NewResponse(&v1pb.SearchWorksheetsResponse{
		Worksheets: v1pbWorksheets,
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

	worksheetUID, err := common.GetWorksheetUID(request.Worksheet.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if worksheetUID <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid worksheet id %d, must be positive integer", worksheetUID))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	worksheet, err := s.store.GetWorkSheet(ctx, &store.FindWorkSheetMessage{
		UID: &worksheetUID,
	}, user.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get worksheet: %v", err))
	}
	if worksheet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("worksheet %q not found", request.Worksheet.Name))
	}
	ok, err = s.canWriteWorksheet(ctx, worksheet)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access with error: %v", err))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("cannot write worksheet %s", worksheet.Title))
	}

	worksheetPatch := &store.PatchWorkSheetMessage{
		UID: worksheet.UID,
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
			database, err := getDatabaseMessage(ctx, s.store, request.Worksheet.Database)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to found database %v", request.Worksheet.Database))
			}
			if database == nil || database.Deleted {
				return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %v not found", request.Worksheet.Database))
			}
			worksheetPatch.InstanceID, worksheetPatch.DatabaseName = &database.InstanceID, &database.DatabaseName
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update mask path %q", path))
		}
	}
	if err := s.store.PatchWorkSheet(ctx, worksheetPatch); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to update worksheet: %v", err))
	}

	worksheet, err = s.store.GetWorkSheet(ctx, &store.FindWorkSheetMessage{
		UID: &worksheetUID,
	}, user.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get worksheet: %v", err))
	}
	if worksheet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("worksheet %q not found", request.Worksheet.Name))
	}
	v1pbWorksheet, err := s.convertToAPIWorksheetMessage(ctx, worksheet)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(v1pbWorksheet), nil
}

// DeleteWorksheet deletes a worksheet.
func (s *WorksheetService) DeleteWorksheet(
	ctx context.Context,
	req *connect.Request[v1pb.DeleteWorksheetRequest],
) (*connect.Response[emptypb.Empty], error) {
	request := req.Msg
	worksheetUID, err := common.GetWorksheetUID(request.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	worksheet, err := s.store.GetWorkSheet(ctx, &store.FindWorkSheetMessage{
		UID: &worksheetUID,
	}, user.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get worksheet: %v", err))
	}
	if worksheet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("worksheet with id %d not found", worksheetUID))
	}
	ok, err = s.canWriteWorksheet(ctx, worksheet)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access with error: %v", err))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("cannot write worksheet %s", worksheet.Title))
	}

	if err := s.store.DeleteWorkSheet(ctx, worksheetUID); err != nil {
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
	worksheetUID, err := common.GetWorksheetUID(request.Organizer.Worksheet)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if worksheetUID <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid worksheet id %d, must be positive integer", worksheetUID))
	}

	worksheet, err := s.findWorksheet(ctx, &store.FindWorkSheetMessage{
		UID: &worksheetUID,
	})
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
	worksheetOrganizerUpsert := &store.WorksheetOrganizerMessage{
		WorksheetUID: worksheetUID,
		PrincipalUID: user.ID,
	}

	for _, path := range request.UpdateMask.Paths {
		if path == "starred" {
			worksheetOrganizerUpsert.Starred = request.Organizer.Starred
		}
	}

	organizer, err := s.store.UpsertWorksheetOrganizer(ctx, worksheetOrganizerUpsert)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to upsert organizer for worksheet %s with error: %v", request.Organizer.Worksheet, err))
	}

	return connect.NewResponse(&v1pb.WorksheetOrganizer{
		Worksheet: request.Organizer.Worksheet,
		Starred:   organizer.Starred,
	}), nil
}

func (s *WorksheetService) findWorksheet(ctx context.Context, find *store.FindWorkSheetMessage) (*store.WorkSheetMessage, error) {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	worksheet, err := s.store.GetWorkSheet(ctx, find, user.ID)
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
	if worksheet.CreatorID == user.ID {
		return true, nil
	}
	ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionWorksheetsManage, user)
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
		return s.checkWorksheetPermission(ctx, worksheet.ProjectID, user, iam.PermissionWorksheetsManage)
	case store.ProjectWriteWorkSheet:
		// For READ visibility, needs "bb.worksheets.get" permission in the project.
		return s.checkWorksheetPermission(ctx, worksheet.ProjectID, user, iam.PermissionWorksheetsGet)
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
	if worksheet.CreatorID == user.ID {
		return true, nil
	}
	ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionWorksheetsManage, user)
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
		return s.checkWorksheetPermission(ctx, worksheet.ProjectID, user, iam.PermissionWorksheetsGet)
	default:
		return false, nil
	}
}

func (s *WorksheetService) checkWorksheetPermission(
	ctx context.Context,
	projectID string,
	user *store.UserMessage,
	permission iam.Permission,
) (bool, error) {
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return false, err
	}
	ok, err := s.iamManager.CheckPermission(ctx, permission, user, project.ResourceID)
	if err != nil {
		return false, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
	}
	return ok, nil
}

func (s *WorksheetService) convertToAPIWorksheetMessage(ctx context.Context, worksheet *store.WorkSheetMessage) (*v1pb.Worksheet, error) {
	databaseParent := ""
	if worksheet.InstanceID != nil && worksheet.DatabaseName != nil {
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			ProjectID:    &worksheet.ProjectID,
			InstanceID:   worksheet.InstanceID,
			DatabaseName: worksheet.DatabaseName,
		})
		if err != nil {
			slog.Debug("failed to found database for worksheet", log.BBError(err), slog.Int("id", worksheet.UID), slog.String("instance", *worksheet.InstanceID), slog.String("database", *worksheet.DatabaseName))
		}
		if database != nil {
			databaseParent = common.FormatDatabase(database.InstanceID, database.DatabaseName)
		}
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

	creator, err := s.store.GetUserByID(ctx, worksheet.CreatorID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get creator: %v", err))
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &worksheet.ProjectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get project: %v", err))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project with id %s not found", worksheet.ProjectID))
	}
	return &v1pb.Worksheet{
		Name:        fmt.Sprintf("%s%d", common.WorksheetIDPrefix, worksheet.UID),
		Project:     common.FormatProject(project.ResourceID),
		Database:    databaseParent,
		Title:       worksheet.Title,
		Creator:     fmt.Sprintf("users/%s", creator.Email),
		CreateTime:  timestamppb.New(worksheet.CreatedAt),
		UpdateTime:  timestamppb.New(worksheet.UpdatedAt),
		Content:     []byte(worksheet.Statement),
		ContentSize: worksheet.Size,
		Visibility:  visibility,
		Starred:     worksheet.Starred,
	}, nil
}

func convertToStoreWorksheetMessage(project *store.ProjectMessage, database *store.DatabaseMessage, creatorID int, worksheet *v1pb.Worksheet) (*store.WorkSheetMessage, error) {
	visibility, err := convertToStoreWorksheetVisibility(worksheet.Visibility)
	if err != nil {
		return nil, err
	}

	worksheetMessage := &store.WorkSheetMessage{
		ProjectID:    project.ResourceID,
		InstanceID:   &database.InstanceID,
		DatabaseName: &database.DatabaseName,
		CreatorID:    creatorID,
		Title:        worksheet.Title,
		Statement:    string(worksheet.Content),
		Visibility:   visibility,
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
