package v1

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (s *DatabaseService) ListRevisions(ctx context.Context, request *v1pb.ListRevisionsRequest) (*v1pb.ListRevisionsResponse, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get instance and database from %v, err: %v", request.Parent, err)
	}

	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find instance %v, err: %v", instanceID, err)
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %v not found", instanceID)
	}

	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find database %v, err: %v", request.Parent, err)
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %v not found", request.Parent)
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
		DatabaseUID: &database.UID,
		Limit:       &limitPlusOne,
		Offset:      &offset.offset,
		ShowDeleted: request.ShowDeleted,
	}

	revisions, err := s.store.ListRevisions(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find revisions, err: %v", err)
	}

	var nextPageToken string
	if len(revisions) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
		revisions = revisions[:offset.limit]
	}

	converted, err := convertToRevisions(ctx, s.store, request.Parent, revisions)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to revisions, err: %v", err)
	}

	return &v1pb.ListRevisionsResponse{
		Revisions:     converted,
		NextPageToken: nextPageToken,
	}, nil
}

func (s *DatabaseService) GetRevision(ctx context.Context, request *v1pb.GetRevisionRequest) (*v1pb.Revision, error) {
	instanceName, databaseName, revisionUID, err := common.GetInstanceDatabaseRevisionID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get revision UID from %v, err: %v", request.Name, err)
	}
	revision, err := s.store.GetRevision(ctx, revisionUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete revision %v, err: %v", revisionUID, err)
	}
	parent := fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, instanceName, common.DatabaseIDPrefix, databaseName)
	converted, err := convertToRevision(ctx, s.store, parent, revision)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to revision, err: %v", err)
	}
	return converted, nil
}

func (s *DatabaseService) CreateRevision(ctx context.Context, request *v1pb.CreateRevisionRequest) (*v1pb.Revision, error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	instanceID, databaseID, err := common.GetInstanceDatabaseID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get instance and database from %v, err: %v", request.Parent, err)
	}
	if request.Revision == nil {
		return nil, status.Errorf(codes.InvalidArgument, "request.Revision is not set")
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get database, err: %v", err)
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", request.Parent)
	}
	_, sheetUID, err := common.GetProjectResourceIDSheetUID(request.Revision.Sheet)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get sheet from %v, err: %v", request.Revision.Sheet, err)
	}
	sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get sheet, err: %v", err)
	}
	if sheet == nil {
		return nil, status.Errorf(codes.NotFound, "sheet %q not found", request.Revision.Sheet)
	}

	if request.Revision.TaskRun != "" {
		projectID, rolloutID, stageID, taskID, taskRunID, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(request.Revision.TaskRun)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to get taskRun from %q", request.Revision.TaskRun)
		}
		taskRun, err := s.store.GetTaskRun(ctx, taskRunID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get taskRun, err: %v", err)
		}
		if taskRun == nil {
			return nil, status.Errorf(codes.NotFound, "taskRun %q not found", request.Revision.TaskRun)
		}
		if taskRun.ProjectID != projectID ||
			taskRun.PipelineUID != rolloutID ||
			taskRun.StageUID != stageID ||
			taskRun.TaskUID != taskID {
			return nil, status.Errorf(codes.NotFound, "taskRun %q not found", request.Revision.TaskRun)
		}
	}

	if request.Revision.Issue != "" {
		_, _, err := common.GetProjectIDIssueUID(request.Revision.Issue)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to get issue from %q", request.Revision.Issue)
		}
	}

	if (request.Revision.Release == "") != (request.Revision.File == "") {
		return nil, status.Errorf(codes.InvalidArgument, "revision.release and revision.file must be set or unset")
	}
	if request.Revision.Release != "" && request.Revision.File != "" {
		if !strings.HasPrefix(request.Revision.File, request.Revision.Release) {
			return nil, status.Errorf(codes.InvalidArgument, "file %q is not in release %q", request.Revision.File, request.Revision.Release)
		}
		_, releaseUID, fileID, err := common.GetProjectReleaseUIDFile(request.Revision.File)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to get release and file from %q", request.Revision.File)
		}
		release, err := s.store.GetRelease(ctx, releaseUID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get release, err: %v", err)
		}
		if release == nil {
			return nil, status.Errorf(codes.NotFound, "release %q not found", request.Revision.Release)
		}
		foundFile := false
		for _, f := range release.Payload.Files {
			if f.Id == fileID {
				foundFile = true
				if f.Sheet != request.Revision.Sheet {
					return nil, status.Errorf(codes.InvalidArgument, "The sheet in file %q is %q which is different from revision.sheet %q", fileID, f.Sheet, request.Revision.Sheet)
				}
				break
			}
		}
		if !foundFile {
			return nil, status.Errorf(codes.InvalidArgument, "file %q not found in release %q", fileID, request.Revision.Release)
		}
	}

	converted := convertRevision(request.Revision, database, sheet)
	revisionM, err := s.store.CreateRevision(ctx, converted, user.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create revision, err: %v", err)
	}
	converted1, err := convertToRevision(ctx, s.store, request.Parent, revisionM)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert to revision, err: %v", err)
	}

	return converted1, nil
}

func (s *DatabaseService) DeleteRevision(ctx context.Context, request *v1pb.DeleteRevisionRequest) (*emptypb.Empty, error) {
	_, _, revisionUID, err := common.GetInstanceDatabaseRevisionID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to get revision UID from %v, err: %v", request.Name, err)
	}
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}
	if err := s.store.DeleteRevision(ctx, revisionUID, user.ID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete revision %v, err: %v", revisionUID, err)
	}
	return &emptypb.Empty{}, nil
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
	creator, err := s.GetUserByID(ctx, revision.CreatorUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get creator")
	}
	if creator == nil {
		return nil, errors.Errorf("creator %v not found", revision.CreatorUID)
	}
	_, sheetUID, err := common.GetProjectResourceIDSheetUID(revision.Payload.Sheet)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheetUID from %q", revision.Payload.Sheet)
	}
	sheet, err := s.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sheet %q", revision.Payload.Sheet)
	}
	if sheet == nil {
		return nil, errors.Errorf("sheet %q not found", revision.Payload.Sheet)
	}

	taskRunName, issueName := revision.Payload.TaskRun, ""
	if taskRunName != "" {
		_, rolloutUID, _, _, _, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(taskRunName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get rollout UID from %q", taskRunName)
		}
		issue, err := s.GetIssueV2(ctx, &store.FindIssueMessage{PipelineID: &rolloutUID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get issue by rollout %q", rolloutUID)
		}
		if issue != nil {
			issueName = common.FormatIssue(issue.Project.ResourceID, issue.UID)
		}
	}

	r := &v1pb.Revision{
		Name:          fmt.Sprintf("%s/%s%d", parent, common.RevisionNamePrefix, revision.UID),
		Release:       revision.Payload.Release,
		CreateTime:    timestamppb.New(revision.CreatedTime),
		Creator:       common.FormatUserEmail(creator.Email),
		Sheet:         revision.Payload.Sheet,
		SheetSha256:   revision.Payload.SheetSha256,
		Statement:     sheet.Statement,
		StatementSize: sheet.Size,
		Version:       revision.Version,
		File:          revision.Payload.File,
		Issue:         issueName,
		TaskRun:       taskRunName,
	}

	if revision.DeleterUID != nil {
		deleter, err := s.GetUserByID(ctx, *revision.DeleterUID)
		if err != nil {
			return nil, errors.Wrapf(err, "faile to get deleter")
		}
		if deleter == nil {
			return nil, errors.Errorf("deleter %v not found", *revision.DeleterUID)
		}
		r.Deleter = common.FormatUserEmail(deleter.Email)
	}
	if revision.DeletedTime != nil {
		r.DeleteTime = timestamppb.New(*revision.DeletedTime)
	}

	return r, nil
}

func convertRevision(revision *v1pb.Revision, database *store.DatabaseMessage, sheet *store.SheetMessage) *store.RevisionMessage {
	r := &store.RevisionMessage{
		DatabaseUID: database.UID,
		Version:     revision.Version,
		Payload: &storepb.RevisionPayload{
			Release:     revision.Release,
			File:        revision.File,
			Sheet:       revision.Sheet,
			SheetSha256: sheet.GetSha256Hex(),
			TaskRun:     revision.TaskRun,
		},
	}
	return r
}
