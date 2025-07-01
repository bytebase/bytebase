import { reactive } from "vue";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { create } from "@bufbuild/protobuf";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";

const state = {
  uid: -101,
};

const nextUID = () => {
  return String(state.uid--);
};

const sheetsByName = reactive(new Map<string, Sheet>());

export const createEmptyLocalSheet = () => {
  return reactive(create(SheetSchema, {}));
};

export const getNextLocalSheetUID = () => {
  return nextUID();
};

export const getLocalSheetByName = (name: string): Sheet => {
  const existed = sheetsByName.get(name);
  if (existed) {
    return existed;
  }
  const sheet = reactive({
    ...createEmptyLocalSheet(),
    name,
  });
  sheetsByName.set(name, sheet);
  return sheet;
};
