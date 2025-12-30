import { create as createProto } from "@bufbuild/protobuf";
import { t } from "@/plugins/i18n";
import { WebhookType } from "../proto-es/v1/common_pb";
import {
  Activity_Type,
  WebhookSchema,
} from "../proto-es/v1/project_service_pb";

export const emptyProjectWebhook = () => {
  return createProto(WebhookSchema, {
    type: WebhookType.SLACK,
    notificationTypes: [Activity_Type.ISSUE_CREATED],
  });
};

type ProjectWebhookV1TypeItem = {
  type: WebhookType;
  name: string;
  urlPrefix: string;
  urlPlaceholder: string;
  docUrl: string;
  supportDirectMessage: boolean;
};

export const projectWebhookV1TypeItemList = (): ProjectWebhookV1TypeItem[] => {
  return [
    {
      type: WebhookType.SLACK,
      name: t("common.slack"),
      urlPrefix: "https://hooks.slack.com/",
      urlPlaceholder: "https://hooks.slack.com/services/...",
      docUrl: "https://api.slack.com/messaging/webhooks",
      supportDirectMessage: true,
    },
    {
      type: WebhookType.DISCORD,
      name: t("common.discord"),
      urlPrefix: "https://discord.com/api/webhooks",
      urlPlaceholder: "https://discord.com/api/webhooks/...",
      docUrl:
        "https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks",
      supportDirectMessage: false,
    },
    {
      type: WebhookType.TEAMS,
      name: t("common.teams"),
      urlPrefix: "",
      urlPlaceholder: "https://acme123.webhook.office.com/webhookb2/...",
      docUrl:
        "https://docs.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook",
      supportDirectMessage: true,
    },
    {
      type: WebhookType.DINGTALK,
      name: t("common.dingtalk"),
      urlPrefix: "https://oapi.dingtalk.com",
      urlPlaceholder: "https://oapi.dingtalk.com/robot/...",
      docUrl:
        "https://developers.dingtalk.com/document/robots/custom-robot-access",
      supportDirectMessage: true,
    },
    {
      type: WebhookType.FEISHU,
      name: t("common.feishu"),
      urlPrefix: "https://open.feishu.cn",
      urlPlaceholder: "https://open.feishu.cn/open-apis/bot/v2/hook/...",
      docUrl:
        "https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot",
      supportDirectMessage: true,
    },
    {
      type: WebhookType.LARK,
      name: t("common.lark"),
      urlPrefix: "https://open.larksuite.com",
      urlPlaceholder: "https://open.larksuite.com/open-apis/bot/v2/hook/...",
      docUrl:
        "https://open.larksuite.com/document/client-docs/bot-v3/add-custom-bot",
      supportDirectMessage: true,
    },
    {
      type: WebhookType.WECOM,
      name: t("common.wecom"),
      urlPrefix: "https://qyapi.weixin.qq.com",
      urlPlaceholder: "https://qyapi.weixin.qq.com/cgi-bin/webhook/...",
      docUrl: "https://open.work.weixin.qq.com/help2/pc/14931",
      supportDirectMessage: true,
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
        activity: Activity_Type.ISSUE_CREATED,
        supportDirectMessage: false,
      },
      {
        title: t("project.webhook.activity-item.issue-approval-notify.title"),
        label: t("project.webhook.activity-item.issue-approval-notify.label"),
        activity: Activity_Type.ISSUE_APPROVAL_REQUESTED,
        supportDirectMessage: true,
      },
      {
        title: t("project.webhook.activity-item.issue-sent-back.title"),
        label: t("project.webhook.activity-item.issue-sent-back.label"),
        activity: Activity_Type.ISSUE_SENT_BACK,
        supportDirectMessage: true,
      },
      {
        title: t("project.webhook.activity-item.pipeline-failed.title"),
        label: t("project.webhook.activity-item.pipeline-failed.label"),
        activity: Activity_Type.PIPELINE_FAILED,
        supportDirectMessage: false,
      },
      {
        title: t("project.webhook.activity-item.pipeline-completed.title"),
        label: t("project.webhook.activity-item.pipeline-completed.label"),
        activity: Activity_Type.PIPELINE_COMPLETED,
        supportDirectMessage: false,
      },
    ];
  };
