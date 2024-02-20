import { computedAsync } from "@vueuse/core";
import { computed, ref } from "vue";
import { useSheetV1Store } from "@/store";
import { ESTABLISH_BASELINE_SQL } from "@/types";
import { Task_Type } from "@/types/proto/v1/rollout_service";
import {
  extractSheetUID,
  getSheetStatement,
  setSheetStatement,
  sheetNameOfTaskV1,
} from "@/utils";
import { getLocalSheetByName, useIssueContext } from "../../logic";

export const useTaskSheet = () => {
  const sheetStore = useSheetV1Store();
  const { selectedTask } = useIssueContext();

  const sheetName = computed(() => {
    return sheetNameOfTaskV1(selectedTask.value);
  });
  const sheetReady = ref(false);
  const sheet = computedAsync(
    async () => {
      const name = sheetName.value;
      const uid = extractSheetUID(name);
      if (uid.startsWith("-")) {
        return getLocalSheetByName(name);
      }

      // Use any (basic or full) view of sheets here to save data size
      return sheetStore.getOrFetchSheetByName(name);
    },
    undefined,
    {
      evaluating: sheetReady,
    }
  );
  const sheetStatement = computed({
    get() {
      if (selectedTask.value.type === Task_Type.DATABASE_SCHEMA_BASELINE) {
        return ESTABLISH_BASELINE_SQL;
      }

      if (!sheetReady.value || !sheet.value) return "";
      return getSheetStatement(sheet.value);
    },
    set(statement) {
      if (!sheetReady.value || !sheet.value) return;
      setSheetStatement(sheet.value, statement);
    },
  });

  return {
    sheet,
    sheetName,
    sheetReady,
    sheetStatement,
  };
};
