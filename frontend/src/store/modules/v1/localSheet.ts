import { create as createProto } from "@bufbuild/protobuf";
import { defineStore } from "pinia";
import { reactive } from "vue";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import { extractProjectResourceName } from "@/utils";
import { useSheetV1Store } from "./sheet";

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

  const createLocalSheet = (name: string, defaults: Partial<Sheet> = {}) => {
    const sheet = createProto(SheetSchema, {
      name,
      content: defaults.content || new Uint8Array(),
      contentSize: defaults.contentSize || BigInt(0),
    });
    return reactive(sheet);
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

  const saveLocalSheetToRemote = async (localSheet: Sheet) => {
    const project = extractProjectResourceName(localSheet.name);
    const remoteSheet = await useSheetV1Store().createSheet(
      `projects/${project}`,
      localSheet
    );
    return remoteSheet;
  };

  return {
    nextUID,
    createLocalSheet,
    getOrCreateSheetByName,
    saveLocalSheetToRemote,
  };
});
