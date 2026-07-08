import { create } from "@bufbuild/protobuf";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  projectServiceClientConnect,
  rolloutServiceClientConnect,
  userServiceClientConnect,
} from "@/connect";
import { useAppStore } from "@/react/stores/app";
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
  routeQuery: Record<string, unknown> = {}
): Promise<PlanDetailFetchPatch> => {
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
          })
        );
  planPromise?.catch(() => undefined);

  const [project, currentUser] = await Promise.all([
    projectServiceClientConnect.getProject(
      create(GetProjectRequestSchema, {
        name: `${PROJECT_NAME_PREFIX}${projectId}`,
      })
    ),
    userServiceClientConnect.getCurrentUser({}),
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
          .getIssue(create(GetIssueRequestSchema, { name: plan.issue }))
          .catch(() => undefined)
      : Promise.resolve(undefined),
    planServiceClientConnect
      .getPlanCheckRun(
        create(GetPlanCheckRunRequestSchema, {
          name: `${plan.name}/planCheckRun`,
        })
      )
      .then((run) => [run] as PlanCheckRun[])
      .catch(() => []),
    // Fetch through the store so the shared rollout cache stays seeded —
    // cache-first consumers (e.g. the task-run log viewer) then resolve tasks
    // without their own GetRollout round trip, and polling keeps it fresh.
    rolloutName
      ? useAppStore
          .getState()
          .fetchRolloutByName(rolloutName)
          .catch(() => undefined)
      : Promise.resolve(undefined),
    rolloutName
      ? rolloutServiceClientConnect
          .listTaskRuns(
            create(ListTaskRunsRequestSchema, {
              parent: `${rolloutName}/stages/-/tasks/-`,
            })
          )
          .then((response) => response.taskRuns)
          .catch(() => [])
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
