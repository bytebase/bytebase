import { create } from "@bufbuild/protobuf";
import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  GetIssueRequestSchema,
  GetReviewRunRequestSchema,
  Issue_Type,
  ReviewRun_Status,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  GetPlanCheckRunRequestSchema,
  GetPlanRequestSchema,
  PlanCheckRun_Status,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  GetRolloutRequestSchema,
  ListTaskRunsRequestSchema,
  Task_Status,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { getRolloutFromPlan } from "@/utils";
import { shouldRedirectToIssueDetail } from "@/utils/v1/issue/plan";
import type { AgentEvidence } from "../types";

const TASK_RUN_PAGE_SIZE = 10;
const MAX_FAILED_TASKS = 10;

type Available<T> = { status: "available" } & T;
type Unavailable = { status: "unavailable" };
type NotConfigured = { status: "not_configured" };
type NotStarted = { status: "not_started" };

type ReviewFacet =
  | Available<{ runStatus: string }>
  | Unavailable
  | NotConfigured
  | NotStarted;

type IssueOverview = {
  project: string;
  issue:
    | Available<{
        name: string;
        title: string;
        type: string;
        issueStatus: string;
        approvalStatus: string;
      }>
    | Unavailable;
  plan:
    | Available<{
        name: string;
        title: string;
        hasRollout: boolean;
        reviewOnPlan: boolean;
      }>
    | Unavailable
    | NotConfigured;
  planCheck:
    | Available<{
        runStatus: string;
        resultCounts: { success: number; warning: number; error: number };
      }>
    | Unavailable
    | NotConfigured
    | NotStarted;
  reviews: { rule: ReviewFacet; ai: ReviewFacet };
  rollout:
    | Available<{
        name: string;
        stageCount: number;
        failedTasks: string[];
        failedTasksPartial: boolean;
      }>
    | Unavailable
    | NotStarted;
  taskRuns:
    | Available<{
        partial: boolean;
        taskRuns: { name: string; status: string }[];
      }>
    | Unavailable
    | NotStarted;
  evidence: AgentEvidence;
};

type IssueOverviewArgs = { issueName?: unknown };

function isProjectName(value: string | undefined): value is string {
  return !!value && /^projects\/[^/]+$/.test(value);
}

function isIssueNameInProject(
  projectName: string,
  issueName: string | undefined
): issueName is string {
  if (!issueName) return false;
  const [projectPrefix, projectId] = projectName.split("/");
  const [
    issueProjectPrefix,
    issueProjectId,
    collection,
    issueId,
    ...remainingSegments
  ] = issueName.split("/");
  return (
    projectPrefix === "projects" &&
    issueProjectPrefix === projectPrefix &&
    issueProjectId === projectId &&
    collection === "issues" &&
    Boolean(issueId) &&
    remainingSegments.length === 0
  );
}

function isPlanNameInProject(projectName: string, planName: string): boolean {
  const [projectPrefix, projectId] = projectName.split("/");
  const [
    planProjectPrefix,
    planProjectId,
    collection,
    planId,
    ...remainingSegments
  ] = planName.split("/");
  return (
    projectPrefix === "projects" &&
    planProjectPrefix === projectPrefix &&
    planProjectId === projectId &&
    collection === "plans" &&
    Boolean(planId) &&
    remainingSegments.length === 0
  );
}

function isTaskNameInStage(
  rolloutName: string,
  stageName: string,
  taskName: string
): boolean {
  const stagePrefix = `${rolloutName}/stages/`;
  if (!stageName.startsWith(stagePrefix)) return false;
  const stageId = stageName.slice(stagePrefix.length);
  if (!/^(?:[A-Za-z0-9_][A-Za-z0-9_-]*|-)$/.test(stageId)) return false;
  const taskPrefix = `${stageName}/tasks/`;
  if (!taskName.startsWith(taskPrefix)) return false;
  return /^[A-Za-z0-9_][A-Za-z0-9_-]*$/.test(taskName.slice(taskPrefix.length));
}

function enumName(values: Record<number, string | number>, value: number) {
  const name = values[value];
  return typeof name === "string" ? name : "UNSPECIFIED";
}

function isNotFound(error: unknown) {
  return ConnectError.from(error).code === Code.NotFound;
}

function unavailableOverview(project: string): IssueOverview {
  return {
    project,
    issue: { status: "unavailable" },
    plan: { status: "unavailable" },
    planCheck: { status: "unavailable" },
    reviews: {
      rule: { status: "unavailable" },
      ai: { status: "unavailable" },
    },
    rollout: { status: "unavailable" },
    taskRuns: { status: "unavailable" },
    evidence: { project, references: [] },
  };
}

function countPlanCheckResults(
  results: Awaited<
    ReturnType<typeof planServiceClientConnect.getPlanCheckRun>
  >["results"]
) {
  const resultCounts = { success: 0, warning: 0, error: 0 };
  for (const result of results) {
    if (result.status === Advice_Level.SUCCESS) {
      resultCounts.success += 1;
    } else if (result.status === Advice_Level.WARNING) {
      resultCounts.warning += 1;
    } else if (result.status === Advice_Level.ERROR) {
      resultCounts.error += 1;
    }
  }
  return resultCounts;
}

function parseReviewResult(
  result: PromiseSettledResult<
    Awaited<ReturnType<typeof issueServiceClientConnect.getReviewRun>>
  >,
  expectedName: string
): ReviewFacet {
  if (result.status === "fulfilled") {
    if (result.value.name !== expectedName) {
      return { status: "unavailable" };
    }
    return {
      status: "available",
      runStatus: enumName(ReviewRun_Status, result.value.status),
    };
  }
  return isNotFound(result.reason)
    ? { status: "not_started" }
    : { status: "unavailable" };
}

export async function collectIssueOverview(
  projectName: string | undefined,
  issueName: string | undefined
): Promise<IssueOverview | null> {
  if (
    !isProjectName(projectName) ||
    !isIssueNameInProject(projectName, issueName)
  ) {
    return null;
  }

  const silentOptions = {
    contextValues: createContextValues().set(silentContextKey, true),
  };
  const issueResult = await Promise.allSettled([
    issueServiceClientConnect.getIssue(
      create(GetIssueRequestSchema, { name: issueName }),
      silentOptions
    ),
  ]);
  if (issueResult[0].status !== "fulfilled") {
    return unavailableOverview(projectName);
  }

  const issue = issueResult[0].value;
  if (
    !isIssueNameInProject(projectName, issue.name) ||
    issue.name !== issueName
  ) {
    return unavailableOverview(projectName);
  }

  const issueFacet: IssueOverview["issue"] = {
    status: "available",
    name: issue.name,
    title: issue.title,
    type: enumName(Issue_Type, issue.type),
    issueStatus: enumName(IssueStatus, issue.status),
    approvalStatus: enumName(ApprovalStatus, issue.approvalStatus),
  };
  const evidence: AgentEvidence = {
    project: projectName,
    references: [{ kind: "issue", name: issue.name, title: issue.title }],
  };

  if (!issue.plan) {
    return {
      project: projectName,
      issue: issueFacet,
      plan: { status: "not_configured" },
      planCheck: { status: "not_configured" },
      reviews: {
        rule: { status: "not_configured" },
        ai: { status: "not_configured" },
      },
      rollout: { status: "not_started" },
      taskRuns: { status: "not_started" },
      evidence,
    };
  }

  if (!isPlanNameInProject(projectName, issue.plan)) {
    return {
      ...unavailableOverview(projectName),
      issue: issueFacet,
      evidence,
    };
  }

  const [planResult, planCheckResult, ruleReviewResult, aiReviewResult] =
    await Promise.allSettled([
      planServiceClientConnect.getPlan(
        create(GetPlanRequestSchema, { name: issue.plan }),
        silentOptions
      ),
      planServiceClientConnect.getPlanCheckRun(
        create(GetPlanCheckRunRequestSchema, {
          name: `${issue.plan}/planCheckRun`,
        }),
        silentOptions
      ),
      issueServiceClientConnect.getReviewRun(
        create(GetReviewRunRequestSchema, {
          name: `${issue.name}/reviewRuns/rule`,
        }),
        silentOptions
      ),
      issueServiceClientConnect.getReviewRun(
        create(GetReviewRunRequestSchema, {
          name: `${issue.name}/reviewRuns/ai`,
        }),
        silentOptions
      ),
    ]);

  const plan =
    planResult.status === "fulfilled" &&
    planResult.value.name === issue.plan &&
    isPlanNameInProject(projectName, planResult.value.name)
      ? {
          status: "available" as const,
          name: planResult.value.name,
          title: planResult.value.title,
          hasRollout: planResult.value.hasRollout,
          reviewOnPlan: !shouldRedirectToIssueDetail(planResult.value, issue),
        }
      : { status: "unavailable" as const };
  const planCheck =
    planCheckResult.status === "fulfilled"
      ? planCheckResult.value.name === `${plan.name}/planCheckRun`
        ? {
            status: "available" as const,
            runStatus: enumName(
              PlanCheckRun_Status,
              planCheckResult.value.status
            ),
            resultCounts: countPlanCheckResults(planCheckResult.value.results),
          }
        : { status: "unavailable" as const }
      : isNotFound(planCheckResult.reason)
        ? { status: "not_started" as const }
        : { status: "unavailable" as const };
  const reviews = {
    rule: parseReviewResult(ruleReviewResult, `${issue.name}/reviewRuns/rule`),
    ai: parseReviewResult(aiReviewResult, `${issue.name}/reviewRuns/ai`),
  };

  if (plan.status !== "available") {
    return {
      project: projectName,
      issue: issueFacet,
      plan,
      planCheck,
      reviews,
      rollout: { status: "unavailable" },
      taskRuns: { status: "unavailable" },
      evidence,
    };
  }

  if (!plan.hasRollout) {
    return {
      project: projectName,
      issue: issueFacet,
      plan,
      planCheck,
      reviews,
      rollout: { status: "not_started" },
      taskRuns: { status: "not_started" },
      evidence,
    };
  }

  const rolloutName = getRolloutFromPlan(plan.name);
  const rolloutResult = await Promise.allSettled([
    rolloutServiceClientConnect.getRollout(
      create(GetRolloutRequestSchema, { name: rolloutName }),
      silentOptions
    ),
  ]);
  if (
    rolloutResult[0].status !== "fulfilled" ||
    rolloutResult[0].value.name !== rolloutName
  ) {
    return {
      project: projectName,
      issue: issueFacet,
      plan,
      planCheck,
      reviews,
      rollout: { status: "unavailable" },
      taskRuns: { status: "unavailable" },
      evidence,
    };
  }

  const rollout = rolloutResult[0].value;
  const failedTasks: string[] = [];
  let failedTasksPartial = false;
  for (const stage of rollout.stages) {
    for (const task of stage.tasks) {
      if (
        task.status !== Task_Status.FAILED ||
        !isTaskNameInStage(rollout.name, stage.name, task.name)
      ) {
        continue;
      }
      if (failedTasks.length === MAX_FAILED_TASKS) {
        failedTasksPartial = true;
        break;
      }
      failedTasks.push(task.name);
    }
    if (failedTasksPartial) break;
  }
  const taskRunsResult = await Promise.allSettled([
    rolloutServiceClientConnect.listTaskRuns(
      create(ListTaskRunsRequestSchema, {
        parent: `${rollout.name}/stages/-/tasks/-`,
        pageSize: TASK_RUN_PAGE_SIZE,
      }),
      silentOptions
    ),
  ]);
  const taskRuns =
    taskRunsResult[0].status === "fulfilled" &&
    Array.isArray(taskRunsResult[0].value.taskRuns) &&
    taskRunsResult[0].value.taskRuns.every((taskRun) =>
      taskRun.name.startsWith(`${rollout.name}/stages/`)
    )
      ? {
          status: "available" as const,
          partial: Boolean(taskRunsResult[0].value.nextPageToken),
          taskRuns: taskRunsResult[0].value.taskRuns.map((taskRun) => ({
            name: taskRun.name,
            status: enumName(TaskRun_Status, taskRun.status),
          })),
        }
      : { status: "unavailable" as const };

  return {
    project: projectName,
    issue: issueFacet,
    plan,
    planCheck,
    reviews,
    rollout: {
      status: "available",
      name: rollout.name,
      stageCount: rollout.stages.length,
      failedTasks,
      failedTasksPartial,
    },
    taskRuns,
    evidence,
  };
}

export function extractIssueOverviewEvidence(
  result: string
): AgentEvidence | undefined {
  try {
    const payload = JSON.parse(result) as { evidence?: unknown };
    if (!payload.evidence || typeof payload.evidence !== "object") {
      return undefined;
    }
    const evidence = payload.evidence as {
      project?: unknown;
      references?: unknown;
    };
    const project = evidence.project;
    if (
      typeof project !== "string" ||
      !isProjectName(project) ||
      !Array.isArray(evidence.references)
    ) {
      return undefined;
    }

    const references = evidence.references.flatMap((reference) => {
      if (!reference || typeof reference !== "object") return [];
      const value = reference as Record<string, unknown>;
      if (
        value.kind !== "issue" ||
        typeof value.name !== "string" ||
        !isIssueNameInProject(project, value.name) ||
        typeof value.title !== "string"
      ) {
        return [];
      }
      return [
        {
          kind: "issue" as const,
          name: value.name as string,
          title: value.title,
        },
      ];
    });

    return references.length > 0 ? { project, references } : undefined;
  } catch {
    return undefined;
  }
}

export function createIssueOverviewTool({
  projectName,
}: {
  projectName?: string;
}) {
  return async (args: IssueOverviewArgs = {}): Promise<string> => {
    const issueName =
      typeof args.issueName === "string" ? args.issueName : undefined;
    const overview = await collectIssueOverview(projectName, issueName);
    return JSON.stringify(
      overview ?? {
        error: "The Issue must belong to the project bound to this chat.",
      }
    );
  };
}
