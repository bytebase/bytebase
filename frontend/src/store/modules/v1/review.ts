import { defineStore } from "pinia";

import { reviewServiceClient } from "@/grpcweb";
import { useActivityV1Store } from "./activity";
import { IdType, ActivityIssueCommentCreatePayload } from "@/types";
import { projectNamePrefix, reviewNamePrefix } from "./common";

export const useReviewV1Store = defineStore("review_v1", () => {
  const createReviewComment = async ({
    reviewId,
    comment,
    payload,
  }: {
    reviewId: IdType;
    comment: string;
    payload?: ActivityIssueCommentCreatePayload;
  }) => {
    await reviewServiceClient.createReviewComment({
      parent: `${projectNamePrefix}-/${reviewNamePrefix}${reviewId}`,
      reviewComment: {
        comment,
        payload: JSON.stringify(payload ?? {}),
      },
    });
    await useActivityV1Store().fetchActivityListByIssueId(reviewId);
  };

  const updateReviewComment = async ({
    commentId,
    reviewId,
    comment,
  }: {
    commentId: string;
    reviewId: IdType;
    comment: string;
  }) => {
    await reviewServiceClient.updateReviewComment({
      parent: `${projectNamePrefix}-/${reviewNamePrefix}${reviewId}`,
      reviewComment: {
        uid: commentId,
        comment,
      },
      updateMask: ["comment"],
    });
    await useActivityV1Store().fetchActivityListByIssueId(reviewId);
  };

  return {
    createReviewComment,
    updateReviewComment,
  };
});
