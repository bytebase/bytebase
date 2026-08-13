package v1

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
	"github.com/bytebase/bytebase/backend/component/iam"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// SavedQueryService implements the saved query service.
type SavedQueryService struct {
	v1connect.UnimplementedSavedQueryServiceHandler
	store      *store.Store
	iamManager *iam.Manager
}

// NewSavedQueryService creates a new SavedQueryService.
func NewSavedQueryService(store *store.Store, iamManager *iam.Manager) *SavedQueryService {
	return &SavedQueryService{
		store:      store,
		iamManager: iamManager,
	}
}

// CreateSavedQuery creates a new saved query. The bb.savedQueries.create
// permission is enforced at the interceptor; the store's insert transaction
// fences against a concurrent project purge (active project required).
func (s *SavedQueryService) CreateSavedQuery(
	ctx context.Context,
	req *connect.Request[v1pb.CreateSavedQueryRequest],
) (*connect.Response[v1pb.SavedQuery], error) {
	request := req.Msg
	if request.SavedQuery == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("saved query must be set"))
	}
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	projectResourceID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	database := ""
	if request.SavedQuery.Database != "" {
		database, err = s.validateSavedQueryDatabase(ctx, projectResourceID, request.SavedQuery.Database)
		if err != nil {
			return nil, err
		}
	}
	folder, err := store.NormalizeSavedQueryFolder(request.SavedQuery.Folder)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	savedQuery, err := s.store.CreateSavedQuery(ctx, convertToStoreSavedQueryMessage(projectResourceID, database, user.Email, folder, request.SavedQuery))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create saved query: %v", err))
	}
	return connect.NewResponse(convertToAPISavedQuery(savedQuery)), nil
}

// GetSavedQuery returns the requested saved query, cutoff the content if the content is too long and the `raw` flag in request is false.
func (s *SavedQueryService) GetSavedQuery(
	ctx context.Context,
	req *connect.Request[v1pb.GetSavedQueryRequest],
) (*connect.Response[v1pb.SavedQuery], error) {
	projectID, savedQueryID, err := common.GetProjectIDSavedQueryID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	savedQuery, err := s.findSavedQuery(ctx, projectID, savedQueryID, true /* loadFull */)
	if err != nil {
		return nil, err
	}

	ok, err := s.canReadSavedQuery(ctx, savedQuery)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access with error: %v", err))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("cannot access saved query %s", savedQuery.Title))
	}

	v1pbSavedQuery := convertToAPISavedQuery(savedQuery)
	return connect.NewResponse(v1pbSavedQuery), nil
}

// ListSavedQueries returns a list of saved queries.
func (s *SavedQueryService) ListSavedQueries(
	ctx context.Context,
	req *connect.Request[v1pb.ListSavedQueriesRequest],
) (*connect.Response[v1pb.ListSavedQueriesResponse], error) {
	request := req.Msg
	if request.PageSize < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("page size must be non-negative: %d", request.PageSize))
	}

	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	projectIDs := []string{projectID}
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	if projectID == "-" {
		projectIDs = nil
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
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

	filterQ, err := store.GetListSavedQueryFilter(request.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	savedQueryList, err := s.store.ListSavedQueries(ctx, &store.FindSavedQueryMessage{
		ProjectIDs:     projectIDs,
		Workspace:      workspaceID,
		PrincipalEmail: user.Email,
		FilterQ:        filterQ,
		Limit:          &limitPlusOne,
		Offset:         &offset.offset,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list saved queries: %v", err))
	}

	var nextPageToken string
	if len(savedQueryList) == limitPlusOne {
		savedQueryList = savedQueryList[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get next page token"))
		}
	}

	v1pbSavedQueries := make([]*v1pb.SavedQuery, 0, len(savedQueryList))
	for _, savedQuery := range savedQueryList {
		v1pbSavedQuery := convertToAPISavedQuery(savedQuery)
		v1pbSavedQueries = append(v1pbSavedQueries, v1pbSavedQuery)
	}
	return connect.NewResponse(&v1pb.ListSavedQueriesResponse{
		SavedQueries:  v1pbSavedQueries,
		NextPageToken: nextPageToken,
	}), nil
}

// SearchSavedQueryFolders returns the caller's saved query folder paths in a
// project, including every ancestor prefix.
func (s *SavedQueryService) SearchSavedQueryFolders(
	ctx context.Context,
	req *connect.Request[v1pb.SearchSavedQueryFoldersRequest],
) (*connect.Response[v1pb.SearchSavedQueryFoldersResponse], error) {
	request := req.Msg
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if projectID == "-" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(`SearchSavedQueryFolders does not support parent "projects/-"`))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	if err := s.checkDiscoverSavedQueries(ctx, user, projectID); err != nil {
		return nil, err
	}

	filterQ, err := store.GetSearchSavedQueryFilter(ctx, s.store, user.Email, request.Filter, false /* allowTitleContains */)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Folders are derived from rows, so they inherit the rows' read rule.
	// Without the admin backstop the caller only ever sees folders holding
	// their own saved queries, whatever the filter asks for.
	canManage, err := s.iamManager.CheckPermission(ctx, permission.SavedQueriesManage, user, common.GetWorkspaceIDFromContext(ctx), projectID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
	}
	creator := &user.Email
	if canManage {
		creator = nil
	}

	paths, err := s.store.ListSavedQueryFolderPaths(ctx, projectID, creator, filterQ)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to search saved query folders: %v", err))
	}

	// Expand every ancestor prefix so the tree can render intermediate
	// folders that only exist through their children.
	prefixSet := make(map[string]bool)
	for _, path := range paths {
		segments := strings.Split(path, "/")
		for i := range segments {
			prefixSet[strings.Join(segments[:i+1], "/")] = true
		}
	}
	folders := make([]string, 0, len(prefixSet))
	for folder := range prefixSet {
		folders = append(folders, folder)
	}
	slices.Sort(folders)

	return connect.NewResponse(&v1pb.SearchSavedQueryFoldersResponse{
		Folders: folders,
	}), nil
}

// SearchSavedQueries returns a list of saved queries based on the search filters.
func (s *SavedQueryService) SearchSavedQueries(
	ctx context.Context,
	req *connect.Request[v1pb.SearchSavedQueriesRequest],
) (*connect.Response[v1pb.SearchSavedQueriesResponse], error) {
	request := req.Msg
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if projectID == "-" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(`SearchSavedQueries does not support parent "projects/-"`))
	}
	if err := s.checkDiscoverSavedQueries(ctx, user, projectID); err != nil {
		return nil, err
	}
	if request.PageSize < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("page size cannot be negative"))
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

	savedQueryFind := &store.FindSavedQueryMessage{
		ProjectIDs:     []string{projectID},
		PrincipalEmail: user.Email,
		Limit:          &limitPlusOne,
		Offset:         &offset.offset,
	}

	filterQ, err := store.GetSearchSavedQueryFilter(ctx, s.store, user.Email, request.Filter, true /* allowTitleContains */)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	savedQueryFind.FilterQ = filterQ

	savedQueryList, err := s.store.ListSavedQueries(ctx, savedQueryFind)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list saved queries: %v", err))
	}
	nextPageToken := ""
	if len(savedQueryList) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to generate next page token: %v", err))
		}
		savedQueryList = savedQueryList[:offset.limit]
	}

	var v1pbSavedQueries []*v1pb.SavedQuery
	for _, savedQuery := range savedQueryList {
		ok, err := s.canReadSavedQuery(ctx, savedQuery)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access with error: %v", err))
		}
		if !ok {
			slog.Warn("cannot access saved query", slog.String("name", savedQuery.Title))
			continue
		}
		v1pbSavedQuery := convertToAPISavedQuery(savedQuery)
		v1pbSavedQueries = append(v1pbSavedQueries, v1pbSavedQuery)
	}
	return connect.NewResponse(&v1pb.SearchSavedQueriesResponse{
		SavedQueries:  v1pbSavedQueries,
		NextPageToken: nextPageToken,
	}), nil
}

// UpdateSavedQuery updates a saved query.
func (s *SavedQueryService) UpdateSavedQuery(
	ctx context.Context,
	req *connect.Request[v1pb.UpdateSavedQueryRequest],
) (*connect.Response[v1pb.SavedQuery], error) {
	request := req.Msg
	if request.SavedQuery == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("saved query cannot be empty"))
	}
	if request.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update mask cannot be empty"))
	}
	if request.SavedQuery.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("saved query name cannot be empty"))
	}

	projectID, savedQueryID, err := common.GetProjectIDSavedQueryID(request.SavedQuery.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	savedQuery, err := s.findSavedQuery(ctx, projectID, savedQueryID, false /* loadFull */)
	if err != nil {
		return nil, err
	}
	ok, err = s.canWriteSavedQuery(ctx, savedQuery)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access with error: %v", err))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("cannot write saved query %s", savedQuery.Title))
	}

	savedQueryPatch := &store.PatchSavedQueryMessage{
		ResourceID: savedQuery.ResourceID,
	}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			savedQueryPatch.Title = &request.SavedQuery.Title
		case "content":
			statement := string(request.SavedQuery.Content)
			savedQueryPatch.Statement = &statement
		case "folder":
			folder, err := store.NormalizeSavedQueryFolder(request.SavedQuery.Folder)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			savedQueryPatch.Folder = &folder
		case "database":
			switch request.SavedQuery.Database {
			case savedQuery.Database:
				// Unchanged — skip validation entirely. The reference is a
				// soft link: autosave re-sends the stored value on every
				// write, and re-validating would brick content saves once
				// the database is deleted or transferred. A stale stored
				// value keeps dangling, exactly like the read path.
			case "":
				emptyStr := ""
				savedQueryPatch.Database = &emptyStr
			default:
				database, err := s.validateSavedQueryDatabase(ctx, savedQuery.ProjectID, request.SavedQuery.Database)
				if err != nil {
					return nil, err
				}
				savedQueryPatch.Database = &database
			}
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update mask path %q", path))
		}
	}
	if err := s.store.PatchSavedQuery(ctx, savedQueryPatch); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to update saved query: %v", err))
	}

	savedQuery, err = s.store.GetSavedQuery(ctx, &store.FindSavedQueryMessage{
		ProjectIDs:     []string{savedQuery.ProjectID},
		ResourceID:     &savedQueryID,
		PrincipalEmail: user.Email,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get saved query: %v", err))
	}
	if savedQuery == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("saved query %q not found", request.SavedQuery.Name))
	}
	v1pbSavedQuery := convertToAPISavedQuery(savedQuery)
	return connect.NewResponse(v1pbSavedQuery), nil
}

// DeleteSavedQuery deletes a saved query.
func (s *SavedQueryService) DeleteSavedQuery(
	ctx context.Context,
	req *connect.Request[v1pb.DeleteSavedQueryRequest],
) (*connect.Response[emptypb.Empty], error) {
	projectID, savedQueryID, err := common.GetProjectIDSavedQueryID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	savedQuery, err := s.findSavedQuery(ctx, projectID, savedQueryID, false /* loadFull */)
	if err != nil {
		return nil, err
	}
	ok, err := s.canWriteSavedQuery(ctx, savedQuery)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access with error: %v", err))
	}
	if !ok {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("cannot write saved query %s", savedQuery.Title))
	}

	if err := s.store.DeleteSavedQuery(ctx, savedQuery.ResourceID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to delete saved query: %v", err))
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// UpdateSavedQueryStar stars or unstars a saved query for the caller. A
// star is always the caller's own per-user marker; the gate is read access
// to the saved query being starred.
func (s *SavedQueryService) UpdateSavedQueryStar(
	ctx context.Context,
	req *connect.Request[v1pb.UpdateSavedQueryStarRequest],
) (*connect.Response[v1pb.SavedQuery], error) {
	request := req.Msg
	projectID, savedQueryID, err := common.GetProjectIDSavedQueryID(request.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	savedQuery, err := s.findSavedQuery(ctx, projectID, savedQueryID, false /* loadFull */)
	if err != nil {
		return nil, err
	}
	ok, err := s.canReadSavedQuery(ctx, savedQuery)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access with error: %v", err))
	}
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot find the saved query"))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	applied, err := s.store.SetSavedQueryStar(ctx, savedQuery.ResourceID, user.Email, request.Starred)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to update star: %v", err))
	}
	if !applied {
		// Deleted between the read above and the star write.
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot find the saved query"))
	}

	savedQuery.Starred = request.Starred
	return connect.NewResponse(convertToAPISavedQuery(savedQuery)), nil
}

// BatchUpdateSavedQueries re-files the matched saved queries into a folder.
// Only rows the caller may re-file are updated: their own, or any in scope
// for admins.
func (s *SavedQueryService) BatchUpdateSavedQueries(
	ctx context.Context,
	req *connect.Request[v1pb.BatchUpdateSavedQueriesRequest],
) (*connect.Response[v1pb.BatchUpdateSavedQueriesResponse], error) {
	request := req.Msg
	if request.SavedQuery == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("saved_query cannot be empty"))
	}
	if request.UpdateMask == nil || len(request.UpdateMask.Paths) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update mask cannot be empty"))
	}
	for _, path := range request.UpdateMask.Paths {
		if path != "folder" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid update mask path %q; only \"folder\" is supported", path))
		}
	}

	// Normalize before any work: a folder the filter could never match must
	// not be written to a single row.
	folder, err := store.NormalizeSavedQueryFolder(request.SavedQuery.Folder)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if projectID == "-" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("projects/- is not supported for batch update saved queries"))
	}

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	filterQ, err := store.GetSearchSavedQueryFilter(ctx, s.store, user.Email, request.Filter, false /* allowTitleContains */)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	savedQueryList, err := s.store.ListSavedQueries(ctx, &store.FindSavedQueryMessage{
		ProjectIDs:     []string{projectID},
		PrincipalEmail: user.Email,
		FilterQ:        filterQ,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to list saved queries: %v", err))
	}

	var savedQueryIDs []string
	for _, savedQuery := range savedQueryList {
		ok, err := s.canWriteSavedQuery(ctx, savedQuery)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check access with error: %v", err))
		}
		if !ok {
			continue
		}
		savedQueryIDs = append(savedQueryIDs, savedQuery.ResourceID)
	}

	updated, err := s.store.BatchUpdateSavedQueryFolder(ctx, savedQueryIDs, folder)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to batch update saved queries: %v", err))
	}
	return connect.NewResponse(&v1pb.BatchUpdateSavedQueriesResponse{
		UpdatedCount: int32(updated),
	}), nil
}

// checkDiscoverSavedQueries gates the per-project discovery surfaces
// (Search, folder list): bb.savedQueries.search, with bb.savedQueries.manage
// as the admin backstop that must be able to enumerate what it manages.
func (s *SavedQueryService) checkDiscoverSavedQueries(ctx context.Context, user *store.UserMessage, projectID string) error {
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	for _, p := range []permission.Permission{permission.SavedQueriesSearch, permission.SavedQueriesManage} {
		ok, err := s.iamManager.CheckPermission(ctx, p, user, workspaceID, projectID)
		if err != nil {
			return connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err))
		}
		if ok {
			return nil
		}
	}
	return connect.NewError(connect.CodePermissionDenied, errors.Errorf("permission denied: %s", permission.SavedQueriesSearch))
}

func (s *SavedQueryService) findSavedQuery(ctx context.Context, projectID string, savedQueryID string, loadFull bool) (*store.SavedQueryMessage, error) {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	savedQuery, err := s.store.GetSavedQuery(ctx, &store.FindSavedQueryMessage{
		ProjectIDs:     []string{projectID},
		ResourceID:     &savedQueryID,
		LoadFull:       loadFull,
		PrincipalEmail: user.Email,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get saved query: %v", err))
	}
	if savedQuery == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot find the saved query"))
	}
	return savedQuery, nil
}

// validateSavedQueryDatabase checks that `name` refers to an existing
// database in the saved query's own project and is canonical for its
// instance's scope (project form for a project instance, workspace form
// otherwise), then returns it as the canonical name to store. The stored
// reference is soft: it is validated only at write time and may dangle
// afterwards.
func (s *SavedQueryService) validateSavedQueryDatabase(ctx context.Context, projectID, name string) (string, error) {
	targetProjectID, instanceID, databaseName, err := common.GetDatabaseResourceName(name)
	if err != nil {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", name))
	}
	if targetProjectID != nil && *targetProjectID != projectID {
		return "", connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found in project %q", name, projectID))
	}
	database, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		Workspace:    common.GetWorkspaceIDFromContext(ctx),
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return "", connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
	}
	if database == nil || database.ProjectID != projectID {
		return "", connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found in project %q", name, projectID))
	}
	instance, err := s.store.GetInstance(ctx, &store.FindInstanceMessage{
		Workspace:  common.GetWorkspaceIDFromContext(ctx),
		ResourceID: &instanceID,
	})
	if err != nil {
		return "", connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get instance %q", instanceID))
	}
	if instance == nil || instance.Deleted {
		return "", connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", instanceID))
	}
	if (targetProjectID == nil) != (instance.ProjectID == nil) ||
		targetProjectID != nil && *targetProjectID != *instance.ProjectID {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("database name %q is not canonical for its instance", name))
	}
	return name, nil
}

// canWriteSavedQuery check if the principal can write the saved query.
// Saved queries are private: only the creator, or a caller holding
// "bb.savedQueries.manage" in scope (the admin backstop), can write.
func (s *SavedQueryService) canWriteSavedQuery(ctx context.Context, savedQuery *store.SavedQueryMessage) (bool, error) {
	return s.canAccessSavedQuery(ctx, savedQuery)
}

// canReadSavedQuery check if the principal can read the saved query.
// The access is the same as canWriteSavedQuery.
func (s *SavedQueryService) canReadSavedQuery(ctx context.Context, savedQuery *store.SavedQueryMessage) (bool, error) {
	return s.canAccessSavedQuery(ctx, savedQuery)
}

func (s *SavedQueryService) canAccessSavedQuery(ctx context.Context, savedQuery *store.SavedQueryMessage) (bool, error) {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return false, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	if savedQuery.Creator == user.Email {
		return true, nil
	}
	ok, err := s.iamManager.CheckPermission(ctx, permission.SavedQueriesManage, user, common.GetWorkspaceIDFromContext(ctx), savedQuery.ProjectID)
	if err != nil {
		return false, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
	}
	return ok, nil
}

// convertToAPISavedQuery converts a store message to the API shape. The
// stored database reference is already the canonical name and is returned
// as is — it is soft and may dangle, in which case the UI degrades to "no
// database" (no per-row resolution, no hard error).
func convertToAPISavedQuery(savedQuery *store.SavedQueryMessage) *v1pb.SavedQuery {
	return &v1pb.SavedQuery{
		Name:        common.FormatSavedQuery(savedQuery.ProjectID, savedQuery.ResourceID),
		Project:     common.FormatProject(savedQuery.ProjectID),
		Database:    savedQuery.Database,
		Title:       savedQuery.Title,
		Creator:     fmt.Sprintf("users/%s", savedQuery.Creator),
		CreateTime:  timestamppb.New(savedQuery.CreatedAt),
		UpdateTime:  timestamppb.New(savedQuery.UpdatedAt),
		Content:     []byte(savedQuery.Statement),
		ContentSize: savedQuery.Size,
		Starred:     savedQuery.Starred,
		Folder:      savedQuery.Folder,
	}
}

func convertToStoreSavedQueryMessage(projectID string, database string, creator string, folder string, savedQuery *v1pb.SavedQuery) *store.SavedQueryMessage {
	return &store.SavedQueryMessage{
		ProjectID: projectID,
		Creator:   creator,
		Title:     savedQuery.Title,
		Statement: string(savedQuery.Content),
		Database:  database,
		Folder:    folder,
	}
}
