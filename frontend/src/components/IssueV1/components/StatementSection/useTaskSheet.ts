import { useSheetByName } from "@/store";
import { sheetNameOfTaskV1 } from "@/utils";
import { computed } from "vue";
import { useIssueContext } from "../../logic";

export const useTaskSheet = () => {
  const { selectedTask } = useIssueContext();

  const sheetName = computed(() => sheetNameOfTaskV1(selectedTask.value));
  const { sheet, ready: sheetReady } = useSheetByName(sheetName);
  const sheetStatement = computed(() => {
    if (!sheetReady.value || !sheet.value) return "";
    return new TextDecoder().decode(sheet.value.content);
  });

  return {
    sheet,
    sheetName,
    sheetReady,
    sheetStatement,
  };
};
