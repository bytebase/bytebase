import { reactive } from "vue";
import { Sheet } from "@/types/proto/api/v1alpha/sheet_service";

const sheetsByName = reactive(new Map<string, Sheet>());

export const createEmptyLocalSheet = () => {
  return reactive(Sheet.fromJSON({}));
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
