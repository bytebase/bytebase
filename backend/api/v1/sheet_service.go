package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sheet"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// SheetService implements the sheet service.
type SheetService struct {
	v1pb.UnimplementedSheetServiceServer
	store          *store.Store
	sheetManager   *sheet.Manager
	licenseService enterprise.LicenseService
	iamManager     *iam.Manager
	profile        *config.Profile
}

// NewSheetService creates a new SheetService.
func NewSheetService(store *store.Store, sheetManager *sheet.Manager, licenseService enterprise.LicenseService, iamManager *iam.Manager, profile *config.Profile) *SheetService {
	return &SheetService{
		store:          store,
		sheetManager:   sheetManager,
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
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project with resource id %q, err: %v", projectResourceID, err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project with resource id %q not found", projectResourceID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, "project with resource id %q had deleted", projectResourceID)
	}

	storeSheetCreate, err := convertToStoreSheetMessage(project.ResourceID, principalID, request.Sheet)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to convert sheet: %v", err)
	}
	sheet, err := s.sheetManager.CreateSheet(ctx, storeSheetCreate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create sheet: %v", err)
	}
	v1pbSheet, err := s.convertToAPISheetMessage(ctx, sheet)
	if err != nil {
		return nil, err
	}
	return v1pbSheet, nil
}

func (s *SheetService) BatchCreateSheets(ctx context.Context, request *v1pb.BatchCreateSheetsRequest) (*v1pb.BatchCreateSheetsResponse, error) {
	if len(request.Requests) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "requests must be set")
	}
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}

	projectResourceID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project with resource id %q, err: %v", projectResourceID, err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project with resource id %q not found", projectResourceID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, "project with resource id %q had deleted", projectResourceID)
	}

	var sheetCreates []*store.SheetMessage
	for _, r := range request.Requests {
		if r.Parent != "" && r.Parent != request.Parent {
			return nil, status.Errorf(codes.InvalidArgument, "Sheet Parent %q does not match BatchCreateSheetsRequest.Parent %q", r.Parent, request.Parent)
		}

		storeSheetCreate, err := convertToStoreSheetMessage(project.ResourceID, user.ID, r.Sheet)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to convert sheet: %v", err)
		}

		sheetCreates = append(sheetCreates, storeSheetCreate)
	}

	sheets, err := s.sheetManager.BatchCreateSheets(ctx, sheetCreates, project.ResourceID, user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create sheet: %v", err)
	}
	response := &v1pb.BatchCreateSheetsResponse{}
	for _, sheet := range sheets {
		v1pbSheet, err := s.convertToAPISheetMessage(ctx, sheet)
		if err != nil {
			return nil, err
		}
		response.Sheets = append(response.Sheets, v1pbSheet)
	}
	return response, nil
}

// GetSheet returns the requested sheet, cutoff the content if the content is too long and the `raw` flag in request is false.
func (s *SheetService) GetSheet(ctx context.Context, request *v1pb.GetSheetRequest) (*v1pb.Sheet, error) {
	projectResourceID, sheetUID, err := common.GetProjectResourceIDSheetUID(request.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if sheetUID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid sheet id %d, must be positive integer", sheetUID)
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "project with resource id %s not found", projectResourceID)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project with resource id %s not found", projectResourceID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, "project with resource id %q had deleted", projectResourceID)
	}

	find := &store.FindSheetMessage{
		ProjectID: &project.ResourceID,
		UID:       &sheetUID,
		LoadFull:  request.Raw,
	}
	sheet, err := s.findSheet(ctx, find)
	if err != nil {
		return nil, err
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
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if sheetUID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid sheet id %d, must be positive integer", sheetUID)
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "project with resource id %s not found", projectResourceID)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project with resource id %s not found", projectResourceID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, "project with resource id %q had deleted", projectResourceID)
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{
		UID:       &sheetUID,
		ProjectID: &project.ResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get sheet: %v", err)
	}
	if sheet == nil {
		return nil, status.Errorf(codes.NotFound, "sheet %q not found", request.Sheet.Name)
	}

	sheetPatch := &store.PatchSheetMessage{
		UID:       sheet.UID,
		UpdaterID: principalID,
	}

	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "content":
			statement := string(request.Sheet.Content)
			sheetPatch.Statement = &statement
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid update mask path %q", path)
		}
	}
	storeSheet, err := s.store.PatchSheet(ctx, sheetPatch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update sheet: %v", err)
	}
	v1pbSheet, err := s.convertToAPISheetMessage(ctx, storeSheet)
	if err != nil {
		return nil, err
	}

	return v1pbSheet, nil
}

func (s *SheetService) findSheet(ctx context.Context, find *store.FindSheetMessage) (*store.SheetMessage, error) {
	sheet, err := s.store.GetSheet(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get sheet: %v", err)
	}
	if sheet == nil {
		return nil, status.Errorf(codes.NotFound, "cannot find the sheet")
	}
	return sheet, nil
}

func (s *SheetService) convertToAPISheetMessage(ctx context.Context, sheet *store.SheetMessage) (*v1pb.Sheet, error) {
	creator, err := s.store.GetUserByID(ctx, sheet.CreatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get creator: %v", err)
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &sheet.ProjectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project with id %s not found", sheet.ProjectID)
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

func convertToStoreSheetMessage(projectID string, creatorID int, sheet *v1pb.Sheet) (*store.SheetMessage, error) {
	sheetMessage := &store.SheetMessage{
		ProjectID: projectID,
		CreatorID: creatorID,
		Title:     sheet.Title,
		Statement: string(sheet.Content),
		Payload:   &storepb.SheetPayload{},
	}
	sheetMessage.Payload.Engine = convertEngine(sheet.Engine)

	return sheetMessage, nil
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
