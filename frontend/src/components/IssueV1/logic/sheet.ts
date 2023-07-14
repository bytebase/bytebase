import { reactive } from "vue";
import {
  Sheet,
  Sheet_Source,
  Sheet_Type,
  Sheet_Visibility,
} from "@/types/proto/v1/sheet_service";

const sheetsByName = reactive(new Map<string, Sheet>());

export const getLocalSheetByName = (name: string) => {
  const existed = sheetsByName.get(name);
  if (existed) {
    return existed;
  }
  const sheet = reactive(
    Sheet.fromJSON({
      name: name,
      source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
      visibility: Sheet_Visibility.VISIBILITY_PROJECT,
      type: Sheet_Type.TYPE_SQL,
    })
  );
  sheetsByName.set(name, sheet);
  return sheet;
};
