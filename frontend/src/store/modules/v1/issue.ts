import { defineStore } from "pinia";

import { issueServiceClient } from "@/grpcweb";
import { useActivityV1Store } from "./activity";
import { IdType, ActivityIssueCommentCreatePayload } from "@/types";
import { projectNamePrefix, reviewNamePrefix } from "./common";

export const useIssueV1Store = defineStore("issue_v1", () => {
  const createIssueComment = async ({
    reviewId,
    comment,
    payload,
  }: {
    reviewId: IdType;
    comment: string;
    payload?: ActivityIssueCommentCreatePayload;
  }) => {
    await issueServiceClient.createIssueComment({
      parent: `${projectNamePrefix}-/${reviewNamePrefix}${reviewId}`,
      issueComment: {
        comment,
        payload: JSON.stringify(payload ?? {}),
      },
    });
    await useActivityV1Store().fetchActivityListByIssueId(reviewId);
  };

  const updateIssueComment = async ({
    commentId,
    reviewId,
    comment,
  }: {
    commentId: string;
    reviewId: IdType;
    comment: string;
  }) => {
    await issueServiceClient.updateIssueComment({
      parent: `${projectNamePrefix}-/${reviewNamePrefix}${reviewId}`,
      issueComment: {
        uid: commentId,
        comment,
      },
      updateMask: ["comment"],
    });
    await useActivityV1Store().fetchActivityListByIssueId(reviewId);
  };

  return {
    createIssueComment,
    updateIssueComment,
  };
});
