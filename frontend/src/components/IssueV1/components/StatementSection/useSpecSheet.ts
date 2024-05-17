import { computedAsync } from "@vueuse/core";
import { computed, ref } from "vue";
import { useSheetV1Store } from "@/store";
import {
  extractSheetUID,
  getSheetStatement,
  setSheetStatement,
  sheetNameOfSpec,
} from "@/utils";
import { getLocalSheetByName, useIssueContext } from "../../logic";

export const useSpecSheet = () => {
  const sheetStore = useSheetV1Store();
  const { selectedSpec } = useIssueContext();

  const sheetName = computed(() => {
    return sheetNameOfSpec(selectedSpec.value);
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

      return sheetStore.getOrFetchSheetByName(name);
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
