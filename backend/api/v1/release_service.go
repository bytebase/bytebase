package v1

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
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

	// Set defaults for release ID template and timezone if not provided.
	releaseIDTemplate := req.Msg.ReleaseIdTemplate
	if releaseIDTemplate == "" {
		releaseIDTemplate = "release_{date}-RC{iteration}"
	}
	releaseIDTimezone := req.Msg.ReleaseIdTimezone
	if releaseIDTimezone == "" {
		releaseIDTimezone = "UTC"
	}

	// Compute train from template and timezone.
	train, err := renderTrain(releaseIDTemplate, releaseIDTimezone, time.Now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to render train"))
	}

	releaseMessage := &store.ReleaseMessage{
		ProjectID: project.ResourceID,
		Train:     train,
		Category:  req.Msg.Release.Category,
		Payload: &storepb.ReleasePayload{
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
	projectID, releaseID, err := common.GetProjectReleaseID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get release id"))
	}
	releaseMessage, err := s.store.GetRelease(ctx, &store.FindReleaseMessage{
		ProjectID: &projectID,
		ReleaseID: &releaseID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get release"))
	}
	if releaseMessage == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("release %s not found in project %s", releaseID, projectID))
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

	// Parse filter for category
	if req.Msg.Filter != "" {
		category, err := parseCategoryFilter(req.Msg.Filter)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid filter"))
		}
		if category != "" {
			releaseFind.Category = &category
		}
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

func (*ReleaseService) UpdateRelease(_ context.Context, _ *connect.Request[v1pb.UpdateReleaseRequest]) (*connect.Response[v1pb.Release], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.Errorf("update release is not supported, create a new release iteration instead"))
}

func (s *ReleaseService) DeleteRelease(ctx context.Context, req *connect.Request[v1pb.DeleteReleaseRequest]) (*connect.Response[emptypb.Empty], error) {
	projectID, releaseID, err := common.GetProjectReleaseID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get release id"))
	}
	release, err := s.store.GetRelease(ctx, &store.FindReleaseMessage{
		ProjectID: &projectID,
		ReleaseID: &releaseID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get release"))
	}
	if release == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("release %s not found in project %s", releaseID, projectID))
	}

	if _, err := s.store.UpdateRelease(ctx, &store.UpdateReleaseMessage{
		ProjectID: projectID,
		ReleaseID: releaseID,
		Deleted:   &deletePatch,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to delete release"))
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *ReleaseService) UndeleteRelease(ctx context.Context, req *connect.Request[v1pb.UndeleteReleaseRequest]) (*connect.Response[v1pb.Release], error) {
	projectID, releaseID, err := common.GetProjectReleaseID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get release id"))
	}
	release, err := s.store.GetRelease(ctx, &store.FindReleaseMessage{
		ProjectID:   &projectID,
		ReleaseID:   &releaseID,
		ShowDeleted: true,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get release"))
	}
	if release == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("release %s not found in project %s", releaseID, projectID))
	}

	releaseMessage, err := s.store.UpdateRelease(ctx, &store.UpdateReleaseMessage{
		ProjectID: projectID,
		ReleaseID: releaseID,
		Deleted:   &undeletePatch,
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
		Name:       common.FormatReleaseName(release.ProjectID, release.ReleaseID),
		Category:   release.Category,
		Creator:    common.FormatUserEmail(release.Creator),
		CreateTime: timestamppb.New(release.At),
		VcsSource:  convertToReleaseVcsSource(release.Payload.VcsSource),
		State:      convertDeletedToState(release.Deleted),
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

func (s *ReleaseService) ListReleaseCategories(ctx context.Context, req *connect.Request[v1pb.ListReleaseCategoriesRequest]) (*connect.Response[v1pb.ListReleaseCategoriesResponse], error) {
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

	categories, err := s.store.ListReleaseCategories(ctx, project.ResourceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list release categories"))
	}

	return connect.NewResponse(&v1pb.ListReleaseCategoriesResponse{
		Categories: categories,
	}), nil
}

// parseCategoryFilter parses CEL filter expression and extracts the category value.
// Supports: category == "value"
func parseCategoryFilter(filter string) (string, error) {
	if filter == "" {
		return "", nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return "", errors.Wrap(err, "failed to create CEL environment")
	}

	ast, iss := e.Parse(filter)
	if iss != nil {
		return "", errors.Errorf("failed to parse filter: %v", iss.String())
	}

	category, err := extractCategoryFromExpr(ast.NativeRep().Expr())
	if err != nil {
		return "", errors.Wrap(err, "failed to extract category")
	}

	return category, nil
}

// extractCategoryFromExpr walks the CEL AST to extract the category value.
func extractCategoryFromExpr(expr celast.Expr) (string, error) {
	switch expr.Kind() {
	case celast.CallKind:
		call := expr.AsCall()
		functionName := call.FunctionName()

		// Handle: category == "value"
		if functionName == celoperators.Equals {
			variable, value := getVariableAndValueFromExpr(expr)
			if variable == "category" {
				if categoryValue, ok := value.(string); ok {
					return categoryValue, nil
				}
				return "", errors.Errorf("category value must be a string, got %T", value)
			}
			return "", errors.Errorf("unsupported filter variable: %s", variable)
		}

		return "", errors.Errorf("unsupported operator: %s (only '==' is supported)", functionName)

	default:
		return "", errors.Errorf("unsupported expression type")
	}
}

func renderTrain(template, timezone string, t time.Time) (string, error) {
	// Validate template
	if err := validateTemplate(template); err != nil {
		return "", err
	}

	// Validate timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return "", errors.Wrapf(err, "invalid timezone: %s", timezone)
	}

	now := t.In(loc)

	train := template
	train = strings.ReplaceAll(train, "{date}", now.Format("20060102"))
	train = strings.ReplaceAll(train, "{time}", now.Format("1504"))
	train = strings.ReplaceAll(train, "{timestamp}", now.Format("20060102_1504"))
	train = strings.ReplaceAll(train, "{iteration}", "")

	return train, nil
}

func validateTemplate(template string) error {
	// Must contain {iteration}
	if !strings.Contains(template, "{iteration}") {
		return errors.New("template must contain {iteration} placeholder")
	}

	// {iteration} must be at the end of the template
	if !strings.HasSuffix(template, "{iteration}") {
		return errors.New("{iteration} must be at the end of the template")
	}

	// Must contain at least one time variable
	hasTimeVar := strings.Contains(template, "{date}") ||
		strings.Contains(template, "{time}") ||
		strings.Contains(template, "{timestamp}")
	if !hasTimeVar {
		return errors.New("template must contain at least one of: {date}, {time}, {timestamp}")
	}

	return nil
}
