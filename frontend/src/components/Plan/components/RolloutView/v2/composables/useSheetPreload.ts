import { ref, watch } from "vue";
import { useSheetV1Store } from "@/store";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { sheetNameOfTaskV1 } from "@/utils";

/**
 * Prefetches sheets for visible tasks in parallel to improve performance
 */
export const useSheetPreload = (tasks: () => Task[]) => {
  const sheetStore = useSheetV1Store();
  const fetchedSheets = ref(new Set<string>());

  watch(
    tasks,
    (currentTasks) => {
      const uniqueSheetNames = new Set<string>();

      // Extract all unique sheet names from tasks
      for (const task of currentTasks) {
        const sheetName = sheetNameOfTaskV1(task);
        if (sheetName) {
          uniqueSheetNames.add(sheetName);
        }
      }

      // Only fetch sheets we haven't fetched yet
      const newSheets = Array.from(uniqueSheetNames).filter(
        (name) => !fetchedSheets.value.has(name)
      );

      if (newSheets.length > 0) {
        // Batch fetch new sheets in parallel
        Promise.all(
          newSheets.map((sheetName) =>
            sheetStore.getOrFetchSheetByName(sheetName, "BASIC")
          )
        );

        // Mark these sheets as fetched
        newSheets.forEach((name) => fetchedSheets.value.add(name));
      }
    },
    { immediate: true }
  );
};
