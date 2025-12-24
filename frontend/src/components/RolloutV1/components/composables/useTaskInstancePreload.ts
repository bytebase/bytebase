import { ref, watch } from "vue";
import { useDatabaseV1Store } from "@/store";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";

/**
 * Preloads databases (and their instances) referenced by tasks
 * This ensures database and engine icons display correctly
 */
export const useTaskInstancePreload = (stages: () => Stage[]) => {
  const databaseStore = useDatabaseV1Store();
  const fetchedDatabases = ref(new Set<string>());

  watch(
    stages,
    async (currentStages) => {
      const uniqueDatabaseNames = new Set<string>();

      // Extract all unique database names from task targets
      for (const stage of currentStages) {
        for (const task of stage.tasks) {
          if (task.target) {
            uniqueDatabaseNames.add(task.target);
          }
        }
      }

      // Only fetch databases we haven't fetched yet
      const newDatabases = Array.from(uniqueDatabaseNames).filter(
        (name) => !fetchedDatabases.value.has(name)
      );

      if (newDatabases.length > 0) {
        // Batch fetch databases (which also fetches their instances)
        try {
          await databaseStore.batchGetOrFetchDatabases(newDatabases);
          // Mark these databases as fetched
          newDatabases.forEach((name) => fetchedDatabases.value.add(name));
        } catch {
          // Ignore errors - this is just for pre-loading data
        }
      }
    },
    { immediate: true }
  );
};
