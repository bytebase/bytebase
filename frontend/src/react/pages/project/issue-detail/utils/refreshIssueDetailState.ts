import { create } from "@bufbuild/protobuf";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/connect";
import { GetIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import {
  GetPlanCheckRunRequestSchema,
  GetPlanRequestSchema,
  type PlanCheckRun,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  GetRolloutRequestSchema,
  ListTaskRunsRequestSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { getRolloutFromPlan } from "@/utils";
import type { IssueDetailPageState } from "../hooks/useIssueDetailPage";

export async function refreshIssueDetailState(
  page: Pick<IssueDetailPageState, "issue" | "patchState" | "plan">
) {
  if (!page.plan?.name) {
    return;
  }

  const nextPlan = await planServiceClientConnect.getPlan(
    create(GetPlanRequestSchema, {
      name: page.plan.name,
    })
  );

  const [issueResult, planCheckRunResult, rolloutResult] = await Promise.all([
    page.issue?.name
      ? issueServiceClientConnect
          .getIssue(
            create(GetIssueRequestSchema, {
              name: page.issue.name,
            })
          )
          .catch(() => undefined)
      : Promise.resolve(undefined),
    planServiceClientConnect
      .getPlanCheckRun(
        create(GetPlanCheckRunRequestSchema, {
          name: `${page.plan.name}/planCheckRun`,
        })
      )
      .then((result) => [result] as PlanCheckRun[])
      .catch(() => []),
    nextPlan.hasRollout
      ? rolloutServiceClientConnect
          .getRollout(
            create(GetRolloutRequestSchema, {
              name: getRolloutFromPlan(nextPlan.name),
            })
          )
          .catch(() => undefined)
      : Promise.resolve(undefined),
  ]);

  const nextTaskRuns =
    rolloutResult !== undefined
      ? await rolloutServiceClientConnect
          .listTaskRuns(
            create(ListTaskRunsRequestSchema, {
              parent: `${rolloutResult.name}/stages/-/tasks/-`,
            })
          )
          .then((response) => response.taskRuns)
          .catch(() => [])
      : [];

  page.patchState({
    issue: issueResult,
    plan: nextPlan,
    planCheckRuns: planCheckRunResult,
    rollout: rolloutResult,
    taskRuns: nextTaskRuns,
  });
}
