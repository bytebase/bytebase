import { defineStore } from "pinia";
import { ref } from "vue";
import { uniq, uniqBy } from "lodash-es";

import { Issue, ProjectRoleTypeDeveloper, ProjectRoleTypeOwner } from "@/types";
import {
  Review,
  ApprovalStep,
  ApprovalNode_Type,
  ApprovalNode_GroupValue,
} from "@/types/proto/v1/review_service";
import { convertUserToPrincipal, extractUserEmail, useUserStore } from "./user";
import { useMemberStore } from "./member";
import { reviewServiceClient } from "@/grpcweb";
import { User, UserType } from "@/types/proto/v1/auth_service";
import { extractRoleResourceName, memberListInProjectV1 } from "@/utils";
import { useProjectV1Store } from "./v1";

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
      updateMask: ["approval_finding_done"],
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
  const workspaceMemberList = memberStore.memberList.filter(
    (member) => member.principal.type === "END_USER"
  );
  const project = useProjectV1Store().getProjectByUID(String(issue.project.id));
  const projectMemberList = memberListInProjectV1(project, project.iamPolicy)
    .filter((member) => member.user.userType === UserType.USER)
    .map((member) => ({
      ...member,
      principal: convertUserToPrincipal(member.user),
    }));

  const candidates = step.nodes.flatMap((node) => {
    const {
      type,
      groupValue = ApprovalNode_GroupValue.UNRECOGNIZED,
      role,
    } = node;
    if (type !== ApprovalNode_Type.ANY_IN_GROUP) return [];

    const candidatesForSystemRoles = (groupValue: ApprovalNode_GroupValue) => {
      if (groupValue === ApprovalNode_GroupValue.PROJECT_MEMBER) {
        return projectMemberList
          .filter((member) => member.roleList.includes("roles/DEVELOPER"))
          .map((member) => member.principal);
      }
      if (groupValue === ApprovalNode_GroupValue.PROJECT_OWNER) {
        return projectMemberList
          .filter((member) => member.roleList.includes("roles/OWNER"))
          .map((member) => member.principal);
      }
      if (groupValue === ApprovalNode_GroupValue.WORKSPACE_DBA) {
        return workspaceMemberList
          .filter((member) => member.role === "DBA")
          .map((member) => member.principal);
      }
      if (groupValue === ApprovalNode_GroupValue.WORKSPACE_OWNER) {
        return workspaceMemberList
          .filter((member) => member.role === "OWNER")
          .map((member) => member.principal);
      }
      return [];
    };
    const candidatesForCustomRoles = (role: string) => {
      const project = useProjectV1Store().getProjectByUID(
        String(issue.project.id)
      );
      const memberList = memberListInProjectV1(project, project.iamPolicy)
      return memberList
        .filter((member) => member.user.userType === UserType.USER)
        .filter((member) => member.roleList.includes(role))
        .map((member) => convertUserToPrincipal(member.user));
    };

    if (groupValue !== ApprovalNode_GroupValue.UNRECOGNIZED) {
      return candidatesForSystemRoles(groupValue);
    }
    if (role) {
      return candidatesForCustomRoles(role);
    }
    return [];
  });

  return uniq(candidates.map((principal) => `users/${principal.id}`));
};
