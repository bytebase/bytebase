import { create } from "@bufbuild/protobuf";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
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

// Returns the raw statement bytes of a change-database spec's local sheet.
// setSheetStatement replaces `content` with a fresh Uint8Array on every edit,
// so the reference doubles as a cheap change signature: comparing references
// detects edits in O(1) without decoding the (possibly huge) blob. Both the
// parent staleness gate and the draft-check runner read it through here.
export const getSpecStatementContent = (
  spec: Plan_Spec
): Uint8Array | undefined => {
  if (spec.config.case !== "changeDatabaseConfig") return undefined;
  return getLocalSheetByName(spec.config.value.sheet).content;
};

// Byte-equality for statement content. The reference check is the O(1) fast
// path for the common "unchanged since checks ran" case; the byte fallback
// covers edit-then-revert, where setSheetStatement minted a fresh Uint8Array
// with identical bytes — the prior checks are still valid for that statement,
// so a reference mismatch alone must not hide them.
export const isSameStatementContent = (
  a: Uint8Array | undefined,
  b: Uint8Array | undefined
): boolean => {
  if (a === b) return true;
  if (!a || !b || a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false;
  }
  return true;
};
