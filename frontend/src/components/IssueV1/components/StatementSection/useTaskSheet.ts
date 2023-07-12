import { computed } from "vue";
import { useSheetByName } from "@/store";
import { Task_Type } from "@/types/proto/v1/rollout_service";
import { sheetNameOfTaskV1 } from "@/utils";
import { useIssueContext } from "../../logic";

export const ESTABLISH_BASELINE_SQL =
  "/* Establish baseline using current schema */";

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
    if (selectedTask.value.type === Task_Type.DATABASE_SCHEMA_BASELINE) {
      return ESTABLISH_BASELINE_SQL;
    }

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
