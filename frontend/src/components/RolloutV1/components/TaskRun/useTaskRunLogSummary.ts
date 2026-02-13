import type { ComputedRef, MaybeRefOrGetter } from "vue";
import { computed, onUnmounted, ref, toValue, watch } from "vue";
import { usePlanContext } from "@/components/Plan/logic";
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
  entries: TaskRunLogEntry[];
}

export interface UseTaskRunLogSummaryReturn {
  taskRunLog: ComputedRef<TaskRunLog | undefined>;
  summary: ComputedRef<TaskRunLogSummary>;
  isLoading: ComputedRef<boolean>;
}

const EMPTY_SUMMARY: TaskRunLogSummary = {
  totalAffectedRows: BigInt(0),
  hasAffectedRows: false,
  entries: [],
};

const calculateSummary = (log: TaskRunLog | undefined): TaskRunLogSummary => {
  if (!log?.entries?.length) return EMPTY_SUMMARY;

  let totalAffectedRows = BigInt(0);
  let hasAffectedRows = false;

  for (const entry of log.entries) {
    if (entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE) {
      const affectedRows = entry.commandExecute?.response?.affectedRows;
      if (affectedRows !== undefined) {
        totalAffectedRows += affectedRows;
        hasAffectedRows = true;
      }
    }
  }

  return {
    totalAffectedRows,
    hasAffectedRows,
    entries: log.entries,
  };
};

/**
 * Composable for fetching and summarizing task run logs.
 *
 * Features:
 * - Fetches logs when taskRun changes or shouldFetch becomes true
 * - Auto-refetches when taskRun status changes (to get final results)
 * - Auto-refetches when poller refreshes taskRuns
 * - Calculates total affected rows from command executions
 *
 * @param taskRun - The task run to fetch logs for
 * @param shouldFetch - Whether to actively fetch (e.g., only when UI is visible)
 */
export const useTaskRunLogSummary = (
  taskRun: MaybeRefOrGetter<TaskRun | undefined>,
  shouldFetch: MaybeRefOrGetter<boolean>
): UseTaskRunLogSummaryReturn => {
  const { events } = usePlanContext();
  const taskRunLogStore = useTaskRunLogStore();
  const isLoading = ref(false);

  // Helper to get current values
  const getTaskRunName = () => toValue(taskRun)?.name;
  const canFetch = () => toValue(shouldFetch) && !!getTaskRunName();

  const fetchLog = async (skipCache = false) => {
    const taskRunName = getTaskRunName();
    if (!taskRunName) return;

    isLoading.value = true;
    try {
      await taskRunLogStore.fetchTaskRunLog(taskRunName, { skipCache });
    } finally {
      isLoading.value = false;
    }
  };

  // Fetch when taskRun name changes or shouldFetch becomes true
  watch(
    [getTaskRunName, () => toValue(shouldFetch)],
    async ([taskRunName, fetch], [, oldFetch]) => {
      if (!fetch || !taskRunName) return;
      // Skip cache when transitioning to fetch mode (e.g., item expanded)
      const skipCache = !oldFetch && fetch;
      await fetchLog(skipCache);
    },
    { immediate: true }
  );

  // Refetch when taskRun status changes to get final results
  watch(
    () => toValue(taskRun)?.status,
    async (newStatus, oldStatus) => {
      if (newStatus !== oldStatus && canFetch()) {
        await fetchLog(true);
      }
    }
  );

  // Refetch when poller refreshes taskRuns
  const unsubscribe = events.on(
    "resource-refresh-completed",
    async ({ resources }) => {
      if (resources.includes("taskRuns") && canFetch()) {
        await fetchLog(true);
      }
    }
  );

  onUnmounted(unsubscribe);

  const taskRunLog = computed(() => {
    const name = getTaskRunName();
    return name ? taskRunLogStore.getTaskRunLog(name) : undefined;
  });

  const summary = computed(() => calculateSummary(taskRunLog.value));

  return {
    taskRunLog,
    summary,
    isLoading: computed(() => isLoading.value),
  };
};
