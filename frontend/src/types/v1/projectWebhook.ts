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
  urlPlaceholder: string;
  docUrl: string;
  supportDirectMessage: boolean;
};

export const projectWebhookV1TypeItemList = (): ProjectWebhookV1TypeItem[] => {
  return [
    {
      type: Webhook_Type.TYPE_SLACK,
      name: t("common.slack"),
      urlPrefix: "https://hooks.slack.com/",
      urlPlaceholder: "https://hooks.slack.com/services/...",
      docUrl: "https://api.slack.com/messaging/webhooks",
      supportDirectMessage: true,
    },
    {
      type: Webhook_Type.TYPE_DISCORD,
      name: t("common.discord"),
      urlPrefix: "https://discord.com/api/webhooks",
      urlPlaceholder: "https://discord.com/api/webhooks/...",
      docUrl:
        "https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks",
      supportDirectMessage: false,
    },
    {
      type: Webhook_Type.TYPE_TEAMS,
      name: t("common.teams"),
      urlPrefix: "",
      urlPlaceholder: "https://acme123.webhook.office.com/webhookb2/...",
      docUrl:
        "https://docs.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook",
      supportDirectMessage: false,
    },
    {
      type: Webhook_Type.TYPE_DINGTALK,
      name: t("common.dingtalk"),
      urlPrefix: "https://oapi.dingtalk.com",
      urlPlaceholder: "https://oapi.dingtalk.com/robot/...",
      docUrl:
        "https://developers.dingtalk.com/document/robots/custom-robot-access",
      supportDirectMessage: false,
    },
    {
      type: Webhook_Type.TYPE_FEISHU,
      name: t("common.feishu"),
      urlPrefix: "https://open.feishu.cn",
      urlPlaceholder: "https://open.feishu.cn/open-apis/bot/v2/hook/...",
      docUrl:
        "https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot",
      supportDirectMessage: true,
    },
    {
      type: Webhook_Type.TYPE_WECOM,
      name: t("common.wecom"),
      urlPrefix: "https://qyapi.weixin.qq.com",
      urlPlaceholder: "https://qyapi.weixin.qq.com/cgi-bin/webhook/...",
      docUrl: "https://open.work.weixin.qq.com/help2/pc/14931",
      supportDirectMessage: true,
    },
    {
      type: Webhook_Type.TYPE_CUSTOM,
      name: t("common.custom"),
      urlPrefix:
        "https://bytebase.com/docs/change-database/webhook#custom?source=console",
      urlPlaceholder: "https://example.com/api/webhook/...",
      docUrl:
        "https://www.bytebase.com/docs/change-database/webhook#custom?source=console",
      supportDirectMessage: false,
    },
  ];
};

type ProjectWebhookV1ActivityItem = {
  title: string;
  label: string;
  activity: Activity_Type;
  supportDirectMessage: boolean;
};

export const projectWebhookV1ActivityItemList =
  (): ProjectWebhookV1ActivityItem[] => {
    return [
      {
        title: t("project.webhook.activity-item.issue-creation.title"),
        label: t("project.webhook.activity-item.issue-creation.label"),
        activity: Activity_Type.TYPE_ISSUE_CREATE,
        supportDirectMessage: false,
      },
      {
        title: t("project.webhook.activity-item.issue-status-change.title"),
        label: t("project.webhook.activity-item.issue-status-change.label"),
        activity: Activity_Type.TYPE_ISSUE_STATUS_UPDATE,
        supportDirectMessage: false,
      },
      {
        title: t(
          "project.webhook.activity-item.issue-stage-status-change.title"
        ),
        label: t(
          "project.webhook.activity-item.issue-stage-status-change.label"
        ),
        activity: Activity_Type.TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE,
        supportDirectMessage: false,
      },
      {
        title: t(
          "project.webhook.activity-item.issue-task-status-change.title"
        ),
        label: t(
          "project.webhook.activity-item.issue-task-status-change.label"
        ),
        activity: Activity_Type.TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE,
        supportDirectMessage: false,
      },
      {
        title: t("project.webhook.activity-item.issue-info-change.title"),
        label: t("project.webhook.activity-item.issue-info-change.label"),
        activity: Activity_Type.TYPE_ISSUE_FIELD_UPDATE,
        supportDirectMessage: false,
      },
      {
        title: t("project.webhook.activity-item.issue-comment-creation.title"),
        label: t("project.webhook.activity-item.issue-comment-creation.label"),
        activity: Activity_Type.TYPE_ISSUE_COMMENT_CREATE,
        supportDirectMessage: false,
      },
      {
        title: t("project.webhook.activity-item.issue-approval-notify.title"),
        label: t("project.webhook.activity-item.issue-approval-notify.label"),
        activity: Activity_Type.TYPE_ISSUE_APPROVAL_NOTIFY,
        supportDirectMessage: true,
      },
      {
        title: t("project.webhook.activity-item.notify-issue-approved.title"),
        label: t("project.webhook.activity-item.notify-issue-approved.label"),
        activity: Activity_Type.TYPE_NOTIFY_ISSUE_APPROVED,
        supportDirectMessage: true,
      },
      {
        title: t("project.webhook.activity-item.notify-pipeline-rollout.title"),
        label: t("project.webhook.activity-item.notify-pipeline-rollout.label"),
        activity: Activity_Type.TYPE_NOTIFY_PIPELINE_ROLLOUT,
        supportDirectMessage: true,
      },
    ];
  };
