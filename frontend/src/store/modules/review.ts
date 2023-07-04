import { defineStore } from "pinia";
import { ref } from "vue";
import { uniq, uniqBy } from "lodash-es";

import { Issue, PresetRoleType } from "@/types";
import {
  Review,
  ApprovalStep,
  ApprovalNode_Type,
  ApprovalNode_GroupValue,
} from "@/types/proto/v1/issue_service";
import { useUserStore } from "./user";
import { issueServiceClient } from "@/grpcweb";
import { User, UserRole, UserType } from "@/types/proto/v1/auth_service";
import { extractUserResourceName, memberListInProjectV1 } from "@/utils";
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

  const getIssueByIssue = (issue: Issue) => {
    return reviewsByName.value.get(reviewName(issue)) ?? emptyReview(issue);
  };

  const setReviewByIssue = async (issue: Issue, review: Review) => {
    await fetchReviewApproversAndCandidates(issue, review);
    reviewsByName.value.set(reviewName(issue), review);
  };

  const fetchReviewByIssue = async (issue: Issue, force = false) => {
    const name = reviewName(issue);

    try {
      const review = await issueServiceClient.getIssue({
        name,
        force,
      });
      await setReviewByIssue(issue, review);
      return review;
    } catch (error) {
      return Review.fromJSON({});
    }
  };

  const approveReview = async (issue: Issue, comment?: string) => {
    const review = await issueServiceClient.approveReview({
      name: reviewName(issue),
      comment,
    });
    await setReviewByIssue(issue, review);
  };

  const rejectReview = async (issue: Issue, comment?: string) => {
    const review = await issueServiceClient.rejectReview({
      name: reviewName(issue),
      comment,
    });
    await setReviewByIssue(issue, review);
  };

  const requestReview = async (issue: Issue, comment?: string) => {
    const review = await issueServiceClient.requestReview({
      name: reviewName(issue),
      comment,
    });
    await setReviewByIssue(issue, review);
  };

  const regenerateReview = async (issue: Issue) => {
    const review = await issueServiceClient.updateIssue({
      review: {
        name: reviewName(issue),
        approvalFindingDone: false,
      },
      updateMask: ["approval_finding_done"],
    });
    await setReviewByIssue(issue, review);
  };

  return {
    getIssueByIssue,
    fetchReviewByIssue,
    approveReview,
    rejectReview,
    requestReview,
    regenerateReview,
  };
});

const fetchReviewApproversAndCandidates = async (
  issue: Issue,
  review: Review
) => {
  const userStore = useUserStore();
  const approvers = review.approvers.map((approver) => {
    return userStore.getUserByEmail(
      extractUserResourceName(approver.principal)
    );
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
  const workspaceMemberList = useUserStore().activeUserList.filter(
    (user) => user.userType === UserType.USER
  );
  const project = useProjectV1Store().getProjectByUID(String(issue.project.id));
  const projectMemberList = memberListInProjectV1(project, project.iamPolicy)
    .filter((member) => member.user.userType === UserType.USER)
    .map((member) => ({
      ...member,
      user: member.user,
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
          .filter((member) =>
            member.roleList.includes(PresetRoleType.DEVELOPER)
          )
          .map((member) => member.user);
      }
      if (groupValue === ApprovalNode_GroupValue.PROJECT_OWNER) {
        return projectMemberList
          .filter((member) => member.roleList.includes(PresetRoleType.OWNER))
          .map((member) => member.user);
      }
      if (groupValue === ApprovalNode_GroupValue.WORKSPACE_DBA) {
        return workspaceMemberList.filter(
          (member) => member.userRole === UserRole.DBA
        );
      }
      if (groupValue === ApprovalNode_GroupValue.WORKSPACE_OWNER) {
        return workspaceMemberList.filter(
          (member) => member.userRole === UserRole.OWNER
        );
      }
      return [];
    };
    const candidatesForCustomRoles = (role: string) => {
      const project = useProjectV1Store().getProjectByUID(
        String(issue.project.id)
      );
      const memberList = memberListInProjectV1(project, project.iamPolicy);
      return memberList
        .filter((member) => member.user.userType === UserType.USER)
        .filter((member) => member.roleList.includes(role))
        .map((member) => member.user);
    };

    if (groupValue !== ApprovalNode_GroupValue.UNRECOGNIZED) {
      return candidatesForSystemRoles(groupValue);
    }
    if (role) {
      return candidatesForCustomRoles(role);
    }
    return [];
  });

  return uniq(candidates.map((user) => user.name));
};
