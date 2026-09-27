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
