import { reactive } from "vue";
import { useLocalSheetStore } from "@/store";
import { Changelist_Change as Change } from "@/types/proto-es/v1/changelist_service_pb";

export const emptyRawSQLChange = (project: string) => {
  return reactive(
    Change.fromPartial({
      sheet: `${project}/sheets/${useLocalSheetStore().nextUID()}`,
      source: "",
    })
  );
};
