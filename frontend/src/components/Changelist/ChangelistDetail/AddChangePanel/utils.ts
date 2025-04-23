import { reactive } from "vue";
import { useLocalSheetStore } from "@/store";
import { Changelist_Change as Change } from "@/types/proto/api/v1alpha/changelist_service";

export const emptyRawSQLChange = (project: string) => {
  return reactive(
    Change.fromPartial({
      sheet: `${project}/sheets/${useLocalSheetStore().nextUID()}`,
      source: "",
    })
  );
};
