import { computed, ref, watch } from "vue";
import { useSheetV1Store } from "@/store";
import {
  extractSheetUID,
  getSheetStatement,
  setSheetStatement,
  sheetNameOfTaskV1,
} from "@/utils";
import { getLocalSheetByName, useIssueContext } from "../../logic";
import { ESTABLISH_BASELINE_SQL } from "@/types";
import { Task_Type } from "@/types/proto/v1/rollout_service";

export const useTaskSheet = () => {
  const sheetStore = useSheetV1Store();
  const { selectedTask } = useIssueContext();

  const sheetName = computed(() => {
    return sheetNameOfTaskV1(selectedTask.value);
  });
  const sheetReady = ref(false);
  const sheet = computed(() => {
    const name = sheetName.value;
    const uid = extractSheetUID(name);
    if (uid.startsWith("-")) {
      return getLocalSheetByName(name);
    }
    return sheetStore.getSheetByName(name);
  });
  watch(
    sheetName,
    (sheetName) => {
      const uid = extractSheetUID(sheetName);
      if (uid.startsWith("-")) {
        sheetReady.value = true;
      } else {
        sheetReady.value = false;
        sheetStore.getOrFetchSheetByName(sheetName).finally(() => {
          sheetReady.value = true;
        });
      }
    },
    { immediate: true }
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
