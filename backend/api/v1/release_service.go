package v1

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/sheet"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

type ReleaseService struct {
	v1connect.UnimplementedReleaseServiceHandler
	store        *store.Store
	sheetManager *sheet.Manager
	dbFactory    *dbfactory.DBFactory
}

func NewReleaseService(
	store *store.Store,
	sheetManager *sheet.Manager,
	dbFactory *dbfactory.DBFactory,
) *ReleaseService {
	return &ReleaseService{
		store:        store,
		sheetManager: sheetManager,
		dbFactory:    dbFactory,
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

	sanitizedFiles, err := validateAndSanitizeReleaseFiles(ctx, s.store, req.Msg.Release.Files, req.Msg.Release.Type)
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
				Statement: string(file.Statement),
			}
			sheetsToCreate = append(sheetsToCreate, sheet)
			filesWithoutSheet = append(filesWithoutSheet, file)
		}
	}

	// Batch create sheets if needed.
	if len(sheetsToCreate) > 0 {
		createdSheets, err := s.store.CreateSheets(ctx, sheetsToCreate...)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create sheets"))
		}
		if len(createdSheets) != len(sheetsToCreate) {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create all sheets, expected %d but got %d", len(sheetsToCreate), len(createdSheets)))
		}

		// Map created sheets back to files.
		for i, sheet := range createdSheets {
			filesWithoutSheet[i].Sheet = common.FormatSheet(project.ResourceID, sheet.Sha256)
		}
	}

	releaseMessage := &store.ReleaseMessage{
		ProjectID: project.ResourceID,
		Digest:    req.Msg.Release.Digest,
		Payload: &storepb.ReleasePayload{
			Title:     req.Msg.Release.Title,
			VcsSource: convertReleaseVcsSource(req.Msg.Release.VcsSource),
			Type:      storepb.SchemaChangeType(req.Msg.Release.Type),
		},
	}

	var sheetSha256s []string
	for _, f := range sanitizedFiles {
		_, sheetSha256, err := common.GetProjectResourceIDSheetSha256(f.Sheet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheetSha256 from %q", f.Sheet)
		}
		sheetSha256s = append(sheetSha256s, sheetSha256)
	}
	exist, err := s.store.HasSheets(ctx, sheetSha256s...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check sheets")
	}
	if !exist {
		return nil, errors.Errorf("some sheets are not found")
	}

	for i, f := range sanitizedFiles {
		releaseMessage.Payload.Files = append(releaseMessage.Payload.Files, &storepb.ReleasePayload_File{
			Path:        f.Path,
			SheetSha256: sheetSha256s[i],
			Version:     f.Version,
			EnableGhost: f.EnableGhost,
		})
	}

	release, err := s.store.CreateRelease(ctx, releaseMessage, user.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create release"))
	}

	converted := convertToRelease(release)
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
	release := convertToRelease(releaseMessage)
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

	releases := convertToReleases(releaseMessages)
	return connect.NewResponse(&v1pb.ListReleasesResponse{
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
		if path == "type" {
			payload := release.Payload
			payload.Type = storepb.SchemaChangeType(req.Msg.Release.Type)
			update.Payload = payload
		}
	}

	releaseMessage, err := s.store.UpdateRelease(ctx, update)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to update release"))
	}
	converted := convertToRelease(releaseMessage)
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
	releaseConverted := convertToRelease(releaseMessage)
	return connect.NewResponse(releaseConverted), nil
}

func convertToReleases(releases []*store.ReleaseMessage) []*v1pb.Release {
	var rs []*v1pb.Release
	for _, release := range releases {
		rs = append(rs, convertToRelease(release))
	}
	return rs
}

func convertToRelease(release *store.ReleaseMessage) *v1pb.Release {
	r := &v1pb.Release{
		Name:       common.FormatReleaseName(release.ProjectID, release.UID),
		Title:      release.Payload.Title,
		Creator:    common.FormatUserEmail(release.Creator),
		CreateTime: timestamppb.New(release.At),
		VcsSource:  convertToReleaseVcsSource(release.Payload.VcsSource),
		State:      convertDeletedToState(release.Deleted),
		Digest:     release.Digest,
		Type:       v1pb.Release_Type(release.Payload.Type),
	}

	for _, f := range release.Payload.Files {
		// Sheets are now project-agnostic, no need to check projectID
		r.Files = append(r.Files, &v1pb.Release_File{
			Path:        f.Path,
			Sheet:       common.FormatSheet(release.ProjectID, f.SheetSha256),
			SheetSha256: f.SheetSha256,
			Version:     f.Version,
			EnableGhost: f.EnableGhost,
		})
	}
	return r
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
// It also checks for duplicate versions in versioned releases and sorts the files by version.
//
// The input files are modified in place.
// If a sheet is provided, it populates the statement from the sheet.
// If a statement is provided, it computes the sheetSha256 from the statement.
func validateAndSanitizeReleaseFiles(ctx context.Context, s *store.Store, files []*v1pb.Release_File, releaseType v1pb.Release_Type) ([]*v1pb.Release_File, error) {
	if len(files) == 0 {
		return nil, errors.Errorf("release files cannot be empty")
	}

	versionSet := map[string]struct{}{}

	for _, f := range files {
		switch {
		// Validate that either sheet or statement is provided
		// For DECLARATIVE releases, empty content is allowed (represents dropping all objects)
		case f.Sheet == "" && len(f.Statement) == 0:
			if releaseType != v1pb.Release_DECLARATIVE {
				return nil, errors.Errorf("either sheet or statement must be set for file %q", f.Path)
			}
			// For declarative releases with empty content, set empty statement and compute SHA256
			f.Statement = []byte{}
			h := sha256.Sum256(f.Statement)
			f.SheetSha256 = hex.EncodeToString(h[:])
		case f.Sheet != "" && len(f.Statement) > 0:
			return nil, errors.Errorf("cannot set both sheet and statement for file %q", f.Path)

		// If sheet is provided but statement is not, populate statement from sheet
		case f.Sheet != "":
			_, sheetSha256, err := common.GetProjectResourceIDSheetSha256(f.Sheet)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get sheet SHA256 from %q", f.Sheet)
			}
			sheet, err := s.GetSheetFull(ctx, sheetSha256)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get sheet %q", f.Sheet)
			}
			if sheet == nil {
				return nil, errors.Errorf("sheet %q not found", f.Sheet)
			}
			// Sheets are now project-agnostic, no need to check projectID
			f.Statement = []byte(sheet.Statement)
			f.SheetSha256 = sheet.Sha256
		case len(f.Statement) > 0:
			// populate sheetSha256 from statement
			h := sha256.Sum256(f.Statement)
			f.SheetSha256 = hex.EncodeToString(h[:])
		default:
		}

		// Version validation for versioned releases only.
		// Declarative releases can have multiple files with the same version.
		if releaseType == v1pb.Release_VERSIONED {
			if _, ok := versionSet[f.Version]; ok {
				return nil, errors.Errorf("found duplicate version %q", f.Version)
			}
			versionSet[f.Version] = struct{}{}
		}
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
