import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import dayjs from "dayjs";
import { uniq } from "lodash-es";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import type { IssueFilter } from "@/types";
import { ApprovalStatus, RiskLevel } from "@/types/proto-es/v1/common_pb";
import {
  CreateIssueRequestSchema,
  GetIssueRequestSchema,
  type Issue,
  Issue_Type,
  IssueStatus,
  SearchIssuesRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  CreatePlanRequestSchema,
  type Plan,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { AppSliceCreator, IssueSlice } from "./types";

// Resource names are `projects/{project}/issues/{issue}` — yank the project
// resource out of the head without importing the heavy `@/utils/v1/project`
// barrel (which pulls the Pinia actuator chain into the app-store graph).
function projectResourceFromIssueName(issueName: string): string {
  const match = issueName.match(/^(projects\/[^/]+)\//);
  return match?.[1] ?? "";
}

export const buildIssueFilter = (find: IssueFilter): string => {
  const filter: string[] = [];
  if (find.creator) {
    filter.push(`creator == "${find.creator}"`);
  }
  if (find.currentApprover) {
    filter.push(`current_approver == "${find.currentApprover}"`);
  }
  if (find.approvalStatus) {
    filter.push(`approval_status == "${ApprovalStatus[find.approvalStatus]}"`);
  }
  if (find.statusList && find.statusList.length > 0) {
    filter.push(
      `status in [${find.statusList.map((s) => `"${IssueStatus[s]}"`).join(",")}]`
    );
  }
  if (find.riskLevelList && find.riskLevelList.length > 0) {
    filter.push(
      `risk_level in [${find.riskLevelList.map((r) => `"${RiskLevel[r]}"`).join(",")}]`
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
  if (find.typeList && find.typeList.length > 0) {
    filter.push(
      `type in [${find.typeList.map((t) => `"${Issue_Type[t]}"`).join(",")}]`
    );
  }
  if (find.labels && find.labels.length > 0) {
    filter.push(`labels in [${find.labels.map((l) => `"${l}"`).join(",")}]`);
  }
  return filter.join(" && ");
};

export interface CreateIssueByPlanOptions {
  skipRollout?: boolean;
}

/**
 * Creates a plan, then an issue referencing it, and (unless skipped) a rollout.
 * Relocated from the legacy Pinia `experimental-issue` module — a stateless
 * orchestration helper over the plan/issue/rollout services.
 */
export const experimentalCreateIssueByPlan = async (
  project: Project,
  issueCreate: Issue,
  planCreate: Plan,
  options?: CreateIssueByPlanOptions
) => {
  const createdPlan = await planServiceClientConnect.createPlan(
    createProto(CreatePlanRequestSchema, {
      parent: project.name,
      plan: planCreate,
    })
  );
  issueCreate.plan = createdPlan.name;

  const createdIssue = await issueServiceClientConnect.createIssue(
    createProto(CreateIssueRequestSchema, {
      parent: project.name,
      issue: issueCreate,
    })
  );

  // Skip rollout creation for plans that create rollout on-demand
  // (e.g., database creation).
  if (options?.skipRollout) {
    return { createdPlan, createdIssue, createdRollout: undefined };
  }

  const createdRollout = await rolloutServiceClientConnect.createRollout(
    createProto(CreateRolloutRequestSchema, {
      parent: createdPlan.name,
    })
  );

  return { createdPlan, createdIssue, createdRollout };
};

/**
 * Port of the legacy Pinia `useIssueV1Store`. Stateless fetches (no per-issue
 * cache — callers consume the result once) that also prime the owning
 * project(s) in the app store, matching the Pinia behavior so downstream code
 * that reads a project synchronously after the call sees it cached.
 */
export const createIssueSlice: AppSliceCreator<IssueSlice> = (_set, get) => ({
  fetchIssueByName: async (name, silent = false) => {
    const issue = await issueServiceClientConnect.getIssue(
      createProto(GetIssueRequestSchema, { name }),
      {
        contextValues: createContextValues().set(silentContextKey, silent),
      }
    );
    const projectName = projectResourceFromIssueName(issue.name);
    if (projectName) {
      await get().fetchProject(projectName);
    }
    return issue;
  },

  listIssues: async ({ find, pageSize, pageToken }) => {
    const resp = await issueServiceClientConnect.searchIssues(
      createProto(SearchIssuesRequestSchema, {
        parent: find.project,
        filter: buildIssueFilter(find),
        query: find.query,
        pageSize,
        pageToken,
        orderBy: find.orderBy,
      })
    );
    const issues = resp.issues;
    await get().batchFetchProjects(
      uniq(
        issues.map((issue) => projectResourceFromIssueName(issue.name))
      ).filter(Boolean)
    );
    await get().batchGetOrFetchUsers(
      uniq(issues.map((issue) => issue.creator))
    );
    return { nextPageToken: resp.nextPageToken, issues };
  },
});
