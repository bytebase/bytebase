import { Sheet } from "@/types/proto/v1/sheet_service";
import { Worksheet } from "@/types/proto/v1/worksheet_service";
import { getStatementSize } from "@/utils";

export const extractSheetUID = (name: string) => {
  const pattern = /(?:^|\/)sheets\/([^/]+)(?:$|\/)/;
  const matches = name.match(pattern);
  return matches?.[1] ?? "-1";
};

export const isLocalSheet = (name: string) => {
  return extractSheetUID(name).startsWith("-");
};

export const setSheetStatement = (
  sheet: Sheet | Worksheet,
  statement: string
) => {
  sheet.content = new TextEncoder().encode(statement);
  sheet.contentSize = getStatementSize(statement);
};

export const getSheetStatement = (sheet: Sheet | Worksheet) => {
  return new TextDecoder().decode(sheet.content);
};
