import { create } from "@bufbuild/protobuf";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";

const state = {
  uid: -101,
};

const localSheetsByName = new Map<string, Sheet>();

export const createEmptyLocalSheet = () => {
  return create(SheetSchema, {});
};

export const getNextLocalSheetUID = () => {
  return String(state.uid--);
};

export const getLocalSheetByName = (name: string): Sheet => {
  const existing = localSheetsByName.get(name);
  if (existing) {
    return existing;
  }
  const sheet = create(SheetSchema, {
    ...createEmptyLocalSheet(),
    name,
  });
  localSheetsByName.set(name, sheet);
  return sheet;
};

export const removeLocalSheet = (name: string): void => {
  localSheetsByName.delete(name);
};
