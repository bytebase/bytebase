package v1

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

func convertToProtoAny(i any) (*anypb.Any, error) {
	switch deltas := i.(type) {
	case []*v1pb.BindingDelta:
		auditData := v1pb.AuditData{
			PolicyDelta: &v1pb.PolicyDelta{
				BindingDeltas: deltas,
			},
		}
		return anypb.New(&auditData)
	default:
		return &anypb.Any{}, nil
	}
}

func convertToStoreProjectWebhookMessage(webhook *v1pb.Webhook) (*store.ProjectWebhookMessage, error) {
	storeType, err := convertToStoreWebhookType(webhook.Type)
	if err != nil {
		return nil, err
	}

	activityTypes, err := convertToStoreActivityTypes(webhook.NotificationTypes)
	if err != nil {
		return nil, err
	}
	return &store.ProjectWebhookMessage{
		Payload: &storepb.ProjectWebhook{
			Type:          storeType,
			Title:         webhook.Title,
			Url:           webhook.Url,
			Activities:    activityTypes,
			DirectMessage: webhook.DirectMessage,
		},
	}, nil
}

func convertToStoreActivityTypes(types []v1pb.Activity_Type) ([]storepb.Activity_Type, error) {
	var result []storepb.Activity_Type
	for _, tp := range types {
		switch tp {
		case v1pb.Activity_TYPE_UNSPECIFIED:
			return nil, common.Errorf(common.Invalid, "activity type must not be unspecified")
		case v1pb.Activity_ISSUE_CREATE:
			result = append(result, storepb.Activity_ISSUE_CREATE)
		case v1pb.Activity_ISSUE_COMMENT_CREATE:
			result = append(result, storepb.Activity_ISSUE_COMMENT_CREATE)
		case v1pb.Activity_ISSUE_FIELD_UPDATE:
			result = append(result, storepb.Activity_ISSUE_FIELD_UPDATE)
		case v1pb.Activity_ISSUE_STATUS_UPDATE:
			result = append(result, storepb.Activity_ISSUE_STATUS_UPDATE)
		case v1pb.Activity_ISSUE_APPROVAL_NOTIFY:
			result = append(result, storepb.Activity_ISSUE_APPROVAL_NOTIFY)
		case v1pb.Activity_ISSUE_PIPELINE_STAGE_STATUS_UPDATE:
			result = append(result, storepb.Activity_ISSUE_PIPELINE_STAGE_STATUS_UPDATE)
		case v1pb.Activity_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE:
			result = append(result, storepb.Activity_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE)
		case v1pb.Activity_NOTIFY_ISSUE_APPROVED:
			result = append(result, storepb.Activity_NOTIFY_ISSUE_APPROVED)
		case v1pb.Activity_NOTIFY_PIPELINE_ROLLOUT:
			result = append(result, storepb.Activity_NOTIFY_PIPELINE_ROLLOUT)
		default:
			return nil, common.Errorf(common.Invalid, "unsupported activity type: %v", tp)
		}
	}
	return result, nil
}

func convertToV1ActivityTypes(types []storepb.Activity_Type) []v1pb.Activity_Type {
	var result []v1pb.Activity_Type
	for _, tp := range types {
		switch tp {
		case storepb.Activity_ISSUE_CREATE:
			result = append(result, v1pb.Activity_ISSUE_CREATE)
		case storepb.Activity_ISSUE_COMMENT_CREATE:
			result = append(result, v1pb.Activity_ISSUE_COMMENT_CREATE)
		case storepb.Activity_ISSUE_FIELD_UPDATE:
			result = append(result, v1pb.Activity_ISSUE_FIELD_UPDATE)
		case storepb.Activity_ISSUE_STATUS_UPDATE:
			result = append(result, v1pb.Activity_ISSUE_STATUS_UPDATE)
		case storepb.Activity_ISSUE_APPROVAL_NOTIFY:
			result = append(result, v1pb.Activity_ISSUE_APPROVAL_NOTIFY)
		case storepb.Activity_ISSUE_PIPELINE_STAGE_STATUS_UPDATE:
			result = append(result, v1pb.Activity_ISSUE_PIPELINE_STAGE_STATUS_UPDATE)
		case storepb.Activity_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE:
			result = append(result, v1pb.Activity_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE)
		case storepb.Activity_NOTIFY_ISSUE_APPROVED:
			result = append(result, v1pb.Activity_NOTIFY_ISSUE_APPROVED)
		case storepb.Activity_NOTIFY_PIPELINE_ROLLOUT:
			result = append(result, v1pb.Activity_NOTIFY_PIPELINE_ROLLOUT)
		default:
			result = append(result, v1pb.Activity_TYPE_UNSPECIFIED)
		}
	}
	return result
}

func convertToStoreWebhookType(tp v1pb.WebhookType) (storepb.WebhookType, error) {
	switch tp {
	case v1pb.WebhookType_WEBHOOK_TYPE_UNSPECIFIED:
		return storepb.WebhookType_WEBHOOK_TYPE_UNSPECIFIED, common.Errorf(common.Invalid, "webhook type must not be unspecified")
	case v1pb.WebhookType_SLACK:
		return storepb.WebhookType_SLACK, nil
	case v1pb.WebhookType_DISCORD:
		return storepb.WebhookType_DISCORD, nil
	case v1pb.WebhookType_TEAMS:
		return storepb.WebhookType_TEAMS, nil
	case v1pb.WebhookType_DINGTALK:
		return storepb.WebhookType_DINGTALK, nil
	case v1pb.WebhookType_FEISHU:
		return storepb.WebhookType_FEISHU, nil
	case v1pb.WebhookType_WECOM:
		return storepb.WebhookType_WECOM, nil
	case v1pb.WebhookType_LARK:
		return storepb.WebhookType_LARK, nil
	default:
		return storepb.WebhookType_WEBHOOK_TYPE_UNSPECIFIED, common.Errorf(common.Invalid, "webhook type %q is not supported", tp)
	}
}

func convertToV1WebhookType(tp storepb.WebhookType) v1pb.WebhookType {
	switch tp {
	case storepb.WebhookType_SLACK:
		return v1pb.WebhookType_SLACK
	case storepb.WebhookType_DISCORD:
		return v1pb.WebhookType_DISCORD
	case storepb.WebhookType_TEAMS:
		return v1pb.WebhookType_TEAMS
	case storepb.WebhookType_DINGTALK:
		return v1pb.WebhookType_DINGTALK
	case storepb.WebhookType_FEISHU:
		return v1pb.WebhookType_FEISHU
	case storepb.WebhookType_WECOM:
		return v1pb.WebhookType_WECOM
	case storepb.WebhookType_LARK:
		return v1pb.WebhookType_LARK
	default:
		return v1pb.WebhookType_WEBHOOK_TYPE_UNSPECIFIED
	}
}

func convertToV1MemberInBinding(_ context.Context, _ *store.Store, member string) string {
	if strings.HasPrefix(member, common.UserNamePrefix) {
		email, err := common.GetUserEmail(member)
		if err != nil {
			slog.Error("failed to get user email from member", slog.String("member", member), log.BBError(err))
			return ""
		}
		return fmt.Sprintf("%s%s", common.UserBindingPrefix, email)
	} else if strings.HasPrefix(member, common.GroupPrefix) {
		email, err := common.GetGroupEmail(member)
		if err != nil {
			slog.Error("failed to parse group email from member", slog.String("member", member), log.BBError(err))
			return ""
		}
		return fmt.Sprintf("%s%s", common.GroupBindingPrefix, email)
	}
	// handle allUsers.
	return member
}

func convertToV1IamPolicy(ctx context.Context, stores *store.Store, iamPolicy *store.IamPolicyMessage) (*v1pb.IamPolicy, error) {
	var bindings []*v1pb.Binding

	for _, binding := range iamPolicy.Policy.Bindings {
		var members []string
		for _, member := range binding.Members {
			memberInBinding := convertToV1MemberInBinding(ctx, stores, member)
			if memberInBinding == "" {
				continue
			}
			members = append(members, memberInBinding)
		}
		if len(members) == 0 {
			continue
		}
		v1pbBinding := &v1pb.Binding{
			Role:      binding.Role,
			Members:   members,
			Condition: binding.Condition,
		}
		if v1pbBinding.Condition == nil {
			v1pbBinding.Condition = &expr.Expr{}
		}
		if v1pbBinding.Condition.Expression != "" {
			e, err := cel.NewEnv(common.IAMPolicyConditionCELAttributes...)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create cel environment"))
			}
			ast, issues := e.Parse(v1pbBinding.Condition.Expression)
			if issues != nil && issues.Err() != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to parse expression with error: %v", issues.Err()))
			}
			expr, err := cel.AstToParsedExpr(ast)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert ast to parsed expression with error"))
			}
			v1pbBinding.ParsedExpr = expr.Expr
		}
		bindings = append(bindings, v1pbBinding)
	}

	return &v1pb.IamPolicy{
		Bindings: bindings,
		Etag:     iamPolicy.Etag,
	}, nil
}

func convertToStoreIamPolicy(ctx context.Context, stores *store.Store, iamPolicy *v1pb.IamPolicy) (*storepb.IamPolicy, error) {
	var bindings []*storepb.Binding

	for _, binding := range iamPolicy.Bindings {
		var members []string
		for _, member := range utils.Uniq(binding.Members) {
			storeMember, err := convertToStoreIamPolicyMember(ctx, stores, member)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert iam member with error"))
			}
			members = append(members, storeMember)
		}
		if len(members) == 0 {
			continue
		}

		storeBinding := &storepb.Binding{
			Role:      binding.Role,
			Members:   members,
			Condition: binding.Condition,
		}
		if storeBinding.Condition == nil {
			storeBinding.Condition = &expr.Expr{}
		}
		bindings = append(bindings, storeBinding)
	}

	if len(bindings) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("policy binding is empty"))
	}

	return &storepb.IamPolicy{
		Bindings: bindings,
	}, nil
}

func convertToStoreIamPolicyMember(ctx context.Context, stores *store.Store, member string) (string, error) {
	if strings.HasPrefix(member, common.UserBindingPrefix) {
		email := strings.TrimPrefix(member, common.UserBindingPrefix)
		user, err := stores.GetUserByEmail(ctx, email)
		if err != nil {
			return "", connect.NewError(connect.CodeInternal, err)
		}
		if user == nil {
			return "", connect.NewError(connect.CodeNotFound, errors.Errorf("user %q not found", member))
		}
		return common.FormatUserEmail(user.Email), nil
	} else if strings.HasPrefix(member, common.GroupBindingPrefix) {
		email := strings.TrimPrefix(member, common.GroupBindingPrefix)
		return common.FormatGroupEmail(email), nil
	} else if member == common.AllUsers {
		return member, nil
	}
	return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport member %s", member))
}

func convertToProject(projectMessage *store.ProjectMessage) *v1pb.Project {
	var projectWebhooks []*v1pb.Webhook
	for _, webhook := range projectMessage.Webhooks {
		projectWebhooks = append(projectWebhooks, &v1pb.Webhook{
			Name:              fmt.Sprintf("%s/%s%d", common.FormatProject(projectMessage.ResourceID), common.WebhookIDPrefix, webhook.ID),
			Type:              convertToV1WebhookType(webhook.Payload.GetType()),
			Title:             webhook.Payload.GetTitle(),
			Url:               webhook.Payload.GetUrl(),
			NotificationTypes: convertToV1ActivityTypes(webhook.Payload.GetActivities()),
			DirectMessage:     webhook.Payload.GetDirectMessage(),
		})
	}

	var issueLabels []*v1pb.Label
	for _, label := range projectMessage.Setting.IssueLabels {
		issueLabels = append(issueLabels, &v1pb.Label{
			Value: label.Value,
			Color: label.Color,
			Group: label.Group,
		})
	}

	return &v1pb.Project{
		Name:                       common.FormatProject(projectMessage.ResourceID),
		State:                      convertDeletedToState(projectMessage.Deleted),
		Title:                      projectMessage.Title,
		Webhooks:                   projectWebhooks,
		DataClassificationConfigId: projectMessage.DataClassificationConfigID,
		IssueLabels:                issueLabels,
		ForceIssueLabels:           projectMessage.Setting.ForceIssueLabels,
		EnforceIssueTitle:          projectMessage.Setting.EnforceIssueTitle,
		EnforceSqlReview:           projectMessage.Setting.EnforceSqlReview,
		AutoEnableBackup:           projectMessage.Setting.AutoEnableBackup,
		SkipBackupErrors:           projectMessage.Setting.SkipBackupErrors,
		PostgresDatabaseTenantMode: projectMessage.Setting.PostgresDatabaseTenantMode,
		AllowSelfApproval:          projectMessage.Setting.AllowSelfApproval,
		ExecutionRetryPolicy:       convertToV1ExecutionRetryPolicy(projectMessage.Setting.ExecutionRetryPolicy),
		CiSamplingSize:             projectMessage.Setting.CiSamplingSize,
		ParallelTasksPerRollout:    projectMessage.Setting.ParallelTasksPerRollout,
		Labels:                     projectMessage.Setting.Labels,
		RequireIssueApproval:       projectMessage.Setting.RequireIssueApproval,
		RequirePlanCheckNoError:    projectMessage.Setting.RequirePlanCheckNoError,
	}
}

func convertToV1ExecutionRetryPolicy(policy *storepb.Project_ExecutionRetryPolicy) *v1pb.Project_ExecutionRetryPolicy {
	if policy == nil {
		return &v1pb.Project_ExecutionRetryPolicy{
			MaximumRetries: 0,
		}
	}
	return &v1pb.Project_ExecutionRetryPolicy{
		MaximumRetries: policy.MaximumRetries,
	}
}

func convertToStoreExecutionRetryPolicy(policy *v1pb.Project_ExecutionRetryPolicy) *storepb.Project_ExecutionRetryPolicy {
	if policy == nil {
		return &storepb.Project_ExecutionRetryPolicy{
			MaximumRetries: 0,
		}
	}
	return &storepb.Project_ExecutionRetryPolicy{
		MaximumRetries: policy.MaximumRetries,
	}
}

func convertToProjectMessage(resourceID string, project *v1pb.Project) *store.ProjectMessage {
	setting := &storepb.Project{
		EnforceIssueTitle:          project.EnforceIssueTitle,
		AutoEnableBackup:           project.AutoEnableBackup,
		SkipBackupErrors:           project.SkipBackupErrors,
		PostgresDatabaseTenantMode: project.PostgresDatabaseTenantMode,
		AllowSelfApproval:          project.AllowSelfApproval,
		CiSamplingSize:             project.CiSamplingSize,
		ParallelTasksPerRollout:    project.ParallelTasksPerRollout,
		Labels:                     project.Labels,
		EnforceSqlReview:           project.EnforceSqlReview,
		RequireIssueApproval:       project.RequireIssueApproval,
		RequirePlanCheckNoError:    project.RequirePlanCheckNoError,
	}
	return &store.ProjectMessage{
		ResourceID: resourceID,
		Title:      project.Title,
		Setting:    setting,
	}
}
