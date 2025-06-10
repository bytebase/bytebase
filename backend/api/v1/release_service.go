package v1

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type ReleaseService struct {
	v1pb.UnimplementedReleaseServiceServer
	store        *store.Store
	sheetManager *sheet.Manager
	schemaSyncer *schemasync.Syncer
	dbFactory    *dbfactory.DBFactory
}

func NewReleaseService(
	store *store.Store,
	sheetManager *sheet.Manager,
	schemaSyncer *schemasync.Syncer,
	dbFactory *dbfactory.DBFactory,
) *ReleaseService {
	return &ReleaseService{
		store:        store,
		sheetManager: sheetManager,
		schemaSyncer: schemaSyncer,
		dbFactory:    dbFactory,
	}
}

func (s *ReleaseService) CreateRelease(ctx context.Context, request *v1pb.CreateReleaseRequest) (*v1pb.Release, error) {
	if request.Release == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request.Release cannot be nil")
	}

	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}

	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get project id, err: %v", err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find project, err: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %v not found", projectID)
	}

	request.Release.Files, err = validateAndSanitizeReleaseFiles(ctx, s.store, request.Release.Files)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid release files, err: %v", err)
	}
	sheetsToCreate := []*store.SheetMessage{}
	var filesWithoutSheet []*v1pb.Release_File
	// Prepare sheets to create for files with missing sheets.
	// Check versions.
	for _, file := range request.Release.Files {
		if file.Sheet == "" {
			// statement must be present due to validation in validateAndSanitizeReleaseFiles
			sheet := &store.SheetMessage{
				Title:     fmt.Sprintf("File %s", file.Path),
				Statement: string(file.Statement),
			}
			sheetsToCreate = append(sheetsToCreate, sheet)
			filesWithoutSheet = append(filesWithoutSheet, file)
		}
	}

	// Batch create sheets if needed.
	if len(sheetsToCreate) > 0 {
		createdSheets, err := s.sheetManager.BatchCreateSheets(ctx, sheetsToCreate, project.ResourceID, user.ID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create sheets, err: %v", err)
		}
		if len(createdSheets) != len(sheetsToCreate) {
			return nil, status.Errorf(codes.Internal, "failed to create all sheets, expected %d but got %d", len(sheetsToCreate), len(createdSheets))
		}

		// Map created sheets back to files.
		for i, sheet := range createdSheets {
			filesWithoutSheet[i].Sheet = common.FormatSheet(project.ResourceID, sheet.UID)
		}
	}

	files, err := convertReleaseFiles(ctx, s.store, request.Release.Files)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert files, err: %v", err)
	}

	releaseMessage := &store.ReleaseMessage{
		ProjectID: project.ResourceID,
		Payload: &storepb.ReleasePayload{
			Title:     request.Release.Title,
			Files:     files,
			VcsSource: convertReleaseVcsSource(request.Release.VcsSource),
		},
	}

	release, err := s.store.CreateRelease(ctx, releaseMessage, user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create release, err: %v", err)
	}

	converted, err := convertToRelease(ctx, s.store, release)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert release, err: %v", err)
	}

	return converted, nil
}

func (s *ReleaseService) GetRelease(ctx context.Context, request *v1pb.GetReleaseRequest) (*v1pb.Release, error) {
	_, releaseUID, err := common.GetProjectReleaseUID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get release uid, err: %v", err)
	}
	releaseMessage, err := s.store.GetRelease(ctx, releaseUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get release, err: %v", err)
	}
	if releaseMessage == nil {
		return nil, status.Errorf(codes.NotFound, "release %v not found", releaseUID)
	}
	release, err := convertToRelease(ctx, s.store, releaseMessage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to release, err: %v", err)
	}
	return release, nil
}

func (s *ReleaseService) ListReleases(ctx context.Context, request *v1pb.ListReleasesRequest) (*v1pb.ListReleasesResponse, error) {
	if request.PageSize < 0 {
		return nil, status.Errorf(codes.InvalidArgument, "page size must be non-negative: %d", request.PageSize)
	}

	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get project id, err: %v", err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find project, err: %v", err)
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %v not found", projectID)
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

	releaseFind := &store.FindReleaseMessage{
		ProjectID:   &project.ResourceID,
		Limit:       &limitPlusOne,
		Offset:      &offset.offset,
		ShowDeleted: request.ShowDeleted,
	}

	releaseMessages, err := s.store.ListReleases(ctx, releaseFind)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list releases, err: %v", err)
	}

	var nextPageToken string
	if len(releaseMessages) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
		releaseMessages = releaseMessages[:offset.limit]
	}

	releases, err := convertToReleases(ctx, s.store, releaseMessages)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to release, err: %v", err)
	}

	return &v1pb.ListReleasesResponse{
		Releases:      releases,
		NextPageToken: nextPageToken,
	}, nil
}

func (s *ReleaseService) UpdateRelease(ctx context.Context, request *v1pb.UpdateReleaseRequest) (*v1pb.Release, error) {
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	_, releaseUID, err := common.GetProjectReleaseUID(request.Release.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get release uid, err: %v", err)
	}
	release, err := s.store.GetRelease(ctx, releaseUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get release, err: %v", err)
	}
	if release == nil {
		return nil, status.Errorf(codes.NotFound, "release %v not found", releaseUID)
	}
	if release.Deleted {
		return nil, status.Errorf(codes.FailedPrecondition, "release %d is deleted", releaseUID)
	}

	update := &store.UpdateReleaseMessage{
		UID: releaseUID,
	}
	for _, path := range request.UpdateMask.Paths {
		if path == "title" {
			payload := release.Payload
			payload.Title = request.Release.Title
			update.Payload = payload
		}
	}

	releaseMessage, err := s.store.UpdateRelease(ctx, update)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update release, err: %v", err)
	}
	converted, err := convertToRelease(ctx, s.store, releaseMessage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert release, err: %v", err)
	}
	return converted, nil
}

func (s *ReleaseService) DeleteRelease(ctx context.Context, request *v1pb.DeleteReleaseRequest) (*emptypb.Empty, error) {
	_, releaseUID, err := common.GetProjectReleaseUID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get release uid, err: %v", err)
	}
	if _, err := s.store.UpdateRelease(ctx, &store.UpdateReleaseMessage{
		UID:     releaseUID,
		Deleted: &deletePatch,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete release, err: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *ReleaseService) UndeleteRelease(ctx context.Context, request *v1pb.UndeleteReleaseRequest) (*v1pb.Release, error) {
	_, releaseUID, err := common.GetProjectReleaseUID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get release uid, err: %v", err)
	}
	releaseMessage, err := s.store.UpdateRelease(ctx, &store.UpdateReleaseMessage{
		UID:     releaseUID,
		Deleted: &undeletePatch,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to undelete release, err: %v", err)
	}
	release, err := convertToRelease(ctx, s.store, releaseMessage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert release, err: %v", err)
	}
	return release, nil
}

func convertToReleases(ctx context.Context, s *store.Store, releases []*store.ReleaseMessage) ([]*v1pb.Release, error) {
	var rs []*v1pb.Release
	for _, release := range releases {
		r, err := convertToRelease(ctx, s, release)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert to release")
		}
		rs = append(rs, r)
	}
	return rs, nil
}

func convertToRelease(ctx context.Context, s *store.Store, release *store.ReleaseMessage) (*v1pb.Release, error) {
	r := &v1pb.Release{
		Title:      release.Payload.Title,
		CreateTime: timestamppb.New(release.At),
		VcsSource:  convertToReleaseVcsSource(release.Payload.VcsSource),
		State:      convertDeletedToState(release.Deleted),
	}

	files, err := convertToReleaseFiles(ctx, s, release.Payload.Files)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert release files")
	}
	r.Files = files

	project, err := s.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &release.ProjectID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find project")
	}
	if project == nil {
		return nil, errors.Wrapf(err, "project %s not found", release.ProjectID)
	}
	r.Name = common.FormatReleaseName(project.ResourceID, release.UID)

	creator, err := s.GetUserByID(ctx, release.CreatorUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get release creator")
	}
	r.Creator = common.FormatUserEmail(creator.Email)

	return r, nil
}

func convertToReleaseFiles(ctx context.Context, s *store.Store, files []*storepb.ReleasePayload_File) ([]*v1pb.Release_File, error) {
	if files == nil {
		return nil, nil
	}
	var v1Files []*v1pb.Release_File
	for _, f := range files {
		_, sheetUID, err := common.GetProjectResourceIDSheetUID(f.Sheet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheetUID from %q", f.Sheet)
		}
		sheet, err := s.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet %q", f.Sheet)
		}
		if sheet == nil {
			return nil, errors.Errorf("sheet %q not found", f.Sheet)
		}
		v1Files = append(v1Files, &v1pb.Release_File{
			Id:            f.Id,
			Path:          f.Path,
			Sheet:         f.Sheet,
			SheetSha256:   f.SheetSha256,
			Type:          v1pb.ReleaseFileType(f.Type),
			Version:       f.Version,
			Statement:     []byte(sheet.Statement),
			StatementSize: sheet.Size,
			ChangeType:    v1pb.Release_File_ChangeType(f.ChangeType),
		})
	}
	return v1Files, nil
}

func convertToReleaseVcsSource(vs *storepb.ReleasePayload_VCSSource) *v1pb.Release_VCSSource {
	if vs == nil {
		return nil
	}
	return &v1pb.Release_VCSSource{
		VcsType: v1pb.VCSType(vs.VcsType),
		Url:     vs.Url,
	}
}

func convertReleaseFiles(ctx context.Context, s *store.Store, files []*v1pb.Release_File) ([]*storepb.ReleasePayload_File, error) {
	if files == nil {
		return nil, nil
	}
	var rFiles []*storepb.ReleasePayload_File
	for _, f := range files {
		_, sheetUID, err := common.GetProjectResourceIDSheetUID(f.Sheet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheetUID from %q", f.Sheet)
		}
		sheet, err := s.GetSheet(ctx, &store.FindSheetMessage{
			UID:      &sheetUID,
			LoadFull: false,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet %q", f.Sheet)
		}
		if sheet == nil {
			return nil, errors.Errorf("sheet %q not found", f.Sheet)
		}

		rFiles = append(rFiles, &storepb.ReleasePayload_File{
			Id:          f.Id,
			Path:        f.Path,
			Sheet:       f.Sheet,
			SheetSha256: sheet.GetSha256Hex(),
			Type:        storepb.ReleaseFileType(f.Type),
			Version:     f.Version,
			ChangeType:  storepb.ReleasePayload_File_ChangeType(f.ChangeType),
		})
	}
	return rFiles, nil
}

func convertReleaseVcsSource(vs *v1pb.Release_VCSSource) *storepb.ReleasePayload_VCSSource {
	if vs == nil {
		return nil
	}
	return &storepb.ReleasePayload_VCSSource{
		VcsType: storepb.VCSType(vs.VcsType),
		Url:     vs.Url,
	}
}

// validateAndSanitizeReleaseFiles validates and sanitizes the release files inputs.
// It ensures that each file has either a sheet or a statement, and that the sheet is valid.
// It also checks for duplicate versions and sorts the files by version.
// If a sheet is provided, it populates the statement from the sheet.
// If a statement is provided, it computes the sheetSha256 from the statement.
// It returns an error if any validation fails.
// The function also generates a unique ID for each file.
// The files are sorted by version in ascending order.
func validateAndSanitizeReleaseFiles(ctx context.Context, s *store.Store, files []*v1pb.Release_File) ([]*v1pb.Release_File, error) {
	versionSet := map[string]struct{}{}

	for _, f := range files {
		f.Id = uuid.NewString()

		switch {
		// Validate that either sheet or statement is provided
		case f.Sheet == "" && len(f.Statement) == 0:
			return nil, errors.Errorf("either sheet or statement must be set for file %q", f.Path)
		case f.Sheet != "" && len(f.Statement) > 0:
			return nil, errors.Errorf("cannot set both sheet and statement for file %q", f.Path)

		// If sheet is provided but statement is not, populate statement from sheet
		case f.Sheet != "":
			_, sheetUID, err := common.GetProjectResourceIDSheetUID(f.Sheet)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get sheet UID from %q", f.Sheet)
			}
			sheet, err := s.GetSheet(ctx, &store.FindSheetMessage{
				UID:      &sheetUID,
				LoadFull: true,
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get sheet %q", f.Sheet)
			}
			if sheet == nil {
				return nil, errors.Errorf("sheet %q not found", f.Sheet)
			}
			f.Statement = []byte(sheet.Statement)
			f.SheetSha256 = sheet.GetSha256Hex()
		case len(f.Statement) > 0:
			// populate sheetSha256 from statement
			h := sha256.Sum256(f.Statement)
			f.SheetSha256 = hex.EncodeToString(h[:])
		}

		switch f.Type {
		case v1pb.ReleaseFileType_VERSIONED:
		default:
			return nil, errors.Errorf("unexpected file type %q", f.Type.String())
		}

		if _, ok := versionSet[f.Version]; ok {
			return nil, errors.Errorf("found duplicate version %q", f.Version)
		}
		versionSet[f.Version] = struct{}{}
	}

	// Create files with additional parsed version data for sorting.
	type fileWithVersion struct {
		file    *v1pb.Release_File
		version *model.Version
	}
	var filesWithVersions []fileWithVersion
	for _, f := range files {
		version, err := model.NewVersion(f.Version)
		if err != nil {
			return nil, err
		}
		filesWithVersions = append(filesWithVersions, fileWithVersion{
			file:    f,
			version: version,
		})
	}
	slices.SortFunc(filesWithVersions, func(a, b fileWithVersion) int {
		if a.version.LessThan(b.version) {
			return -1
		}
		return 1
	})

	return slices.Collect(func(yield func(*v1pb.Release_File) bool) {
		for _, f := range filesWithVersions {
			if !yield(f.file) {
				return
			}
		}
	}), nil
}
