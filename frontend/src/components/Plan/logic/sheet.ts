import { reactive } from "vue";
import { Sheet } from "@/types/proto/v1/sheet_service";

const state = {
  uid: -101,
};

const nextUID = () => {
  return String(state.uid--);
};

const sheetsByName = reactive(new Map<string, Sheet>());

export const createEmptyLocalSheet = () => {
  return reactive(Sheet.fromPartial({}));
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
