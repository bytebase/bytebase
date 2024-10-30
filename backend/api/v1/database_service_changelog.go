package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (s *DatabaseService) ListChangelogs(ctx context.Context, request *v1pb.ListChangelogsRequest) (*v1pb.ListChangelogsResponse, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(request.Parent)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
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

	// TODO(p0ny): support view and filter
	find := &store.FindChangelogMessage{
		DatabaseUID: &database.UID,
		Limit:       &limitPlusOne,
		Offset:      &offset.offset,
	}

	changelogs, err := s.store.ListChangelogs(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list changelogs, errors: %v", err)
	}

	nextPageToken := ""
	if len(changelogs) == limitPlusOne {
		if nextPageToken, err = offset.getPageToken(); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
		changelogs = changelogs[:offset.limit]
	}

	// no subsequent pages
	converted, err := s.convertToChangelogs(ctx, database, changelogs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert changelogs, error: %v", err)
	}
	return &v1pb.ListChangelogsResponse{
		Changelogs:    converted,
		NextPageToken: nextPageToken,
	}, nil
}

func (s *DatabaseService) GetChangelog(ctx context.Context, request *v1pb.GetChangelogRequest) (*v1pb.Changelog, error) {
	instanceID, databaseName, changelogUID, err := common.GetInstanceDatabaseChangelogUID(request.Name)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// TODO(p0ny): support view and filter.
	changelog, err := s.store.GetChangelog(ctx, changelogUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list changelogs, errors: %v", err)
	}
	if changelog == nil {
		return nil, status.Errorf(codes.NotFound, "changelog %q not found", changelogUID)
	}

	// Get related database to convert changelog from store.
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if instance == nil {
		return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}

	converted, err := s.convertToChangelog(ctx, database, changelog)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert changelog, error: %v", err)
	}
	return converted, nil
}

func (s *DatabaseService) convertToChangelogs(ctx context.Context, d *store.DatabaseMessage, cs []*store.ChangelogMessage) ([]*v1pb.Changelog, error) {
	var changelogs []*v1pb.Changelog
	for _, c := range cs {
		changelog, err := s.convertToChangelog(ctx, d, c)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert to changelog")
		}
		changelogs = append(changelogs, changelog)
	}
	return changelogs, nil
}

func (s *DatabaseService) convertToChangelog(ctx context.Context, d *store.DatabaseMessage, c *store.ChangelogMessage) (*v1pb.Changelog, error) {
	cl := &v1pb.Changelog{
		Name:             common.FormatChangelog(d.InstanceID, d.DatabaseName, c.UID),
		Creator:          "",
		CreateTime:       timestamppb.New(c.CreatedTime),
		Status:           v1pb.Changelog_Status(c.Payload.GetTask().GetStatus()),
		Statement:        "",
		StatementSize:    0,
		StatementSheet:   "",
		Schema:           "",
		SchemaSize:       0,
		PrevSchema:       "",
		PrevSchemaSize:   0,
		Issue:            c.Payload.GetTask().GetIssue(),
		TaskRun:          c.Payload.GetTask().GetTaskRun(),
		Version:          c.Payload.GetTask().GetVersion(),
		Revision:         "",
		ChangedResources: convertToChangedResources(c.Payload.GetTask().GetChangedResources()),
	}

	if sheet := c.Payload.GetTask().GetSheet(); sheet != "" {
		_, sheetUID, err := common.GetProjectResourceIDSheetUID(sheet)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheetUID from %q", sheet)
		}
		sheetM, err := s.store.GetSheet(ctx, &store.FindSheetMessage{UID: &sheetUID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sheet %q", sheet)
		}
		if sheetM == nil {
			return nil, errors.Errorf("sheet %q not found", sheet)
		}

		cl.StatementSheet = sheet
		cl.Statement = sheetM.Statement
		cl.StatementSize = sheetM.Size
	}

	if id := c.Payload.GetTask().GetRevision(); id != 0 {
		cl.Revision = common.FormatRevision(d.InstanceID, d.DatabaseName, id)
	}

	creator, err := s.store.GetUserByID(ctx, c.CreatorUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get creator")
	}
	cl.Creator = common.FormatUserEmail(creator.Email)

	if id := c.Payload.GetTask().GetPrevSyncHistoryId(); id != 0 {
		h, err := s.store.GetSyncHistoryByUID(ctx, id)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sync history %d", id)
		}
		cl.PrevSchema = h.Schema
		cl.PrevSchemaSize = int64(len(cl.PrevSchema))
	}

	if id := c.Payload.GetTask().GetSyncHistoryId(); id != 0 {
		h, err := s.store.GetSyncHistoryByUID(ctx, id)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get sync history %d", id)
		}
		cl.Schema = h.Schema
		cl.SchemaSize = int64(len(cl.PrevSchema))
	}

	return cl, nil
}
