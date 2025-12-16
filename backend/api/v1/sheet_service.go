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
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	projectResourceID, err := common.GetProjectID(request.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
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

	storeSheetCreate := convertToStoreSheetMessage(project.ResourceID, user.Email, request.Msg.Sheet)
	sheets, err := s.sheetManager.CreateSheets(ctx, project.ResourceID, user.Email, storeSheetCreate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create sheet"))
	}
	sheet := sheets[0]
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
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	projectResourceID, err := common.GetProjectID(request.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
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

		storeSheetCreate := convertToStoreSheetMessage(project.ResourceID, user.Email, r.Sheet)

		sheetCreates = append(sheetCreates, storeSheetCreate)
	}

	sheets, err := s.sheetManager.CreateSheets(ctx, project.ResourceID, user.Email, sheetCreates...)
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

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
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
	creator, err := s.store.GetUserByEmail(ctx, sheet.Creator)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get creator"))
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
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
		v1SheetPayload.Commands = convertToRanges(sheet.Payload.GetCommands())
	} else {
		v1SheetPayload.Commands = []*v1pb.Range{
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

func convertToStoreSheetMessage(projectID string, creator string, sheet *v1pb.Sheet) *store.SheetMessage {
	sheetMessage := &store.SheetMessage{
		ProjectID: projectID,
		Creator:   creator,
		Title:     sheet.Title,
		Statement: string(sheet.Content),
		Payload:   &storepb.SheetPayload{},
	}
	sheetMessage.Payload.Engine = convertEngine(sheet.Engine)

	return sheetMessage
}

func convertToRanges(commands []*storepb.Range) []*v1pb.Range {
	var cs []*v1pb.Range
	for _, command := range commands {
		c := &v1pb.Range{
			Start: command.Start,
			End:   command.End,
		}
		cs = append(cs, c)
	}
	return cs
}
