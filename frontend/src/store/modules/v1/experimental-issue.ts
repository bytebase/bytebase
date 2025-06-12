import { orderBy } from "lodash-es";
import {
  issueServiceClient,
  planServiceClient,
  rolloutServiceClient,
} from "@/grpcweb";
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
      const plan = await planServiceClient.getPlan({
        name: issue.plan,
      });
      issue.planEntity = plan;
    }

    if (hasProjectPermissionV2(projectEntity, "bb.planCheckRuns.list")) {
      // Only show the latest plan check runs.
      const { planCheckRuns } = await planServiceClient.listPlanCheckRuns({
        parent: issue.plan,
        latestOnly: true,
      });
      issue.planCheckRunList = orderBy(planCheckRuns, "name", "desc");
    }
  }
  if (config.withRollout && issue.rollout) {
    if (hasProjectPermissionV2(projectEntity, "bb.rollouts.get")) {
      issue.rolloutEntity = await rolloutServiceClient.getRollout({
        name: issue.rollout,
      });
    }

    if (hasProjectPermissionV2(projectEntity, "bb.taskRuns.list")) {
      const { taskRuns } = await rolloutServiceClient.listTaskRuns({
        parent: `${issue.rollout}/stages/-/tasks/-`,
        pageSize: DEFAULT_PAGE_SIZE,
      });
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

  const rawIssue = await issueServiceClient.getIssue({
    name: `${project}/issues/${uid}`,
  });

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
  const createdPlan = await planServiceClient.createPlan({
    parent: project.name,
    plan: planCreate,
  });
  issueCreate.plan = createdPlan.name;
  await hooks?.planCreated?.(planCreate);

  const createdIssue = await issueServiceClient.createIssue({
    parent: project.name,
    issue: issueCreate,
  });
  await hooks?.issueCreated?.(createdIssue, createdPlan);
  const createdRollout = await rolloutServiceClient.createRollout({
    parent: project.name,
    rollout: {
      plan: createdPlan.name,
    },
  });
  createdIssue.rollout = createdRollout.name;
  await hooks?.rolloutCreated?.(createdIssue, createdPlan, createdRollout);

  return { createdPlan, createdIssue, createdRollout };
};
