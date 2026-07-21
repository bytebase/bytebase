import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { PlanCheckRun_Status } from "@/types/proto-es/v1/plan_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";

type T = (key: string, options?: Record<string, unknown>) => string;

const isDatabaseGroupName = (name: string): boolean =>
  name.includes("/databaseGroups/");

export interface PlanCheckSummary {
  error: number;
  running: number;
  success: number;
  total: number;
  warning: number;
}

export const getPlanCheckSummary = (plan: Plan): PlanCheckSummary => {
  const statusCount = plan.planCheckRunStatusCount || {};
  const running =
    statusCount[PlanCheckRun_Status[PlanCheckRun_Status.RUNNING]] || 0;
  const success = statusCount[Advice_Level[Advice_Level.SUCCESS]] || 0;
  const warning = statusCount[Advice_Level[Advice_Level.WARNING]] || 0;
  const error = statusCount[Advice_Level[Advice_Level.ERROR]] || 0;
  const failed =
    statusCount[PlanCheckRun_Status[PlanCheckRun_Status.FAILED]] || 0;
  const totalError = error + failed;

  return {
    error: totalError,
    running,
    success,
    total: running + success + warning + totalError,
    warning,
  };
};

const targetsForSpec = (spec: Plan["specs"][number]): string[] => {
  if (spec.config?.case === "changeDatabaseConfig") {
    return spec.config.value.targets || [];
  }
  if (spec.config?.case === "exportDataConfig") {
    return spec.config.value.targets || [];
  }
  return [];
};

const extractPrincipalEmail = (principal: string): string =>
  principal.replace(/^users\//, "");

export const buildChangesSummary = (plan: Plan, t: T): string => {
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
  if (checks.total > 0) {
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

export const buildReviewSummary = (issue: Issue | undefined, t: T): string => {
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
