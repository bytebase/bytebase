import { v1 as uuidv1 } from "uuid";
import type { EditStatus } from "../types";

// Inline-edit affordance for table-grid cells. The input carries a subtle
// resting border so users can immediately tell the cell is an editable field
// (a hover-only border was too hard to discover). The border darkens on hover
// and turns accent on focus. `text-sm` overrides the smaller font from the
// `size="xs"` input variant so inline-edit cells match the table's default
// plain-text cells (also text-sm), and `h-7 px-3` overrides the cramped
// `h-6 px-2` from that variant: roomier horizontal padding but a shorter height
// so rows stay compact.
export const INLINE_EDIT_INPUT_CLASS =
  "h-7 border border-control-border/50 bg-transparent px-3 text-sm shadow-none enabled:hover:border-control-border focus-visible:border-accent focus-visible:ring-1";

interface ObjectWithHiddenProps {
  __uuid?: string;
  __status_before_drop?: EditStatus;
}

export const markUUID = (obj: object): string => {
  const target = obj as ObjectWithHiddenProps;
  if (!target.__uuid) {
    Object.defineProperty(obj, "__uuid", {
      enumerable: false,
      writable: false,
      configurable: false,
      value: uuidv1(),
    });
  }
  return target.__uuid as string;
};

export const markEditStatusBeforeDrop = (obj: object, status: EditStatus) => {
  const target = obj as ObjectWithHiddenProps;
  Object.defineProperty(target, "__status_before_drop", {
    enumerable: false,
    writable: true,
    configurable: true,
    value: status,
  });
};

export const getEditStatusBeforeDrop = (
  obj: object
): EditStatus | undefined => {
  return (obj as ObjectWithHiddenProps).__status_before_drop;
};
