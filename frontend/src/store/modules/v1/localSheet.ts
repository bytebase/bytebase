import { defineStore } from "pinia";
import { reactive } from "vue";
import {
  Sheet,
  Sheet_Source,
  Sheet_Type,
  Sheet_Visibility,
  DeepPartial,
} from "@/types/proto/v1/sheet_service";

const state = reactive({
  uid: -101,
});

/**
 * The `local_sheet` store is used for creating and editing local sheets pending
 * create. Once used they should be saved and organized by the `sheet` store.
 */
export const useLocalSheetStore = defineStore("local_sheet", () => {
  const sheetsByName = reactive(new Map<string, Sheet>());

  const nextUID = () => {
    return state.uid--;
  };

  const createLocalSheet = (
    name: string,
    defaults: DeepPartial<Sheet> = {}
  ) => {
    return reactive(
      Sheet.fromPartial({
        name,
        source: Sheet_Source.SOURCE_BYTEBASE_ARTIFACT,
        visibility: Sheet_Visibility.VISIBILITY_PROJECT,
        type: Sheet_Type.TYPE_SQL,
        ...defaults,
      })
    );
  };

  const getOrCreateSheetByName = (name: string) => {
    const existed = sheetsByName.get(name);
    if (existed) {
      return existed;
    }
    const sheet = createLocalSheet(name);
    sheetsByName.set(name, sheet);
    return sheet;
  };

  return {
    nextUID,
    createLocalSheet,
    getOrCreateSheetByName,
  };
});
