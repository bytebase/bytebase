import {
  getPlanCheckSummaryWithFallback,
  type PlanCheckSummary,
} from "@/lib/plan/check";
import {
  ApprovalStatus,
  IssueStatus,
  State,
} from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import type { AutomatedReviewRow } from "./planGates";
import { reviewSummaryPart } from "./reviewLabels";

export type { PlanCheckSummary } from "@/lib/plan/check";

type T = (key: string, options?: Record<string, unknown>) => string;

const isDatabaseGroupName = (name: string): boolean =>
  name.includes("/databaseGroups/");

// Mirrors the eligibility gates that can permanently prevent RolloutCreator
// from materializing a rollout. Transient approval and check gates are handled
// separately below.
export const supportsAutomaticRollout = (plan: Plan): boolean =>
  plan.state === State.ACTIVE &&
  plan.specs.length > 0 &&
  plan.specs.every((spec) => spec.config?.case === "changeDatabaseConfig");

export const getPlanCheckSummary = (plan: Plan): PlanCheckSummary =>
  getPlanCheckSummaryWithFallback([], plan.planCheckRunStatusCount);

export const isRolloutExpected = ({
  issue,
  plan,
}: {
  issue?: Issue;
  plan: Plan;
}): boolean => {
  if (plan.hasRollout) {
    return true;
  }
  if (!supportsAutomaticRollout(plan)) {
    return false;
  }
  if (issue?.status === IssueStatus.DONE) {
    return true;
  }
  const approvalPassed =
    issue?.approvalStatus === ApprovalStatus.APPROVED ||
    issue?.approvalStatus === ApprovalStatus.SKIPPED ||
    issue?.approvalTemplate?.flow?.roles.length === 0;
  if (issue?.status !== IssueStatus.OPEN || issue.draft || !approvalPassed) {
    return false;
  }
  const checks = getPlanCheckSummary(plan);
  return (
    checks.running === 0 &&
    checks.error === 0 &&
    (plan.planCheckRunStatusCount?.AVAILABLE ?? 0) === 0
  );
};

const targetsForSpec = (spec: Plan["specs"][number]): string[] => {
  if (spec.config?.case === "changeDatabaseConfig") {
    return spec.config.value.targets || [];
  }
  return [];
};

const extractPrincipalEmail = (principal: string): string =>
  principal.replace(/^users\//, "");

// Under SQL Review V2 the checks are reviewers in the Review section, so the
// Changes summary leaves them out.
export const buildChangesSummary = (
  plan: Plan,
  t: T,
  { includeChecks = true }: { includeChecks?: boolean } = {}
): string => {
  const parts: string[] = [];

  parts.push(t("plan.summary.n-changes", { n: plan.specs.length }));

  const targets = Array.from(new Set(plan.specs.flatMap(targetsForSpec)));
  if (targets.length > 0) {
    const dbGroupCount = targets.filter(isDatabaseGroupName).length;
    const dbCount = targets.length - dbGroupCount;
    const targetParts: string[] = [];
    if (dbCount > 0) {
      targetParts.push(t("plan.summary.n-databases", { n: dbCount }));
    }
    if (dbGroupCount > 0) {
      targetParts.push(
        t("plan.summary.n-database-groups", { n: dbGroupCount })
      );
    }
    parts.push(targetParts.join(` ${t("common.and")} `));
  }

  const checks = getPlanCheckSummary(plan);
  if (includeChecks && checks.total > 0) {
    const checkParts: string[] = [];
    if (checks.success > 0) {
      checkParts.push(t("plan.summary.n-passed", { n: checks.success }));
    }
    if (checks.warning > 0) {
      checkParts.push(t("plan.summary.n-warning", { n: checks.warning }));
    }
    if (checks.error > 0) {
      checkParts.push(t("plan.summary.n-error", { n: checks.error }));
    }
    if (checkParts.length > 0) {
      parts.push(checkParts.join(", "));
    }
  }

  return parts.join(" · ");
};

export const buildReviewSummary = (
  issue: Issue | undefined,
  t: T,
  unresolvedThreads = 0
): string => {
  if (!issue) return "";

  const roles = issue.approvalTemplate?.flow?.roles ?? [];
  const approvers = issue.approvers;
  const parts: string[] = [];

  if (roles.length > 0) {
    parts.push(
      t("plan.summary.n-of-m-approved", {
        m: roles.length,
        n: approvers.length,
      })
    );
  }

  if (approvers.length > 0) {
    const last = approvers[approvers.length - 1];
    const name = extractPrincipalEmail(last.principal).split("@")[0];
    parts.push(t("plan.summary.last-approved-by", { name }));
  }

  if (unresolvedThreads > 0) {
    parts.push(
      t("plan.summary.n-unresolved-threads", { count: unresolvedThreads })
    );
  }

  return parts.join(" · ");
};

// The Review summary under SQL Review V2: the approval outcome, then each
// reviewer's conclusion, so a reader can tell people resolved the findings.
export const buildReviewSummaryV2 = (
  issue: Issue | undefined,
  t: T,
  rows: readonly AutomatedReviewRow[],
  openDiscussions: number
): string => {
  if (!issue) return "";
  const parts: string[] = [];
  const roles = issue.approvalTemplate?.flow?.roles ?? [];
  if (!issue.draft) {
    if (
      issue.approvalStatus === ApprovalStatus.SKIPPED ||
      (issue.approvalTemplate && roles.length === 0)
    ) {
      parts.push(t("plan.summary.no-approval-required"));
    } else if (issue.approvalStatus === ApprovalStatus.APPROVED) {
      const names = issue.approvers.map(
        (approver) => extractPrincipalEmail(approver.principal).split("@")[0]
      );
      parts.push(t("plan.summary.approved-by", { names: names.join(", ") }));
    } else if (issue.approvalStatus === ApprovalStatus.REJECTED) {
      parts.push(t("plan.lifecycle.gate-review-rejected"));
    } else if (roles.length > 0) {
      parts.push(
        t("plan.summary.n-of-m-approved", {
          m: roles.length,
          n: issue.approvers.length,
        })
      );
    }
  }
  for (const row of rows) {
    parts.push(reviewSummaryPart(t, row));
  }
  if (openDiscussions > 0) {
    parts.push(
      t("plan.deploy-waiting.open-discussions", { count: openDiscussions })
    );
  }
  return parts.join(" · ");
};

export const buildDeploySummary = (
  rollout: Rollout | undefined,
  t: T
): string => {
  if (!rollout) return "";

  const allTasks = rollout.stages.flatMap((stage) => stage.tasks);
  const completed = allTasks.filter(
    (task) =>
      task.status === Task_Status.DONE || task.status === Task_Status.SKIPPED
  ).length;

  return t("plan.summary.n-of-m-tasks", {
    m: allTasks.length,
    n: completed,
  });
};
