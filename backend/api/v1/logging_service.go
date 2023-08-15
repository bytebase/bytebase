package v1

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// LoggingService implements the logging service.
type LoggingService struct {
	v1pb.UnimplementedLoggingServiceServer
	store *store.Store
}

// NewLoggingService creates a new LoggingService.
func NewLoggingService(store *store.Store) *LoggingService {
	return &LoggingService{
		store: store,
	}
}

var resourceActionTypeMap = map[string][]api.ActivityType{
	"users": {
		api.ActivityMemberCreate,
		api.ActivityMemberRoleUpdate,
		api.ActivityMemberActivate,
		api.ActivityMemberDeactivate,
	},
	"instances": {
		api.ActivitySQLEditorQuery,
		api.ActivitySQLExport,
	},
	"projects": {
		api.ActivityProjectRepositoryPush,
		api.ActivityProjectDatabaseTransfer,
		api.ActivityProjectMemberCreate,
		api.ActivityProjectMemberDelete,
		api.ActivityDatabaseRecoveryPITRDone,
	},
	"pipelines": {
		api.ActivityPipelineStageStatusUpdate,
		api.ActivityPipelineTaskStatusUpdate,
		api.ActivityPipelineTaskFileCommit,
		api.ActivityPipelineTaskStatementUpdate,
		api.ActivityPipelineTaskEarliestAllowedTimeUpdate,
	},
	"issues": {
		api.ActivityIssueCreate,
		api.ActivityIssueCommentCreate,
		api.ActivityIssueFieldUpdate,
		api.ActivityIssueStatusUpdate,
		api.ActivityIssueApprovalNotify,
	},
}

// ListLogs lists the logs.
func (s *LoggingService) ListLogs(ctx context.Context, request *v1pb.ListLogsRequest) (*v1pb.ListLogsResponse, error) {
	var pageToken storepb.PageToken
	if request.PageToken != "" {
		if err := unmarshalPageToken(request.PageToken, &pageToken); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid page token: %v", err)
		}
		if pageToken.Limit != request.PageSize {
			return nil, status.Errorf(codes.InvalidArgument, "request page size does not match the page token")
		}
	} else {
		pageToken.Limit = request.PageSize
	}

	limit := int(pageToken.Limit)
	if limit <= 0 {
		limit = 10
	}
	if limit > 1000 {
		limit = 1000
	}
	limitPlusOne := limit + 1
	offset := int(pageToken.Offset)

	activityFind := &store.FindActivityMessage{
		Limit:  &limitPlusOne,
		Offset: &offset,
	}

	filters, err := parseFilter(request.Filter)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	if request.OrderBy != "" {
		orderByKeys, err := parseOrderBy(request.OrderBy)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		if len(orderByKeys) != 1 || orderByKeys[0].key != "create_time" {
			return nil, status.Errorf(codes.InvalidArgument, `invalid order_by, only support order by "create_time" for now`)
		}
		order := api.DESC
		if orderByKeys[0].isAscend {
			order = api.ASC
		}
		activityFind.Order = &order
	}

	for _, spec := range filters {
		switch spec.key {
		case "creator":
			if spec.operator != comparatorTypeEqual {
				return nil, status.Errorf(codes.InvalidArgument, `only support "=" operation for "creator" filter`)
			}
			creatorEmail := strings.TrimPrefix(spec.value, "users/")
			if creatorEmail == "" {
				return nil, status.Errorf(codes.InvalidArgument, "invalid empty creator identifier")
			}
			user, err := s.store.GetUser(ctx, &store.FindUserMessage{
				Email:       &creatorEmail,
				ShowDeleted: true,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, `failed to find user "%s" with error: %v`, creatorEmail, err.Error())
			}
			if user == nil {
				return nil, errors.Errorf("cannot found user %s", creatorEmail)
			}
			activityFind.CreatorUID = &user.ID
		case "resource":
			if spec.operator != comparatorTypeEqual {
				return nil, status.Errorf(codes.InvalidArgument, `only support "=" operation for "resource" filter`)
			}
			sections := strings.Split(spec.value, "/")
			if len(sections) != 2 {
				return nil, status.Errorf(codes.InvalidArgument, `invalid resource "%s" for filter`, spec.value)
			}
			typeList, ok := resourceActionTypeMap[sections[0]]
			if !ok {
				return nil, status.Errorf(codes.InvalidArgument, `unsupport resource %s`, spec.value)
			}
			activityFind.TypeList = append(activityFind.TypeList, typeList...)
			switch fmt.Sprintf("%s/", sections[0]) {
			case common.UserNamePrefix:
				user, err := s.store.GetUser(ctx, &store.FindUserMessage{
					Email:       &sections[1],
					ShowDeleted: true,
				})
				if err != nil {
					return nil, status.Errorf(codes.Internal, err.Error())
				}
				if user == nil {
					return nil, status.Errorf(codes.NotFound, "user %q not found", spec.value)
				}
				activityFind.ContainerUID = &user.ID
			case common.InstanceNamePrefix:
				instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
					ResourceID:  &sections[1],
					ShowDeleted: true,
				})
				if err != nil {
					return nil, status.Errorf(codes.Internal, err.Error())
				}
				if instance == nil {
					return nil, status.Errorf(codes.NotFound, "instance %q not found", spec.value)
				}
				activityFind.ContainerUID = &instance.UID
			case common.ProjectNamePrefix:
				project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
					ResourceID:  &sections[1],
					ShowDeleted: true,
				})
				if err != nil {
					return nil, status.Errorf(codes.Internal, err.Error())
				}
				if project == nil {
					return nil, status.Errorf(codes.NotFound, "project %q not found", spec.value)
				}
				activityFind.ContainerUID = &project.UID
			case common.PipelineNamePrefix, common.IssueNamePrefix:
				uid, err := strconv.Atoi(sections[1])
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, `invalid resource id "%s"`, spec.value)
				}
				activityFind.ContainerUID = &uid
			default:
				return nil, status.Errorf(codes.InvalidArgument, `resource "%s" in filter is not support`, spec.value)
			}
		case "level":
			if spec.operator != comparatorTypeEqual {
				return nil, status.Errorf(codes.InvalidArgument, `only support "=" operation for "level" filter`)
			}
			for _, level := range strings.Split(spec.value, " | ") {
				activityLevel, err := convertToActivityLevel(v1pb.LogEntity_Level(v1pb.LogEntity_Level_value[level]))
				if err != nil {
					return nil, err
				}
				activityFind.LevelList = append(activityFind.LevelList, activityLevel)
			}
		case "action":
			if spec.operator != comparatorTypeEqual {
				return nil, status.Errorf(codes.InvalidArgument, `only support "=" operation for "action" filter`)
			}
			for _, action := range strings.Split(spec.value, " | ") {
				activityType, err := convertToActivityType(v1pb.LogEntity_Action(v1pb.LogEntity_Action_value[action]))
				if err != nil {
					return nil, err
				}
				activityFind.TypeList = append(activityFind.TypeList, activityType)
			}
		case "create_time":
			if spec.operator != comparatorTypeGreaterEqual && spec.operator != comparatorTypeLessEqual {
				return nil, status.Errorf(codes.InvalidArgument, `only support "<=" or ">=" operation for "create_time" filter`)
			}
			t, err := time.Parse(time.RFC3339, spec.value)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid start_time filter %q", spec.value)
			}
			ts := t.Unix()
			if spec.operator == comparatorTypeGreaterEqual {
				activityFind.CreatedTsAfter = &ts
			} else {
				activityFind.CreatedTsBefore = &ts
			}
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid filter %s", spec.key)
		}
	}

	activityList, err := s.store.ListActivityV2(ctx, activityFind)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list activity: %v", err.Error())
	}

	nextPageToken := ""
	if len(activityList) == limitPlusOne {
		activityList = activityList[:limit]
		if nextPageToken, err = marshalPageToken(&storepb.PageToken{
			Limit:  int32(limit),
			Offset: int32(limit + offset),
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal next page token, error: %v", err)
		}
	}

	resp := &v1pb.ListLogsResponse{
		NextPageToken: nextPageToken,
	}
	for _, activity := range activityList {
		logEntity, err := convertToLogEntity(ctx, s.store, activity)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert log entity, error: %v", err)
		}
		resp.LogEntities = append(resp.LogEntities, logEntity)
	}

	return resp, nil
}

// GetLog gets the log.
func (s *LoggingService) GetLog(ctx context.Context, request *v1pb.GetLogRequest) (*v1pb.LogEntity, error) {
	activityUID, err := common.GetUIDFromName(request.Name, common.LogNamePrefix)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	activity, err := s.store.GetActivityV2(ctx, activityUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find activity with error: %v", err.Error())
	}
	if activity == nil {
		return nil, status.Errorf(codes.NotFound, "cannot found activity %s", request.Name)
	}

	logEntity, err := convertToLogEntity(ctx, s.store, activity)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert log entity, error: %v", err)
	}
	return logEntity, nil
}

func convertToLogEntity(ctx context.Context, db *store.Store, activity *store.ActivityMessage) (*v1pb.LogEntity, error) {
	resource := ""
	switch activity.Type {
	case
		api.ActivityMemberCreate,
		api.ActivityMemberRoleUpdate,
		api.ActivityMemberActivate,
		api.ActivityMemberDeactivate:
		user, err := db.GetUserByID(ctx, activity.ContainerUID)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, errors.Errorf("cannot found user with id %d", activity.ContainerUID)
		}
		resource = fmt.Sprintf("%s%s", common.UserNamePrefix, user.Email)
	case
		api.ActivityIssueCreate,
		api.ActivityIssueCommentCreate,
		api.ActivityIssueFieldUpdate,
		api.ActivityIssueStatusUpdate,
		api.ActivityIssueApprovalNotify:
		resource = fmt.Sprintf("%s%d", common.IssueNamePrefix, activity.ContainerUID)
	case
		api.ActivityPipelineStageStatusUpdate,
		api.ActivityPipelineTaskStatusUpdate,
		api.ActivityPipelineTaskFileCommit,
		api.ActivityPipelineTaskStatementUpdate,
		api.ActivityPipelineTaskEarliestAllowedTimeUpdate:
		resource = fmt.Sprintf("%s%d", common.PipelineNamePrefix, activity.ContainerUID)
	case
		api.ActivityProjectRepositoryPush,
		api.ActivityProjectDatabaseTransfer,
		api.ActivityProjectMemberCreate,
		api.ActivityProjectMemberDelete,
		api.ActivityDatabaseRecoveryPITRDone:
		project, err := db.GetProjectV2(ctx, &store.FindProjectMessage{
			UID: &activity.ContainerUID,
		})
		if err != nil {
			return nil, err
		}
		if project == nil {
			return nil, errors.Errorf("failed to find project by id %d", activity.ContainerUID)
		}
		resource = fmt.Sprintf("%s%s", common.ProjectNamePrefix, project.ResourceID)
	case
		api.ActivitySQLEditorQuery,
		api.ActivitySQLExport:
		instance, err := db.GetInstanceV2(ctx, &store.FindInstanceMessage{
			UID: &activity.ContainerUID,
		})
		if err != nil {
			return nil, err
		}
		if instance == nil {
			return nil, errors.Errorf("failed to find instance by id %d", activity.ContainerUID)
		}
		resource = fmt.Sprintf("%s%s", common.InstanceNamePrefix, instance.ResourceID)
	}

	user, err := db.GetUserByID(ctx, activity.CreatorUID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.Errorf("cannot found user with id %d", activity.ContainerUID)
	}

	return &v1pb.LogEntity{
		Name:       fmt.Sprintf("%s%d", common.LogNamePrefix, activity.UID),
		Creator:    fmt.Sprintf("%s%s", common.UserNamePrefix, user.Email),
		Resource:   resource,
		Action:     convertToActionType(activity.Type),
		Level:      convertToLogLevel(activity.Level),
		CreateTime: timestamppb.New(time.Unix(activity.CreatedTs, 0)),
		UpdateTime: timestamppb.New(time.Unix(activity.UpdatedTs, 0)),
		Comment:    activity.Comment,
		Payload:    activity.Payload,
	}, nil
}

func convertToActivityType(action v1pb.LogEntity_Action) (api.ActivityType, error) {
	switch action {
	case v1pb.LogEntity_ACTION_MEMBER_CREATE:
		return api.ActivityMemberCreate, nil
	case v1pb.LogEntity_ACTION_MEMBER_ROLE_UPDATE:
		return api.ActivityMemberRoleUpdate, nil
	case v1pb.LogEntity_ACTION_MEMBER_ACTIVATE:
		return api.ActivityMemberActivate, nil
	case v1pb.LogEntity_ACTION_MEMBER_DEACTIVE:
		return api.ActivityMemberDeactivate, nil

	case v1pb.LogEntity_ACTION_ISSUE_CREATE:
		return api.ActivityIssueCreate, nil
	case v1pb.LogEntity_ACTION_ISSUE_COMMENT_CREATE:
		return api.ActivityIssueCommentCreate, nil
	case v1pb.LogEntity_ACTION_ISSUE_FIELD_UPDATE:
		return api.ActivityIssueFieldUpdate, nil
	case v1pb.LogEntity_ACTION_ISSUE_STATUS_UPDATE:
		return api.ActivityIssueStatusUpdate, nil
	case v1pb.LogEntity_ACTION_ISSUE_APPROVAL_NOTIFY:
		return api.ActivityIssueApprovalNotify, nil

	case v1pb.LogEntity_ACTION_PIPELINE_STAGE_STATUS_UPDATE:
		return api.ActivityPipelineStageStatusUpdate, nil
	case v1pb.LogEntity_ACTION_PIPELINE_TASK_STATUS_UPDATE:
		return api.ActivityPipelineTaskStatusUpdate, nil
	case v1pb.LogEntity_ACTION_PIPELINE_TASK_FILE_COMMIT:
		return api.ActivityPipelineTaskFileCommit, nil
	case v1pb.LogEntity_ACTION_PIPELINE_TASK_STATEMENT_UPDATE:
		return api.ActivityPipelineTaskStatementUpdate, nil
	case v1pb.LogEntity_ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE:
		return api.ActivityPipelineTaskEarliestAllowedTimeUpdate, nil

	case v1pb.LogEntity_ACTION_PROJECT_REPOSITORY_PUSH:
		return api.ActivityProjectRepositoryPush, nil
	case v1pb.LogEntity_ACTION_PROJECT_DATABASE_TRANSFER:
		return api.ActivityProjectDatabaseTransfer, nil
	case v1pb.LogEntity_ACTION_PROJECT_MEMBER_CREATE:
		return api.ActivityProjectMemberCreate, nil
	case v1pb.LogEntity_ACTION_PROJECT_MEMBER_DELETE:
		return api.ActivityProjectMemberDelete, nil
	case v1pb.LogEntity_ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE:
		return api.ActivityDatabaseRecoveryPITRDone, nil

	case v1pb.LogEntity_ACTION_DATABASE_SQL_EDITOR_QUERY:
		return api.ActivitySQLEditorQuery, nil
	case v1pb.LogEntity_ACTION_DATABASE_SQL_EXPORT:
		return api.ActivitySQLExport, nil
	default:
		return api.ActivityMemberCreate, status.Errorf(codes.InvalidArgument, "unsupported action type: %v", action)
	}
}

func convertToActionType(activityType api.ActivityType) v1pb.LogEntity_Action {
	switch activityType {
	case api.ActivityMemberCreate:
		return v1pb.LogEntity_ACTION_MEMBER_CREATE
	case api.ActivityMemberRoleUpdate:
		return v1pb.LogEntity_ACTION_MEMBER_ROLE_UPDATE
	case api.ActivityMemberActivate:
		return v1pb.LogEntity_ACTION_MEMBER_ACTIVATE
	case api.ActivityMemberDeactivate:
		return v1pb.LogEntity_ACTION_MEMBER_DEACTIVE

	case api.ActivityIssueCreate:
		return v1pb.LogEntity_ACTION_ISSUE_CREATE
	case api.ActivityIssueCommentCreate:
		return v1pb.LogEntity_ACTION_ISSUE_COMMENT_CREATE
	case api.ActivityIssueFieldUpdate:
		return v1pb.LogEntity_ACTION_ISSUE_FIELD_UPDATE
	case api.ActivityIssueStatusUpdate:
		return v1pb.LogEntity_ACTION_ISSUE_STATUS_UPDATE
	case api.ActivityIssueApprovalNotify:
		return v1pb.LogEntity_ACTION_ISSUE_APPROVAL_NOTIFY

	case api.ActivityPipelineStageStatusUpdate:
		return v1pb.LogEntity_ACTION_PIPELINE_STAGE_STATUS_UPDATE
	case api.ActivityPipelineTaskStatusUpdate:
		return v1pb.LogEntity_ACTION_PIPELINE_TASK_STATUS_UPDATE
	case api.ActivityPipelineTaskFileCommit:
		return v1pb.LogEntity_ACTION_PIPELINE_TASK_FILE_COMMIT
	case api.ActivityPipelineTaskStatementUpdate:
		return v1pb.LogEntity_ACTION_PIPELINE_TASK_STATEMENT_UPDATE
	case api.ActivityPipelineTaskEarliestAllowedTimeUpdate:
		return v1pb.LogEntity_ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE

	case api.ActivityProjectRepositoryPush:
		return v1pb.LogEntity_ACTION_PROJECT_REPOSITORY_PUSH
	case api.ActivityProjectDatabaseTransfer:
		return v1pb.LogEntity_ACTION_PROJECT_DATABASE_TRANSFER
	case api.ActivityProjectMemberCreate:
		return v1pb.LogEntity_ACTION_PROJECT_MEMBER_CREATE
	case api.ActivityProjectMemberDelete:
		return v1pb.LogEntity_ACTION_PROJECT_MEMBER_DELETE
	case api.ActivityDatabaseRecoveryPITRDone:
		return v1pb.LogEntity_ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE

	case api.ActivitySQLEditorQuery:
		return v1pb.LogEntity_ACTION_DATABASE_SQL_EDITOR_QUERY
	case api.ActivitySQLExport:
		return v1pb.LogEntity_ACTION_DATABASE_SQL_EXPORT
	default:
		return v1pb.LogEntity_ACTION_UNSPECIFIED
	}
}

func convertToActivityLevel(logLevel v1pb.LogEntity_Level) (api.ActivityLevel, error) {
	switch logLevel {
	case v1pb.LogEntity_LEVEL_ERROR:
		return api.ActivityError, nil
	case v1pb.LogEntity_LEVEL_WARNING:
		return api.ActivityWarn, nil
	case v1pb.LogEntity_LEVEL_INFO:
		return api.ActivityInfo, nil
	default:
		return api.ActivityError, status.Errorf(codes.InvalidArgument, "unsupport log level %v", logLevel)
	}
}

func convertToLogLevel(activityLevel api.ActivityLevel) v1pb.LogEntity_Level {
	switch activityLevel {
	case api.ActivityInfo:
		return v1pb.LogEntity_LEVEL_INFO
	case api.ActivityWarn:
		return v1pb.LogEntity_LEVEL_WARNING
	case api.ActivityError:
		return v1pb.LogEntity_LEVEL_ERROR
	default:
		return v1pb.LogEntity_LEVEL_UNSPECIFIED
	}
}
