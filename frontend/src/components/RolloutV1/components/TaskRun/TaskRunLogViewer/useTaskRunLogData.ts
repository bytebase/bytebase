import { create } from "@bufbuild/protobuf";
import { computedAsync } from "@vueuse/core";
import type { MaybeRefOrGetter, Ref } from "vue";
import { computed, ref, toValue, watch } from "vue";
import {
  rolloutServiceClientConnect,
  sheetServiceClientConnect,
} from "@/connect";
import { useReleaseStore, useRolloutByName } from "@/store";
import {
  GetTaskRunLogRequestSchema,
  type TaskRunLogEntry,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import {
  extractRolloutNameFromTaskRunName,
  extractTaskNameFromTaskRunName,
  isReleaseBasedTask,
  releaseNameOfTaskV1,
  sheetNameOfTaskV1,
} from "@/utils";

export interface UseTaskRunLogDataReturn {
  entries: Ref<TaskRunLogEntry[]>;
  sheet: Ref<Sheet | undefined>;
  sheetsMap: Ref<Map<string, Sheet>>;
}

/**
 * Composable that fetches task run log entries and associated sheets.
 * Handles both regular tasks (single sheet) and release tasks (multiple sheets by version).
 */
export const useTaskRunLogData = (
  taskRunName: MaybeRefOrGetter<string | undefined>
): UseTaskRunLogDataReturn => {
  const releaseStore = useReleaseStore();

  // Sheet for non-release tasks
  const sheet = ref<Sheet | undefined>(undefined);
  // Map of version -> Sheet for release tasks
  const sheetsMap = ref<Map<string, Sheet>>(new Map());

  // Fetch task run log entries
  const taskRunLog = computedAsync(async () => {
    const name = toValue(taskRunName);
    if (!name) return undefined;
    const request = create(GetTaskRunLogRequestSchema, {
      parent: name,
    });
    return await rolloutServiceClientConnect.getTaskRunLog(request);
  }, undefined);

  const entries = computed(() => taskRunLog.value?.entries ?? []);

  // Extract rollout name from task run name to get the task
  const rolloutName = computed(() => {
    const name = toValue(taskRunName);
    if (!name) return "";
    return extractRolloutNameFromTaskRunName(name);
  });

  const { rollout } = useRolloutByName(rolloutName);

  // Find the task from the rollout
  const task = computed(() => {
    const name = toValue(taskRunName);
    if (!name || !rollout.value) return undefined;

    const taskName = extractTaskNameFromTaskRunName(name);
    if (!taskName) return undefined;

    for (const stage of rollout.value.stages) {
      const foundTask = stage.tasks.find((t) => t.name === taskName);
      if (foundTask) return foundTask;
    }
    return undefined;
  });

  // Track fetch version to handle race conditions
  let fetchVersion = 0;

  const fetchReleaseSheets = async (
    releaseName: string,
    version: number
  ): Promise<void> => {
    const release = await releaseStore.fetchReleaseByName(releaseName, true);
    if (version !== fetchVersion) return;

    const results = await Promise.all(
      release.files
        .filter((f) => f.sheet && f.version)
        .map(async (file) => {
          try {
            const s = await sheetServiceClientConnect.getSheet({
              name: file.sheet,
              raw: true,
            });
            return { version: file.version, sheet: s };
          } catch {
            return null;
          }
        })
    );
    if (version !== fetchVersion) return;

    const newMap = new Map<string, Sheet>();
    for (const r of results) {
      if (r?.sheet) newMap.set(r.version, r.sheet);
    }
    sheetsMap.value = newMap;
  };

  const fetchSingleSheet = async (
    sheetName: string,
    version: number
  ): Promise<void> => {
    try {
      const fetched = await sheetServiceClientConnect.getSheet({
        name: sheetName,
        raw: true,
      });
      if (version !== fetchVersion) return;
      sheet.value = fetched;
    } catch {
      if (version === fetchVersion) {
        sheet.value = undefined;
      }
    }
  };

  // Fetch sheet(s) based on task type
  watch(
    task,
    async (currentTask) => {
      const currentVersion = ++fetchVersion;

      if (!currentTask) {
        sheet.value = undefined;
        sheetsMap.value = new Map();
        return;
      }

      try {
        if (isReleaseBasedTask(currentTask)) {
          const releaseName = releaseNameOfTaskV1(currentTask);
          if (releaseName) {
            await fetchReleaseSheets(releaseName, currentVersion);
          }
        } else {
          const sheetName = sheetNameOfTaskV1(currentTask);
          if (sheetName) {
            await fetchSingleSheet(sheetName, currentVersion);
          }
        }
      } catch {
        // Ignore fetch errors
      }
    },
    { immediate: true }
  );

  return {
    entries,
    sheet,
    sheetsMap,
  };
};
