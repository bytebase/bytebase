import { ref, watch } from "vue";
import { useSheetV1Store } from "@/store";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { sheetNameOfTaskV1 } from "@/utils";

/**
 * Prefetches sheets for the given tasks in parallel.
 */
export const useSheetPreload = (tasks: () => Task[]) => {
  const sheetStore = useSheetV1Store();
  const fetchedSheets = ref(new Set<string>());

  watch(
    tasks,
    (currentTasks) => {
      // Extract unique sheet names
      const sheetNames = new Set<string>();
      for (const task of currentTasks) {
        const sheetName = sheetNameOfTaskV1(task);
        if (sheetName) {
          sheetNames.add(sheetName);
        }
      }

      // Filter to only unfetched sheets
      const newSheets = Array.from(sheetNames).filter(
        (name) => !fetchedSheets.value.has(name)
      );

      if (newSheets.length === 0) return;

      // Batch fetch in parallel
      Promise.all(
        newSheets.map((name) => sheetStore.getOrFetchSheetByName(name, "BASIC"))
      );

      // Mark as fetched
      for (const name of newSheets) {
        fetchedSheets.value.add(name);
      }
    },
    { immediate: true }
  );
};
