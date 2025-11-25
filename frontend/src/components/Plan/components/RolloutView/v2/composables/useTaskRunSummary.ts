import type { ComputedRef } from "vue";
import { computed, ref, watch } from "vue";
import { useTaskRunLogStore } from "@/store/modules/v1/taskRunLog";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRunLogEntry_Type } from "@/types/proto-es/v1/rollout_service_pb";

export interface UseTaskRunSummaryReturn {
  totalAffectedRows: ComputedRef<bigint | undefined>;
  isLoadingLog: ComputedRef<boolean>;
}

export const useTaskRunSummary = (
  latestTaskRun: () => TaskRun | undefined,
  isExpanded: () => boolean
): UseTaskRunSummaryReturn => {
  const taskRunLogStore = useTaskRunLogStore();
  const isLoadingLog = ref(false);

  // Fetch log when expanded and has task run
  watch(
    [() => latestTaskRun()?.name, isExpanded],
    async ([taskRunName, expanded]) => {
      if (expanded && taskRunName) {
        isLoadingLog.value = true;
        try {
          await taskRunLogStore.fetchTaskRunLog(taskRunName);
        } finally {
          isLoadingLog.value = false;
        }
      }
    },
    { immediate: true }
  );

  const taskRunLog = computed(() => {
    const taskRun = latestTaskRun();
    if (!taskRun?.name) return undefined;
    return taskRunLogStore.getTaskRunLog(taskRun.name);
  });

  const totalAffectedRows = computed((): bigint | undefined => {
    const log = taskRunLog.value;
    if (!log?.entries?.length) return undefined;

    let total = BigInt(0);
    let hasAffectedRows = false;

    for (const entry of log.entries) {
      if (entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE) {
        const response = entry.commandExecute?.response;
        if (response?.affectedRows !== undefined) {
          total += response.affectedRows;
          hasAffectedRows = true;
        }
      }
    }

    return hasAffectedRows ? total : undefined;
  });

  return {
    totalAffectedRows,
    isLoadingLog: computed(() => isLoadingLog.value),
  };
};
