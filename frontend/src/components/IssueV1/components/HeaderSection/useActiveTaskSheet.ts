import { computed, ref, watch } from "vue";
import { useIssueContext } from "@/components/IssueV1/logic";
import { useSheetV1Store } from "@/store";
import { extractSheetUID, sheetNameOfTaskV1 } from "@/utils";

export const useActiveTaskSheet = () => {
  const sheetStore = useSheetV1Store();
  const { isCreating, activeTask } = useIssueContext();

  const sheetName = computed(() => {
    return sheetNameOfTaskV1(activeTask.value);
  });
  const sheetReady = ref(false);
  const sheet = computed(() => {
    if (isCreating.value) {
      return undefined;
    }
    const name = sheetName.value;
    return sheetStore.getSheetByName(name);
  });
  watch(
    [sheetName, isCreating],
    ([sheetName, isCreating]) => {
      if (isCreating) {
        sheetReady.value = true;
        return;
      }
      const uid = extractSheetUID(sheetName);
      if (!uid) return;
      sheetReady.value = false;
      sheetStore.getOrFetchSheetByName(sheetName).finally(() => {
        sheetReady.value = true;
      });
    },
    { immediate: true }
  );

  return {
    sheet,
    sheetName,
    sheetReady,
  };
};
