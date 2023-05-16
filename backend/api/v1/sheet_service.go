package v1

import (
	"context"
	"fmt"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// SheetService implements the sheet service.
type SheetService struct {
	v1pb.UnimplementedSheetServiceServer
	store *store.Store
}

// NewSheetService creates a new SheetService.
func NewSheetService(store *store.Store) *SheetService {
	return &SheetService{
		store: store,
	}
}

// CreateSheet creates a new sheet.
func (s *SheetService) CreateSheet(ctx context.Context, request *v1pb.CreateSheetRequest) (*v1pb.Sheet, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateSheet not implemented")
}

// GetSheet returns the requested sheet, cutoff the content if the content is too long and the `raw` flag in request is false.
func (s *SheetService) GetSheet(ctx context.Context, request *v1pb.GetSheetRequest) (*v1pb.Sheet, error) {
	projectResourceID, sheetID, err := getProjectResourceIDSheetID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	sheetIntID, err := strconv.Atoi(sheetID)
	if err != nil || sheetIntID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid sheet id %s, must be positive integer", sheetID))
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %s not found", projectResourceID))
	}

	currentPrincipalID := ctx.Value(common.PrincipalIDContextKey).(int)
	sheet, err := s.store.GetSheetV2(ctx, &api.SheetFind{
		ID:        &sheetIntID,
		LoadFull:  request.Raw,
		ProjectID: &project.UID,
	}, currentPrincipalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get sheet: %v", err))
	}
	if sheet == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("sheet with id %d not found", sheetIntID))
	}

	v1pbSheet, err := s.convertToAPISheetMessage(ctx, projectResourceID, sheet)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert sheet: %v", err))
	}
	return v1pbSheet, nil
}

// SearchSheets returns a list of sheets based on the search filters.
func (s *SheetService) SearchSheets(ctx context.Context, request *v1pb.SearchSheetsRequest) (*v1pb.SearchSheetsResponse, error) {
	projectResourceID, err := getProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	currentPrincipalID := ctx.Value(common.PrincipalIDContextKey).(int)

	sheetFind := &api.SheetFind{}
	if projectResourceID != "-" {
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID: &projectResourceID,
		})
		if err != nil {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %s not found", projectResourceID))
		}
		sheetFind.ProjectID = &project.UID
	}

	// TODO(zp): It is difficult to find all the sheets visible to a principal atomically
	// without adding a new store layer method, which has two parts:
	// 1. creator = principal && visibility in (PROJECT, PUBLIC, PRIVATE)
	// 2. creator ! = principal && visibility in (PROJECT, PUBLIC)
	// So we don't allow empty filter for now.
	if request.Filter == "" {
		return nil, status.Errorf(codes.InvalidArgument, "filter should not be empty")
	}

	specs, err := parseFilter(request.Filter)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	for _, spec := range specs {
		switch spec.key {
		case "creator":
			creatorEmail := getUserEmailFromIdentifier(spec.value)
			if creatorEmail == "" {
				return nil, status.Errorf(codes.InvalidArgument, "invalid empty creator identifier")
			}
			user, err := s.store.GetUser(ctx, &store.FindUserMessage{
				Email: &creatorEmail,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get user: %s", err.Error()))
			}
			if user == nil {
				return nil, status.Errorf(codes.NotFound, fmt.Sprintf("user with email %s not found", creatorEmail))
			}
			switch spec.operator {
			case comparatorTypeEqual:
				sheetFind.CreatorID = &user.ID
				sheetFind.Visibilities = []api.SheetVisibility{api.ProjectSheet, api.PublicSheet, api.PrivateSheet}
			case comparatorTypeNotEqual:
				sheetFind.ExcludedCreatorID = &user.ID
				sheetFind.Visibilities = []api.SheetVisibility{api.ProjectSheet, api.PublicSheet}
				sheetFind.PrincipalID = &user.ID
			default:
				return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid operator %q for creator", spec.operator))
			}
		case "starred":
			if spec.operator != comparatorTypeEqual {
				return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid operator %q for starred", spec.operator))
			}
			switch spec.value {
			case "true":
				sheetFind.OrganizerPrincipalIDStarred = &currentPrincipalID
			case "false":
				sheetFind.OrganizerPrincipalIDNotStarred = &currentPrincipalID
			default:
				return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid value %q for starred", spec.value))
			}
		default:
			return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid filter key %q", spec.key))
		}
	}
	sheetList, err := s.store.ListSheetsV2(ctx, sheetFind, currentPrincipalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to list sheets: %v", err))
	}

	var v1pbSheets []*v1pb.Sheet
	for _, sheet := range sheetList {
		v1pbSheet, err := s.convertToAPISheetMessage(ctx, projectResourceID, sheet)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert sheet: %v", err))
		}
		v1pbSheets = append(v1pbSheets, v1pbSheet)
	}
	return &v1pb.SearchSheetsResponse{
		Sheets: v1pbSheets,
	}, nil
}

func (_ *SheetService) UpdateSheet(context.Context, *v1pb.UpdateSheetRequest) (*v1pb.Sheet, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateSheet not implemented")
}

func (_ *SheetService) DeleteSheet(context.Context, *v1pb.DeleteSheetRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteSheet not implemented")
}

func (_ *SheetService) SyncSheets(context.Context, *v1pb.SyncSheetsRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SyncSheets not implemented")
}

func (s *SheetService) convertToAPISheetMessage(ctx context.Context, projectResourceID string, sheet *store.SheetMessage) (*v1pb.Sheet, error) {
	databaseParent := ""
	if sheet.DatabaseID != nil {
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			UID: sheet.DatabaseID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get database: %v", err))
		}
		if database == nil {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("database with id %d not found", *sheet.DatabaseID))
		}
		databaseParent = fmt.Sprintf("%s%s/%s%d", instanceNamePrefix, database.InstanceID, databaseIDPrefix, database.UID)
	}

	visibility := v1pb.Sheet_VISIBILITY_UNSPECIFIED
	switch sheet.Visibility {
	case api.PublicSheet:
		visibility = v1pb.Sheet_VISIBILITY_PUBLIC
	case api.ProjectSheet:
		visibility = v1pb.Sheet_VISIBILITY_PROJECT
	case api.PrivateSheet:
		visibility = v1pb.Sheet_VISIBILITY_PRIVATE
	}

	source := v1pb.Sheet_SOURCE_UNSPECIFIED
	switch sheet.Source {
	case api.SheetFromBytebase:
		source = v1pb.Sheet_SOURCE_BYTEBASE
	case api.SheetFromBytebaseArtifact:
		source = v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT
	case api.SheetFromGitLab:
		source = v1pb.Sheet_SOURCE_GITLAB
	case api.SheetFromGitHub:
		source = v1pb.Sheet_SOURCE_GITHUB
	case api.SheetFromBitbucket:
		source = v1pb.Sheet_SOURCE_BITBUCKET
	}

	tp := v1pb.Sheet_TYPE_UNSPECIFIED
	switch sheet.Type {
	case api.SheetForSQL:
		tp = v1pb.Sheet_TYPE_SQL
	default:
	}

	return &v1pb.Sheet{
		Name:        fmt.Sprintf("%s%s/%s%d", projectNamePrefix, projectResourceID, sheetIDPrefix, sheet.UID),
		Database:    databaseParent,
		Title:       sheet.Name,
		Creator:     getUserIdentifier(sheet.Creator.Email),
		CreateTime:  timestamppb.New(sheet.CreatedTime),
		UpdateTime:  timestamppb.New(sheet.UpdatedTime),
		Content:     []byte(sheet.Statement),
		ContentSize: sheet.Size,
		Visibility:  visibility,
		Source:      source,
		Type:        tp,
		Starred:     sheet.Starred,
	}, nil
}
