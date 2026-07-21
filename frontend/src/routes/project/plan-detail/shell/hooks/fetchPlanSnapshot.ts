import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  projectServiceClientConnect,
  rolloutServiceClientConnect,
  userServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import {
  GetIssueRequestSchema,
  type Issue,
  IssueStatus,
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
  issue?: Issue | undefined;
  rollout?: Rollout | undefined;
  planCheckRuns?: PlanCheckRun[];
  taskRuns?: TaskRun[];
}

// Keep the rollout and task-run request shape in one place. They are applied as
// one consistency group below: if either request fails, neither field is
// patched, so a transient poll failure cannot erase last-known-good deploy data.
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

const requestRolloutState = async (rolloutName: string) => {
  const [rollout, taskRuns] = await Promise.all([
    requestRollout(rolloutName),
    requestRolloutTaskRuns(rolloutName),
  ]);
  return { rollout, taskRuns };
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

  let plan = await planPromise;

  // The rollout name is derived from the plan name, so task runs can load in
  // parallel with the rollout instead of being serialized behind it.
  let rolloutName = plan.hasRollout ? getRolloutFromPlan(plan.name) : "";
  const [issueResult, planCheckRunsResult, rolloutStateResult] =
    await Promise.allSettled([
      plan.issue
        ? issueServiceClientConnect.getIssue(
            create(GetIssueRequestSchema, { name: plan.issue }),
            silentCtx
          )
        : Promise.resolve(undefined),
      planServiceClientConnect
        .getPlanCheckRun(
          create(GetPlanCheckRunRequestSchema, {
            name: `${plan.name}/planCheckRun`,
          }),
          silentCtx
        )
        .then((run) => [run] as PlanCheckRun[]),
      rolloutName
        ? requestRolloutState(rolloutName)
        : Promise.resolve({ rollout: undefined, taskRuns: [] as TaskRun[] }),
    ]);

  // Rollout creation commits plan.hasRollout before marking the issue DONE. If
  // those two RPCs straddle that commit, refresh the older plan half once so a
  // single page snapshot cannot combine DONE with a pre-rollout plan.
  let rolloutState =
    rolloutStateResult.status === "fulfilled"
      ? rolloutStateResult.value
      : undefined;
  if (
    !plan.hasRollout &&
    issueResult.status === "fulfilled" &&
    issueResult.value?.status === IssueStatus.DONE
  ) {
    // The initial no-rollout result belongs to the stale plan half. Do not
    // publish it if reconciliation itself fails; the caller must retain its
    // last-known-good deploy state and let the next poll retry.
    rolloutState = undefined;
    const refreshedPlan = await planServiceClientConnect
      .getPlan(create(GetPlanRequestSchema, { name: plan.name }), silentCtx)
      .catch(() => undefined);
    if (refreshedPlan?.hasRollout) {
      plan = refreshedPlan;
      rolloutName = getRolloutFromPlan(plan.name);
      rolloutState = await requestRolloutState(rolloutName).catch(
        () => undefined
      );
    }
  }

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
    ...(issueResult.status === "fulfilled" ? { issue: issueResult.value } : {}),
    ...(planCheckRunsResult.status === "fulfilled"
      ? { planCheckRuns: planCheckRunsResult.value }
      : {}),
    ...(rolloutState ? rolloutState : {}),
  };
};
