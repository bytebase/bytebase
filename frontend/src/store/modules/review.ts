import { defineStore } from "pinia";

import { Issue } from "@/types";
import {
  Review,
  ApprovalStep,
  ApprovalNode_Type,
  ApprovalNode_GroupValue,
} from "@/types/proto/v1/review_service";
import { ref } from "vue";
import { extractUserEmail, useUserStore } from "./user";
import { useMemberStore } from "./member";
import { uniq, uniqBy } from "lodash-es";
import { reviewServiceClient } from "@/grpcweb";
import { User } from "@/types/proto/v1/auth_service";

const reviewName = (issue: Issue) => {
  return `projects/${issue.project.id}/reviews/${issue.id}`;
};

const emptyReview = (issue: Issue) => {
  return Review.fromJSON({
    name: reviewName(issue),
    approvalFindingDone: false,
  });
};

export const useReviewStore = defineStore("review", () => {
  const reviewsByName = ref(new Map<string, Review>());

  const getReviewByIssue = (issue: Issue) => {
    return reviewsByName.value.get(reviewName(issue)) ?? emptyReview(issue);
  };

  const setReviewByIssue = async (issue: Issue, review: Review) => {
    await fetchReviewApproversAndCandidates(issue, review);
    reviewsByName.value.set(reviewName(issue), review);
  };

  const fetchReviewByIssue = async (issue: Issue, ignoreCache = false) => {
    const name = reviewName(issue);

    try {
      const review = await reviewServiceClient.getReview({
        name,
      });
      await setReviewByIssue(issue, review);
      return review;
    } catch (error) {
      return Review.fromJSON({});
    }
  };

  const approveReview = async (issue: Issue) => {
    const review = await reviewServiceClient.approveReview({
      name: reviewName(issue),
    });
    await setReviewByIssue(issue, review);
  };

  const regenerateReview = async (issue: Issue) => {
    const review = await reviewServiceClient.updateReview({
      review: {
        name: reviewName(issue),
        approvalFindingDone: false,
      },
      updateMask: ["review.approval_finding_done"],
    });
    await setReviewByIssue(issue, review);
  };

  return {
    getReviewByIssue,
    fetchReviewByIssue,
    approveReview,
    regenerateReview,
  };
});

const fetchReviewApproversAndCandidates = async (
  issue: Issue,
  review: Review
) => {
  const userStore = useUserStore();
  const approvers = review.approvers.map((approver) => {
    return userStore.getUserByEmail(extractUserEmail(approver.principal));
  });
  const candidates = review.approvalTemplates
    .flatMap((template) => {
      const steps = template.flow?.steps ?? [];
      return steps.flatMap((step) => candidatesOfApprovalStep(issue, step));
    })
    .map((user) => userStore.getUserByName(user));
  const users = [...approvers, ...candidates].filter(
    (user) => user !== undefined
  ) as User[];
  return uniqBy(users, "name");
};

export const candidatesOfApprovalStep = (issue: Issue, step: ApprovalStep) => {
  const memberStore = useMemberStore();

  const candidates = step.nodes.flatMap((node) => {
    const { type, groupValue } = node;
    if (type !== ApprovalNode_Type.ANY_IN_GROUP) return [];
    if (groupValue === ApprovalNode_GroupValue.PROJECT_MEMBER) {
      return issue.project.memberList
        .filter((member) => member.role === "DEVELOPER")
        .map((member) => member.principal);
    }
    if (groupValue === ApprovalNode_GroupValue.PROJECT_OWNER) {
      return issue.project.memberList
        .filter((member) => member.role === "OWNER")
        .map((member) => member.principal);
    }
    if (groupValue === ApprovalNode_GroupValue.WORKSPACE_DBA) {
      return memberStore.memberList
        .filter((member) => member.role === "DBA")
        .map((member) => member.principal);
    }
    if (groupValue === ApprovalNode_GroupValue.WORKSPACE_OWNER) {
      return memberStore.memberList
        .filter((member) => member.role === "OWNER")
        .map((member) => member.principal);
    }
    return [];
  });

  return uniq(candidates.map((principal) => `users/${principal.id}`));
};
