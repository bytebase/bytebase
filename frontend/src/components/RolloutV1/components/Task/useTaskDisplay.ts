import type { ComputedRef } from "vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status, Task_Type } from "@/types/proto-es/v1/rollout_service_pb";

export interface UseTaskDisplayReturn {
  taskTypeDisplay: ComputedRef<string>;
  executorEmail: ComputedRef<string>;
  collapsedContextInfo: ComputedRef<string>;
  collapsedStatusText: ComputedRef<string>;
}

/**
 * Composable for task display formatting.
 * Handles task type labels, executor info, and collapsed view display.
 */
export const useTaskDisplay = (
  task: () => Task,
  latestTaskRun: () => TaskRun | undefined,
  timingDisplay: () => string,
  affectedRowsDisplay: () => string
): UseTaskDisplayReturn => {
  const { t } = useI18n();

  const taskTypeDisplay = computed(() => {
    switch (task().type) {
      case Task_Type.DATABASE_CREATE:
        return t("task.type.database-create");
      case Task_Type.DATABASE_MIGRATE:
        return t("task.type.migrate");
      case Task_Type.DATABASE_EXPORT:
        return t("task.type.database-export");
      case Task_Type.GENERAL:
        return t("task.type.general");
      default:
        return "";
    }
  });

  const executorEmail = computed(() => {
    const creator = latestTaskRun()?.creator || "";
    const match = creator.match(/users\/([^/]+)/);
    return match?.[1] || "";
  });

  // Collapsed view: contextual info (duration, affected rows for completed tasks)
  const collapsedContextInfo = computed(() => {
    if (task().status !== Task_Status.DONE) return "";

    const parts: string[] = [];
    if (timingDisplay()) parts.push(timingDisplay());
    if (affectedRowsDisplay()) parts.push(affectedRowsDisplay());
    return parts.join(" · ");
  });

  // Collapsed view: error or skip reason
  const collapsedStatusText = computed(() => {
    const detail = latestTaskRun()?.detail || "";
    if (detail) {
      return detail;
    }

    const currentTask = task();
    if (
      currentTask.status === Task_Status.SKIPPED &&
      currentTask.skippedReason
    ) {
      return currentTask.skippedReason;
    }

    return "";
  });

  return {
    taskTypeDisplay,
    executorEmail,
    collapsedContextInfo,
    collapsedStatusText,
  };
};
