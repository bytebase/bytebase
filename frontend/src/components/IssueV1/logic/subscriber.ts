import { pull } from "lodash-es";
import { issueServiceClient } from "@/grpcweb";
import { t } from "@/plugins/i18n";
import { pushNotification, useCurrentUserV1 } from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import { Issue } from "@/types/proto/v1/issue_service";

export const updateIssueSubscribers = async (
  issue: Issue,
  subscribers: string[],
  silent?: boolean // If silent is true, no notification will be pushed
) => {
  const issuePatch = Issue.fromPartial({
    ...issue,
    subscribers,
  });
  const updated = await issueServiceClient.updateIssue({
    issue: issuePatch,
    updateMask: ["subscribers"],
  });
  Object.assign(issue, updated);
  if (!silent) {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  }
};

export const toggleSubscribeIssue = async (issue: Issue) => {
  const subscribers = [...issue.subscribers];
  const userTag = `${userNamePrefix}${useCurrentUserV1().value.email}`;
  const isSubscribed = subscribers.includes(userTag);
  if (isSubscribed) {
    pull(subscribers, userTag);
  } else {
    subscribers.push(userTag);
  }
  return await updateIssueSubscribers(issue, subscribers);
};
