import dayjs from "dayjs";
import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import { issueServiceClient } from "@/grpcweb";
import {
  ActivityIssueCommentCreatePayload,
  PresetRoleType,
  ComposedIssue,
  IssueFilter,
} from "@/types";
import { UserRole, UserType } from "@/types/proto/v1/auth_service";
import {
  issueStatusToJSON,
  ApprovalStep,
  ApprovalNode_Type,
  ApprovalNode_GroupValue,
} from "@/types/proto/v1/issue_service";
import { memberListInProjectV1 } from "@/utils";
import { useUserStore } from "../user";
import { useActivityV1Store } from "./activity";
import { projectNamePrefix, issueNamePrefix } from "./common";
import { shallowComposeIssue } from "./experimental-issue";

export type ListIssueParams = {
  find: IssueFilter;
  pageSize?: number;
  pageToken?: string;
};

export const buildIssueFilter = (find: IssueFilter): string => {
  const filter: string[] = [];
  if (find.principal) {
    filter.push(`principal = "${find.principal}"`);
  }
  if (find.creator) {
    filter.push(`creator = "${find.creator}"`);
  }
  if (find.assignee) {
    filter.push(`assignee = "${find.assignee}"`);
  }
  if (find.subscriber) {
    filter.push(`subscriber = "${find.subscriber}"`);
  }
  if (find.statusList) {
    filter.push(
      `status = "${find.statusList
        .map((s) => issueStatusToJSON(s))
        .join(" | ")}"`
    );
  }
  if (find.createdTsAfter) {
    filter.push(
      `create_time >= "${dayjs(find.createdTsAfter).utc().format()}"`
    );
  }
  if (find.createdTsBefore) {
    filter.push(
      `create_time <= "${dayjs(find.createdTsBefore).utc().format()}"`
    );
  }
  if (find.type) {
    filter.push(`type = "${find.type}"`);
  }
  if (find.instance) {
    filter.push(`instance = "${find.instance}"`);
  }
  if (find.database) {
    filter.push(`database = "${find.database}"`);
  }
  return filter.join(" && ");
};

export const useIssueV1Store = defineStore("issue_v1", () => {
  const regenerateReviewV1 = async (name: string) => {
    await issueServiceClient.updateIssue({
      issue: {
        name,
        approvalFindingDone: false,
      },
      updateMask: ["approval_finding_done"],
    });
  };

  const createIssueComment = async ({
    issueId,
    comment,
    payload,
  }: {
    issueId: string;
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
    await useActivityV1Store().fetchActivityListByIssueUID(issueId);
  };

  const updateIssueComment = async ({
    commentId,
    issueId,
    comment,
  }: {
    commentId: string;
    issueId: string;
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
    await useActivityV1Store().fetchActivityListByIssueUID(issueId);
  };

  const listIssues = async ({ find, pageSize, pageToken }: ListIssueParams) => {
    const resp = await issueServiceClient.listIssues({
      parent: find.project,
      filter: buildIssueFilter(find),
      query: find.query,
      pageSize,
      pageToken,
    });

    const composedIssues = await Promise.all(
      resp.issues.map((issue) => shallowComposeIssue(issue))
    );
    return {
      nextPageToken: resp.nextPageToken,
      issues: composedIssues,
    };
  };

  return {
    listIssues,
    regenerateReviewV1,
    createIssueComment,
    updateIssueComment,
  };
});

export const candidatesOfApprovalStepV1 = (
  issue: ComposedIssue,
  step: ApprovalStep
) => {
  const workspaceMemberList = useUserStore().activeUserList.filter(
    (user) => user.userType === UserType.USER
  );
  const project = issue.projectEntity;
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
