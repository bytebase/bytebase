package v1

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// QueryHistoryService implements the query history service.
type QueryHistoryService struct {
	v1connect.UnimplementedQueryHistoryServiceHandler
	store *store.Store
}

// NewQueryHistoryService creates a new QueryHistoryService.
func NewQueryHistoryService(store *store.Store) *QueryHistoryService {
	return &QueryHistoryService{
		store: store,
	}
}

// SearchQueryHistories lists the caller's own query histories in the parent
// project. The AIP-159 wildcard "projects/-" is rejected: cross-project reads
// are reserved for the IAM-gated ListQueryHistories.
func (s *QueryHistoryService) SearchQueryHistories(ctx context.Context, req *connect.Request[v1pb.SearchQueryHistoriesRequest]) (*connect.Response[v1pb.SearchQueryHistoriesResponse], error) {
	request := req.Msg
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	find := &store.FindQueryHistoryMessage{Creator: &user.Email}
	if err := s.resolveQueryHistoryParent(ctx, request.Parent, find, false); err != nil {
		return nil, err
	}

	filterQ, err := store.GetListQueryHistoryFilter(request.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.FilterQ = filterQ

	histories, nextPageToken, err := s.paginatedQueryHistories(ctx, find, request.PageToken, request.PageSize)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1pb.SearchQueryHistoriesResponse{
		QueryHistories: histories,
		NextPageToken:  nextPageToken,
	}), nil
}

// ListQueryHistories lists query histories of all users in a project. The IAM
// interceptor checks bb.queryHistories.list on the parent project, or on the
// workspace when the parent is the AIP-159 wildcard "projects/-".
func (s *QueryHistoryService) ListQueryHistories(ctx context.Context, req *connect.Request[v1pb.ListQueryHistoriesRequest]) (*connect.Response[v1pb.ListQueryHistoriesResponse], error) {
	request := req.Msg
	find := &store.FindQueryHistoryMessage{}
	if err := s.resolveQueryHistoryParent(ctx, request.Parent, find, true); err != nil {
		return nil, err
	}

	creator, err := store.GetListQueryHistoriesCreatorFilter(request.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.Creator = creator

	histories, nextPageToken, err := s.paginatedQueryHistories(ctx, find, request.PageToken, request.PageSize)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1pb.ListQueryHistoriesResponse{
		QueryHistories: histories,
		NextPageToken:  nextPageToken,
	}), nil
}

// GetQueryHistory gets a single query history. Query histories are private to
// their creator, so only the creator may retrieve one.
func (s *QueryHistoryService) GetQueryHistory(ctx context.Context, req *connect.Request[v1pb.GetQueryHistoryRequest]) (*connect.Response[v1pb.QueryHistory], error) {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	projectID, historyID, err := common.GetProjectIDQueryHistoryID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	history, err := s.store.GetQueryHistory(ctx, historyID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get query history: %v", err.Error()))
	}
	// Hide existence from non-creators and project mismatches by returning the
	// same not-found error for all three cases.
	if history == nil || history.Project != projectID || history.Creator != user.Email {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("query history %q not found", req.Msg.Name))
	}

	queryHistory, err := s.convertToV1QueryHistory(ctx, history)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert query history"))
	}
	return connect.NewResponse(queryHistory), nil
}

// paginatedQueryHistories applies pagination to find, fetches one extra row to
// detect whether a next page exists, and converts the results to the v1 shape.
func (s *QueryHistoryService) paginatedQueryHistories(ctx context.Context, find *store.FindQueryHistoryMessage, pageToken string, requestedPageSize int32) ([]*v1pb.QueryHistory, string, error) {
	offset, err := parseLimitAndOffset(&pageSize{
		token:   pageToken,
		limit:   int(requestedPageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, "", err
	}
	limitPlusOne := offset.limit + 1
	find.Limit = &limitPlusOne
	find.Offset = &offset.offset

	historyList, err := s.store.ListQueryHistories(ctx, find)
	if err != nil {
		return nil, "", connect.NewError(connect.CodeInternal, errors.Errorf("failed to list history: %v", err.Error()))
	}

	nextPageToken := ""
	if len(historyList) == limitPlusOne {
		historyList = historyList[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, "", connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to marshal next page token"))
		}
	}

	var histories []*v1pb.QueryHistory
	for _, history := range historyList {
		queryHistory, err := s.convertToV1QueryHistory(ctx, history)
		if err != nil {
			return nil, "", connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert log entity"))
		}
		if queryHistory == nil {
			continue
		}
		histories = append(histories, queryHistory)
	}
	return histories, nextPageToken, nil
}

// resolveQueryHistoryParent applies the parent scope to find: a concrete
// "projects/{id}" parent must exist in the caller's workspace and scopes by
// project. When allowWildcard is set, the AIP-159 wildcard "projects/-"
// scopes by workspace instead, which is mandatory because query_history rows
// are workspace-scoped only through their project.
func (s *QueryHistoryService) resolveQueryHistoryParent(ctx context.Context, parent string, find *store.FindQueryHistoryMessage, allowWildcard bool) error {
	projectID, err := common.GetProjectID(parent)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if projectID == "-" {
		if !allowWildcard {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`the wildcard parent "projects/-" is not supported; specify a concrete project`))
		}
		workspaceID := common.GetWorkspaceIDFromContext(ctx)
		if workspaceID == "" {
			return connect.NewError(connect.CodeInternal, errors.Errorf("workspace not found in context"))
		}
		find.Workspace = workspaceID
		return nil
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
		Workspace:  common.GetWorkspaceIDFromContext(ctx),
		ResourceID: &projectID,
	})
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get project"))
	}
	if project == nil {
		return connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", parent))
	}
	find.Project = &projectID
	return nil
}

func (s *QueryHistoryService) convertToV1QueryHistory(ctx context.Context, history *store.QueryHistoryMessage) (*v1pb.QueryHistory, error) {
	creator, err := s.store.GetAccountByEmail(ctx, history.Creator)
	if err != nil {
		return nil, err
	}
	if creator == nil {
		return nil, errors.Errorf("cannot found user with email %s", history.Creator)
	}

	historyType := v1pb.QueryHistory_TYPE_UNSPECIFIED
	switch history.Type {
	case store.QueryHistoryTypeExport:
		historyType = v1pb.QueryHistory_EXPORT
	case store.QueryHistoryTypeQuery:
		historyType = v1pb.QueryHistory_QUERY
	default:
	}

	return &v1pb.QueryHistory{
		Name:       fmt.Sprintf("%s/queryHistories/%s", common.FormatProject(history.Project), history.ResourceID),
		Statement:  history.Statement,
		Error:      history.Payload.Error,
		Database:   history.Database,
		Creator:    common.FormatUserEmail(creator.Email),
		CreateTime: timestamppb.New(history.CreatedAt),
		Duration:   history.Payload.Duration,
		Type:       historyType,
	}, nil
}
