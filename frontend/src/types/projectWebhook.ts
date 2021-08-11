import { ActivityType } from "./activity";
import { MemberId, ProjectId } from "./id";
import { Principal } from "./principal";

type ProjectWebhookTypeItem = {
  type: string;
  name: string;
  logo: string;
  urlPrefix: string;
};

export const PROJECT_HOOK_TYPE_ITEM_LIST: ProjectWebhookTypeItem[] = [
  {
    type: "bb.plugin.webhook.slack",
    name: "Slack",
    logo: "slack-logo.png",
    urlPrefix: "https://hooks.slack.com/",
  },
  {
    type: "bb.plugin.webhook.discord",
    name: "Discord",
    logo: "discord-logo.svg",
    urlPrefix: "https://discord.com/api/webhooks",
  },
  {
    type: "bb.plugin.webhook.teams",
    name: "Teams",
    logo: "teams-logo.svg",
    urlPrefix: "",
  },
  {
    type: "bb.plugin.webhook.dingtalk",
    name: "DingTalk",
    logo: "dingtalk-logo.png",
    urlPrefix: "https://oapi.dingtalk.com",
  },
  {
    type: "bb.plugin.webhook.feishu",
    name: "Feishu",
    logo: "feishu-logo.png",
    urlPrefix: "https://open.feishu.cn",
  },
  {
    type: "bb.plugin.webhook.wecom",
    name: "WeCom",
    logo: "wecom-logo.png",
    urlPrefix: "https://qyapi.weixin.qq.com",
  },
];

type ProjectWebhookActivityItem = {
  title: string;
  label: string;
  activity: ActivityType;
};

export const PROJECT_HOOK_ACTIVITY_ITEM_LIST: ProjectWebhookActivityItem[] = [
  {
    title: "Issue creation",
    label: "When new issue has been created",
    activity: "bb.issue.create",
  },
  {
    title: "Issue status change",
    label: "When issue status has changed",
    activity: "bb.issue.status.update",
  },
  {
    title: "Issue task status change",
    label: "When issue's enclosing task status has changed",
    activity: "bb.pipeline.task.status.update",
  },
  {
    title: "Issue info change",
    label: "When issue info (e.g. assignee, title, description) has changed",
    activity: "bb.issue.field.update",
  },
  {
    title: "Issue comment creation",
    label: "When new issue comment has been created",
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
