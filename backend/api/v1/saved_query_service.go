package v1

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/permission"
	"github.com/bytebase/bytebase/backend/component/iam"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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

	// Resource resolution returns archived projects (GetProject reports a
	// project regardless of its deleted state), so this is what rejects a
	// create into one. The store's fence covers the window after it.
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		Workspace:  common.GetWorkspaceIDFromContext(ctx),
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project %q", projectResourceID))
	}
	if project == nil || project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectResourceID))
	}

	// The interceptor already checked bb.savedQueries.create, but it evaluates
	// conditions with request.time only and passes the rest, so a grant scoped
	// to a slice of databases would satisfy it. A saved query is a project-wide
	// row; re-check the source here and reject those bindings.
	if err := s.checkProjectWide(ctx, user, permission.SavedQueriesCreate, projectResourceID); err != nil {
		return nil, err
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
		if common.ErrorCode(err) == common.NotFound {
			// The fence lost to an archive or purge racing this create.
			return nil, connect.NewError(connect.CodeNotFound, errors.Wrapf(err, "project %q not found", projectResourceID))
		}
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

	access, err := s.savedQueryAccess(ctx, savedQuery)
	if err != nil {
		return nil, err
	}
	if !access.canRead() {
		// Same answer as a name that does not exist: a saved query the caller
		// cannot read must not be probeable, by existence or by title.
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot find the saved query"))
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

	// Same source re-check as CreateSavedQuery: bb.savedQueries.list reads
	// everyone's content, which a database-scoped grant must not confer.
	if err := s.checkProjectWide(ctx, user, permission.SavedQueriesList, projectIDs...); err != nil {
		return nil, err
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
		// Governance reads are whole-statement reads. Search previews and
		// offers GetSavedQuery for the rest, but a caller holding only
		// bb.savedQueries.list cannot Get another creator's row, so a
		// truncated statement here would have no way back to the full one.
		LoadFull: true,
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

	accessMembers, err := s.iamManager.PrincipalMembers(ctx, common.GetWorkspaceIDFromContext(ctx), user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to resolve principals: %v", err))
	}

	filterQ, err := store.GetSearchSavedQueryFilter(ctx, s.store, user.Email, accessMembers, request.Filter, false /* allowTitleContains */)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Folders inherit the rows' read rule, exactly as SearchSavedQueries does:
	// the caller's own saved queries plus those a binding shares with them, and
	// no admin widening. The filter is what splits My (creator == me) from
	// Shared (shared == true) — scoping to the caller's own rows here instead
	// would hide a shared saved query its creator filed, since the client seeds
	// its folder tree from this call and cannot expand into a folder it never
	// learns about.
	paths, err := s.store.ListSavedQueryFolderPaths(ctx, projectID, user.Email, accessMembers, filterQ)
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

	// Search is caller-scoped: the caller's own saved queries plus those a
	// binding shares with them, and nothing else — the admin backstop does not
	// widen it. An admin wanting everyone's rows uses ListSavedQueries, the
	// surface built for that, which is gated on bb.savedQueries.list and
	// returns whole statements.
	//
	// The predicate goes into the query rather than filtering the page
	// afterwards, which would return short pages: nine of ten rows dropped
	// still consumes the whole page.
	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	accessMembers, err := s.iamManager.PrincipalMembers(ctx, workspaceID, user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to resolve principals: %v", err))
	}
	savedQueryFind.AccessMembers = accessMembers

	filterQ, err := store.GetSearchSavedQueryFilter(ctx, s.store, user.Email, accessMembers, request.Filter, true /* allowTitleContains */)
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

	v1pbSavedQueries := make([]*v1pb.SavedQuery, 0, len(savedQueryList))
	for _, savedQuery := range savedQueryList {
		v1pbSavedQueries = append(v1pbSavedQueries, convertToAPISavedQuery(savedQuery))
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
	access, err := s.savedQueryAccess(ctx, savedQuery)
	if err != nil {
		return nil, err
	}
	if !access.canRead() {
		// Invisible to this caller, so answer as if it were missing rather
		// than confirming the name exists.
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot find the saved query"))
	}
	if !access.canWrite() {
		// A VIEWER can already see the row, so there is nothing left to hide.
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("permission denied to update the saved query"))
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
			// Filing is organization rather than editing: you never re-file
			// someone else's saved query, whatever your binding says.
			if !access.canManage() {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("only the creator or an admin can re-file a saved query"))
			}
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
		// Only Search truncates. The client caches this response as its FULL
		// view, so returning a cut-off statement would silently shorten a
		// script larger than common.MaxSheetSize.
		LoadFull: true,
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
	access, err := s.savedQueryAccess(ctx, savedQuery)
	if err != nil {
		return nil, err
	}
	if !access.canRead() {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot find the saved query"))
	}
	// An EDITOR binding carries content writes, never deletion, so sharing a
	// saved query for editing cannot cost its owner the saved query.
	if !access.canManage() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("only the creator or an admin can delete a saved query"))
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

	savedQuery, err := s.findSavedQuery(ctx, projectID, savedQueryID, true /* loadFull */)
	if err != nil {
		return nil, err
	}
	access, err := s.savedQueryAccess(ctx, savedQuery)
	if err != nil {
		return nil, err
	}
	if !access.canRead() {
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

	accessMembers, err := s.iamManager.PrincipalMembers(ctx, common.GetWorkspaceIDFromContext(ctx), user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to resolve principals: %v", err))
	}

	filterQ, err := store.GetSearchSavedQueryFilter(ctx, s.store, user.Email, accessMembers, request.Filter, false /* allowTitleContains */)
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
		access, err := s.savedQueryAccess(ctx, savedQuery)
		if err != nil {
			return nil, err
		}
		// Re-filing is creator/admin only, and matched rows the caller may not
		// re-file are skipped rather than failing the batch — so the response
		// count is what changed, not what matched.
		if !access.canManage() {
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
// checkProjectWide re-checks an IAM-gated permission, rejecting bindings whose
// condition scopes resources. The generic interceptor cannot express this: it
// evaluates request.time and passes the residual condition through.
func (s *SavedQueryService) checkProjectWide(ctx context.Context, user *store.UserMessage, p permission.Permission, projectIDs ...string) error {
	ok, err := s.iamManager.CheckProjectWidePermission(ctx, p, user, common.GetWorkspaceIDFromContext(ctx), projectIDs...)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err))
	}
	if !ok {
		return connect.NewError(connect.CodePermissionDenied, errors.Errorf("permission denied: %s", p))
	}
	return nil
}

func (s *SavedQueryService) checkDiscoverSavedQueries(ctx context.Context, user *store.UserMessage, projectID string) error {
	// bb.savedQueries.search alone, not "search or manage". The search family
	// is caller-scoped, so manage would let a role through the gate to see
	// only its own rows — no use to it, and one more thing to explain. Admins
	// reach everyone's saved queries through ListSavedQueries instead, and
	// every predefined role holding manage holds search too.
	ok, err := s.iamManager.CheckProjectWidePermission(ctx, permission.SavedQueriesSearch, user, common.GetWorkspaceIDFromContext(ctx), projectID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err))
	}
	if !ok {
		return connect.NewError(connect.CodePermissionDenied, errors.Errorf("permission denied: %s", permission.SavedQueriesSearch))
	}
	return nil
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

// GetSavedQueryPolicy returns a saved query's grants. Anyone who can read the
// saved query can read its policy — that is how a grantee learns whether they
// may edit it, now that nothing caller-relative rides on the resource.
func (s *SavedQueryService) GetSavedQueryPolicy(
	ctx context.Context,
	req *connect.Request[v1pb.GetSavedQueryPolicyRequest],
) (*connect.Response[v1pb.SavedQueryPolicy], error) {
	projectID, savedQueryID, err := common.GetProjectIDSavedQueryID(req.Msg.Resource)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	savedQuery, err := s.findSavedQuery(ctx, projectID, savedQueryID, false /* loadFull */)
	if err != nil {
		return nil, err
	}
	access, err := s.savedQueryAccess(ctx, savedQuery)
	if err != nil {
		return nil, err
	}
	if !access.canRead() {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot find the saved query"))
	}

	policy, err := convertToAPISavedQueryPolicy(savedQuery.Bindings)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to build policy: %v", err))
	}
	return connect.NewResponse(policy), nil
}

// SetSavedQueryPolicy replaces a saved query's grants under compare-and-swap.
func (s *SavedQueryService) SetSavedQueryPolicy(
	ctx context.Context,
	req *connect.Request[v1pb.SetSavedQueryPolicyRequest],
) (*connect.Response[v1pb.SavedQueryPolicy], error) {
	request := req.Msg
	if request.Policy == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("policy cannot be empty"))
	}
	projectID, savedQueryID, err := common.GetProjectIDSavedQueryID(request.Resource)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	savedQuery, err := s.findSavedQuery(ctx, projectID, savedQueryID, false /* loadFull */)
	if err != nil {
		return nil, err
	}
	access, err := s.savedQueryAccess(ctx, savedQuery)
	if err != nil {
		return nil, err
	}
	if !access.canRead() {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot find the saved query"))
	}
	if !access.canManage() {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("only the creator or an admin can share a saved query"))
	}

	bindings, err := convertToStoreSavedQueryBindings(request.Policy.Bindings)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	applied, err := s.store.SetSavedQueryBindings(ctx, projectID, savedQuery.ResourceID, bindings, request.Policy.Etag)
	if err != nil {
		if errors.Is(err, store.ErrSavedQueryEtagMismatch) {
			// The policy moved under this write. Aborted tells the caller to
			// refetch and reapply rather than clobber a concurrent revocation.
			return nil, connect.NewError(connect.CodeAborted, errors.Errorf("the saved query policy was modified; refetch and retry"))
		}
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to set policy: %v", err))
	}
	if !applied {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot find the saved query"))
	}

	policy, err := convertToAPISavedQueryPolicy(bindings)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to build policy: %v", err))
	}
	return connect.NewResponse(policy), nil
}

// convertToAPISavedQueryPolicy converts stored bindings to the API shape,
// stamping the etag the next compare-and-swap write must present.
func convertToAPISavedQueryPolicy(bindings []*storepb.SavedQueryBinding) (*v1pb.SavedQueryPolicy, error) {
	etag, err := store.SavedQueryPolicyEtag(bindings)
	if err != nil {
		return nil, err
	}
	policy := &v1pb.SavedQueryPolicy{Etag: etag}
	for _, binding := range bindings {
		members := make([]string, 0, len(binding.Members))
		for _, member := range binding.Members {
			members = append(members, convertToV1SavedQueryMember(member))
		}
		policy.Bindings = append(policy.Bindings, &v1pb.SavedQueryBinding{
			Level:   v1pb.SavedQueryBinding_Level(binding.Level),
			Members: members,
		})
	}
	return policy, nil
}

// convertToStoreSavedQueryMember maps a submitted member to the resource name
// the store holds. Only users and groups are accepted; serviceAccount:,
// workloadIdentity:, and allUsers are not grantees on a saved query.
func convertToStoreSavedQueryMember(member string) (string, error) {
	if email, ok := strings.CutPrefix(member, common.UserBindingPrefix); ok && email != "" {
		return common.FormatUserEmail(email), nil
	}
	if email, ok := strings.CutPrefix(member, common.GroupBindingPrefix); ok && email != "" {
		return common.FormatGroupEmail(email), nil
	}
	return "", errors.Errorf("invalid member %q: only %q and %q are supported", member, common.UserBindingPrefix, common.GroupBindingPrefix)
}

func convertToV1SavedQueryMember(member string) string {
	if email, ok := strings.CutPrefix(member, common.UserNamePrefix); ok {
		return common.UserBindingPrefix + email
	}
	if email, ok := strings.CutPrefix(member, common.GroupPrefix); ok {
		return common.GroupBindingPrefix + email
	}
	return member
}

// convertToStoreSavedQueryBindings validates a submitted policy and converts it
// for storage, converting each member from the API's binding form to the
// resource name the store holds -- "user:a@corp.com" becomes "users/a@corp.com"
// -- exactly as convertToStoreIamPolicyMember does for an IAM policy. Doing it
// once here rather than at every read is what lets the access queries compare
// stored members against iam.PrincipalMembers with no conversion at all.
//
// A member may appear at one binding only, so the stored policy never needs a
// tie-break rule. Members are prefix-checked, not resolved, the same shape
// project and workspace IAM use in validateMember, narrowed to the two types a
// saved query accepts. A service account named under a "user:" prefix is inert
// rather than rejected: a caller's members come from their principal type, so
// such a binding matches nobody.
func convertToStoreSavedQueryBindings(bindings []*v1pb.SavedQueryBinding) ([]*storepb.SavedQueryBinding, error) {
	var converted []*storepb.SavedQueryBinding
	seenLevel := make(map[v1pb.SavedQueryBinding_Level]bool, len(bindings))
	seenMember := make(map[string]bool)
	for _, binding := range bindings {
		switch binding.Level {
		case v1pb.SavedQueryBinding_VIEWER, v1pb.SavedQueryBinding_EDITOR:
		default:
			return nil, errors.Errorf("invalid level %q", binding.Level)
		}
		if seenLevel[binding.Level] {
			return nil, errors.Errorf("duplicate binding for level %q", binding.Level)
		}
		seenLevel[binding.Level] = true

		if len(binding.Members) == 0 {
			return nil, errors.Errorf("binding for level %q has no members", binding.Level)
		}
		members := make([]string, 0, len(binding.Members))
		for _, member := range binding.Members {
			if seenMember[member] {
				return nil, errors.Errorf("member %q appears at more than one binding", member)
			}
			seenMember[member] = true
			stored, err := convertToStoreSavedQueryMember(member)
			if err != nil {
				return nil, err
			}
			members = append(members, stored)
		}

		converted = append(converted, &storepb.SavedQueryBinding{
			Level:   storepb.SavedQueryBinding_Level(binding.Level),
			Members: members,
		})
	}
	return converted, nil
}

// savedQueryAccess is how one caller stands relative to one saved query. The
// three sources of access are resolved once and the predicates derived from
// them, so read, write, share, and delete cannot drift apart.
type savedQueryAccess struct {
	isCreator bool
	// Holds bb.savedQueries.manage in scope.
	isAdmin bool
	// The highest level this caller's bindings grant, directly or through one
	// of their groups. LEVEL_UNSPECIFIED when no binding names them.
	level storepb.SavedQueryBinding_Level
}

// EDITOR > VIEWER in the enum, so the comparisons below read as the levels do.
func (a savedQueryAccess) canRead() bool {
	return a.isCreator || a.isAdmin || a.level >= storepb.SavedQueryBinding_VIEWER
}

func (a savedQueryAccess) canWrite() bool {
	return a.isCreator || a.isAdmin || a.level >= storepb.SavedQueryBinding_EDITOR
}

// canManage covers sharing, deleting, and re-filing: a binding never confers
// any of them, so sharing a saved query for editing cannot cost its owner the
// saved query.
func (a savedQueryAccess) canManage() bool {
	return a.isCreator || a.isAdmin
}

func (s *SavedQueryService) savedQueryAccess(ctx context.Context, savedQuery *store.SavedQueryMessage) (savedQueryAccess, error) {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return savedQueryAccess{}, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	access := savedQueryAccess{isCreator: savedQuery.Creator == user.Email}
	if access.isCreator {
		// Nothing a binding or the backstop adds can exceed what the owner
		// already has, so skip both lookups.
		return access, nil
	}

	workspaceID := common.GetWorkspaceIDFromContext(ctx)
	isAdmin, err := s.iamManager.CheckProjectWidePermission(ctx, permission.SavedQueriesManage, user, workspaceID, savedQuery.ProjectID)
	if err != nil {
		return savedQueryAccess{}, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check permission with error: %v", err.Error()))
	}
	access.isAdmin = isAdmin

	if len(savedQuery.Bindings) > 0 {
		principals, err := s.iamManager.PrincipalMembers(ctx, workspaceID, user)
		if err != nil {
			return savedQueryAccess{}, connect.NewError(connect.CodeInternal, errors.Errorf("failed to resolve principals: %v", err))
		}
		principalSet := make(map[string]bool, len(principals))
		for _, principal := range principals {
			principalSet[principal] = true
		}
		for _, binding := range savedQuery.Bindings {
			if binding.Level <= access.level {
				continue
			}
			for _, member := range binding.Members {
				if principalSet[member] {
					access.level = binding.Level
					break
				}
			}
		}
	}
	return access, nil
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
