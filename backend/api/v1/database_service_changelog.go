package v1

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/transform"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *DatabaseService) ListChangelogs(ctx context.Context, req *connect.Request[v1pb.ListChangelogsRequest]) (*connect.Response[v1pb.ListChangelogsResponse], error) {
	database, err := getDatabaseMessage(ctx, s.store, req.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if database == nil || database.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", req.Msg.Parent))
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

	find := &store.FindChangelogMessage{
		InstanceID:   &database.InstanceID,
		DatabaseName: &database.DatabaseName,
		Limit:        &limitPlusOne,
		Offset:       &offset.offset,
	}
	if req.Msg.View == v1pb.ChangelogView_CHANGELOG_VIEW_FULL {
		find.ShowFull = true
	}

	filters, err := ParseFilter(req.Msg.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	for _, expr := range filters {
		if expr.Operator != ComparatorTypeEqual {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New(`only support "=" operation for filter`))
		}
		switch expr.Key {
		case "type":
			find.TypeList = strings.Split(expr.Value, " | ")
		case "table":
			resourcesFilter := expr.Value
			find.ResourcesFilter = &resourcesFilter
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid filter key %q", expr.Key))
		}
	}

	changelogs, err := s.store.ListChangelogs(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list changelogs"))
	}

	nextPageToken := ""
	if len(changelogs) == limitPlusOne {
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get next page token"))
		}
		changelogs = changelogs[:offset.limit]
	}

	// no subsequent pages
	converted := s.convertToChangelogs(database, changelogs)
	return connect.NewResponse(&v1pb.ListChangelogsResponse{
		Changelogs:    converted,
		NextPageToken: nextPageToken,
	}), nil
}

func (s *DatabaseService) GetChangelog(ctx context.Context, req *connect.Request[v1pb.GetChangelogRequest]) (*connect.Response[v1pb.Changelog], error) {
	instanceID, databaseName, changelogUID, err := common.GetInstanceDatabaseChangelogUID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	find := &store.FindChangelogMessage{
		UID: &changelogUID,
	}
	if req.Msg.View == v1pb.ChangelogView_CHANGELOG_VIEW_FULL {
		find.ShowFull = true
	}

	changelog, err := s.store.GetChangelog(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list changelogs"))
	}
	if changelog == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("changelog %q not found", changelogUID))
	}

	// Get related database to convert changelog from store.
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if instance == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", instanceID))
	}
	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		InstanceID:      &instanceID,
		DatabaseName:    &databaseName,
		IsCaseSensitive: store.IsObjectCaseSensitive(instance),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if database == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", databaseName))
	}

	converted := s.convertToChangelog(database, changelog)
	if req.Msg.SdlFormat {
		switch instance.Metadata.GetEngine() {
		case storepb.Engine_MYSQL, storepb.Engine_TIDB, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
			sdlSchema, err := transform.SchemaTransform(storepb.Engine_MYSQL, converted.Schema)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert schema to sdl format"))
			}
			converted.Schema = sdlSchema
			converted.SchemaSize = int64(len(sdlSchema))
			sdlSchema, err = transform.SchemaTransform(storepb.Engine_MYSQL, converted.PrevSchema)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert previous schema to sdl format"))
			}
			converted.PrevSchema = sdlSchema
			converted.PrevSchemaSize = int64(len(sdlSchema))
		}
	}
	return connect.NewResponse(converted), nil
}

func (s *DatabaseService) convertToChangelogs(d *store.DatabaseMessage, cs []*store.ChangelogMessage) []*v1pb.Changelog {
	var changelogs []*v1pb.Changelog
	for _, c := range cs {
		changelog := s.convertToChangelog(d, c)
		changelogs = append(changelogs, changelog)
	}
	return changelogs
}

func (*DatabaseService) convertToChangelog(d *store.DatabaseMessage, c *store.ChangelogMessage) *v1pb.Changelog {
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

	return cl
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
