import { uniq } from "lodash-es";
import { computed, type Ref } from "vue";
import { useI18n } from "vue-i18n";
import { extractUserEmail } from "@/store";
import { isValidDatabaseGroupName } from "@/types";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { targetsForSpec } from "../../logic/plan";
import { usePlanCheckStatus } from "../../logic/usePlanCheckStatus";

export const usePhaseSummaries = (
  plan: Ref<Plan>,
  issue: Ref<Issue | undefined>,
  rollout: Ref<Rollout | undefined>
) => {
  const { t } = useI18n();
  const { statusSummary } = usePlanCheckStatus(plan);

  const changesSummary = computed(() => {
    const parts: string[] = [];

    const specCount = plan.value.specs.length;
    parts.push(t("plan.summary.n-changes", { n: specCount }));

    const targets = uniq(plan.value.specs.flatMap(targetsForSpec));
    if (targets.length > 0) {
      const dbGroupCount = targets.filter(isValidDatabaseGroupName).length;
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

    const checks = statusSummary.value;
    if (checks.total > 0) {
      const checkParts: string[] = [];
      if (checks.success > 0)
        checkParts.push(t("plan.summary.n-passed", { n: checks.success }));
      if (checks.warning > 0)
        checkParts.push(t("plan.summary.n-warning", { n: checks.warning }));
      if (checks.error > 0)
        checkParts.push(t("plan.summary.n-error", { n: checks.error }));
      parts.push(checkParts.join(", "));
    }

    return parts.join(" · ");
  });

  const reviewSummary = computed(() => {
    if (!issue.value) return "";
    const roles = issue.value.approvalTemplate?.flow?.roles ?? [];
    const approvers = issue.value.approvers;
    const parts: string[] = [];

    if (roles.length > 0) {
      parts.push(
        t("plan.summary.n-of-m-approved", {
          n: approvers.length,
          m: roles.length,
        })
      );
    }

    if (approvers.length > 0) {
      const last = approvers[approvers.length - 1];
      const name = extractUserEmail(last.principal).split("@")[0];
      parts.push(t("plan.summary.last-approved-by", { name }));
    }

    return parts.join(" · ");
  });

  const deploySummary = computed(() => {
    if (!rollout.value) return "";
    const allTasks = rollout.value.stages.flatMap((s) => s.tasks);
    const total = allTasks.length;
    const completed = allTasks.filter(
      (task) =>
        task.status === Task_Status.DONE || task.status === Task_Status.SKIPPED
    ).length;
    return t("plan.summary.n-of-m-tasks", { n: completed, m: total });
  });

  return { changesSummary, reviewSummary, deploySummary };
};
