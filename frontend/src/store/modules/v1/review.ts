import { defineStore } from "pinia";

import { issueServiceClient } from "@/grpcweb";
import { useActivityV1Store } from "./activity";
import { IdType, ActivityIssueCommentCreatePayload } from "@/types";
import { projectNamePrefix, issueNamePrefix } from "./common";

export const useIssueV1Store = defineStore("issue_v1", () => {
  const createIssueComment = async ({
    issueId,
    comment,
    payload,
  }: {
    issueId: IdType;
    comment: string;
    payload?: ActivityIssueCommentCreatePayload;
  }) => {
    await issueServiceClient.createIssueComment({
      parent: `${projectNamePrefix}-/${issueNamePrefix}${issueId}`,
      issueComment: {
        comment,
        payload: JSON.stringify(payload ?? {}),
      },
    });
    await useActivityV1Store().fetchActivityListByIssueId(issueId);
  };

  const updateIssueComment = async ({
    commentId,
    issueId,
    comment,
  }: {
    commentId: string;
    issueId: IdType;
    comment: string;
  }) => {
    await issueServiceClient.updateIssueComment({
      parent: `${projectNamePrefix}-/${issueNamePrefix}${issueId}`,
      issueComment: {
        uid: commentId,
        comment,
      },
      updateMask: ["comment"],
    });
    await useActivityV1Store().fetchActivityListByIssueId(issueId);
  };

  return {
    createIssueComment,
    updateIssueComment,
  };
});
