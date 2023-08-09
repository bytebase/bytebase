import { Issue } from "@/types/proto/v1/issue_service";
import { issueServiceClient } from "@/grpcweb";
import { pushNotification } from "@/store";
import { t } from "@/plugins/i18n";
import { User } from "@/types/proto/v1/auth_service";
import { pull } from "lodash-es";

export const updateIssueSubscribers = async (
  issue: Issue,
  subscribers: string[]
) => {
  const issuePatch = Issue.fromJSON({
    ...issue,
    subscribers,
  });
  const updated = await issueServiceClient.updateIssue({
    issue: issuePatch,
    updateMask: ["subscribers"],
  });
  Object.assign(issue, updated);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

export const toggleSubscribeIssue = async (issue: Issue, user: User) => {
  const subscribers = [...issue.subscribers];
  const userTag = `users/${user.email}`;
  const isSubscribed = subscribers.includes(userTag);
  if (isSubscribed) {
    pull(subscribers, userTag);
  } else {
    subscribers.push(userTag);
  }
  return await updateIssueSubscribers(issue, subscribers);
};

export const doSubscribeIssue = async (issue: Issue, user: User) => {
  const subscribers = [...issue.subscribers];
  const userTag = `users/${user.email}`;
  const isSubscribed = subscribers.includes(userTag);
  if (isSubscribed) {
    return;
  }
  subscribers.push(userTag);
  return await updateIssueSubscribers(issue, subscribers);
};
