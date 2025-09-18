package v1

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// SheetService implements the sheet service.
type SheetService struct {
	v1connect.UnimplementedSheetServiceHandler
	store          *store.Store
	sheetManager   *sheet.Manager
	licenseService *enterprise.LicenseService
	iamManager     *iam.Manager
	profile        *config.Profile
}

// NewSheetService creates a new SheetService.
func NewSheetService(store *store.Store, sheetManager *sheet.Manager, licenseService *enterprise.LicenseService, iamManager *iam.Manager, profile *config.Profile) *SheetService {
	return &SheetService{
		store:          store,
		sheetManager:   sheetManager,
		licenseService: licenseService,
		iamManager:     iamManager,
		profile:        profile,
	}
}

// CreateSheet creates a new sheet.
func (s *SheetService) CreateSheet(ctx context.Context, request *connect.Request[v1pb.CreateSheetRequest]) (*connect.Response[v1pb.Sheet], error) {
	if request.Msg.Sheet == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("sheet must be set"))
	}
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("principal ID not found"))
	}

	projectResourceID, err := common.GetProjectID(request.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project with resource id %q", projectResourceID))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project with resource id %q not found", projectResourceID))
	}
	if project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project with resource id %q had deleted", projectResourceID))
	}

	storeSheetCreate := convertToStoreSheetMessage(project.ResourceID, principalID, request.Msg.Sheet)
	sheet, err := s.sheetManager.CreateSheet(ctx, storeSheetCreate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create sheet"))
	}
	v1pbSheet, err := s.convertToAPISheetMessage(ctx, sheet)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(v1pbSheet), nil
}

func (s *SheetService) BatchCreateSheets(ctx context.Context, request *connect.Request[v1pb.BatchCreateSheetsRequest]) (*connect.Response[v1pb.BatchCreateSheetsResponse], error) {
	if len(request.Msg.Requests) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("requests must be set"))
	}
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	projectResourceID, err := common.GetProjectID(request.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project with resource id %q", projectResourceID))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project with resource id %q not found", projectResourceID))
	}
	if project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project with resource id %q had deleted", projectResourceID))
	}

	var sheetCreates []*store.SheetMessage
	for _, r := range request.Msg.Requests {
		if r.Parent != "" && r.Parent != request.Msg.Parent {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("Sheet Parent %q does not match BatchCreateSheetsRequest.Parent %q", r.Parent, request.Msg.Parent))
		}

		storeSheetCreate := convertToStoreSheetMessage(project.ResourceID, user.ID, r.Sheet)

		sheetCreates = append(sheetCreates, storeSheetCreate)
	}

	sheets, err := s.sheetManager.BatchCreateSheets(ctx, sheetCreates, project.ResourceID, user.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create sheet"))
	}
	response := &v1pb.BatchCreateSheetsResponse{}
	for _, sheet := range sheets {
		v1pbSheet, err := s.convertToAPISheetMessage(ctx, sheet)
		if err != nil {
			return nil, err
		}
		response.Sheets = append(response.Sheets, v1pbSheet)
	}
	return connect.NewResponse(response), nil
}

// GetSheet returns the requested sheet, cutoff the content if the content is too long and the `raw` flag in request is false.
func (s *SheetService) GetSheet(ctx context.Context, request *connect.Request[v1pb.GetSheetRequest]) (*connect.Response[v1pb.Sheet], error) {
	projectResourceID, sheetUID, err := common.GetProjectResourceIDSheetUID(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if sheetUID <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid sheet id %d, must be positive integer", sheetUID))
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project with resource id %s not found", projectResourceID))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project with resource id %s not found", projectResourceID))
	}
	if project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project with resource id %q had deleted", projectResourceID))
	}

	find := &store.FindSheetMessage{
		ProjectID: &project.ResourceID,
		UID:       &sheetUID,
		LoadFull:  request.Msg.Raw,
	}
	sheet, err := s.findSheet(ctx, find)
	if err != nil {
		return nil, err
	}

	v1pbSheet, err := s.convertToAPISheetMessage(ctx, sheet)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(v1pbSheet), nil
}

// UpdateSheet updates a sheet.
func (s *SheetService) UpdateSheet(ctx context.Context, request *connect.Request[v1pb.UpdateSheetRequest]) (*connect.Response[v1pb.Sheet], error) {
	if request.Msg.Sheet == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("sheet cannot be empty"))
	}
	if request.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update mask cannot be empty"))
	}
	if request.Msg.Sheet.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("sheet name cannot be empty"))
	}

	projectResourceID, sheetUID, err := common.GetProjectResourceIDSheetUID(request.Msg.Sheet.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if sheetUID <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid sheet id %d, must be positive integer", sheetUID))
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project with resource id %s not found", projectResourceID))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project with resource id %s not found", projectResourceID))
	}
	if project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project with resource id %q had deleted", projectResourceID))
	}

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{
		UID:       &sheetUID,
		ProjectID: &project.ResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get sheet"))
	}
	if sheet == nil {
		if request.Msg.AllowMissing {
			ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionSheetsCreate, user, project.ResourceID)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to check permission"))
			}
			if !ok {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionSheetsCreate))
			}
			return s.CreateSheet(ctx, connect.NewRequest(&v1pb.CreateSheetRequest{
				Parent: common.FormatProject(project.ResourceID),
				Sheet:  request.Msg.Sheet,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("sheet %q not found", request.Msg.Sheet.Name))
	}

	sheetPatch := &store.PatchSheetMessage{
		UID:       sheet.UID,
		UpdaterID: user.ID,
	}

	for _, path := range request.Msg.UpdateMask.Paths {
		switch path {
		case "content":
			statement := string(request.Msg.Sheet.Content)
			sheetPatch.Statement = &statement
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update mask path %q", path))
		}
	}
	storeSheet, err := s.store.PatchSheet(ctx, sheetPatch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update sheet"))
	}
	v1pbSheet, err := s.convertToAPISheetMessage(ctx, storeSheet)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(v1pbSheet), nil
}

func (s *SheetService) findSheet(ctx context.Context, find *store.FindSheetMessage) (*store.SheetMessage, error) {
	sheet, err := s.store.GetSheet(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get sheet"))
	}
	if sheet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot find the sheet"))
	}
	return sheet, nil
}

func (s *SheetService) convertToAPISheetMessage(ctx context.Context, sheet *store.SheetMessage) (*v1pb.Sheet, error) {
	creator, err := s.store.GetUserByID(ctx, sheet.CreatorID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get creator"))
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &sheet.ProjectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project with id %s not found", sheet.ProjectID))
	}
	v1SheetPayload := &v1pb.SheetPayload{}
	if len(sheet.Payload.GetCommands()) > 0 {
		v1SheetPayload.Commands = convertToSheetCommands(sheet.Payload.GetCommands())
	} else {
		v1SheetPayload.Commands = []*v1pb.SheetCommand{
			{Start: 0, End: int32(sheet.Size)},
		}
	}

	return &v1pb.Sheet{
		Name:        common.FormatSheet(project.ResourceID, sheet.UID),
		Title:       sheet.Title,
		Creator:     fmt.Sprintf("users/%s", creator.Email),
		CreateTime:  timestamppb.New(sheet.CreatedAt),
		Content:     []byte(sheet.Statement),
		ContentSize: sheet.Size,
		Payload:     v1SheetPayload,
		Engine:      convertToEngine(sheet.Payload.GetEngine()),
	}, nil
}

func convertToStoreSheetMessage(projectID string, creatorID int, sheet *v1pb.Sheet) *store.SheetMessage {
	sheetMessage := &store.SheetMessage{
		ProjectID: projectID,
		CreatorID: creatorID,
		Title:     sheet.Title,
		Statement: string(sheet.Content),
		Payload:   &storepb.SheetPayload{},
	}
	sheetMessage.Payload.Engine = convertEngine(sheet.Engine)

	return sheetMessage
}

func convertToSheetCommands(commands []*storepb.SheetCommand) []*v1pb.SheetCommand {
	var cs []*v1pb.SheetCommand
	for _, command := range commands {
		c := &v1pb.SheetCommand{
			Start: command.Start,
			End:   command.End,
		}
		cs = append(cs, c)
	}
	return cs
}
