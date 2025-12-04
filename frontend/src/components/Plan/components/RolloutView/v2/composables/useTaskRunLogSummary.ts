import type { ComputedRef, MaybeRefOrGetter } from "vue";
import { computed, ref, toValue, watch } from "vue";
import { useTaskRunLogStore } from "@/store/modules/v1/taskRunLog";
import type {
  TaskRun,
  TaskRunLog,
  TaskRunLogEntry,
} from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRunLogEntry_Type } from "@/types/proto-es/v1/rollout_service_pb";

export interface TaskRunLogSummary {
  totalAffectedRows: bigint;
  hasAffectedRows: boolean;
  latestEntries: TaskRunLogEntry[];
}

export interface UseTaskRunLogSummaryReturn {
  taskRunLog: ComputedRef<TaskRunLog | undefined>;
  summary: ComputedRef<TaskRunLogSummary>;
  isLoading: ComputedRef<boolean>;
}

/**
 * Composable for fetching and summarizing task run logs.
 * Provides affected rows count and log entries.
 * Data is cached in the store, so multiple calls with the same taskRun are efficient.
 */
export const useTaskRunLogSummary = (
  taskRun: MaybeRefOrGetter<TaskRun | undefined>,
  shouldFetch: MaybeRefOrGetter<boolean>
): UseTaskRunLogSummaryReturn => {
  const taskRunLogStore = useTaskRunLogStore();
  const isLoading = ref(false);

  watch(
    [() => toValue(taskRun)?.name, () => toValue(shouldFetch)],
    async ([taskRunName, fetch]) => {
      if (fetch && taskRunName) {
        isLoading.value = true;
        try {
          await taskRunLogStore.fetchTaskRunLog(taskRunName);
        } finally {
          isLoading.value = false;
        }
      }
    },
    { immediate: true }
  );

  const taskRunLog = computed(() => {
    const run = toValue(taskRun);
    if (!run?.name) return undefined;
    return taskRunLogStore.getTaskRunLog(run.name);
  });

  const summary = computed((): TaskRunLogSummary => {
    const log = taskRunLog.value;
    const result: TaskRunLogSummary = {
      totalAffectedRows: BigInt(0),
      hasAffectedRows: false,
      latestEntries: [],
    };

    if (!log?.entries?.length) return result;

    for (const entry of log.entries) {
      if (entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE) {
        const affectedRows = entry.commandExecute?.response?.affectedRows;
        if (affectedRows !== undefined) {
          result.totalAffectedRows += affectedRows;
          result.hasAffectedRows = true;
        }
      }
    }

    result.latestEntries = [...log.entries].reverse();
    return result;
  });

  return {
    taskRunLog,
    summary,
    isLoading: computed(() => isLoading.value),
  };
};
