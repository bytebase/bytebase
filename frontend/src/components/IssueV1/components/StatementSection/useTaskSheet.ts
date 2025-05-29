import { computedAsync } from "@vueuse/core";
import { computed, ref } from "vue";
import { getLocalSheetByName } from "@/components/Plan";
import { useSheetV1Store } from "@/store";
import {
  extractSheetUID,
  getSheetStatement,
  setSheetStatement,
  sheetNameOfTaskV1,
} from "@/utils";
import { useIssueContext } from "../../logic";

export const useTaskSheet = () => {
  const sheetStore = useSheetV1Store();
  const { selectedTask } = useIssueContext();

  const sheetName = computed(() => {
    return sheetNameOfTaskV1(selectedTask.value);
  });
  const isFetchingSheet = ref(false);
  const sheetReady = computed(() => {
    return !isFetchingSheet.value;
  });
  const sheet = computedAsync(
    async () => {
      const name = sheetName.value;
      const uid = extractSheetUID(name);
      if (uid.startsWith("-")) {
        return getLocalSheetByName(name);
      }
      // Use any view(basic or full) of sheets here to save data size.
      return await sheetStore.getOrFetchSheetByName(name);
    },
    undefined,
    {
      evaluating: isFetchingSheet,
    }
  );
  const sheetStatement = computed({
    get() {
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
