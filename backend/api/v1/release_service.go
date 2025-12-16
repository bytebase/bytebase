package v1

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

type ReleaseService struct {
	v1connect.UnimplementedReleaseServiceHandler
	store        *store.Store
	sheetManager *sheet.Manager
	schemaSyncer *schemasync.Syncer
	dbFactory    *dbfactory.DBFactory
	iamManager   *iam.Manager
}

func NewReleaseService(
	store *store.Store,
	sheetManager *sheet.Manager,
	schemaSyncer *schemasync.Syncer,
	dbFactory *dbfactory.DBFactory,
	iamManager *iam.Manager,
) *ReleaseService {
	return &ReleaseService{
		store:        store,
		sheetManager: sheetManager,
		schemaSyncer: schemaSyncer,
		dbFactory:    dbFactory,
		iamManager:   iamManager,
	}
}

func (s *ReleaseService) CreateRelease(ctx context.Context, req *connect.Request[v1pb.CreateReleaseRequest]) (*connect.Response[v1pb.Release], error) {
	if req.Msg.Release == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("request.Release cannot be nil"))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	projectID, err := common.GetProjectID(req.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get project id"))
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %s not found", projectID))
	}

	sanitizedFiles, err := validateAndSanitizeReleaseFiles(ctx, s.store, req.Msg.Release.Files, true)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid release files"))
	}
	sheetsToCreate := []*store.SheetMessage{}
	var filesWithoutSheet []*v1pb.Release_File
	// Prepare sheets to create for files with missing sheets.
	// Check versions.
	for _, file := range sanitizedFiles {
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
		createdSheets, err := s.sheetManager.CreateSheets(ctx, project.ResourceID, user.Email, sheetsToCreate...)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create sheets"))
		}
		if len(createdSheets) != len(sheetsToCreate) {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create all sheets, expected %d but got %d", len(sheetsToCreate), len(createdSheets)))
		}

		// Map created sheets back to files.
		for i, sheet := range createdSheets {
			filesWithoutSheet[i].Sheet = common.FormatSheet(project.ResourceID, sheet.UID)
		}
	}

	files, err := convertReleaseFiles(ctx, s.store, sanitizedFiles)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert files"))
	}

	releaseMessage := &store.ReleaseMessage{
		ProjectID: project.ResourceID,
		Digest:    req.Msg.Release.Digest,
		Payload: &storepb.ReleasePayload{
			Title:     req.Msg.Release.Title,
			Files:     files,
			VcsSource: convertReleaseVcsSource(req.Msg.Release.VcsSource),
		},
	}

	release, err := s.store.CreateRelease(ctx, releaseMessage, user.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create release"))
	}

	converted, err := convertToRelease(ctx, s.store, release)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert release"))
	}

	return connect.NewResponse(converted), nil
}

func (s *ReleaseService) GetRelease(ctx context.Context, req *connect.Request[v1pb.GetReleaseRequest]) (*connect.Response[v1pb.Release], error) {
	projectID, releaseUID, err := common.GetProjectReleaseUID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get release uid"))
	}
	releaseMessage, err := s.store.GetRelease(ctx, &store.FindReleaseMessage{
		ProjectID: &projectID,
		UID:       &releaseUID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get release"))
	}
	if releaseMessage == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("release %d not found in project %s", releaseUID, projectID))
	}
	release, err := convertToRelease(ctx, s.store, releaseMessage)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to release"))
	}
	return connect.NewResponse(release), nil
}

func (s *ReleaseService) ListReleases(ctx context.Context, req *connect.Request[v1pb.ListReleasesRequest]) (*connect.Response[v1pb.ListReleasesResponse], error) {
	if req.Msg.PageSize < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("page size must be non-negative: %d", req.Msg.PageSize))
	}

	projectID, err := common.GetProjectID(req.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get project id"))
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %s not found", projectID))
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   req.Msg.PageToken,
		limit:   int(req.Msg.PageSize),
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
		ShowDeleted: req.Msg.ShowDeleted,
	}

	releaseMessages, err := s.store.ListReleases(ctx, releaseFind)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list releases"))
	}

	var nextPageToken string
	if len(releaseMessages) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get next page token"))
		}
		releaseMessages = releaseMessages[:offset.limit]
	}

	releases, err := convertToReleases(ctx, s.store, releaseMessages)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to release"))
	}

	return connect.NewResponse(&v1pb.ListReleasesResponse{
		Releases:      releases,
		NextPageToken: nextPageToken,
	}), nil
}

func (s *ReleaseService) SearchReleases(ctx context.Context, req *connect.Request[v1pb.SearchReleasesRequest]) (*connect.Response[v1pb.SearchReleasesResponse], error) {
	if req.Msg.PageSize < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("page size must be non-negative: %d", req.Msg.PageSize))
	}

	projectID, err := common.GetProjectID(req.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get project id"))
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &projectID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find project"))
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %s not found", projectID))
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   req.Msg.PageToken,
		limit:   int(req.Msg.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	releaseFind := &store.FindReleaseMessage{
		ProjectID: &project.ResourceID,
		Limit:     &limitPlusOne,
		Offset:    &offset.offset,
	}

	if req.Msg.Digest != nil {
		releaseFind.Digest = req.Msg.Digest
	}

	releaseMessages, err := s.store.ListReleases(ctx, releaseFind)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list releases"))
	}

	var nextPageToken string
	if len(releaseMessages) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get next page token"))
		}
		releaseMessages = releaseMessages[:offset.limit]
	}

	releases, err := convertToReleases(ctx, s.store, releaseMessages)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to release"))
	}

	return connect.NewResponse(&v1pb.SearchReleasesResponse{
		Releases:      releases,
		NextPageToken: nextPageToken,
	}), nil
}

func (s *ReleaseService) UpdateRelease(ctx context.Context, req *connect.Request[v1pb.UpdateReleaseRequest]) (*connect.Response[v1pb.Release], error) {
	if req.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update_mask must be set"))
	}

	projectID, releaseUID, err := common.GetProjectReleaseUID(req.Msg.Release.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get release uid"))
	}

	// Use GetRelease with FindReleaseMessage for database-level filtering
	release, err := s.store.GetRelease(ctx, &store.FindReleaseMessage{
		ProjectID: &projectID,
		UID:       &releaseUID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get release"))
	}
	if release == nil {
		if req.Msg.AllowMissing {
			project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &projectID})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find project"))
			}
			if project == nil {
				return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %s not found", projectID))
			}
			return s.CreateRelease(ctx, connect.NewRequest(&v1pb.CreateReleaseRequest{
				Parent:  common.FormatProject(project.ResourceID),
				Release: req.Msg.Release,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("release %d not found in project %s", releaseUID, projectID))
	}

	if release.Deleted {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("release %d is deleted", releaseUID))
	}

	update := &store.UpdateReleaseMessage{
		UID: releaseUID,
	}
	for _, path := range req.Msg.UpdateMask.Paths {
		if path == "title" {
			payload := release.Payload
			payload.Title = req.Msg.Release.Title
			update.Payload = payload
		}
	}

	releaseMessage, err := s.store.UpdateRelease(ctx, update)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update release"))
	}
	converted, err := convertToRelease(ctx, s.store, releaseMessage)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert release"))
	}
	return connect.NewResponse(converted), nil
}

func (s *ReleaseService) DeleteRelease(ctx context.Context, req *connect.Request[v1pb.DeleteReleaseRequest]) (*connect.Response[emptypb.Empty], error) {
	projectID, releaseUID, err := common.GetProjectReleaseUID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get release uid"))
	}
	release, err := s.store.GetRelease(ctx, &store.FindReleaseMessage{
		ProjectID: &projectID,
		UID:       &releaseUID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get release"))
	}
	if release == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("release %d not found in project %s", releaseUID, projectID))
	}

	if _, err := s.store.UpdateRelease(ctx, &store.UpdateReleaseMessage{
		UID:     releaseUID,
		Deleted: &deletePatch,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to delete release"))
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *ReleaseService) UndeleteRelease(ctx context.Context, req *connect.Request[v1pb.UndeleteReleaseRequest]) (*connect.Response[v1pb.Release], error) {
	projectID, releaseUID, err := common.GetProjectReleaseUID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get release uid"))
	}
	release, err := s.store.GetRelease(ctx, &store.FindReleaseMessage{
		ProjectID:   &projectID,
		UID:         &releaseUID,
		ShowDeleted: true,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get release"))
	}
	if release == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("release %d not found in project %s", releaseUID, projectID))
	}

	releaseMessage, err := s.store.UpdateRelease(ctx, &store.UpdateReleaseMessage{
		UID:     releaseUID,
		Deleted: &undeletePatch,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to undelete release"))
	}
	releaseConverted, err := convertToRelease(ctx, s.store, releaseMessage)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert release"))
	}
	return connect.NewResponse(releaseConverted), nil
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
		Digest:     release.Digest,
	}

	files, err := convertToReleaseFiles(ctx, s, release.Payload.Files)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert release files")
	}
	r.Files = files

	project, err := s.GetProject(ctx, &store.FindProjectMessage{ResourceID: &release.ProjectID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find project")
	}
	if project == nil {
		return nil, errors.Errorf("project %s not found", release.ProjectID)
	}
	r.Name = common.FormatReleaseName(project.ResourceID, release.UID)

	creator, err := s.GetUserByEmail(ctx, release.Creator)
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
		projectID, sheetUID, err := common.GetProjectResourceIDSheetUID(f.Sheet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheetUID from %q", f.Sheet)
		}
		sheet, err := s.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID, ProjectID: &projectID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet %q", f.Sheet)
		}
		if sheet == nil {
			return nil, errors.Errorf("sheet %d not found in project %s", sheetUID, projectID)
		}
		v1Files = append(v1Files, &v1pb.Release_File{
			Id:            f.Id,
			Path:          f.Path,
			Sheet:         f.Sheet,
			SheetSha256:   f.SheetSha256,
			Type:          v1pb.Release_File_Type(f.Type),
			Version:       f.Version,
			Statement:     []byte(sheet.Statement),
			StatementSize: sheet.Size,
			EnableGhost:   f.EnableGhost,
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
		projectID, sheetUID, err := common.GetProjectResourceIDSheetUID(f.Sheet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheetUID from %q", f.Sheet)
		}
		sheet, err := s.GetSheet(ctx, &store.FindSheetMessage{
			UID:       &sheetUID,
			ProjectID: &projectID,
			LoadFull:  false,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet %q", f.Sheet)
		}
		if sheet == nil {
			return nil, errors.Errorf("sheet %d not found in project %s", sheetUID, projectID)
		}

		rFiles = append(rFiles, &storepb.ReleasePayload_File{
			Id:          f.Id,
			Path:        f.Path,
			Sheet:       f.Sheet,
			SheetSha256: sheet.GetSha256Hex(),
			Type:        storepb.SchemaChangeType(f.Type),
			Version:     f.Version,
			EnableGhost: f.EnableGhost,
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
// For creating release with declarative files, it checks if there is only one file.
//
// The input files are modified in place.
// If a sheet is provided, it populates the statement from the sheet.
// If a statement is provided, it computes the sheetSha256 from the statement.
func validateAndSanitizeReleaseFiles(ctx context.Context, s *store.Store, files []*v1pb.Release_File, createRelease bool) ([]*v1pb.Release_File, error) {
	if len(files) == 0 {
		return nil, errors.Errorf("release files cannot be empty")
	}

	versionSet := map[string]struct{}{}
	fileTypeCount := map[v1pb.Release_File_Type]int{}

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
			projectID, sheetUID, err := common.GetProjectResourceIDSheetUID(f.Sheet)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get sheet UID from %q", f.Sheet)
			}
			sheet, err := s.GetSheet(ctx, &store.FindSheetMessage{
				UID:       &sheetUID,
				ProjectID: &projectID,
				LoadFull:  true,
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get sheet %q", f.Sheet)
			}
			if sheet == nil {
				return nil, errors.Errorf("sheet %d not found in project %s", sheetUID, projectID)
			}
			f.Statement = []byte(sheet.Statement)
			f.SheetSha256 = sheet.GetSha256Hex()
		case len(f.Statement) > 0:
			// populate sheetSha256 from statement
			h := sha256.Sum256(f.Statement)
			f.SheetSha256 = hex.EncodeToString(h[:])
		default:
		}

		switch f.Type {
		case v1pb.Release_File_VERSIONED:
			if _, ok := versionSet[f.Version]; ok {
				return nil, errors.Errorf("found duplicate version %q", f.Version)
			}
			versionSet[f.Version] = struct{}{}
		case v1pb.Release_File_DECLARATIVE:
			versionSet[f.Version] = struct{}{}
			if len(versionSet) > 1 {
				return nil, errors.Errorf("declarative files should have the same version, found %v", versionSet)
			}
		default:
			return nil, errors.Errorf("unexpected file type %q", f.Type.String())
		}

		fileTypeCount[f.Type]++
	}

	if fileTypeCount[v1pb.Release_File_VERSIONED] > 0 && fileTypeCount[v1pb.Release_File_DECLARATIVE] > 0 {
		return nil, errors.Errorf("cannot have both versioned and declarative files")
	}
	if createRelease && fileTypeCount[v1pb.Release_File_DECLARATIVE] > 1 {
		return nil, errors.Errorf("cannot have multiple declarative files when creating release")
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
