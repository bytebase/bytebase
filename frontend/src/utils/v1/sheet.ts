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

export const getSheetStatement = (sheet: Sheet | SavedQuery) => {
  return new TextDecoder().decode(sheet.content);
};

// Whether the sheet carries its full content rather than a truncated preview
// (fetches without `raw` return at most a size-capped prefix). `content` is
// already the encoded bytes, so this is an O(1) size check.
export const isSheetContentComplete = (sheet: Sheet | SavedQuery): boolean =>
  BigInt(sheet.content.byteLength) >= sheet.contentSize;
