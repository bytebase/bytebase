package v1

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// ChangelogService implements the changelog service.
type ChangelogService struct {
	v1connect.UnimplementedChangelogServiceHandler
	store *store.Store
}

// NewChangelogService creates a new ChangelogService.
func NewChangelogService(store *store.Store) *ChangelogService {
	return &ChangelogService{
		store: store,
	}
}

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
				default:
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %v", variable))
				}
			case celoperators.GreaterEquals, celoperators.LessEquals:
				variable, rawValue := getVariableAndValueFromExpr(expr)
				value, ok := rawValue.(string)
				if !ok {
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("expect string, got %T, hint: filter literals should be string", rawValue))
				}
				if variable != "create_time" {
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`">=" and "<=" are only supported for "create_time"`))
				}
				t, err := time.Parse(time.RFC3339, value)
				if err != nil {
					return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse time %v, error: %v", value, err))
				}
				if functionName == celoperators.GreaterEquals {
					find.CreatedAtAfter = &t
				} else {
					find.CreatedAtBefore = &t
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

func (s *ChangelogService) ListChangelogs(ctx context.Context, req *connect.Request[v1pb.ListChangelogsRequest]) (*connect.Response[v1pb.ListChangelogsResponse], error) {
	instance, database, err := getInstanceDatabaseMessage(ctx, s.store, req.Msg.Parent)
	if err != nil {
		return nil, err
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
		InstanceID:   database.InstanceID,
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
	converted := convertToChangelogs(instance, database, changelogs)
	return connect.NewResponse(&v1pb.ListChangelogsResponse{
		Changelogs:    converted,
		NextPageToken: nextPageToken,
	}), nil
}

func (s *ChangelogService) GetChangelog(ctx context.Context, req *connect.Request[v1pb.GetChangelogRequest]) (*connect.Response[v1pb.Changelog], error) {
	parent, changelogID, err := getChangelogParentAndID(req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", req.Msg.Name))
	}
	instance, database, err := getInstanceDatabaseMessage(ctx, s.store, parent)
	if err != nil {
		return nil, err
	}

	find := &store.FindChangelogMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: &database.DatabaseName,
		ResourceID:   &changelogID,
	}
	if req.Msg.View == v1pb.ChangelogView_CHANGELOG_VIEW_FULL {
		find.ShowFull = true
	}

	changelog, err := s.store.GetChangelog(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list changelogs"))
	}
	if changelog == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("changelog %q not found", changelogID))
	}

	converted := convertToChangelog(instance, database, changelog)
	return connect.NewResponse(converted), nil
}

func getInstanceDatabaseMessage(ctx context.Context, stores *store.Store, parent string) (*store.InstanceMessage, *store.DatabaseMessage, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(parent)
	if err != nil {
		if projectID, nestedInstanceID, nestedDatabaseName, projectErr := common.GetProjectIDInstanceDatabaseID(parent); projectErr == nil {
			instanceID, databaseName = nestedInstanceID, nestedDatabaseName
			instance, getErr := getInstanceMessage(ctx, stores, common.FormatProjectInstance(projectID, instanceID))
			if getErr != nil {
				return nil, nil, getErr
			}
			database, getErr := getDatabaseMessage(ctx, stores, instanceID, databaseName, parent)
			return instance, database, getErr
		}
		return nil, nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to parse %q", parent))
	}

	instance, err := getInstanceMessage(ctx, stores, common.FormatInstance(instanceID))
	if err != nil {
		return nil, nil, err
	}
	database, err := getDatabaseMessage(ctx, stores, instanceID, databaseName, parent)
	return instance, database, err
}

func getDatabaseMessage(ctx context.Context, stores *store.Store, instanceID, databaseName, requestName string) (*store.DatabaseMessage, error) {
	database, err := stores.GetDatabase(ctx, &store.FindDatabaseMessage{
		Workspace:    common.GetWorkspaceIDFromContext(ctx),
		InstanceID:   &instanceID,
		DatabaseName: &databaseName,
		ShowDeleted:  true,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database"))
	}
	if database == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %q not found", requestName))
	}
	return database, nil
}

func getChangelogParentAndID(name string) (string, string, error) {
	if projectID, instanceID, databaseName, changelogID, err := common.GetProjectIDInstanceDatabaseChangelogID(name); err == nil {
		return common.FormatProjectDatabase(projectID, instanceID, databaseName), changelogID, nil
	}
	instanceID, databaseName, changelogID, err := common.GetInstanceDatabaseChangelogID(name)
	if err != nil {
		return "", "", err
	}
	return common.FormatDatabase(instanceID, databaseName), changelogID, nil
}

func convertToChangelogs(instance *store.InstanceMessage, d *store.DatabaseMessage, cs []*store.ChangelogMessage) []*v1pb.Changelog {
	var changelogs []*v1pb.Changelog
	for _, c := range cs {
		changelog := convertToChangelog(instance, d, c)
		changelogs = append(changelogs, changelog)
	}
	return changelogs
}

func convertToChangelog(instance *store.InstanceMessage, d *store.DatabaseMessage, c *store.ChangelogMessage) *v1pb.Changelog {
	name := common.FormatChangelog(d.InstanceID, d.DatabaseName, c.ResourceID)
	if instance.ProjectID != nil {
		name = common.FormatProjectChangelog(*instance.ProjectID, d.InstanceID, d.DatabaseName, c.ResourceID)
	}
	cl := &v1pb.Changelog{
		Name:       name,
		CreateTime: timestamppb.New(c.CreatedAt),
		Status:     convertToChangelogStatus(c.Status),
		Schema:     "",
		SchemaSize: 0,
		TaskRun:    c.Payload.GetTaskRun(),
		PlanTitle:  c.PlanTitle,
	}

	if c.SyncHistory != nil {
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
