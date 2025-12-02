import type { ComputedRef } from "vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status, Task_Type } from "@/types/proto-es/v1/rollout_service_pb";

const MAX_ERROR_PREVIEW_LENGTH = 100;

export interface UseTaskDisplayReturn {
  taskTypeDisplay: ComputedRef<string>;
  executorEmail: ComputedRef<string>;
  resultSummary: ComputedRef<string>;
  waitingMessage: ComputedRef<string>;
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
      case Task_Type.DATABASE_SDL:
        return t("task.type.database-sdl");
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

  const resultSummary = computed(() => {
    const taskRun = latestTaskRun();
    if (!taskRun) return "";

    if (taskRun.schemaVersion) {
      return t("task.result.schema-version", {
        version: taskRun.schemaVersion,
      });
    }
    if (taskRun.exportArchiveStatus) {
      return t("task.result.export-archive-ready");
    }
    return "";
  });

  const waitingMessage = computed(() => {
    const taskRun = latestTaskRun();
    if (!taskRun || task().status !== Task_Status.PENDING) return "";

    const cause = taskRun.schedulerInfo?.waitingCause?.cause;
    if (!cause) return "";

    switch (cause.case) {
      case "connectionLimit":
        return t("task.waiting.connection-limit");
      case "parallelTasksLimit":
        return t("task.waiting.parallel-tasks-limit");
      case "task":
        return t("task.waiting.blocking-task");
      default:
        return "";
    }
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
    const currentTask = task();

    if (currentTask.status === Task_Status.FAILED) {
      const detail = latestTaskRun()?.detail || "";
      const firstLine = detail.split("\n")[0];
      return firstLine.length > MAX_ERROR_PREVIEW_LENGTH
        ? `${firstLine.substring(0, MAX_ERROR_PREVIEW_LENGTH)}...`
        : firstLine;
    }

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
    resultSummary,
    waitingMessage,
    collapsedContextInfo,
    collapsedStatusText,
  };
};
