import { useSheetByName } from "@/store";
import { sheetNameOfTaskV1 } from "@/utils";
import { computed } from "vue";
import { useIssueContext } from "../../logic";

export const useTaskSheet = () => {
  const { isCreating, issue, selectedTask } = useIssueContext();

  const sheetName = computed(() => {
    if (isCreating.value) {
      return `${issue.value.project}/sheets/-1`;
    }
    return sheetNameOfTaskV1(selectedTask.value);
  });
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
