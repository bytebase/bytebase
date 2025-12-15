package v1

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// RevisionService implements the revision service.
type RevisionService struct {
	v1connect.UnimplementedRevisionServiceHandler
	store *store.Store
}

// NewRevisionService creates a new RevisionService.
func NewRevisionService(store *store.Store) *RevisionService {
	return &RevisionService{
		store: store,
	}
}

func (s *RevisionService) ListRevisions(
	ctx context.Context,
	req *connect.Request[v1pb.ListRevisionsRequest],
) (*connect.Response[v1pb.ListRevisionsResponse], error) {
	request := req.Msg
	database, err := getDatabaseMessage(ctx, s.store, request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to found database %v", request.Parent))
	}
	if database == nil || database.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %v not found", request.Parent))
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

	find := &store.FindRevisionMessage{
		InstanceID:   &database.InstanceID,
		DatabaseName: &database.DatabaseName,
		Limit:        &limitPlusOne,
		Offset:       &offset.offset,
		ShowDeleted:  request.ShowDeleted,
	}

	revisions, err := s.store.ListRevisions(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find revisions"))
	}

	var nextPageToken string
	if len(revisions) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get next page token, error"))
		}
		revisions = revisions[:offset.limit]
	}

	converted, err := convertToRevisions(ctx, s.store, request.Parent, revisions)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to revisions"))
	}

	return connect.NewResponse(&v1pb.ListRevisionsResponse{
		Revisions:     converted,
		NextPageToken: nextPageToken,
	}), nil
}

func (s *RevisionService) GetRevision(
	ctx context.Context,
	req *connect.Request[v1pb.GetRevisionRequest],
) (*connect.Response[v1pb.Revision], error) {
	request := req.Msg
	instanceID, databaseName, revisionUID, err := common.GetInstanceDatabaseRevisionID(request.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get revision UID from %v", request.Name))
	}
	revision, err := s.store.GetRevision(ctx, revisionUID, instanceID, databaseName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get revision %v", request.Name))
	}
	parent := common.FormatDatabase(instanceID, databaseName)
	converted, err := convertToRevision(ctx, s.store, parent, revision)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to revision"))
	}
	return connect.NewResponse(converted), nil
}

func (s *RevisionService) CreateRevision(
	ctx context.Context,
	req *connect.Request[v1pb.CreateRevisionRequest],
) (*connect.Response[v1pb.Revision], error) {
	request := req.Msg
	if request.Revision == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("request.Revision is not set"))
	}
	// Validate the version format.
	if _, err := model.NewVersion(request.Revision.Version); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse version %q", request.Revision.Version))
	}
	database, err := getDatabaseMessage(ctx, s.store, request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to found database %v", request.Parent))
	}
	if database == nil || database.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %v not found", request.Parent))
	}
	sheetProjectID, sheetUID, err := common.GetProjectResourceIDSheetUID(request.Revision.Sheet)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get sheet from %v", request.Revision.Sheet))
	}
	sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID, ProjectID: &sheetProjectID})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get sheet"))
	}
	if sheet == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("sheet %d not found in project %s", sheetUID, sheetProjectID))
	}

	if request.Revision.TaskRun != "" {
		projectID, rolloutID, stageID, taskID, taskRunID, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(request.Revision.TaskRun)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to get taskRun from %q", request.Revision.TaskRun))
		}
		taskRun, err := s.store.GetTaskRunByUID(ctx, taskRunID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get taskRun"))
		}
		if taskRun == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("taskRun %q not found", request.Revision.TaskRun))
		}
		if taskRun.ProjectID != projectID ||
			taskRun.PipelineUID != rolloutID ||
			taskRun.Environment != formatEnvironmentFromStageID(stageID) ||
			taskRun.TaskUID != taskID {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("taskRun %q not found", request.Revision.TaskRun))
		}
	}

	if request.Revision.Type == v1pb.Revision_TYPE_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("revision.type cannot be TYPE_UNSPECIFIED"))
	}

	if (request.Revision.Release == "") != (request.Revision.File == "") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("revision.release and revision.file must be set or unset"))
	}
	if request.Revision.Release != "" && request.Revision.File != "" {
		if !strings.HasPrefix(request.Revision.File, request.Revision.Release) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("file %q is not in release %q", request.Revision.File, request.Revision.Release))
		}
		_, releaseUID, fileID, err := common.GetProjectReleaseUIDFile(request.Revision.File)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to get release and file from %q", request.Revision.File))
		}
		release, err := s.store.GetReleaseByUID(ctx, releaseUID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get release"))
		}
		if release == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("release %q not found", request.Revision.Release))
		}
		foundFile := false
		for _, f := range release.Payload.Files {
			if f.Id == fileID {
				foundFile = true
				if f.Sheet != request.Revision.Sheet {
					return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("The sheet in file %q is %q which is different from revision.sheet %q", fileID, f.Sheet, request.Revision.Sheet))
				}
				break
			}
		}
		if !foundFile {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("file %q not found in release %q", fileID, request.Revision.Release))
		}
	}

	revisionCreate := convertRevision(request.Revision, database, sheet)
	revisionM, err := s.store.CreateRevision(ctx, revisionCreate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create revision"))
	}
	converted, err := convertToRevision(ctx, s.store, request.Parent, revisionM)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert to revision"))
	}

	return connect.NewResponse(converted), nil
}

func (s *RevisionService) BatchCreateRevisions(
	ctx context.Context,
	req *connect.Request[v1pb.BatchCreateRevisionsRequest],
) (*connect.Response[v1pb.BatchCreateRevisionsResponse], error) {
	request := req.Msg
	database, err := getDatabaseMessage(ctx, s.store, request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to found database %v", request.Parent))
	}
	if database == nil || database.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %v not found", request.Parent))
	}

	var revisions []*v1pb.Revision
	for _, req := range request.Requests {
		// Validate parent matches
		if req.Parent != request.Parent {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("request parent %q does not match batch parent %q", req.Parent, request.Parent))
		}

		// Reuse the CreateRevision logic by calling it directly
		revisionResp, err := s.CreateRevision(ctx, connect.NewRequest(req))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create revision for request"))
		}
		revisions = append(revisions, revisionResp.Msg)
	}

	return connect.NewResponse(&v1pb.BatchCreateRevisionsResponse{
		Revisions: revisions,
	}), nil
}

func (s *RevisionService) DeleteRevision(
	ctx context.Context,
	req *connect.Request[v1pb.DeleteRevisionRequest],
) (*connect.Response[emptypb.Empty], error) {
	request := req.Msg
	instanceID, databaseName, revisionUID, err := common.GetInstanceDatabaseRevisionID(request.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %v", request.Name))
	}
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	if err := s.store.DeleteRevision(ctx, revisionUID, instanceID, databaseName, user.Email); err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to delete revision %v", request.Name))
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func convertToRevisions(ctx context.Context, s *store.Store, parent string, revisions []*store.RevisionMessage) ([]*v1pb.Revision, error) {
	var rs []*v1pb.Revision
	for _, revision := range revisions {
		r, err := convertToRevision(ctx, s, parent, revision)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert to revisions")
		}
		rs = append(rs, r)
	}
	return rs, nil
}

func convertToRevision(ctx context.Context, s *store.Store, parent string, revision *store.RevisionMessage) (*v1pb.Revision, error) {
	sheetProjectID, sheetUID, err := common.GetProjectResourceIDSheetUID(revision.Payload.Sheet)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheetUID from %q", revision.Payload.Sheet)
	}
	sheet, err := s.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID, ProjectID: &sheetProjectID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet %q", revision.Payload.Sheet)
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet %d not found in project %s", sheetUID, sheetProjectID)
	}

	taskRunName := revision.Payload.TaskRun
	r := &v1pb.Revision{
		Name:          fmt.Sprintf("%s/%s%d", parent, common.RevisionNamePrefix, revision.UID),
		Release:       revision.Payload.Release,
		CreateTime:    timestamppb.New(revision.CreatedAt),
		Sheet:         revision.Payload.Sheet,
		SheetSha256:   revision.Payload.SheetSha256,
		Statement:     sheet.Statement,
		StatementSize: sheet.Size,
		Version:       revision.Version,
		File:          revision.Payload.File,
		TaskRun:       taskRunName,
		Type:          convertToRevisionType(revision.Payload.Type),
	}

	if revision.Deleter != nil {
		r.Deleter = common.FormatUserEmail(*revision.Deleter)
	}
	if revision.DeletedAt != nil {
		r.DeleteTime = timestamppb.New(*revision.DeletedAt)
	}

	return r, nil
}

func convertToRevisionType(t storepb.SchemaChangeType) v1pb.Revision_Type {
	//exhaustive:enforce
	switch t {
	case storepb.SchemaChangeType_SCHEMA_CHANGE_TYPE_UNSPECIFIED:
		return v1pb.Revision_TYPE_UNSPECIFIED
	case storepb.SchemaChangeType_VERSIONED:
		return v1pb.Revision_VERSIONED
	case storepb.SchemaChangeType_DECLARATIVE:
		return v1pb.Revision_DECLARATIVE
	default:
		return v1pb.Revision_TYPE_UNSPECIFIED
	}
}

func convertRevision(revision *v1pb.Revision, database *store.DatabaseMessage, sheet *store.SheetMessage) *store.RevisionMessage {
	r := &store.RevisionMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
		Version:      revision.Version,
		Payload: &storepb.RevisionPayload{
			Release:     revision.Release,
			File:        revision.File,
			Sheet:       revision.Sheet,
			SheetSha256: sheet.GetSha256Hex(),
			TaskRun:     revision.TaskRun,
			Type:        convertRevisionType(revision.Type),
		},
	}
	return r
}

func convertRevisionType(t v1pb.Revision_Type) storepb.SchemaChangeType {
	//exhaustive:enforce
	switch t {
	case v1pb.Revision_TYPE_UNSPECIFIED:
		return storepb.SchemaChangeType_SCHEMA_CHANGE_TYPE_UNSPECIFIED
	case v1pb.Revision_VERSIONED:
		return storepb.SchemaChangeType_VERSIONED
	case v1pb.Revision_DECLARATIVE:
		return storepb.SchemaChangeType_DECLARATIVE
	default:
		return storepb.SchemaChangeType_SCHEMA_CHANGE_TYPE_UNSPECIFIED
	}
}
