import { create } from "@bufbuild/protobuf";
import { reactive } from "vue";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";

const localSheetsByName = reactive(new Map<string, Sheet>());

export const createEmptyLocalSheet = () => {
  return reactive(create(SheetSchema, {}));
};

export const getLocalSheetByName = (name: string): Sheet => {
  const existing = localSheetsByName.get(name);
  if (existing) {
    return existing;
  }
  const sheet = reactive({
    ...createEmptyLocalSheet(),
    name,
  });
  localSheetsByName.set(name, sheet);
  return sheet;
};
