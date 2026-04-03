import type { TagProps } from "naive-ui";
import { computed, type Ref } from "vue";
import { useI18n } from "vue-i18n";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import {
  Issue_ApprovalStatus,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import type { PhaseType } from "./useActivePhase";

export type PhaseStatus = "completed" | "closed" | "active" | "future";

export interface PhaseBadge {
  label: string;
  type: TagProps["type"];
}

export interface PhaseConfig {
  status: PhaseStatus;
  badge?: PhaseBadge;
  lineClass: string;
}

export const usePhaseState = (
  isCreating: Ref<boolean>,
  issue: Ref<Issue | undefined>,
  rollout: Ref<Rollout | undefined>
) => {
  const { t } = useI18n();

  const statusMap = computed<Record<PhaseType, PhaseStatus>>(() => {
    const creating = isCreating.value;
    const hasIssue = !!issue.value;
    const hasRollout = !!rollout.value;
    const isIssueClosed =
      issue.value?.status === IssueStatus.CANCELED ||
      issue.value?.status === IssueStatus.DONE;
    const allTasks = rollout.value?.stages.flatMap((s) => s.tasks) ?? [];
    const allDone =
      allTasks.length > 0 &&
      allTasks.every(
        (task) =>
          task.status === Task_Status.DONE ||
          task.status === Task_Status.SKIPPED
      );

    let review: PhaseStatus = "future";
    if (hasIssue) {
      if (issue.value?.status === IssueStatus.CANCELED) {
        review = "closed";
      } else {
        review = hasRollout || isIssueClosed ? "completed" : "active";
      }
    }

    return {
      changes: creating || (!hasIssue && !hasRollout) ? "active" : "completed",
      review,
      deploy: hasRollout ? (allDone ? "completed" : "active") : "future",
    };
  });

  // Badges

  const changesBadge = computed<PhaseBadge | undefined>(() => {
    if (statusMap.value.changes !== "active") return undefined;
    if (isCreating.value) return undefined;
    return { label: t("common.draft"), type: "default" };
  });

  const reviewBadge = computed<PhaseBadge | undefined>(() => {
    if (!issue.value) return undefined;
    if (issue.value.status === IssueStatus.CANCELED) {
      return { label: t("common.closed"), type: "default" };
    }
    // Show badge for both active and completed review phases
    if (statusMap.value.review === "future") return undefined;
    const map: Partial<
      Record<Issue_ApprovalStatus, { label: string; type: TagProps["type"] }>
    > = {
      [Issue_ApprovalStatus.APPROVED]: {
        label: t("issue.table.approved"),
        type: "success",
      },
      [Issue_ApprovalStatus.SKIPPED]: {
        label: t("common.skipped"),
        type: "default",
      },
      [Issue_ApprovalStatus.REJECTED]: {
        label: t("common.rejected"),
        type: "warning",
      },
      [Issue_ApprovalStatus.PENDING]: {
        label: t("common.under-review"),
        type: "info",
      },
    };
    // If review phase is completed but approval is still pending, it was bypassed
    if (
      statusMap.value.review === "completed" &&
      issue.value.approvalStatus === Issue_ApprovalStatus.PENDING
    ) {
      return { label: t("common.bypassed"), type: "default" };
    }
    return map[issue.value.approvalStatus];
  });

  const deployBadge = computed<PhaseBadge | undefined>(() => {
    if (statusMap.value.deploy !== "active" || !rollout.value) return undefined;
    const allTasks = rollout.value.stages.flatMap((s) => s.tasks);
    if (allTasks.some((task) => task.status === Task_Status.FAILED))
      return { label: t("common.failed"), type: "error" };
    if (allTasks.some((task) => task.status === Task_Status.RUNNING))
      return { label: t("common.in-progress"), type: "info" };
    return { label: t("common.not-started"), type: "default" };
  });

  // Line colors

  const lineClass = (from: PhaseType, to: PhaseType): string => {
    const f = statusMap.value[from];
    const s = statusMap.value[to];
    if (f === "closed" || s === "closed")
      return "border-l-2 border-dashed border-control-border";
    if (f === "completed" && s === "completed")
      return "border-l-2 border-success";
    if (f === "completed" && s === "active") return "border-l-2 border-success";
    if (f === "active") return "border-l-2 border-dashed border-accent";
    return "border-l-2 border-dashed border-control-border";
  };

  // Assembled array (indexed: 0=changes, 1=review, 2=deploy)

  const phases = computed<[PhaseConfig, PhaseConfig, PhaseConfig]>(() => [
    {
      status: statusMap.value.changes,
      badge: changesBadge.value,
      lineClass: lineClass("changes", "review"),
    },
    {
      status: statusMap.value.review,
      badge: reviewBadge.value,
      lineClass: lineClass("review", "deploy"),
    },
    {
      status: statusMap.value.deploy,
      badge: deployBadge.value,
      lineClass: "",
    },
  ]);

  return { phases };
};
