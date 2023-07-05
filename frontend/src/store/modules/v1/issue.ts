import { defineStore } from "pinia";
import { ref } from "vue";
import { uniq, uniqBy } from "lodash-es";

import { issueServiceClient } from "@/grpcweb";
import { useActivityV1Store } from "./activity";
import {
  IdType,
  ActivityIssueCommentCreatePayload,
  Issue as LegacyIssue,
  PresetRoleType,
} from "@/types";
import {
  Issue,
  ApprovalStep,
  ApprovalNode_Type,
  ApprovalNode_GroupValue,
} from "@/types/proto/v1/issue_service";
import { projectNamePrefix, issueNamePrefix } from "./common";
import { useUserStore } from "../user";
import { User, UserRole, UserType } from "@/types/proto/v1/auth_service";
import { extractUserResourceName, memberListInProjectV1 } from "@/utils";
import { useProjectV1Store } from ".";

const issueName = (legacyIssue: LegacyIssue) => {
  return `projects/${legacyIssue.project.id}/issues/${legacyIssue.id}`;
};

const emptyIssue = (legacyIssue: LegacyIssue) => {
  return Issue.fromJSON({
    name: issueName(legacyIssue),
    approvalFindingDone: false,
  });
};

export const useIssueV1Store = defineStore("issue_v1", () => {
  const issuesByName = ref(new Map<string, Issue>());

  const getIssueByIssue = (issue: LegacyIssue) => {
    return issuesByName.value.get(issueName(issue)) ?? emptyIssue(issue);
  };

  const setIssueIssue = async (legacyIssue: LegacyIssue, issue: Issue) => {
    await fetchReviewApproversAndCandidates(legacyIssue, issue);
    issuesByName.value.set(issueName(legacyIssue), issue);
  };

  const fetchIssueByLegacyIssue = async (
    legacyIssue: LegacyIssue,
    force = false
  ) => {
    const name = issueName(legacyIssue);

    try {
      const issue = await issueServiceClient.getIssue({
        name,
        force,
      });
      await setIssueIssue(legacyIssue, issue);
      return issue;
    } catch (error) {
      return Issue.fromJSON({});
    }
  };

  const approveIssue = async (legacyIssue: LegacyIssue, comment?: string) => {
    const issue = await issueServiceClient.approveIssue({
      name: issueName(legacyIssue),
      comment,
    });
    await setIssueIssue(legacyIssue, issue);
  };

  const rejectIssue = async (legacyIssue: LegacyIssue, comment?: string) => {
    const issue = await issueServiceClient.rejectIssue({
      name: issueName(legacyIssue),
      comment,
    });
    await setIssueIssue(legacyIssue, issue);
  };

  const requestIssue = async (legacyIssue: LegacyIssue, comment?: string) => {
    const issue = await issueServiceClient.requestIssue({
      name: issueName(legacyIssue),
      comment,
    });
    await setIssueIssue(legacyIssue, issue);
  };

  const regenerateReview = async (legacyIssue: LegacyIssue) => {
    const issue = await issueServiceClient.updateIssue({
      issue: {
        name: issueName(legacyIssue),
        approvalFindingDone: false,
      },
      updateMask: ["approval_finding_done"],
    });
    await setIssueIssue(legacyIssue, issue);
  };

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
    getIssueByIssue,
    fetchIssueByLegacyIssue: fetchIssueByLegacyIssue,
    approveIssue,
    rejectIssue,
    requestIssue,
    regenerateReview,
    createIssueComment,
    updateIssueComment,
  };
});

const fetchReviewApproversAndCandidates = async (
  legacyIssue: LegacyIssue,
  issue: Issue
) => {
  const userStore = useUserStore();
  const approvers = issue.approvers.map((approver) => {
    return userStore.getUserByEmail(
      extractUserResourceName(approver.principal)
    );
  });
  const candidates = issue.approvalTemplates
    .flatMap((template) => {
      const steps = template.flow?.steps ?? [];
      return steps.flatMap((step) =>
        candidatesOfApprovalStep(legacyIssue, step)
      );
    })
    .map((user) => userStore.getUserByName(user));
  const users = [...approvers, ...candidates].filter(
    (user) => user !== undefined
  ) as User[];
  return uniqBy(users, "name");
};

export const candidatesOfApprovalStep = (
  legacyIssue: LegacyIssue,
  step: ApprovalStep
) => {
  const workspaceMemberList = useUserStore().activeUserList.filter(
    (user) => user.userType === UserType.USER
  );
  const project = useProjectV1Store().getProjectByUID(
    String(legacyIssue.project.id)
  );
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
        String(legacyIssue.project.id)
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
