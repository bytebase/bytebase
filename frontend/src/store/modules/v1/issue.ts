import dayjs from "dayjs";
import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import type { WatchCallback } from "vue";
import { ref, watch } from "vue";
import { issueServiceClient } from "@/grpcweb";
import type { ComposedIssue, IssueFilter } from "@/types";
import { isValidProjectName, PresetRoleType } from "@/types";
import { UserType } from "@/types/proto/v1/auth_service";
import type { ApprovalStep } from "@/types/proto/v1/issue_service";
import {
  issueStatusToJSON,
  ApprovalNode_Type,
  ApprovalNode_GroupValue,
} from "@/types/proto/v1/issue_service";
import { extractProjectResourceName, memberListInProjectIAM } from "@/utils";
import { extractUserEmail, useUserStore } from "../user";
import {
  shallowComposeIssue,
  type ComposeIssueConfig,
} from "./experimental-issue";
import { useProjectV1Store } from "./project";
import { useWorkspaceV1Store } from "./workspace";

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
  if (find.taskType) {
    filter.push(`task_type = "${find.taskType}"`);
  }
  if (find.instance) {
    filter.push(`instance = "${find.instance}"`);
  }
  if (find.database) {
    filter.push(`database = "${find.database}"`);
  }
  if (find.labels && find.labels.length > 0) {
    filter.push(`labels = "${find.labels.join(" & ")}"`);
  }
  if (find.hasPipeline !== undefined) {
    filter.push(`has_pipeline = "${find.hasPipeline}"`);
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

  const listIssues = async (
    { find, pageSize, pageToken }: ListIssueParams,
    composeIssueConfig?: ComposeIssueConfig
  ) => {
    const resp = await issueServiceClient.searchIssues({
      parent: find.project,
      filter: buildIssueFilter(find),
      query: find.query,
      pageSize,
      pageToken,
    });

    const projectStore = useProjectV1Store();
    const issues = resp.issues.filter((issue) => {
      const proj = extractProjectResourceName(issue.name);
      return isValidProjectName(
        projectStore.getProjectByName(`projects/${proj}`).name
      );
    });

    const composedIssues = await Promise.all(
      issues.map((issue) => shallowComposeIssue(issue, composeIssueConfig))
    );
    return {
      nextPageToken: resp.nextPageToken,
      issues: composedIssues,
    };
  };

  const fetchIssueByName = async (
    name: string,
    composeIssueConfig?: ComposeIssueConfig
  ) => {
    const issue = await issueServiceClient.getIssue({ name });
    return shallowComposeIssue(issue, composeIssueConfig);
  };

  return {
    listIssues,
    fetchIssueByName,
    regenerateReviewV1,
  };
});

const convertApprovalNodeGroupToRole = (
  group: ApprovalNode_GroupValue
): string => {
  switch (group) {
    case ApprovalNode_GroupValue.PROJECT_MEMBER:
      return PresetRoleType.PROJECT_DEVELOPER;
    case ApprovalNode_GroupValue.PROJECT_OWNER:
      return PresetRoleType.PROJECT_OWNER;
    case ApprovalNode_GroupValue.WORKSPACE_DBA:
      return PresetRoleType.WORKSPACE_DBA;
    case ApprovalNode_GroupValue.WORKSPACE_OWNER:
      return PresetRoleType.WORKSPACE_ADMIN;
  }
  return "";
};

export const candidatesOfApprovalStepV1 = (
  issue: ComposedIssue,
  step: ApprovalStep
) => {
  const userStore = useUserStore();
  const workspaceStore = useWorkspaceV1Store();

  const workspaceMemberList = userStore.activeUserList.filter(
    (user) => user.userType === UserType.USER
  );
  const project = issue.projectEntity;
  const projectMemberList = memberListInProjectIAM(project.iamPolicy).filter(
    (member) => member.user.userType === UserType.USER
  );

  const candidates = step.nodes
    .flatMap((node) => {
      const {
        type,
        groupValue = ApprovalNode_GroupValue.UNRECOGNIZED,
        role,
      } = node;
      if (type !== ApprovalNode_Type.ANY_IN_GROUP) return [];

      const candidatesForSystemRoles = (
        groupValue: ApprovalNode_GroupValue
      ) => {
        if (
          groupValue === ApprovalNode_GroupValue.PROJECT_MEMBER ||
          groupValue === ApprovalNode_GroupValue.PROJECT_OWNER
        ) {
          const targetRole = convertApprovalNodeGroupToRole(groupValue);
          return projectMemberList
            .filter((member) => member.roleList.includes(targetRole))
            .map((member) => member.user);
        }
        if (
          groupValue === ApprovalNode_GroupValue.WORKSPACE_DBA ||
          groupValue === ApprovalNode_GroupValue.WORKSPACE_OWNER
        ) {
          return workspaceMemberList.filter(
            (member) =>
              workspaceStore.emailMapToRoles
                .get(member.email)
                ?.has(convertApprovalNodeGroupToRole(groupValue)) ?? false
          );
        }
        return [];
      };
      const candidatesForCustomRoles = (role: string) => {
        const memberList = memberListInProjectIAM(project.iamPolicy, role);
        return memberList
          .filter((member) => member.user.userType === UserType.USER)
          .map((member) => member.user);
      };

      if (groupValue !== ApprovalNode_GroupValue.UNRECOGNIZED) {
        return candidatesForSystemRoles(groupValue);
      }
      if (role) {
        return candidatesForCustomRoles(role);
      }
      return [];
    })
    // do not show the creator in the candidate list.
    .filter((user) => user.email !== extractUserEmail(issue.creator));

  return uniq(candidates.map((user) => user.name));
};

// expose global list refresh features
const REFRESH_ISSUE_LIST = ref(Math.random());
export const refreshIssueList = () => {
  REFRESH_ISSUE_LIST.value = Math.random();
};
export const useRefreshIssueList = (callback: WatchCallback) => {
  watch(REFRESH_ISSUE_LIST, callback);
};
