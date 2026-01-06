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
	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", request.Parent))
	}
	database, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
	}
	if database == nil {
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
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get next page token"))
		}
		revisions = revisions[:offset.limit]
	}

	converted := convertToRevisions(request.Parent, database.ProjectID, revisions)

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
	database, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
	}
	if database == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database not found"))
	}

	revision, err := s.store.GetRevision(ctx, revisionUID, instanceID, databaseName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get revision %v", request.Name))
	}
	parent := common.FormatDatabase(instanceID, databaseName)
	converted := convertToRevision(parent, database.ProjectID, revision)
	return connect.NewResponse(converted), nil
}

func (s *RevisionService) BatchCreateRevisions(
	ctx context.Context,
	req *connect.Request[v1pb.BatchCreateRevisionsRequest],
) (*connect.Response[v1pb.BatchCreateRevisionsResponse], error) {
	request := req.Msg
	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", request.Parent))
	}
	database, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
	}
	if database == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %v not found", request.Parent))
	}

	var revisions []*v1pb.Revision
	for _, r := range request.Requests {
		// Validate parent matches
		if r.Parent != request.Parent {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("request parent %q does not match batch parent %q", r.Parent, request.Parent))
		}
		revisions = append(revisions, r.Revision)
	}

	createdRevisions, err := s.createRevisions(ctx, request.Parent, revisions, database)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1pb.BatchCreateRevisionsResponse{
		Revisions: createdRevisions,
	}), nil
}

func (s *RevisionService) createRevisions(
	ctx context.Context,
	parent string,
	revisions []*v1pb.Revision,
	database *store.DatabaseMessage,
) ([]*v1pb.Revision, error) {
	var sheetSha256s []string
	for _, revisionReq := range revisions {
		if revisionReq == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("request.Revision is not set"))
		}
		// Validate the version format.
		if _, err := model.NewVersion(revisionReq.Version); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse version %q", revisionReq.Version))
		}

		_, sha, err := common.GetProjectResourceIDSheetSha256(revisionReq.Sheet)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to get sheet from %v", revisionReq.Sheet))
		}
		sheetSha256s = append(sheetSha256s, sha)
	}

	exist, err := s.store.HasSheets(ctx, sheetSha256s...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to check sheets: %v", err))
	}
	if !exist {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("some sheets are not found"))
	}

	var createdRevisions []*v1pb.Revision
	for i, revision := range revisions {
		if revision.TaskRun != "" {
			projectID, planID, stageID, taskID, taskRunID, err := common.GetProjectIDPlanIDStageIDTaskIDTaskRunID(revision.TaskRun)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to get taskRun from %q", revision.TaskRun))
			}
			taskRun, err := s.store.GetTaskRunByUID(ctx, taskRunID)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get taskRun"))
			}
			if taskRun == nil {
				return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("taskRun %q not found", revision.TaskRun))
			}
			if taskRun.ProjectID != projectID ||
				taskRun.PlanUID != planID ||
				taskRun.Environment != formatEnvironmentFromStageID(stageID) ||
				taskRun.TaskUID != taskID {
				return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("taskRun %q not found", revision.TaskRun))
			}
		}

		if revision.Type == v1pb.Revision_TYPE_UNSPECIFIED {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("revision.type cannot be TYPE_UNSPECIFIED"))
		}

		if (revision.Release == "") != (revision.File == "") {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("revision.release and revision.file must be set or unset"))
		}
		if revision.Release != "" && revision.File != "" {
			if !strings.HasPrefix(revision.File, revision.Release) {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("file %q is not in release %q", revision.File, revision.Release))
			}
			_, releaseUID, fileID, err := common.GetProjectReleaseUIDFile(revision.File)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to get release and file from %q", revision.File))
			}
			release, err := s.store.GetReleaseByUID(ctx, releaseUID)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get release"))
			}
			if release == nil {
				return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("release %q not found", revision.Release))
			}
			foundFile := false
			for _, f := range release.Payload.Files {
				if f.Path == fileID {
					foundFile = true
					fileSheet := common.FormatSheet(release.ProjectID, f.SheetSha256)
					if fileSheet != revision.Sheet {
						return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("The sheet in file %q is %q which is different from revision.sheet %q", fileID, fileSheet, revision.Sheet))
					}
					break
				}
			}
			if !foundFile {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("file %q not found in release %q", fileID, revision.Release))
			}
		}

		revisionCreate := convertRevision(revision, database, sheetSha256s[i])
		revisionM, err := s.store.CreateRevision(ctx, revisionCreate)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create revision"))
		}
		converted := convertToRevision(parent, database.ProjectID, revisionM)
		createdRevisions = append(createdRevisions, converted)
	}
	return createdRevisions, nil
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

func convertToRevisions(parent string, projectID string, revisions []*store.RevisionMessage) []*v1pb.Revision {
	var rs []*v1pb.Revision
	for _, revision := range revisions {
		r := convertToRevision(parent, projectID, revision)
		rs = append(rs, r)
	}
	return rs
}

func convertToRevision(parent string, projectID string, revision *store.RevisionMessage) *v1pb.Revision {
	taskRunName := revision.Payload.TaskRun
	r := &v1pb.Revision{
		Name:        fmt.Sprintf("%s/%s%d", parent, common.RevisionNamePrefix, revision.UID),
		Release:     revision.Payload.Release,
		CreateTime:  timestamppb.New(revision.CreatedAt),
		Sheet:       common.FormatSheet(projectID, revision.Payload.SheetSha256),
		SheetSha256: revision.Payload.SheetSha256,
		Version:     revision.Version,
		File:        revision.Payload.File,
		TaskRun:     taskRunName,
		Type:        convertToRevisionType(revision.Payload.Type),
	}

	if revision.Deleter != nil {
		r.Deleter = common.FormatUserEmail(*revision.Deleter)
	}
	if revision.DeletedAt != nil {
		r.DeleteTime = timestamppb.New(*revision.DeletedAt)
	}

	return r
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

func convertRevision(revision *v1pb.Revision, database *store.DatabaseMessage, sheetSha256 string) *store.RevisionMessage {
	r := &store.RevisionMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
		Version:      revision.Version,
		Payload: &storepb.RevisionPayload{
			Release:     revision.Release,
			File:        revision.File,
			SheetSha256: sheetSha256,
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
