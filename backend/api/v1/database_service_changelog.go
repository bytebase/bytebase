package v1

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/transform"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (s *DatabaseService) ListChangelogs(ctx context.Context, request *v1pb.ListChangelogsRequest) (*v1pb.ListChangelogsResponse, error) {
	database, err := getDatabaseMessage(ctx, s.store, request.Parent)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if database == nil || database.Deleted {
		return nil, status.Errorf(codes.NotFound, "database %q not found", request.Parent)
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

	find := &store.FindChangelogMessage{
		InstanceID:   &database.InstanceID,
		DatabaseName: &database.DatabaseName,
		Limit:        &limitPlusOne,
		Offset:       &offset.offset,
	}
	if request.View == v1pb.ChangelogView_CHANGELOG_VIEW_FULL {
		find.ShowFull = true
	}

	filters, err := ParseFilter(request.Filter)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	for _, expr := range filters {
		if expr.Operator != ComparatorTypeEqual {
			return nil, status.Errorf(codes.InvalidArgument, `only support "=" operation for filter`)
		}
		switch expr.Key {
		case "type":
			find.TypeList = strings.Split(expr.Value, " | ")
		case "table":
			resourcesFilter := expr.Value
			find.ResourcesFilter = &resourcesFilter
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid filter key %q", expr.Key)
		}
	}

	changelogs, err := s.store.ListChangelogs(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list changelogs, errors: %v", err)
	}

	nextPageToken := ""
	if len(changelogs) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get next page token, error: %v", err)
		}
		changelogs = changelogs[:offset.limit]
	}

	// no subsequent pages
	converted, err := s.convertToChangelogs(database, changelogs)
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

	find := &store.FindChangelogMessage{
		UID: &changelogUID,
	}
	if request.View == v1pb.ChangelogView_CHANGELOG_VIEW_FULL {
		find.ShowFull = true
	}

	changelog, err := s.store.GetChangelog(ctx, find)
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
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}

	converted, err := s.convertToChangelog(database, changelog)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert changelog, error: %v", err)
	}
	if request.SdlFormat {
		switch instance.Metadata.GetEngine() {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			sdlSchema, err := transform.SchemaTransform(storepb.Engine_MYSQL, converted.Schema)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to convert schema to sdl format, error %v", err.Error())
			}
			converted.Schema = sdlSchema
			converted.SchemaSize = int64(len(sdlSchema))
			sdlSchema, err = transform.SchemaTransform(storepb.Engine_MYSQL, converted.PrevSchema)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to convert previous schema to sdl format, error %v", err.Error())
			}
			converted.PrevSchema = sdlSchema
			converted.PrevSchemaSize = int64(len(sdlSchema))
		}
	}
	return converted, nil
}

func (s *DatabaseService) convertToChangelogs(d *store.DatabaseMessage, cs []*store.ChangelogMessage) ([]*v1pb.Changelog, error) {
	var changelogs []*v1pb.Changelog
	for _, c := range cs {
		changelog, err := s.convertToChangelog(d, c)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert to changelog")
		}
		changelogs = append(changelogs, changelog)
	}
	return changelogs, nil
}

func (*DatabaseService) convertToChangelog(d *store.DatabaseMessage, c *store.ChangelogMessage) (*v1pb.Changelog, error) {
	cl := &v1pb.Changelog{
		Name:             common.FormatChangelog(d.InstanceID, d.DatabaseName, c.UID),
		CreateTime:       timestamppb.New(c.CreatedAt),
		Status:           convertToChangelogStatus(c.Status),
		Statement:        "",
		StatementSize:    0,
		StatementSheet:   "",
		Schema:           "",
		SchemaSize:       0,
		PrevSchema:       "",
		PrevSchemaSize:   0,
		Issue:            c.Payload.GetIssue(),
		TaskRun:          c.Payload.GetTaskRun(),
		Version:          c.Payload.GetVersion(),
		Revision:         "",
		ChangedResources: convertToChangedResources(c.Payload.GetChangedResources()),
		Type:             convertToChangelogType(c.Payload.GetType()),
	}

	if sheet := c.Payload.GetSheet(); sheet != "" {
		cl.StatementSheet = sheet
		cl.Statement = c.Statement
		cl.StatementSize = c.StatementSize
	}

	if id := c.Payload.GetRevision(); id != 0 {
		cl.Revision = common.FormatRevision(d.InstanceID, d.DatabaseName, id)
	}

	if v := c.PrevSyncHistoryUID; v != nil {
		cl.PrevSchema = c.PrevSchema
		cl.PrevSchemaSize = int64(len(cl.PrevSchema))
	}

	if v := c.SyncHistoryUID; v != nil {
		cl.Schema = c.Schema
		cl.SchemaSize = int64(len(cl.Schema))
	}

	return cl, nil
}

func convertToChangelogStatus(s store.ChangelogStatus) v1pb.Changelog_Status {
	switch s {
	case store.ChangelogStatusDone:
		return v1pb.Changelog_DONE
	case store.ChangelogStatusFailed:
		return v1pb.Changelog_FAILED
	case store.ChangelogStatusPending:
		return v1pb.Changelog_PENDING
	default:
		return v1pb.Changelog_STATUS_UNSPECIFIED
	}
}

func convertToChangelogType(t storepb.ChangelogPayload_Type) v1pb.Changelog_Type {
	switch t {
	case storepb.ChangelogPayload_BASELINE:
		return v1pb.Changelog_BASELINE
	case storepb.ChangelogPayload_MIGRATE:
		return v1pb.Changelog_MIGRATE
	case storepb.ChangelogPayload_MIGRATE_SDL:
		return v1pb.Changelog_MIGRATE_SDL
	case storepb.ChangelogPayload_MIGRATE_GHOST:
		return v1pb.Changelog_MIGRATE_GHOST
	case storepb.ChangelogPayload_DATA:
		return v1pb.Changelog_DATA
	default:
		return v1pb.Changelog_TYPE_UNSPECIFIED
	}
}
