import dayjs from "dayjs";
import { uniq } from "lodash-es";
import { defineStore } from "pinia";
import type { WatchCallback } from "vue";
import { ref, watch } from "vue";
import { issueServiceClient } from "@/grpcweb";
import type { ComposedIssue, IssueFilter } from "@/types";
import type { ApprovalStep } from "@/types/proto/v1/issue_service";
import {
  issueStatusToJSON,
  ApprovalNode_Type,
} from "@/types/proto/v1/issue_service";
import { memberMapToRolesInProjectIAM } from "@/utils";
import {
  shallowComposeIssue,
  type ComposeIssueConfig,
} from "./experimental-issue";

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
    const composedIssues = await Promise.all(
      resp.issues.map((issue) => shallowComposeIssue(issue, composeIssueConfig))
    );
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
    const issue = await issueServiceClient.getIssue({ name }, { silent });
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
  issue: ComposedIssue,
  step: ApprovalStep
) => {
  const project = issue.projectEntity;

  const candidates = step.nodes.flatMap((node) => {
    const { type, role } = node;
    if (type !== ApprovalNode_Type.ANY_IN_GROUP) return [];

    const candidatesForRoles = (role: string) => {
      const memberMap = memberMapToRolesInProjectIAM(project.iamPolicy, role);
      return [...memberMap.keys()];
    };
    if (role) {
      return candidatesForRoles(role);
    }
    return [];
  });

  return uniq(
    candidates.filter((user) => {
      if (!issue.projectEntity.allowSelfApproval) {
        return user !== issue.creator;
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
