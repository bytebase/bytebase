import { type ComputedRef, computed, ref, watch } from "vue";
import { getLocalSheetByName } from "@/components/Plan";
import { useSheetV1Store } from "@/store";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import {
  extractSheetUID,
  getSheetStatement,
  setSheetStatement,
  sheetNameOfSpec,
} from "@/utils";

export const useSpecSheet = (spec: ComputedRef<Plan_Spec>) => {
  const sheetStore = useSheetV1Store();

  const isFetchingSheet = ref(false);
  const sheet = ref<Sheet | undefined>(undefined);
  const sheetName = computed(() => sheetNameOfSpec(spec.value));
  const sheetReady = computed(() => {
    return !isFetchingSheet.value;
  });

  const fetchSheet = async () => {
    const name = sheetName.value;
    const uid = extractSheetUID(name);
    isFetchingSheet.value = true;
    try {
      if (uid.startsWith("-")) {
        sheet.value = getLocalSheetByName(name);
      } else {
        sheet.value = await sheetStore.getOrFetchSheetByName(name);
      }
    } finally {
      isFetchingSheet.value = false;
    }
  };

  watch(sheetName, fetchSheet, { immediate: true });
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

  const updateSheetStatement = (statement: string) => {
    sheetStatement.value = statement;
  };

  return {
    sheet,
    sheetName,
    sheetReady,
    sheetStatement,
    updateSheetStatement,
  };
};
