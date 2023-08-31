import { t } from "@/plugins/i18n";
import { EMPTY_ID } from "../const";
import {
  Activity_Type,
  Webhook,
  Webhook_Type,
} from "../proto/v1/project_service";

export const emptyProjectWebhook = () => {
  return Webhook.fromJSON({
    name: `projects/${EMPTY_ID}/webhooks/${EMPTY_ID}`,
    type: Webhook_Type.TYPE_SLACK,
    title: "",
    url: "",
    notificationTypes: [Activity_Type.TYPE_ISSUE_STATUS_UPDATE],
  });
};

type ProjectWebhookV1TypeItem = {
  type: Webhook_Type;
  name: string;
  urlPrefix: string;
};

export const projectWebhookV1TypeItemList = (): ProjectWebhookV1TypeItem[] => {
  return [
    {
      type: Webhook_Type.TYPE_SLACK,
      name: t("common.slack"),
      urlPrefix: "https://hooks.slack.com/",
    },
    {
      type: Webhook_Type.TYPE_DISCORD,
      name: t("common.discord"),
      urlPrefix: "https://discord.com/api/webhooks",
    },
    {
      type: Webhook_Type.TYPE_TEAMS,
      name: t("common.teams"),
      urlPrefix: "",
    },
    {
      type: Webhook_Type.TYPE_DINGTALK,
      name: t("common.dingtalk"),
      urlPrefix: "https://oapi.dingtalk.com",
    },
    {
      type: Webhook_Type.TYPE_FEISHU,
      name: t("common.feishu"),
      urlPrefix: "https://open.feishu.cn",
    },
    {
      type: Webhook_Type.TYPE_WECOM,
      name: t("common.wecom"),
      urlPrefix: "https://qyapi.weixin.qq.com",
    },
    {
      type: Webhook_Type.TYPE_CUSTOM,
      name: t("common.custom"),
      urlPrefix:
        "https://bytebase.com/docs/change-database/webhook#custom?source=console",
    },
  ];
};

type ProjectWebhookV1ActivityItem = {
  title: string;
  label: string;
  activity: Activity_Type;
};

export const projectWebhookV1ActivityItemList =
  (): ProjectWebhookV1ActivityItem[] => {
    return [
      {
        title: t("project.webhook.activity-item.issue-creation.title"),
        label: t("project.webhook.activity-item.issue-creation.label"),
        activity: Activity_Type.TYPE_ISSUE_CREATE,
      },
      {
        title: t("project.webhook.activity-item.issue-status-change.title"),
        label: t("project.webhook.activity-item.issue-status-change.label"),
        activity: Activity_Type.TYPE_ISSUE_STATUS_UPDATE,
      },
      {
        title: t(
          "project.webhook.activity-item.issue-stage-status-change.title"
        ),
        label: t(
          "project.webhook.activity-item.issue-stage-status-change.label"
        ),
        activity: Activity_Type.TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE,
      },
      {
        title: t(
          "project.webhook.activity-item.issue-task-status-change.title"
        ),
        label: t(
          "project.webhook.activity-item.issue-task-status-change.label"
        ),
        activity: Activity_Type.TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE,
      },
      {
        title: t(
          "project.webhook.activity-item.issue-task-run-status-change.title"
        ),
        label: t(
          "project.webhook.activity-item.issue-task-run-status-change.label"
        ),
        activity: Activity_Type.TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE,
      },
      {
        title: t("project.webhook.activity-item.issue-info-change.title"),
        label: t("project.webhook.activity-item.issue-info-change.label"),
        activity: Activity_Type.TYPE_ISSUE_FIELD_UPDATE,
      },
      {
        title: t("project.webhook.activity-item.issue-comment-creation.title"),
        label: t("project.webhook.activity-item.issue-comment-creation.label"),
        activity: Activity_Type.TYPE_ISSUE_COMMENT_CREATE,
      },
      {
        title: t("project.webhook.activity-item.issue-approval-notify.title"),
        label: t("project.webhook.activity-item.issue-approval-notify.label"),
        activity: Activity_Type.TYPE_ISSUE_APPROVAL_NOTIFY,
      },
    ];
  };
