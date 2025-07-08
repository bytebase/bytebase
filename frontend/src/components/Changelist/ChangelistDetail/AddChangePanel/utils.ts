import { create } from "@bufbuild/protobuf";
import { reactive } from "vue";
import { useLocalSheetStore } from "@/store";
import { Changelist_ChangeSchema } from "@/types/proto-es/v1/changelist_service_pb";

export const emptyRawSQLChange = (project: string) => {
  return reactive(
    create(Changelist_ChangeSchema, {
      sheet: `${project}/sheets/${useLocalSheetStore().nextUID()}`,
      source: "",
    })
  );
};
