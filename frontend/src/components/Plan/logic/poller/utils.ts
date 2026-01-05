import { create } from "@bufbuild/protobuf";
import type { Ref } from "vue";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/connect";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { GetIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import {
  GetPlanCheckRunRequestSchema,
  GetPlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Rollout, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import {
  GetRolloutRequestSchema,
  ListTaskRunsRequestSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
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
  if (!hasProjectPermissionV2(project, "bb.planCheckRuns.get")) {
    return;
  }

  const request = create(GetPlanCheckRunRequestSchema, {
    name: `${plan.name}/planCheckRun`,
  });
  try {
    const response = await planServiceClientConnect.getPlanCheckRun(request);
    planCheckRuns.value = [response];
  } catch {
    // Plan check run might not exist yet
    planCheckRuns.value = [];
  }
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
  try {
    const newRollout =
      await rolloutServiceClientConnect.getRollout(rolloutRequest);
    rollout.value = newRollout;
  } catch (error) {
    console.error("Failed to refresh rollout", error);
    // Rollout might not exist yet
  }
};

export const refreshIssue = async (
  issue: Ref<Issue | undefined>
): Promise<void> => {
  if (!issue.value) {
    return;
  }
  const request = create(GetIssueRequestSchema, {
    name: issue.value.name,
  });
  const newIssue = await issueServiceClientConnect.getIssue(request);
  issue.value = newIssue;
};

export interface TaskRunScope {
  stageId?: string;
  taskId?: string;
}

export const refreshTaskRuns = async (
  rollout: Rollout,
  project: Project,
  taskRuns: Ref<TaskRun[]>,
  scope?: TaskRunScope
): Promise<void> => {
  if (!hasProjectPermissionV2(project, "bb.taskRuns.list")) {
    return;
  }

  // Build parent path based on scope
  // - No scope: fetch all task runs for the rollout
  // - stageId only: fetch task runs for a specific stage
  // - stageId + taskId: fetch task runs for a specific task
  const stagePart = scope?.stageId ?? "-";
  const taskPart = scope?.taskId ?? "-";
  const parent = `${rollout.name}/stages/${stagePart}/tasks/${taskPart}`;

  const request = create(ListTaskRunsRequestSchema, {
    parent,
  });
  try {
    const response = await rolloutServiceClientConnect.listTaskRuns(request);
    taskRuns.value = response.taskRuns;
  } catch (error) {
    console.error("Failed to refresh task runs", error);
  }
};
