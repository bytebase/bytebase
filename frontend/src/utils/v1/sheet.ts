import type { SavedQuery } from "@/types/proto-es/v1/saved_query_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";

export const extractSheetUID = (name: string) => {
  const pattern = /(?:^|\/)sheets\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "-1";
};

export const setSheetStatement = (
  sheet: Sheet | SavedQuery,
  statement: string
) => {
  sheet.content = new TextEncoder().encode(statement);
  sheet.contentSize = BigInt(new TextEncoder().encode(statement).length);
};

// Decoded once per content buffer: sheets are immutable and several
// consumers read the same one, and a 2 MB decode is not free.
const decodedStatements = new WeakMap<Uint8Array, string>();
export const getSheetStatement = (sheet: Sheet | SavedQuery) => {
  let statement = decodedStatements.get(sheet.content);
  if (statement === undefined) {
    statement = new TextDecoder().decode(sheet.content);
    decodedStatements.set(sheet.content, statement);
  }
  return statement;
};

// Whether the sheet carries its full content rather than a truncated preview
// (fetches without `raw` return at most a size-capped prefix). `content` is
// already the encoded bytes, so this is an O(1) size check.
export const isSheetContentComplete = (sheet: Sheet | SavedQuery): boolean =>
  BigInt(sheet.content.byteLength) >= sheet.contentSize;

// GetSheet(raw=false) returns at most this many characters, not UTF-8 bytes.
export const SHEET_PREVIEW_CHARACTER_LIMIT = 2 * 1024 * 1024;

export const exceedsSheetPreviewLimit = (
  statement: string,
  limit = SHEET_PREVIEW_CHARACTER_LIMIT
): boolean => {
  if (statement.length <= limit) return false;
  // Without surrogate pairs, UTF-16 length already counts characters.
  if (!/[\uD800-\uDBFF]/.test(statement)) return true;
  let characters = 0;
  for (const _ of statement) {
    if (++characters > limit) return true;
  }
  return false;
};

// Sheet content is immutable; share the character count across review threads.
const cappedPreviews = new WeakMap<Uint8Array, boolean>();
export const isCappedSheetPreview = (sheet: Sheet): boolean => {
  if (isSheetContentComplete(sheet)) return false;
  let capped = cappedPreviews.get(sheet.content);
  if (capped === undefined) {
    capped = exceedsSheetPreviewLimit(
      getSheetStatement(sheet),
      SHEET_PREVIEW_CHARACTER_LIMIT - 1
    );
    cappedPreviews.set(sheet.content, capped);
  }
  return capped;
};
