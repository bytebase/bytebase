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
	tp, err := convertToAPIWebhookTypeString(webhook.Type)
	if err != nil {
		return nil, err
	}
	activityTypes, err := convertToActivityTypeStrings(webhook.NotificationTypes)
	if err != nil {
		return nil, err
	}
	return &store.ProjectWebhookMessage{
		Type:   tp,
		URL:    webhook.Url,
		Title:  webhook.Title,
		Events: activityTypes,
		Payload: &storepb.ProjectWebhookPayload{
			DirectMessage: webhook.DirectMessage,
		},
	}, nil
}

func convertToActivityTypeStrings(types []v1pb.Activity_Type) ([]string, error) {
	var result []string
	for _, tp := range types {
		switch tp {
		case v1pb.Activity_TYPE_UNSPECIFIED:
			return nil, common.Errorf(common.Invalid, "activity type must not be unspecified")
		case v1pb.Activity_ISSUE_CREATE:
			result = append(result, string(common.EventTypeIssueCreate))
		case v1pb.Activity_ISSUE_COMMENT_CREATE:
			result = append(result, string(common.EventTypeIssueCommentCreate))
		case v1pb.Activity_ISSUE_FIELD_UPDATE:
			result = append(result, string(common.EventTypeIssueUpdate))
		case v1pb.Activity_ISSUE_STATUS_UPDATE:
			result = append(result, string(common.EventTypeIssueStatusUpdate))
		case v1pb.Activity_ISSUE_APPROVAL_NOTIFY:
			result = append(result, string(common.EventTypeIssueApprovalCreate))
		case v1pb.Activity_ISSUE_PIPELINE_STAGE_STATUS_UPDATE:
			result = append(result, string(common.EventTypeStageStatusUpdate))
		case v1pb.Activity_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE:
			result = append(result, string(common.EventTypeTaskRunStatusUpdate))
		case v1pb.Activity_NOTIFY_ISSUE_APPROVED:
			result = append(result, string(common.EventTypeIssueApprovalPass))
		case v1pb.Activity_NOTIFY_PIPELINE_ROLLOUT:
			result = append(result, string(common.EventTypeIssueRolloutReady))
		default:
			return nil, common.Errorf(common.Invalid, "unsupported activity type: %v", tp)
		}
	}
	return result, nil
}

func convertNotificationTypeStrings(types []string) []v1pb.Activity_Type {
	var result []v1pb.Activity_Type
	for _, tp := range types {
		switch tp {
		case string(common.EventTypeIssueCreate):
			result = append(result, v1pb.Activity_ISSUE_CREATE)
		case string(common.EventTypeIssueCommentCreate):
			result = append(result, v1pb.Activity_ISSUE_COMMENT_CREATE)
		case string(common.EventTypeIssueUpdate):
			result = append(result, v1pb.Activity_ISSUE_FIELD_UPDATE)
		case string(common.EventTypeIssueStatusUpdate):
			result = append(result, v1pb.Activity_ISSUE_STATUS_UPDATE)
		case string(common.EventTypeIssueApprovalCreate):
			result = append(result, v1pb.Activity_ISSUE_APPROVAL_NOTIFY)
		case string(common.EventTypeStageStatusUpdate):
			result = append(result, v1pb.Activity_ISSUE_PIPELINE_STAGE_STATUS_UPDATE)
		case string(common.EventTypeTaskRunStatusUpdate):
			result = append(result, v1pb.Activity_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE)
		case string(common.EventTypeIssueApprovalPass):
			result = append(result, v1pb.Activity_NOTIFY_ISSUE_APPROVED)
		case string(common.EventTypeIssueRolloutReady):
			result = append(result, v1pb.Activity_NOTIFY_PIPELINE_ROLLOUT)
		default:
			result = append(result, v1pb.Activity_TYPE_UNSPECIFIED)
		}
	}
	return result
}

func convertToAPIWebhookTypeString(tp v1pb.Webhook_Type) (string, error) {
	switch tp {
	case v1pb.Webhook_TYPE_UNSPECIFIED:
		return "", common.Errorf(common.Invalid, "webhook type must not be unspecified")
	// TODO(zp): find a better way to place the "bb.plugin.webhook.*".
	case v1pb.Webhook_SLACK:
		return "bb.plugin.webhook.slack", nil
	case v1pb.Webhook_DISCORD:
		return "bb.plugin.webhook.discord", nil
	case v1pb.Webhook_TEAMS:
		return "bb.plugin.webhook.teams", nil
	case v1pb.Webhook_DINGTALK:
		return "bb.plugin.webhook.dingtalk", nil
	case v1pb.Webhook_FEISHU:
		return "bb.plugin.webhook.feishu", nil
	case v1pb.Webhook_WECOM:
		return "bb.plugin.webhook.wecom", nil
	case v1pb.Webhook_LARK:
		return "bb.plugin.webhook.lark", nil
	default:
		return "", common.Errorf(common.Invalid, "webhook type %q is not supported", tp)
	}
}

func convertWebhookTypeString(tp string) v1pb.Webhook_Type {
	switch tp {
	case "bb.plugin.webhook.slack":
		return v1pb.Webhook_SLACK
	case "bb.plugin.webhook.discord":
		return v1pb.Webhook_DISCORD
	case "bb.plugin.webhook.teams":
		return v1pb.Webhook_TEAMS
	case "bb.plugin.webhook.dingtalk":
		return v1pb.Webhook_DINGTALK
	case "bb.plugin.webhook.feishu":
		return v1pb.Webhook_FEISHU
	case "bb.plugin.webhook.wecom":
		return v1pb.Webhook_WECOM
	case "bb.plugin.webhook.lark":
		return v1pb.Webhook_LARK
	default:
		return v1pb.Webhook_TYPE_UNSPECIFIED
	}
}

func convertToV1MemberInBinding(ctx context.Context, stores *store.Store, member string) string {
	if strings.HasPrefix(member, common.UserNamePrefix) {
		userUID, err := common.GetUserID(member)
		if err != nil {
			slog.Error("failed to user id from member", slog.String("member", member), log.BBError(err))
			return ""
		}
		user, err := stores.GetUserByID(ctx, userUID)
		if err != nil {
			slog.Error("failed to get user", slog.String("member", member), log.BBError(err))
			return ""
		}
		if user == nil {
			return ""
		}
		return fmt.Sprintf("%s%s", common.UserBindingPrefix, user.Email)
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
		return common.FormatUserUID(user.ID), nil
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
			Type:              convertWebhookTypeString(webhook.Type),
			Title:             webhook.Title,
			Url:               webhook.URL,
			NotificationTypes: convertNotificationTypeStrings(webhook.Events),
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
		AllowModifyStatement:       projectMessage.Setting.AllowModifyStatement,
		AutoResolveIssue:           projectMessage.Setting.AutoResolveIssue,
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
		AllowModifyStatement:       project.AllowModifyStatement,
		AutoResolveIssue:           project.AutoResolveIssue,
		EnforceIssueTitle:          project.EnforceIssueTitle,
		AutoEnableBackup:           project.AutoEnableBackup,
		SkipBackupErrors:           project.SkipBackupErrors,
		PostgresDatabaseTenantMode: project.PostgresDatabaseTenantMode,
		AllowSelfApproval:          project.AllowSelfApproval,
		CiSamplingSize:             project.CiSamplingSize,
		ParallelTasksPerRollout:    project.ParallelTasksPerRollout,
		Labels:                     project.Labels,
		EnforceSqlReview:           project.EnforceSqlReview,
	}
	return &store.ProjectMessage{
		ResourceID: resourceID,
		Title:      project.Title,
		Setting:    setting,
	}
}
