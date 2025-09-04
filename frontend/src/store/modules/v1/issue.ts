import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import dayjs from "dayjs";
import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import type { WatchCallback } from "vue";
import { ref, watch } from "vue";
import { issueServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import { SYSTEM_BOT_EMAIL, type IssueFilter } from "@/types";
import {
  GetIssueRequestSchema,
  Issue_Type,
  IssueSchema,
  SearchIssuesRequestSchema,
  UpdateIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import type { ApprovalStep, Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  IssueStatus,
  ApprovalNode_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  extractProjectResourceName,
  memberMapToRolesInProjectIAM,
} from "@/utils";
import { useUserStore } from "../user";
import { projectNamePrefix, userNamePrefix } from "./common";
import {
  shallowComposeIssue,
  type ComposeIssueConfig,
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
  if (find.taskType) {
    filter.push(`task_type == "${find.taskType}"`);
  }
  if (find.instance) {
    filter.push(`instance == "${find.instance}"`);
  }
  if (find.database) {
    filter.push(`database == "${find.database}"`);
  }
  if (find.labels && find.labels.length > 0) {
    filter.push(`labels in [${find.labels.map((l) => `"${l}"`).join(",")}]`);
  }
  if (find.hasPipeline !== undefined) {
    filter.push(`has_pipeline == "${find.hasPipeline}"`);
  }
  return filter.join(" && ");
};

export const useIssueV1Store = defineStore("issue_v1", () => {
  const regenerateReviewV1 = async (name: string) => {
    const request = create(UpdateIssueRequestSchema, {
      issue: create(IssueSchema, {
        name,
        approvalFindingDone: false,
      }),
      updateMask: { paths: ["approval_finding_done"] },
    });
    await issueServiceClientConnect.updateIssue(request);
  };

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
    const composedIssues = await Promise.all(
      issues.map((issue) => shallowComposeIssue(issue, composeIssueConfig))
    );
    // Preprare creator for the issues.
    const users = uniq(composedIssues.map((issue) => issue.creator));
    await useUserStore().batchGetUsers(users);
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
    const newIssue = await issueServiceClientConnect.getIssue(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    });
    const issue = newIssue;
    return shallowComposeIssue(issue, composeIssueConfig);
  };

  return {
    listIssues,
    fetchIssueByName,
    regenerateReviewV1,
  };
});

// candidatesOfApprovalStepV1 return user name list in users/{email} format.
// The list could includs users/ALL_USERS_USER_EMAIL
export const candidatesOfApprovalStepV1 = (
  issue: Issue,
  step: ApprovalStep
) => {
  const project = useProjectV1Store().getProjectByName(
    `${projectNamePrefix}${extractProjectResourceName(issue.name)}`
  );
  const candidates = step.nodes.flatMap((node) => {
    const { type, role } = node;
    if (type !== ApprovalNode_Type.ANY_IN_GROUP) return [];

    const candidatesForRoles = (role: string) => {
      const projectIamPolicyStore = useProjectIamPolicyStore();
      const iamPolicy = projectIamPolicyStore.getProjectIamPolicy(project.name);
      const memberMap = memberMapToRolesInProjectIAM(iamPolicy, role);
      return [...memberMap.keys()];
    };
    if (role) {
      return candidatesForRoles(role);
    }
    return [];
  });

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
