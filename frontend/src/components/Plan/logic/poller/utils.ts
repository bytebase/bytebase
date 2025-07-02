import { create } from "@bufbuild/protobuf";
import { reactive, type Ref } from "vue";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/grpcweb";
import { useIssueCommentStore } from "@/store";
import type { ComposedProject } from "@/types";
import { GetIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import {
  GetPlanRequestSchema,
  ListPlanCheckRunsRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { GetRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { ListIssueCommentsRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { hasProjectPermissionV2 } from "@/utils";

interface RefreshTimestamps {
  plan?: number;
  planCheckRuns?: number;
  rollout?: number;
  issue?: number;
  issueComments?: number;
}

const lastRefreshTime = reactive<RefreshTimestamps>({});

export const getLastRefreshTime = (
  resource: keyof RefreshTimestamps
): number | undefined => {
  return lastRefreshTime[resource];
};

export const refreshPlan = async (plan: Ref<Plan>): Promise<void> => {
  const request = create(GetPlanRequestSchema, {
    name: plan.value.name,
  });
  const response = await planServiceClientConnect.getPlan(request);
  plan.value = response;
  lastRefreshTime.plan = Date.now();
};

export const refreshPlanCheckRuns = async (
  plan: Plan,
  project: ComposedProject,
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
  lastRefreshTime.planCheckRuns = Date.now();
};

export const refreshRollout = async (
  rolloutName: string,
  project: ComposedProject,
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
  lastRefreshTime.rollout = Date.now();
};

export const refreshIssue = async (issue: Ref<Issue>): Promise<void> => {
  const request = create(GetIssueRequestSchema, {
    name: issue.value.name,
  });
  const newIssue = await issueServiceClientConnect.getIssue(request);
  issue.value = newIssue;
  lastRefreshTime.issue = Date.now();
};

export const refreshIssueComments = async (issue: Issue): Promise<void> => {
  const issueCommentStore = useIssueCommentStore();
  await issueCommentStore.listIssueComments(
    create(ListIssueCommentsRequestSchema, {
      parent: issue.name,
      pageSize: 100,
    })
  );
  lastRefreshTime.issueComments = Date.now();
};
