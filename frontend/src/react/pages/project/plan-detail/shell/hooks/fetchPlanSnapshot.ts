import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  projectServiceClientConnect,
  rolloutServiceClientConnect,
  userServiceClientConnect,
} from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import {
  GetIssueRequestSchema,
  type Issue,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  GetPlanCheckRunRequestSchema,
  GetPlanRequestSchema,
  type Plan,
  type PlanCheckRun,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  GetProjectRequestSchema,
  type Project,
} from "@/types/proto-es/v1/project_service_pb";
import {
  GetRolloutRequestSchema,
  ListTaskRunsRequestSchema,
  type Rollout,
  type TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { getRolloutFromPlan, hasProjectPermissionV2 } from "@/utils";
import { createPlanSkeleton } from "../../utils/createPlan";
import { PROJECT_NAME_PREFIX } from "../constants";

export interface PlanDetailFetchPatch {
  currentUser: User;
  plan: Plan;
  project: Project;
  projectTitle: string;
  projectCanCreateRollout: boolean;
  projectRequireIssueApproval: boolean;
  projectRequirePlanCheckNoError: boolean;
  issue: Issue | undefined;
  rollout: Rollout | undefined;
  planCheckRuns: PlanCheckRun[];
  taskRuns: TaskRun[];
}

// The rollout and its task-run listing are fetched by both the full page
// snapshot and the slim status lane; only their error handling differs. Keep the
// request shape — schema, silent context, task-run parent glob — in one place.
// Fetched WITHOUT touching the store here; the caller (patchState) seeds the
// store cache after its staleness guard, so a stale in-flight poll can't
// overwrite the shared cache the log viewer reads.
const requestRollout = (rolloutName: string) =>
  rolloutServiceClientConnect.getRollout(
    create(GetRolloutRequestSchema, { name: rolloutName }),
    { contextValues: createContextValues().set(silentContextKey, true) }
  );

const requestRolloutTaskRuns = (rolloutName: string) =>
  rolloutServiceClientConnect
    .listTaskRuns(
      create(ListTaskRunsRequestSchema, {
        parent: `${rolloutName}/stages/-/tasks/-`,
      }),
      { contextValues: createContextValues().set(silentContextKey, true) }
    )
    .then((response) => response.taskRuns);

// The slim "status lane" poll. While a task is transitioning, the deploy view
// only needs the rollout (task statuses) and its task runs — not the whole page.
// Fetching just these two, instead of the 7-RPC full snapshot, lets the poll run
// at a tight floor without the extra load, so PENDING -> RUNNING -> DONE is
// observed promptly.
export type PlanDetailStatusPatch = Partial<
  Pick<PlanDetailFetchPatch, "rollout" | "taskRuns">
>;

export const fetchRolloutState = async (
  rolloutName: string
): Promise<PlanDetailStatusPatch> => {
  const [rolloutResult, taskRunsResult] = await Promise.allSettled([
    requestRollout(rolloutName),
    requestRolloutTaskRuns(rolloutName),
  ]);
  // Apply the rollout and its task runs together or not at all. A partial patch
  // could advance the rollout to a terminal state while leaving stale task runs,
  // which stops polling (isPlanDone) with the latest run still shown as RUNNING.
  // On any failure keep the existing data untouched and let the next tick retry.
  if (
    rolloutResult.status !== "fulfilled" ||
    taskRunsResult.status !== "fulfilled"
  ) {
    return {};
  }
  return { rollout: rolloutResult.value, taskRuns: taskRunsResult.value };
};

const convertRouteQuery = (query: Record<string, unknown>) => {
  const kv: Record<string, string> = {};
  for (const [key, value] of Object.entries(query)) {
    if (typeof value === "string") {
      kv[key] = value;
    }
  }
  return kv;
};

export const fetchPlanSnapshot = async (
  projectId: string,
  planId: string,
  routeQuery: Record<string, unknown> = {},
  // When true (background poll ticks), suppress the global CRITICAL error toast
  // on transient failures — the poll runs every few seconds, so a flaky backend
  // would otherwise spam toasts. The initial load stays loud (default) so a
  // first-load failure is visible; 404/403 are handled explicitly by the caller.
  silent = false
): Promise<PlanDetailFetchPatch> => {
  const silentCtx = silent
    ? { contextValues: createContextValues().set(silentContextKey, true) }
    : undefined;
  // The plan fetch depends only on route ids, so it runs alongside the
  // project/user wave instead of being serialized behind it. The detached
  // no-op catch keeps a plan failure from surfacing as an unhandled rejection
  // when the first wave throws before the plan is awaited; the await below
  // still propagates the error.
  const planPromise =
    planId.toLowerCase() === "create"
      ? undefined
      : planServiceClientConnect.getPlan(
          create(GetPlanRequestSchema, {
            name: `${PROJECT_NAME_PREFIX}${projectId}/plans/${planId}`,
          }),
          silentCtx
        );
  planPromise?.catch(() => undefined);

  const [project, currentUser] = await Promise.all([
    projectServiceClientConnect.getProject(
      create(GetProjectRequestSchema, {
        name: `${PROJECT_NAME_PREFIX}${projectId}`,
      }),
      silentCtx
    ),
    userServiceClientConnect.getCurrentUser({}, silentCtx),
  ]);

  if (!planPromise) {
    const plan = await createPlanSkeleton(
      project,
      convertRouteQuery(routeQuery)
    );
    return {
      currentUser,
      plan,
      project,
      projectTitle: project.title,
      projectCanCreateRollout: hasProjectPermissionV2(
        project,
        "bb.rollouts.create"
      ),
      projectRequireIssueApproval: project.requireIssueApproval,
      projectRequirePlanCheckNoError: project.requirePlanCheckNoError,
      issue: undefined,
      rollout: undefined,
      planCheckRuns: [],
      taskRuns: [],
    };
  }

  const plan = await planPromise;

  // The rollout name is derived from the plan name, so task runs can load in
  // parallel with the rollout instead of being serialized behind it.
  const rolloutName = plan.hasRollout ? getRolloutFromPlan(plan.name) : "";
  const [issue, planCheckRuns, rollout, taskRuns] = await Promise.all([
    plan.issue
      ? issueServiceClientConnect
          .getIssue(
            create(GetIssueRequestSchema, { name: plan.issue }),
            silentCtx
          )
          .catch(() => undefined)
      : Promise.resolve(undefined),
    planServiceClientConnect
      .getPlanCheckRun(
        create(GetPlanCheckRunRequestSchema, {
          name: `${plan.name}/planCheckRun`,
        }),
        silentCtx
      )
      .then((run) => [run] as PlanCheckRun[])
      .catch(() => []),
    rolloutName
      ? requestRollout(rolloutName).catch(() => undefined)
      : Promise.resolve(undefined),
    rolloutName
      ? requestRolloutTaskRuns(rolloutName).catch(() => [])
      : Promise.resolve([] as TaskRun[]),
  ]);

  return {
    currentUser,
    plan,
    project,
    projectTitle: project.title,
    projectCanCreateRollout: hasProjectPermissionV2(
      project,
      "bb.rollouts.create"
    ),
    projectRequireIssueApproval: project.requireIssueApproval,
    projectRequirePlanCheckNoError: project.requirePlanCheckNoError,
    issue,
    rollout,
    planCheckRuns,
    taskRuns,
  };
};
