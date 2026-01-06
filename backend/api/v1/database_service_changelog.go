package v1

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func parseChangelogFilter(filter string, find *store.FindChangelogMessage) error {
	if filter == "" {
		return nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return connect.NewError(connect.CodeInternal, errors.Errorf("failed to create cel env"))
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String()))
	}

	var parseFilter func(expr celast.Expr) error

	parseFilter = func(expr celast.Expr) error {
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalAnd:
				for _, arg := range expr.AsCall().Args() {
					if err := parseFilter(arg); err != nil {
						return err
					}
				}
			case celoperators.In:
				variable, value := getVariableAndValueFromExpr(expr)
				if variable != "type" {
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %v", variable))
				}
				rawList, ok := value.([]any)
				if !ok {
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid list value %q for %v", value, variable))
				}
				if len(rawList) == 0 {
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("empty list value for filter %v", variable))
				}
				typeList := []string{}
				for _, raw := range rawList {
					typeStr, ok := raw.(string)
					if !ok {
						return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("value for type must be a string"))
					}
					v1Type := v1pb.Changelog_Type_value[typeStr]
					storeType := convertToChangelogStoreType(v1pb.Changelog_Type(v1Type))
					typeList = append(typeList, storeType.String())
				}
				find.TypeList = typeList
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				strValue, ok := value.(string)
				if !ok {
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected string but found %q", value))
				}
				switch variable {
				case "status":
					v1Status := v1pb.Changelog_Status_value[strValue]
					storeStatus := convertToChangelogStoreStatus(v1pb.Changelog_Status(v1Status))
					find.Status = &storeStatus
				case "type":
					v1Type := v1pb.Changelog_Type_value[strValue]
					storeType := convertToChangelogStoreType(v1pb.Changelog_Type(v1Type))
					find.TypeList = []string{storeType.String()}
				default:
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %v", variable))
				}
			default:
				return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected function %v", functionName))
			}
		default:
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected expr kind %v", expr.Kind()))
		}
		return nil
	}

	return parseFilter(ast.NativeRep().Expr())
}

func (s *DatabaseService) ListChangelogs(ctx context.Context, req *connect.Request[v1pb.ListChangelogsRequest]) (*connect.Response[v1pb.ListChangelogsResponse], error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(req.Msg.Parent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", req.Msg.Parent))
	}
	database, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
	}
	if database == nil {
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
	if err := parseChangelogFilter(req.Msg.Filter, find); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse the filter %q", req.Msg.Filter))
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
	converted := s.convertToChangelogs(ctx, database, changelogs)
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

	database, err := s.store.GetDatabase(ctx, &store.FindDatabaseMessage{
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if database == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", databaseName))
	}

	converted := s.convertToChangelog(database, changelog)
	return connect.NewResponse(converted), nil
}

func (s *DatabaseService) convertToChangelogs(_ context.Context, d *store.DatabaseMessage, cs []*store.ChangelogMessage) []*v1pb.Changelog {
	var changelogs []*v1pb.Changelog
	for _, c := range cs {
		changelog := s.convertToChangelog(d, c)
		changelogs = append(changelogs, changelog)
	}
	return changelogs
}

func (*DatabaseService) convertToChangelog(d *store.DatabaseMessage, c *store.ChangelogMessage) *v1pb.Changelog {
	changelogType := convertToChangelogType(c.Payload.GetType())

	cl := &v1pb.Changelog{
		Name:       common.FormatChangelog(d.InstanceID, d.DatabaseName, c.UID),
		CreateTime: timestamppb.New(c.CreatedAt),
		Status:     convertToChangelogStatus(c.Status),
		Schema:     "",
		SchemaSize: 0,
		TaskRun:    c.Payload.GetTaskRun(),
		Type:       changelogType,
		PlanTitle:  c.PlanTitle,
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

func convertToChangelogStoreStatus(s v1pb.Changelog_Status) store.ChangelogStatus {
	switch s {
	case v1pb.Changelog_DONE:
		return store.ChangelogStatusDone
	case v1pb.Changelog_FAILED:
		return store.ChangelogStatusFailed
	case v1pb.Changelog_PENDING:
		return store.ChangelogStatusPending
	default:
		return store.ChangelogStatusDone
	}
}

func convertToChangelogType(t storepb.ChangelogPayload_Type) v1pb.Changelog_Type {
	//exhaustive:enforce
	switch t {
	case storepb.ChangelogPayload_BASELINE:
		return v1pb.Changelog_BASELINE
	case storepb.ChangelogPayload_MIGRATE:
		return v1pb.Changelog_MIGRATE
	case storepb.ChangelogPayload_SDL:
		return v1pb.Changelog_SDL
	case storepb.ChangelogPayload_TYPE_UNSPECIFIED:
		return v1pb.Changelog_TYPE_UNSPECIFIED
	default:
		return v1pb.Changelog_TYPE_UNSPECIFIED
	}
}

func convertToChangelogStoreType(t v1pb.Changelog_Type) storepb.ChangelogPayload_Type {
	//exhaustive:enforce
	switch t {
	case v1pb.Changelog_BASELINE:
		return storepb.ChangelogPayload_BASELINE
	case v1pb.Changelog_MIGRATE:
		return storepb.ChangelogPayload_MIGRATE
	case v1pb.Changelog_SDL:
		return storepb.ChangelogPayload_SDL
	case v1pb.Changelog_TYPE_UNSPECIFIED:
		return storepb.ChangelogPayload_TYPE_UNSPECIFIED
	default:
		return storepb.ChangelogPayload_TYPE_UNSPECIFIED
	}
}
