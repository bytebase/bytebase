import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import dayjs from "dayjs";
import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import type { WatchCallback } from "vue";
import { ref, watch } from "vue";
import { issueServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { type IssueFilter, SYSTEM_BOT_EMAIL } from "@/types";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  GetIssueRequestSchema,
  Issue_ApprovalStatus,
  Issue_Type,
  IssueStatus,
  SearchIssuesRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  extractProjectResourceName,
  memberMapToRolesInProjectIAM,
} from "@/utils";
import { useUserStore } from "../user";
import { projectNamePrefix, userNamePrefix } from "./common";
import {
  type ComposeIssueConfig,
  shallowComposeIssue,
} from "./experimental-issue";
import { useProjectV1Store } from "./project";
import { useProjectIamPolicyStore } from "./projectIamPolicy";

export type ListIssueParams = {
  find: IssueFilter;
  pageSize?: number;
  pageToken?: string;
};

export const buildIssueFilter = (find: IssueFilter): string => {
  const filter: string[] = [];
  if (find.creator) {
    filter.push(`creator == "${find.creator}"`);
  }
  if (find.currentApprover) {
    filter.push(`current_approver == "${find.currentApprover}"`);
  }
  if (find.approvalStatus) {
    filter.push(
      `approval_status == "${Issue_ApprovalStatus[find.approvalStatus]}"`
    );
  }
  if (find.statusList && find.statusList.length > 0) {
    filter.push(
      `status in [${find.statusList.map((s) => `"${IssueStatus[s]}"`).join(",")}]`
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
    filter.push(`type == "${Issue_Type[find.type]}"`);
  }
  if (find.labels && find.labels.length > 0) {
    filter.push(`labels in [${find.labels.map((l) => `"${l}"`).join(",")}]`);
  }
  return filter.join(" && ");
};

export const useIssueV1Store = defineStore("issue_v1", () => {
  const projectStore = useProjectV1Store();

  const listIssues = async (
    { find, pageSize, pageToken }: ListIssueParams,
    composeIssueConfig?: ComposeIssueConfig
  ) => {
    const request = create(SearchIssuesRequestSchema, {
      parent: find.project,
      filter: buildIssueFilter(find),
      query: find.query,
      pageSize,
      pageToken,
    });
    const resp = await issueServiceClientConnect.searchIssues(request);
    const issues = resp.issues;

    const projects = issues.map((issue) => {
      return `projects/${extractProjectResourceName(issue.name)}`;
    });
    await projectStore.batchGetOrFetchProjects(projects);
    const composedIssues = await Promise.all(
      issues.map((issue) => shallowComposeIssue(issue, composeIssueConfig))
    );
    // Preprare creator for the issues.
    const users = uniq(composedIssues.map((issue) => issue.creator));
    await useUserStore().batchGetOrFetchUsers(users);
    return {
      nextPageToken: resp.nextPageToken,
      issues: composedIssues,
    };
  };

  const fetchIssueByName = async (
    name: string,
    composeIssueConfig?: ComposeIssueConfig,
    silent: boolean = false
  ) => {
    const request = create(GetIssueRequestSchema, { name });
    const issue = await issueServiceClientConnect.getIssue(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    });
    await projectStore.getOrFetchProjectByName(
      `projects/${extractProjectResourceName(issue.name)}`
    );
    return shallowComposeIssue(issue, composeIssueConfig);
  };

  return {
    listIssues,
    fetchIssueByName,
  };
});

// candidatesOfApprovalStepV1 return user name list in users/{email} format.
// The list could includs users/ALL_USERS_USER_EMAIL
export const candidatesOfApprovalStepV1 = (issue: Issue, role: string) => {
  const project = useProjectV1Store().getProjectByName(
    `${projectNamePrefix}${extractProjectResourceName(issue.name)}`
  );
  const candidatesForRoles = (role: string) => {
    const projectIamPolicyStore = useProjectIamPolicyStore();
    const iamPolicy = projectIamPolicyStore.getProjectIamPolicy(project.name);
    const memberMap = memberMapToRolesInProjectIAM(iamPolicy, role);
    return [...memberMap.keys()];
  };
  const candidates = role ? candidatesForRoles(role) : [];

  return uniq(
    candidates.filter((user) => {
      // Exclude system bot user.
      if (user === `${userNamePrefix}${SYSTEM_BOT_EMAIL}`) {
        return false;
      }
      // If the project does not allow self-approval, exclude the creator.
      if (!project.allowSelfApproval && user === issue.creator) {
        return false;
      }
      return true;
    })
  );
};

// expose global list refresh features
const REFRESH_ISSUE_LIST = ref(Math.random());
export const refreshIssueList = () => {
  REFRESH_ISSUE_LIST.value = Math.random();
};
export const useRefreshIssueList = (callback: WatchCallback) => {
  watch(REFRESH_ISSUE_LIST, callback);
};
