package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	vcsPlugin "github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// SheetService implements the sheet service.
type SheetService struct {
	v1pb.UnimplementedSheetServiceServer
	store          *store.Store
	licenseService enterpriseAPI.LicenseService
}

// NewSheetService creates a new SheetService.
func NewSheetService(store *store.Store, licenseService enterpriseAPI.LicenseService) *SheetService {
	return &SheetService{
		store:          store,
		licenseService: licenseService,
	}
}

// CreateSheet creates a new sheet.
func (s *SheetService) CreateSheet(ctx context.Context, request *v1pb.CreateSheetRequest) (*v1pb.Sheet, error) {
	if request.Sheet == nil {
		return nil, status.Errorf(codes.InvalidArgument, "sheet must be set")
	}
	currentPrincipalID := ctx.Value(common.PrincipalIDContextKey).(int)

	projectResourceID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get project with resource id %q, err: %s", projectResourceID, err.Error()))
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q not found", projectResourceID))
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q had deleted", projectResourceID))
	}

	var databaseUID *int
	if request.Sheet.Database != "" {
		instanceResourceID, databaseName, err := common.GetInstanceDatabaseID(request.Sheet.Database)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
			ResourceID: &instanceResourceID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get instance with resource id %q, err: %s", instanceResourceID, err.Error()))
		}
		if instance == nil {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("instance with resource id %q not found", instanceResourceID))
		}

		find := &store.FindDatabaseMessage{
			ProjectID:  &projectResourceID,
			InstanceID: &instanceResourceID,
		}
		// It's chaos. We return /instance/{resource id}/databases/{uid} database in find sheet request,
		// but the frontend use both /instance/{resource id}/databases/{uid} and /instance/{resource id}/databases/{name}, sometimes the name will convert to int id incorrectly.
		// For database v1 api, we should only use the /instance/{resource id}/databases/{name}
		// We need to remove legacy code after the migration.
		dbUID, isNumber := isNumber(databaseName)
		if isNumber {
			find.UID = &dbUID
		} else {
			find.DatabaseName = &databaseName
			find.IgnoreCaseSensitive = store.IgnoreDatabaseAndTableCaseSensitive(instance)
		}

		database, err := s.store.GetDatabaseV2(ctx, find)
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get database with name %q, err: %s", databaseName, err.Error()))
		}
		if database == nil {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("database with name %q not found in project %q instance %q", databaseName, projectResourceID, instanceResourceID))
		}
		databaseUID = &database.UID
	}
	storeSheetCreate, err := convertToStoreSheetMessage(project.UID, databaseUID, currentPrincipalID, request.Sheet)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("failed to convert sheet: %v", err))
	}
	sheet, err := s.store.CreateSheet(ctx, storeSheetCreate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to create sheet: %v", err))
	}
	v1pbSheet, err := s.convertToAPISheetMessage(ctx, sheet)
	if err != nil {
		return nil, err
	}
	return v1pbSheet, nil
}

// GetSheet returns the requested sheet, cutoff the content if the content is too long and the `raw` flag in request is false.
func (s *SheetService) GetSheet(ctx context.Context, request *v1pb.GetSheetRequest) (*v1pb.Sheet, error) {
	projectResourceID, sheetID, err := common.GetProjectResourceIDSheetID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	sheetIntID, err := strconv.Atoi(sheetID)
	if err != nil || sheetIntID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid sheet id %s, must be positive integer", sheetID))
	}

	find := &store.FindSheetMessage{
		UID:      &sheetIntID,
		LoadFull: request.Raw,
	}

	// this allows get the sheet only by the id: /projects/-/sheets/{sheet uid}.
	// so that we can easily get the sheet from the issue.
	// we can remove this after migrate the issue to v1 API.
	if projectResourceID != "-" {
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID: &projectResourceID,
		})
		if err != nil {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %s not found", projectResourceID))
		}
		if project == nil {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %s not found", projectResourceID))
		}
		if project.Deleted {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q had deleted", projectResourceID))
		}
		find.ProjectUID = &project.UID
	}

	sheet, err := s.findSheet(ctx, find)
	if err != nil {
		return nil, err
	}

	canAccess, err := s.canReadSheet(ctx, sheet)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to check access with error: %v", err))
	}
	if !canAccess {
		return nil, status.Errorf(codes.PermissionDenied, "cannot access sheet %s", sheet.Name)
	}

	v1pbSheet, err := s.convertToAPISheetMessage(ctx, sheet)
	if err != nil {
		return nil, err
	}
	return v1pbSheet, nil
}

// SearchSheets returns a list of sheets based on the search filters.
func (s *SheetService) SearchSheets(ctx context.Context, request *v1pb.SearchSheetsRequest) (*v1pb.SearchSheetsResponse, error) {
	projectResourceID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	currentPrincipalID := ctx.Value(common.PrincipalIDContextKey).(int)

	sheetFind := &store.FindSheetMessage{}
	if projectResourceID != "-" {
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID: &projectResourceID,
		})
		if err != nil {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %s not found", projectResourceID))
		}
		if project.Deleted {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q had deleted", projectResourceID))
		}
		sheetFind.ProjectUID = &project.UID
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
			creatorEmail := strings.TrimPrefix(spec.value, "users/")
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
				sheetFind.Visibilities = []store.SheetVisibility{store.ProjectSheet, store.PublicSheet, store.PrivateSheet}
			case comparatorTypeNotEqual:
				sheetFind.ExcludedCreatorID = &user.ID
				sheetFind.Visibilities = []store.SheetVisibility{store.ProjectSheet, store.PublicSheet}
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
	sheetList, err := s.store.ListSheets(ctx, sheetFind, currentPrincipalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to list sheets: %v", err))
	}

	var v1pbSheets []*v1pb.Sheet
	for _, sheet := range sheetList {
		canAccess, err := s.canReadSheet(ctx, sheet)
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to check access with error: %v", err))
		}
		if !canAccess {
			log.Warn("cannot access sheet", zap.String("name", sheet.Name))
			continue
		}
		v1pbSheet, err := s.convertToAPISheetMessage(ctx, sheet)
		if err != nil {
			st := status.Convert(err)
			if st.Code() == codes.NotFound {
				log.Debug("failed to found resource for sheet", zap.Error(err), zap.Int("id", sheet.UID), zap.Int("project", sheet.ProjectUID))
				continue
			}
			return nil, err
		}
		v1pbSheets = append(v1pbSheets, v1pbSheet)
	}
	return &v1pb.SearchSheetsResponse{
		Sheets: v1pbSheets,
	}, nil
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

	projectResourceID, sheetID, err := common.GetProjectResourceIDSheetID(request.Sheet.Name)
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
	if project == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %s not found", projectResourceID))
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q had deleted", projectResourceID))
	}

	currentPrincipalID := ctx.Value(common.PrincipalIDContextKey).(int)
	sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{
		UID:        &sheetIntID,
		ProjectUID: &project.UID,
	}, currentPrincipalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get sheet: %v", err))
	}
	if sheet == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("sheet with id %d not found", sheetIntID))
	}
	canAccess, err := s.canWriteSheet(ctx, sheet)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to check access with error: %v", err))
	}
	if !canAccess {
		return nil, status.Errorf(codes.PermissionDenied, "cannot write sheet %s", sheet.Name)
	}

	sheetPatch := &store.PatchSheetMessage{
		UID:       sheet.UID,
		UpdaterID: currentPrincipalID,
	}

	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			sheetPatch.Name = &request.Sheet.Title
		case "content":
			statement := string(request.Sheet.Content)
			sheetPatch.Statement = &statement
		case "visibility":
			visibility, err := convertToStoreSheetVisibility(request.Sheet.Visibility)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid visibility %q", request.Sheet.Visibility))
			}
			stringVisibility := string(visibility)
			sheetPatch.Visibility = &stringVisibility
		case "payload":
			sheetPatch.Payload = &request.Sheet.Payload
		default:
			return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid update mask path %q", path))
		}
	}
	storeSheet, err := s.store.PatchSheet(ctx, sheetPatch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to update sheet: %v", err))
	}
	v1pbSheet, err := s.convertToAPISheetMessage(ctx, storeSheet)
	if err != nil {
		return nil, err
	}

	return v1pbSheet, nil
}

// DeleteSheet deletes a sheet.
func (s *SheetService) DeleteSheet(ctx context.Context, request *v1pb.DeleteSheetRequest) (*emptypb.Empty, error) {
	projectResourceID, sheetID, err := common.GetProjectResourceIDSheetID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	sheetIDInt, err := strconv.Atoi(sheetID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid sheet id %s", sheetID))
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get project with resource id %q, err: %s", projectResourceID, err.Error()))
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q not found", projectResourceID))
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q had deleted", projectResourceID))
	}

	currentPrincipalID := ctx.Value(common.PrincipalIDContextKey).(int)

	sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{
		UID:        &sheetIDInt,
		ProjectUID: &project.UID,
	}, currentPrincipalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get sheet: %v", err))
	}
	if sheet == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("sheet with id %d not found", sheetIDInt))
	}
	canAccess, err := s.canWriteSheet(ctx, sheet)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to check access with error: %v", err))
	}
	if !canAccess {
		return nil, status.Errorf(codes.PermissionDenied, "cannot write sheet %s", sheet.Name)
	}

	if err := s.store.DeleteSheet(ctx, sheetIDInt); err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to delete sheet: %v", err))
	}

	return &emptypb.Empty{}, nil
}

// SyncSheets syncs sheets from VCS.
func (s *SheetService) SyncSheets(ctx context.Context, request *v1pb.SyncSheetsRequest) (*emptypb.Empty, error) {
	// TODO(tianzhou): uncomment this after adding the test harness to using Enterprise version.
	// if !s.licenseService.IsFeatureEnabled(api.FeatureVCSSheetSync) {
	// 	return echo.NewHTTPError(http.StatusForbidden, api.FeatureVCSSheetSync.AccessErrorMessage())
	// }
	currentPrincipalID := ctx.Value(common.PrincipalIDContextKey).(int)

	projectResourceID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &projectResourceID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get project with resource id %q, err: %s", projectResourceID, err.Error()))
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q not found", projectResourceID))
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with resource id %q had deleted", projectResourceID))
	}
	if project.Workflow != api.VCSWorkflow {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("project with resource id %q is not a VCS enabled project", projectResourceID))
	}

	projectRoles, err := s.findProjectRoles(ctx, project.UID, currentPrincipalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to find roles in the project: %v", err))
	}
	if !api.ProjectPermission(api.ProjectPermissionSyncSheet, s.licenseService.GetEffectivePlan(), projectRoles) {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied to sync sheet for project")
	}

	repo, err := s.store.GetRepositoryV2(ctx, &store.FindRepositoryMessage{ProjectResourceID: &project.ResourceID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to find repository for sync sheet: %d", project.UID))
	}
	if repo == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("repository not found for sync sheet: %d", project.UID))
	}

	vcs, err := s.store.GetExternalVersionControlV2(ctx, repo.VCSUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to find VCS for sync sheet, VCSID: %d", repo.VCSUID))
	}
	if vcs == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("VCS not found for sync sheet: %d", repo.VCSUID))
	}

	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find workspace setting with error: %v", err.Error())
	}
	if setting.ExternalUrl == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "external url is required")
	}

	basePath := filepath.Dir(repo.SheetPathTemplate)
	// TODO(Steven): The repo.branchFilter could be `test/*` which cannot be the ref value.
	// TODO(zp): We may need a need VCS interface to get fetch repository file list for a branch instead of a SHA1.
	fileList, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{}).FetchRepositoryFileList(ctx,
		common.OauthContext{
			ClientID:     vcs.ApplicationID,
			ClientSecret: vcs.Secret,
			AccessToken:  repo.AccessToken,
			RefreshToken: repo.RefreshToken,
			Refresher:    utils.RefreshToken(ctx, s.store, repo.WebURL),
			RedirectURL:  fmt.Sprintf("%s/oauth/callback", setting.ExternalUrl),
		},
		vcs.InstanceURL,
		repo.ExternalID,
		repo.BranchFilter,
		basePath,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to fetch repository file list for sync sheet: %d", project.UID))
	}

	for _, file := range fileList {
		sheetInfo, err := parseSheetInfo(file.Path, repo.SheetPathTemplate)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to parse sheet info from template")
		}
		if sheetInfo.SheetName == "" {
			return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("sheet name cannot be empty from sheet path %s with template %s", file.Path, repo.SheetPathTemplate))
		}

		fileContent, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{}).ReadFileContent(ctx,
			common.OauthContext{
				ClientID:     vcs.ApplicationID,
				ClientSecret: vcs.Secret,
				AccessToken:  repo.AccessToken,
				RefreshToken: repo.RefreshToken,
				Refresher:    utils.RefreshToken(ctx, s.store, repo.WebURL),
				RedirectURL:  fmt.Sprintf("%s/oauth/callback", setting.ExternalUrl),
			},
			vcs.InstanceURL,
			repo.ExternalID,
			file.Path,
			repo.BranchFilter,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Failed to fetch file content from VCS, instance URL: %s, repo ID: %s, file path: %s, branch: %s", vcs.InstanceURL, repo.ExternalID, file.Path, repo.BranchFilter))
		}

		fileMeta, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{}).ReadFileMeta(ctx,
			common.OauthContext{
				ClientID:     vcs.ApplicationID,
				ClientSecret: vcs.Secret,
				AccessToken:  repo.AccessToken,
				RefreshToken: repo.RefreshToken,
				Refresher:    utils.RefreshToken(ctx, s.store, repo.WebURL),
				RedirectURL:  fmt.Sprintf("%s/oauth/callback", setting.ExternalUrl),
			},
			vcs.InstanceURL,
			repo.ExternalID,
			file.Path,
			repo.BranchFilter,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Failed to fetch file meta from VCS, instance URL: %s, repo ID: %s, file path: %s, branch: %s", vcs.InstanceURL, repo.ExternalID, file.Path, repo.BranchFilter))
		}

		lastCommit, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{}).FetchCommitByID(ctx,
			common.OauthContext{
				ClientID:     vcs.ApplicationID,
				ClientSecret: vcs.Secret,
				AccessToken:  repo.AccessToken,
				RefreshToken: repo.RefreshToken,
				Refresher:    utils.RefreshToken(ctx, s.store, repo.WebURL),
				RedirectURL:  fmt.Sprintf("%s/oauth/callback", setting.ExternalUrl),
			},
			vcs.InstanceURL,
			repo.ExternalID,
			fileMeta.LastCommitID,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Failed to fetch commit data from VCS, instance URL: %s, repo ID: %s, commit ID: %s", vcs.InstanceURL, repo.ExternalID, fileMeta.LastCommitID))
		}

		sheetVCSPayload := &api.SheetVCSPayload{
			FileName:     fileMeta.Name,
			FilePath:     fileMeta.Path,
			Size:         fileMeta.Size,
			Author:       lastCommit.AuthorName,
			LastCommitID: lastCommit.ID,
			LastSyncTs:   time.Now().Unix(),
		}
		payload, err := json.Marshal(sheetVCSPayload)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Failed to marshal sheetVCSPayload")
		}

		var databaseID *int
		// In non-tenant mode, we can set a databaseId for sheet with ENV_ID and DB_NAME,
		// and ENV_ID and DB_NAME is either both present or neither present.
		if project.TenantMode != api.TenantModeDisabled {
			if sheetInfo.EnvironmentID != "" && sheetInfo.DatabaseName != "" {
				// The database name for PostgreSQL, Oracle, Snowflake and some databases are case sensitive.
				// But the database name for MySQL, TiDB and other databases are case insensitive.
				// So we should find databases by case-insensitive and double-check for case sensitive database engines.
				databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{
					ProjectID:           &project.ResourceID,
					DatabaseName:        &sheetInfo.DatabaseName,
					IgnoreCaseSensitive: true,
				})
				if err != nil {
					return nil, status.Errorf(codes.Internal, fmt.Sprintf("Failed to find database list with name: %s, project ID: %d", sheetInfo.DatabaseName, project.UID))
				}
				for _, database := range databases {
					database := database // create a new var "database".
					instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &database.InstanceID})
					if err != nil {
						return nil, status.Errorf(codes.Internal, fmt.Sprintf("Failed to find instance with ID: %s", database.InstanceID))
					}
					if !store.IgnoreDatabaseAndTableCaseSensitive(instance) && database.DatabaseName != sheetInfo.DatabaseName {
						continue
					}
					if database.EffectiveEnvironmentID == sheetInfo.EnvironmentID {
						databaseID = &database.UID
						break
					}
				}
			}
		}

		var sheetSource store.SheetSource
		switch vcs.Type {
		case vcsPlugin.GitLab:
			sheetSource = store.SheetFromGitLab
		case vcsPlugin.GitHub:
			sheetSource = store.SheetFromGitHub
		case vcsPlugin.Bitbucket:
			sheetSource = store.SheetFromBitbucket
		}
		vscSheetType := store.SheetForSQL
		sheetFind := &store.FindSheetMessage{
			Name:       &sheetInfo.SheetName,
			ProjectUID: &project.UID,
			Source:     &sheetSource,
			Type:       &vscSheetType,
		}
		sheet, err := s.store.GetSheet(ctx, sheetFind, currentPrincipalID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("Failed to find sheet with name: %s, project ID: %d", sheetInfo.SheetName, project.UID))
		}

		if sheet == nil {
			sheetCreate := &store.SheetMessage{
				ProjectUID: project.UID,
				CreatorID:  currentPrincipalID,
				Name:       sheetInfo.SheetName,
				Statement:  fileContent,
				Visibility: store.ProjectSheet,
				Source:     sheetSource,
				Type:       store.SheetForSQL,
				Payload:    string(payload),
			}
			if databaseID != nil {
				sheetCreate.DatabaseUID = databaseID
			}

			if _, err := s.store.CreateSheet(ctx, sheetCreate); err != nil {
				return nil, status.Errorf(codes.Internal, "Failed to create sheet from VCS")
			}
		} else {
			payloadString := string(payload)
			sheetPatch := store.PatchSheetMessage{
				UID:       sheet.UID,
				UpdaterID: currentPrincipalID,
				Statement: &fileContent,
				Payload:   &payloadString,
			}
			if databaseID != nil {
				sheetPatch.DatabaseUID = databaseID
			}

			if _, err := s.store.PatchSheet(ctx, &sheetPatch); err != nil {
				return nil, status.Errorf(codes.Internal, "Failed to patch sheet from VCS")
			}
		}
	}
	return &emptypb.Empty{}, nil
}

// UpdateSheetOrganizer upsert the sheet organizer.
func (s *SheetService) UpdateSheetOrganizer(ctx context.Context, request *v1pb.UpdateSheetOrganizerRequest) (*v1pb.SheetOrganizer, error) {
	_, sheetID, err := common.GetProjectResourceIDSheetID(request.Organizer.Sheet)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	sheetIntID, err := strconv.Atoi(sheetID)
	if err != nil || sheetIntID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid sheet id %s, must be positive integer", sheetID))
	}

	sheet, err := s.findSheet(ctx, &store.FindSheetMessage{
		UID: &sheetIntID,
	})
	if err != nil {
		return nil, err
	}

	canAccess, err := s.canReadSheet(ctx, sheet)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to check access with error: %v", err))
	}
	if !canAccess {
		return nil, status.Errorf(codes.PermissionDenied, "cannot access sheet %s", sheet.Name)
	}

	currentPrincipalID := ctx.Value(common.PrincipalIDContextKey).(int)
	sheetOrganizerUpsert := &store.SheetOrganizerMessage{
		SheetUID:     sheetIntID,
		PrincipalUID: currentPrincipalID,
	}

	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "starred":
			sheetOrganizerUpsert.Starred = request.Organizer.Starred
		case "pinned":
			sheetOrganizerUpsert.Pinned = request.Organizer.Pinned
		}
	}

	organizer, err := s.store.UpsertSheetOrganizerV2(ctx, sheetOrganizerUpsert)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upsert organizer for sheet %s with error: %v", request.Organizer.Sheet, err)
	}

	return &v1pb.SheetOrganizer{
		Sheet:   request.Organizer.Sheet,
		Starred: organizer.Starred,
		Pinned:  organizer.Pinned,
	}, nil
}

func (s *SheetService) findSheet(ctx context.Context, find *store.FindSheetMessage) (*store.SheetMessage, error) {
	currentPrincipalID := ctx.Value(common.PrincipalIDContextKey).(int)
	sheet, err := s.store.GetSheet(ctx, find, currentPrincipalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get sheet: %v", err))
	}
	if sheet == nil {
		return nil, status.Errorf(codes.NotFound, "cannot find the sheet")
	}
	return sheet, nil
}

// canWriteSheet check if the principal can write the sheet.
// sheet if writable when:
// PRIVATE: the creator only.
// PROJECT: the creator or project role can manage sheet, workspace Owner and DBA.
// PUBLIC: the creator only.
func (s *SheetService) canWriteSheet(ctx context.Context, sheet *store.SheetMessage) (bool, error) {
	currentPrincipalID := ctx.Value(common.PrincipalIDContextKey).(int)

	if sheet.CreatorID == currentPrincipalID {
		return true, nil
	}

	if sheet.Visibility == store.ProjectSheet {
		projectRoles, err := s.findProjectRoles(ctx, sheet.ProjectUID, currentPrincipalID)
		if err != nil {
			return false, err
		}
		if len(projectRoles) == 0 {
			return false, nil
		}
		return projectRoles[common.ProjectOwner], nil
	}

	return false, nil
}

// canReadSheet check if the principal can read the sheet.
// sheet is readable when:
// PRIVATE: the creator only.
// PROJECT: the creator and members in the project.
// PUBLIC: everyone in the workspace.
func (s *SheetService) canReadSheet(ctx context.Context, sheet *store.SheetMessage) (bool, error) {
	currentPrincipalID := ctx.Value(common.PrincipalIDContextKey).(int)
	role := ctx.Value(common.RoleContextKey).(api.Role)

	switch sheet.Visibility {
	case store.PrivateSheet:
		return sheet.CreatorID == currentPrincipalID, nil
	case store.PublicSheet:
		return true, nil
	case store.ProjectSheet:
		if role == api.Owner || role == api.DBA {
			return true, nil
		}
		projectRoles, err := s.findProjectRoles(ctx, sheet.ProjectUID, currentPrincipalID)
		if err != nil {
			return false, err
		}
		return len(projectRoles) > 0, nil
	}
	return false, nil
}

func (s *SheetService) findProjectRoles(ctx context.Context, projectUID int, principalUID int) (map[common.ProjectRole]bool, error) {
	policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &projectUID})
	if err != nil {
		return nil, err
	}
	projectRoles := make(map[common.ProjectRole]bool)
	for _, binding := range policy.Bindings {
		for _, member := range binding.Members {
			if member.ID == principalUID {
				projectRoles[common.ProjectRole(binding.Role)] = true
				break
			}
		}
	}
	return projectRoles, nil
}

func (s *SheetService) convertToAPISheetMessage(ctx context.Context, sheet *store.SheetMessage) (*v1pb.Sheet, error) {
	databaseParent := ""
	if sheet.DatabaseUID != nil {
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			UID: sheet.DatabaseUID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get database: %v", err))
		}
		if database == nil {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("database with id %d not found", *sheet.DatabaseUID))
		}
		databaseParent = fmt.Sprintf("%s%s/%s%d", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.UID)
	}

	visibility := v1pb.Sheet_VISIBILITY_UNSPECIFIED
	switch sheet.Visibility {
	case store.PublicSheet:
		visibility = v1pb.Sheet_VISIBILITY_PUBLIC
	case store.ProjectSheet:
		visibility = v1pb.Sheet_VISIBILITY_PROJECT
	case store.PrivateSheet:
		visibility = v1pb.Sheet_VISIBILITY_PRIVATE
	}

	source := v1pb.Sheet_SOURCE_UNSPECIFIED
	switch sheet.Source {
	case store.SheetFromBytebase:
		source = v1pb.Sheet_SOURCE_BYTEBASE
	case store.SheetFromBytebaseArtifact:
		source = v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT
	case store.SheetFromGitLab:
		source = v1pb.Sheet_SOURCE_GITLAB
	case store.SheetFromGitHub:
		source = v1pb.Sheet_SOURCE_GITHUB
	case store.SheetFromBitbucket:
		source = v1pb.Sheet_SOURCE_BITBUCKET
	}

	tp := v1pb.Sheet_TYPE_UNSPECIFIED
	switch sheet.Type {
	case store.SheetForSQL:
		tp = v1pb.Sheet_TYPE_SQL
	default:
	}

	creator, err := s.store.GetUserByID(ctx, sheet.CreatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get creator: %v", err))
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		UID: &sheet.ProjectUID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get project: %v", err))
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project with id %d not found", sheet.ProjectUID))
	}

	return &v1pb.Sheet{
		Name:        fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, project.ResourceID, common.SheetIDPrefix, sheet.UID),
		Database:    databaseParent,
		Title:       sheet.Name,
		Creator:     fmt.Sprintf("users/%s", creator.Email),
		CreateTime:  timestamppb.New(sheet.CreatedTime),
		UpdateTime:  timestamppb.New(sheet.UpdatedTime),
		Content:     []byte(sheet.Statement),
		ContentSize: sheet.Size,
		Visibility:  visibility,
		Source:      source,
		Type:        tp,
		Starred:     sheet.Starred,
		Payload:     sheet.Payload,
	}, nil
}

func convertToStoreSheetMessage(projectUID int, databaseUID *int, creatorID int, sheet *v1pb.Sheet) (*store.SheetMessage, error) {
	visibility, err := convertToStoreSheetVisibility(sheet.Visibility)
	if err != nil {
		return nil, err
	}
	var source store.SheetSource
	switch sheet.Source {
	case v1pb.Sheet_SOURCE_UNSPECIFIED:
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid source %q", sheet.Source))
	case v1pb.Sheet_SOURCE_BYTEBASE:
		source = store.SheetFromBytebase
	case v1pb.Sheet_SOURCE_BYTEBASE_ARTIFACT:
		source = store.SheetFromBytebaseArtifact
	case v1pb.Sheet_SOURCE_GITLAB:
		source = store.SheetFromGitLab
	case v1pb.Sheet_SOURCE_GITHUB:
		source = store.SheetFromGitHub
	case v1pb.Sheet_SOURCE_BITBUCKET:
		source = store.SheetFromBitbucket
	default:
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid source %q", sheet.Source))
	}
	var tp store.SheetType
	switch sheet.Type {
	case v1pb.Sheet_TYPE_UNSPECIFIED:
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid type %q", sheet.Type))
	case v1pb.Sheet_TYPE_SQL:
		tp = store.SheetForSQL
	default:
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid type %q", sheet.Type))
	}

	return &store.SheetMessage{
		ProjectUID:  projectUID,
		DatabaseUID: databaseUID,
		CreatorID:   creatorID,
		Name:        sheet.Title,
		Statement:   string(sheet.Content),
		Visibility:  visibility,
		Source:      source,
		Type:        tp,
		Payload:     sheet.Payload,
	}, nil
}

func convertToStoreSheetVisibility(visibility v1pb.Sheet_Visibility) (store.SheetVisibility, error) {
	switch visibility {
	case v1pb.Sheet_VISIBILITY_UNSPECIFIED:
		return store.SheetVisibility(""), status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid visibility %q", visibility))
	case v1pb.Sheet_VISIBILITY_PUBLIC:
		return store.PublicSheet, nil
	case v1pb.Sheet_VISIBILITY_PROJECT:
		return store.ProjectSheet, nil
	case v1pb.Sheet_VISIBILITY_PRIVATE:
		return store.PrivateSheet, nil
	default:
		return store.SheetVisibility(""), status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid visibility %q", visibility))
	}
}

// SheetInfo represents the sheet related information from sheetPathTemplate.
type SheetInfo struct {
	EnvironmentID string
	DatabaseName  string
	SheetName     string
}

// parseSheetInfo matches sheetPath against sheetPathTemplate. If sheetPath matches, then it will derive SheetInfo from the sheetPath.
// Both sheetPath and sheetPathTemplate are the full file path(including the base directory) of the repository.
func parseSheetInfo(sheetPath string, sheetPathTemplate string) (*SheetInfo, error) {
	placeholderList := []string{
		"ENV_ID",
		"DB_NAME",
		"NAME",
	}
	sheetPathRegex := sheetPathTemplate
	for _, placeholder := range placeholderList {
		sheetPathRegex = strings.ReplaceAll(sheetPathRegex, fmt.Sprintf("{{%s}}", placeholder), fmt.Sprintf("(?P<%s>[a-zA-Z0-9\\+\\-\\=\\_\\#\\!\\$\\. ]+)", placeholder))
	}
	sheetRegex, err := regexp.Compile(fmt.Sprintf("^%s$", sheetPathRegex))
	if err != nil {
		return nil, errors.Wrapf(err, "invalid sheet path template %q", sheetPathTemplate)
	}
	if !sheetRegex.MatchString(sheetPath) {
		return nil, errors.Errorf("sheet path %q does not match sheet path template %q", sheetPath, sheetPathTemplate)
	}

	matchList := sheetRegex.FindStringSubmatch(sheetPath)
	sheetInfo := &SheetInfo{}
	for _, placeholder := range placeholderList {
		index := sheetRegex.SubexpIndex(placeholder)
		if index >= 0 {
			switch placeholder {
			case "ENV_ID":
				sheetInfo.EnvironmentID = matchList[index]
			case "DB_NAME":
				sheetInfo.DatabaseName = matchList[index]
			case "NAME":
				sheetInfo.SheetName = matchList[index]
			}
		}
	}

	return sheetInfo, nil
}
