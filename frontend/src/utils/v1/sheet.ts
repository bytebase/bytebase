import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import type { Worksheet } from "@/types/proto-es/v1/worksheet_service_pb";

export const extractSheetUID = (name: string) => {
  const pattern = /(?:^|\/)sheets\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "-1";
};

export const setSheetStatement = (
  sheet: Sheet | Worksheet,
  statement: string
) => {
  sheet.content = new TextEncoder().encode(statement);
  sheet.contentSize = BigInt(new TextEncoder().encode(statement).length);
};

export const getSheetStatement = (sheet: Sheet | Worksheet) => {
  return new TextDecoder().decode(sheet.content);
};

// Whether the sheet carries its full content rather than a truncated preview
// (fetches without `raw` return at most a size-capped prefix). `content` is
// already the encoded bytes, so this is an O(1) size check.
export const isSheetContentComplete = (sheet: Sheet | Worksheet): boolean =>
  BigInt(sheet.content.byteLength) >= sheet.contentSize;
