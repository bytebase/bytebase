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
import { ListIssueCommentsRequest } from "@/types/proto/v1/issue_service";
import type { Issue } from "@/types/proto/v1/issue_service";
import type { Plan, PlanCheckRun } from "@/types/proto/v1/plan_service";
import type { Rollout } from "@/types/proto/v1/rollout_service";
import { hasProjectPermissionV2 } from "@/utils";
import { convertNewIssueToOld } from "@/utils/v1/issue-conversions";
import {
  convertNewPlanCheckRunToOld,
  convertNewPlanToOld,
} from "@/utils/v1/plan-conversions";
import { convertNewRolloutToOld } from "@/utils/v1/rollout-conversions";

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
  plan.value = convertNewPlanToOld(response);
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
  planCheckRuns.value = response.planCheckRuns.map(convertNewPlanCheckRunToOld);
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
  rollout.value = convertNewRolloutToOld(newRollout);
  lastRefreshTime.rollout = Date.now();
};

export const refreshIssue = async (issue: Ref<Issue>): Promise<void> => {
  const request = create(GetIssueRequestSchema, {
    name: issue.value.name,
  });
  const newIssue = await issueServiceClientConnect.getIssue(request);
  const updatedIssue = convertNewIssueToOld(newIssue);
  issue.value = updatedIssue;
  lastRefreshTime.issue = Date.now();
};

export const refreshIssueComments = async (issue: Issue): Promise<void> => {
  const issueCommentStore = useIssueCommentStore();
  await issueCommentStore.listIssueComments(
    ListIssueCommentsRequest.fromPartial({
      parent: issue.name,
      pageSize: 100,
    })
  );
  lastRefreshTime.issueComments = Date.now();
};
