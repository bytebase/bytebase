import type { ComputedRef, MaybeRefOrGetter } from "vue";
import { computed, onUnmounted, ref, toValue, watch } from "vue";
import { usePlanContext } from "@/components/Plan/logic/context";
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
  totalEntryCount: number;
  isTruncated: boolean;
}

// Limit entries to prevent performance issues with large logs
export const MAX_DISPLAY_ENTRIES = 100;

export interface UseTaskRunLogSummaryReturn {
  taskRunLog: ComputedRef<TaskRunLog | undefined>;
  summary: ComputedRef<TaskRunLogSummary>;
  isLoading: ComputedRef<boolean>;
}

/**
 * Composable for fetching and summarizing task run logs.
 * Provides affected rows count and log entries.
 * Data is cached in the store, so multiple calls with the same taskRun are efficient.
 * Listens to poller's "resource-refresh-completed" event to refetch logs when taskRuns are updated.
 */
export const useTaskRunLogSummary = (
  taskRun: MaybeRefOrGetter<TaskRun | undefined>,
  shouldFetch: MaybeRefOrGetter<boolean>
): UseTaskRunLogSummaryReturn => {
  const { events } = usePlanContext();
  const taskRunLogStore = useTaskRunLogStore();
  const isLoading = ref(false);

  const fetchLog = async (
    taskRunName: string,
    options?: { skipCache?: boolean }
  ) => {
    isLoading.value = true;
    try {
      await taskRunLogStore.fetchTaskRunLog(taskRunName, options);
    } finally {
      isLoading.value = false;
    }
  };

  // Initial fetch when taskRun changes
  watch(
    [() => toValue(taskRun)?.name, () => toValue(shouldFetch)],
    async ([taskRunName, fetch]) => {
      if (fetch && taskRunName) {
        await fetchLog(taskRunName);
      }
    },
    { immediate: true }
  );

  // Refetch logs when poller refreshes taskRuns (skip cache to get latest)
  const unsubscribe = events.on(
    "resource-refresh-completed",
    async ({ resources }) => {
      const taskRunName = toValue(taskRun)?.name;
      if (
        resources.includes("taskRuns") &&
        toValue(shouldFetch) &&
        taskRunName
      ) {
        await fetchLog(taskRunName, { skipCache: true });
      }
    }
  );

  onUnmounted(() => {
    unsubscribe();
  });

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
      totalEntryCount: 0,
      isTruncated: false,
    };

    if (!log?.entries?.length) return result;

    const entries = log.entries;
    result.totalEntryCount = entries.length;
    result.isTruncated = entries.length > MAX_DISPLAY_ENTRIES;

    for (const entry of entries) {
      if (entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE) {
        const affectedRows = entry.commandExecute?.response?.affectedRows;
        if (affectedRows !== undefined) {
          result.totalAffectedRows += affectedRows;
          result.hasAffectedRows = true;
        }
      }
    }

    // Get latest entries (limited for performance)
    const startIndex = Math.max(0, entries.length - MAX_DISPLAY_ENTRIES);
    result.latestEntries = entries.slice(startIndex);
    return result;
  });

  return {
    taskRunLog,
    summary,
    isLoading: computed(() => isLoading.value),
  };
};
