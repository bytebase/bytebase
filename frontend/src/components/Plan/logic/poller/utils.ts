import { create } from "@bufbuild/protobuf";
import type { Ref } from "vue";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/grpcweb";
import { GetIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  GetPlanRequestSchema,
  ListPlanCheckRunsRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Plan, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  GetRolloutRequestSchema,
  ListTaskRunsRequestSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Rollout, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { hasProjectPermissionV2 } from "@/utils";

export const refreshPlan = async (plan: Ref<Plan>): Promise<void> => {
  const request = create(GetPlanRequestSchema, {
    name: plan.value.name,
  });
  const response = await planServiceClientConnect.getPlan(request);
  plan.value = response;
};

export const refreshPlanCheckRuns = async (
  plan: Plan,
  project: Project,
  planCheckRuns: Ref<PlanCheckRun[]>
): Promise<void> => {
  if (!hasProjectPermissionV2(project, "bb.planCheckRuns.list")) {
    return;
  }

  const request = create(ListPlanCheckRunsRequestSchema, {
    parent: plan.name,
    latestOnly: true,
  });
  const response = await planServiceClientConnect.listPlanCheckRuns(request);
  planCheckRuns.value = response.planCheckRuns;
};

export const refreshRollout = async (
  rolloutName: string,
  project: Project,
  rollout: Ref<Rollout | undefined>
): Promise<void> => {
  if (!hasProjectPermissionV2(project, "bb.rollouts.get")) {
    return;
  }

  const rolloutRequest = create(GetRolloutRequestSchema, {
    name: rolloutName,
  });
  const newRollout =
    await rolloutServiceClientConnect.getRollout(rolloutRequest);
  rollout.value = newRollout;
};

export const refreshIssue = async (issue: Ref<Issue>): Promise<void> => {
  const request = create(GetIssueRequestSchema, {
    name: issue.value.name,
  });
  const newIssue = await issueServiceClientConnect.getIssue(request);
  issue.value = newIssue;
};

export const refreshTaskRuns = async (
  rollout: Rollout,
  project: Project,
  taskRuns: Ref<TaskRun[]>
): Promise<void> => {
  if (!hasProjectPermissionV2(project, "bb.taskRuns.list")) {
    return;
  }

  const request = create(ListTaskRunsRequestSchema, {
    parent: `${rollout.name}/stages/-/tasks/-`,
  });
  const response = await rolloutServiceClientConnect.listTaskRuns(request);
  taskRuns.value = response.taskRuns;
};
