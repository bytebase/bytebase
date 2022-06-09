import { ActivityType } from "./activity";
import { MemberId, ProjectId } from "./id";
import { Principal } from "./principal";
import { t } from "../plugins/i18n";

type ProjectWebhookTypeItem = {
  type: string;
  name: string;
  urlPrefix: string;
};

export const PROJECT_HOOK_TYPE_ITEM_LIST: () => ProjectWebhookTypeItem[] =
  () => [
    {
      type: "bb.plugin.webhook.slack",
      name: t("common.slack"),
      urlPrefix: "https://hooks.slack.com/",
    },
    {
      type: "bb.plugin.webhook.discord",
      name: t("common.discord"),
      urlPrefix: "https://discord.com/api/webhooks",
    },
    {
      type: "bb.plugin.webhook.teams",
      name: t("common.teams"),
      urlPrefix: "",
    },
    {
      type: "bb.plugin.webhook.dingtalk",
      name: t("common.dingtalk"),
      urlPrefix: "https://oapi.dingtalk.com",
    },
    {
      type: "bb.plugin.webhook.feishu",
      name: t("common.feishu"),
      urlPrefix: "https://open.feishu.cn",
    },
    {
      type: "bb.plugin.webhook.wecom",
      name: t("common.wecom"),
      urlPrefix: "https://qyapi.weixin.qq.com",
    },
    {
      type: "bb.plugin.webhook.custom",
      name: t("common.custom"),
      urlPrefix:
        "https://www.bytebase.com/docs/use-bytebase/webhook-integration/project-webhook#custom?source=console",
    },
  ];

type ProjectWebhookActivityItem = {
  title: string;
  label: string;
  activity: ActivityType;
};

export const PROJECT_HOOK_ACTIVITY_ITEM_LIST: () => ProjectWebhookActivityItem[] =
  () => [
    {
      title: t("project.webhook.activity-item.issue-creation.title"),
      label: t("project.webhook.activity-item.issue-creation.label"),
      activity: "bb.issue.create",
    },
    {
      title: t("project.webhook.activity-item.issue-status-change.title"),
      label: t("project.webhook.activity-item.issue-status-change.label"),
      activity: "bb.issue.status.update",
    },
    {
      title: t("project.webhook.activity-item.issue-task-status-change.title"),
      label: t("project.webhook.activity-item.issue-task-status-change.label"),
      activity: "bb.pipeline.task.status.update",
    },
    {
      title: t("project.webhook.activity-item.issue-info-change.title"),
      label: t("project.webhook.activity-item.issue-info-change.label"),
      activity: "bb.issue.field.update",
    },
    {
      title: t("project.webhook.activity-item.issue-comment-creation.title"),
      label: t("project.webhook.activity-item.issue-comment-creation.label"),
      activity: "bb.issue.comment.create",
    },
  ];

// Project Member
export type ProjectWebhook = {
  id: MemberId;

  // Related fields
  projectId: ProjectId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  type: string;
  name: string;
  url: string;
  activityList: ActivityType[];
};

export type ProjectWebhookCreate = {
  // Domain specific fields
  type: string;
  name: string;
  url: string;
  activityList: ActivityType[];
};

export type ProjectWebhookPatch = {
  // Domain specific fields
  name?: string;
  url?: string;
  // Comma separated list. Server doesn't support deserialize into pointer to string array (*[]string in Golang)
  activityList?: string;
};

export type ProjectWebhookTestResult = {
  error?: string;
};
