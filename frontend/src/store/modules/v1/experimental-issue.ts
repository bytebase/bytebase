import { orderBy } from "lodash-es";
import { create } from "@bufbuild/protobuf";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/grpcweb";
import {
  GetPlanRequestSchema,
  ListPlanCheckRunsRequestSchema,
  CreatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  GetRolloutRequestSchema,
  ListTaskRunsRequestSchema,
  CreateRolloutRequestSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  CreateIssueRequestSchema,
  GetIssueRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  convertNewPlanToOld,
  convertNewPlanCheckRunToOld,
  convertOldPlanToNew,
} from "@/utils/v1/plan-conversions";
import {
  convertNewIssueToOld,
  convertOldIssueToNew,
} from "@/utils/v1/issue-conversions";
import {
  convertNewRolloutToOld,
  convertNewTaskRunToOld,
} from "@/utils/v1/rollout-conversions";
import { useProjectV1Store } from "@/store";
import type { ComposedIssue, ComposedProject, ComposedTaskRun } from "@/types";
import {
  emptyIssue,
  emptyRollout,
  EMPTY_ID,
  unknownIssue,
  UNKNOWN_ID,
} from "@/types";
import type { Issue } from "@/types/proto/v1/issue_service";
import type { Plan } from "@/types/proto/v1/plan_service";
import { TaskRunLog, type Rollout } from "@/types/proto/v1/rollout_service";
import { extractProjectResourceName, hasProjectPermissionV2 } from "@/utils";
import { DEFAULT_PAGE_SIZE } from "../common";

export interface ComposeIssueConfig {
  withPlan?: boolean;
  withRollout?: boolean;
}

export const composeIssue = async (
  rawIssue: Issue,
  config: ComposeIssueConfig = { withPlan: true, withRollout: true }
): Promise<ComposedIssue> => {
  const project = `projects/${extractProjectResourceName(rawIssue.name)}`;
  const projectEntity =
    await useProjectV1Store().getOrFetchProjectByName(project);

  const issue: ComposedIssue = {
    ...rawIssue,
    planEntity: undefined,
    planCheckRunList: [],
    rolloutEntity: emptyRollout(),
    rolloutTaskRunList: [],
    project,
  };

  if (config.withPlan && issue.plan) {
    if (hasProjectPermissionV2(projectEntity, "bb.plans.get")) {
      const request = create(GetPlanRequestSchema, {
        name: issue.plan,
      });
      const response = await planServiceClientConnect.getPlan(request);
      issue.planEntity = convertNewPlanToOld(response);
    }

    if (hasProjectPermissionV2(projectEntity, "bb.planCheckRuns.list")) {
      // Only show the latest plan check runs.
      const request = create(ListPlanCheckRunsRequestSchema, {
        parent: issue.plan,
        latestOnly: true,
      });
      const response = await planServiceClientConnect.listPlanCheckRuns(request);
      const planCheckRuns = response.planCheckRuns.map(convertNewPlanCheckRunToOld);
      issue.planCheckRunList = orderBy(planCheckRuns, "name", "desc");
    }
  }
  if (config.withRollout && issue.rollout) {
    if (hasProjectPermissionV2(projectEntity, "bb.rollouts.get")) {
      const request = create(GetRolloutRequestSchema, {
        name: issue.rollout,
      });
      const response = await rolloutServiceClientConnect.getRollout(request);
      issue.rolloutEntity = convertNewRolloutToOld(response);
    }

    if (hasProjectPermissionV2(projectEntity, "bb.taskRuns.list")) {
      const request = create(ListTaskRunsRequestSchema, {
        parent: `${issue.rollout}/stages/-/tasks/-`,
        pageSize: DEFAULT_PAGE_SIZE,
      });
      const response = await rolloutServiceClientConnect.listTaskRuns(request);
      const taskRuns = response.taskRuns.map(convertNewTaskRunToOld);
      const composedTaskRuns: ComposedTaskRun[] = [];
      for (const taskRun of taskRuns) {
        const composed: ComposedTaskRun = {
          ...taskRun,
          taskRunLog: TaskRunLog.fromPartial({}),
        };
        composedTaskRuns.push(composed);
      }
      issue.rolloutTaskRunList = composedTaskRuns;
    }
  }

  return issue;
};

export const shallowComposeIssue = async (
  rawIssue: Issue,
  config?: ComposeIssueConfig
): Promise<ComposedIssue> => {
  return composeIssue(
    rawIssue,
    config || { withPlan: false, withRollout: false }
  );
};

export const experimentalFetchIssueByUID = async (
  uid: string,
  project: string
) => {
  if (uid === "undefined") {
    console.warn("undefined issue uid");
    return unknownIssue();
  }

  if (uid === String(EMPTY_ID)) return emptyIssue();
  if (uid === String(UNKNOWN_ID)) return unknownIssue();

  const request = create(GetIssueRequestSchema, {
    name: `${project}/issues/${uid}`,
  });
  const newIssue = await issueServiceClientConnect.getIssue(request);
  const rawIssue = convertNewIssueToOld(newIssue);

  return composeIssue(rawIssue);
};

export type CreateIssueHooks = {
  planCreated: (plan: Plan) => Promise<any>;
  issueCreated: (issue: Issue, plan: Plan) => Promise<any>;
  rolloutCreated: (issue: Issue, plan: Plan, rollout: Rollout) => Promise<any>;
};
export const experimentalCreateIssueByPlan = async (
  project: ComposedProject,
  issueCreate: Issue,
  planCreate: Plan,
  hooks?: Partial<CreateIssueHooks>
) => {
  const newPlan = convertOldPlanToNew(planCreate);
  const request = create(CreatePlanRequestSchema, {
    parent: project.name,
    plan: newPlan,
  });
  const response = await planServiceClientConnect.createPlan(request);
  const createdPlan = convertNewPlanToOld(response);
  issueCreate.plan = createdPlan.name;
  await hooks?.planCreated?.(planCreate);

  const issueRequest = create(CreateIssueRequestSchema, {
    parent: project.name,
    issue: convertOldIssueToNew(issueCreate),
  });
  const newCreatedIssue = await issueServiceClientConnect.createIssue(issueRequest);
  const createdIssue = convertNewIssueToOld(newCreatedIssue);
  await hooks?.issueCreated?.(createdIssue, createdPlan);
  const rolloutRequest = create(CreateRolloutRequestSchema, {
    parent: project.name,
    rollout: {
      plan: createdPlan.name,
    },
  });
  const rolloutResponse = await rolloutServiceClientConnect.createRollout(rolloutRequest);
  const createdRollout = convertNewRolloutToOld(rolloutResponse);
  createdIssue.rollout = createdRollout.name;
  await hooks?.rolloutCreated?.(createdIssue, createdPlan, createdRollout);

  return { createdPlan, createdIssue, createdRollout };
};
