import { create } from "@bufbuild/protobuf";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  projectServiceClientConnect,
  rolloutServiceClientConnect,
  userServiceClientConnect,
} from "@/connect";
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
  const [project, currentUser] = await Promise.all([
    projectServiceClientConnect.getProject(
      create(GetProjectRequestSchema, {
        name: `${PROJECT_NAME_PREFIX}${projectId}`,
      })
    ),
    userServiceClientConnect.getCurrentUser({}),
  ]);

  if (planId.toLowerCase() === "create") {
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

  const plan = await planServiceClientConnect.getPlan(
    create(GetPlanRequestSchema, {
      name: `${PROJECT_NAME_PREFIX}${projectId}/plans/${planId}`,
    })
  );

  const [issue, planCheckRuns, rollout] = await Promise.all([
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
    plan.hasRollout
      ? rolloutServiceClientConnect
          .getRollout(
            create(GetRolloutRequestSchema, {
              name: getRolloutFromPlan(plan.name),
            })
          )
          .catch(() => undefined)
      : Promise.resolve(undefined),
  ]);

  const taskRuns =
    rollout !== undefined
      ? await rolloutServiceClientConnect
          .listTaskRuns(
            create(ListTaskRunsRequestSchema, {
              parent: `${rollout.name}/stages/-/tasks/-`,
            })
          )
          .then((response) => response.taskRuns)
          .catch(() => [])
      : [];

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
