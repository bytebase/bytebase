package v1

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// SheetService implements the sheet service.
type SheetService struct {
	v1connect.UnimplementedSheetServiceHandler
	store *store.Store
}

// NewSheetService creates a new SheetService.
func NewSheetService(store *store.Store) *SheetService {
	return &SheetService{
		store: store,
	}
}

// CreateSheet creates a new sheet.
func (s *SheetService) CreateSheet(ctx context.Context, request *connect.Request[v1pb.CreateSheetRequest]) (*connect.Response[v1pb.Sheet], error) {
	if request.Msg.Sheet == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("sheet must be set"))
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

	storeSheetCreate := convertToStoreSheetMessage(request.Msg.Sheet)
	sheets, err := s.store.CreateSheets(ctx, storeSheetCreate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create sheet"))
	}
	sheet := sheets[0]
	v1pbSheet := convertToAPISheetMessage(project.ResourceID, sheet)
	return connect.NewResponse(v1pbSheet), nil
}

func (s *SheetService) BatchCreateSheets(ctx context.Context, request *connect.Request[v1pb.BatchCreateSheetsRequest]) (*connect.Response[v1pb.BatchCreateSheetsResponse], error) {
	if len(request.Msg.Requests) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("requests must be set"))
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

		storeSheetCreate := convertToStoreSheetMessage(r.Sheet)

		sheetCreates = append(sheetCreates, storeSheetCreate)
	}

	sheets, err := s.store.CreateSheets(ctx, sheetCreates...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create sheet"))
	}
	response := &v1pb.BatchCreateSheetsResponse{}
	for _, sheet := range sheets {
		v1pbSheet := convertToAPISheetMessage(project.ResourceID, sheet)
		response.Sheets = append(response.Sheets, v1pbSheet)
	}
	return connect.NewResponse(response), nil
}

// GetSheet returns the requested sheet, cutoff the content if the content is too long and the `raw` flag in request is false.
func (s *SheetService) GetSheet(ctx context.Context, request *connect.Request[v1pb.GetSheetRequest]) (*connect.Response[v1pb.Sheet], error) {
	projectResourceID, sheetSha256, err := common.GetProjectResourceIDSheetSha256(request.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if sheetSha256 == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid sheet sha256, must be non-empty"))
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

	var sheet *store.SheetMessage
	var sheetErr error
	if request.Msg.Raw {
		sheet, sheetErr = s.store.GetSheetFull(ctx, sheetSha256)
	} else {
		sheet, sheetErr = s.store.GetSheetTruncated(ctx, sheetSha256)
	}
	if sheetErr != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(sheetErr, "failed to get sheet"))
	}
	if sheet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot find the sheet"))
	}

	v1pbSheet := convertToAPISheetMessage(project.ResourceID, sheet)
	return connect.NewResponse(v1pbSheet), nil
}

func convertToAPISheetMessage(projectID string, sheet *store.SheetMessage) *v1pb.Sheet {
	return &v1pb.Sheet{
		Name:        common.FormatSheet(projectID, sheet.Sha256),
		Content:     []byte(sheet.Statement),
		ContentSize: sheet.Size,
	}
}

func convertToStoreSheetMessage(sheet *v1pb.Sheet) *store.SheetMessage {
	sheetMessage := &store.SheetMessage{
		Statement: string(sheet.Content),
	}

	return sheetMessage
}
