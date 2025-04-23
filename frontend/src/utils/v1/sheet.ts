import type { Sheet } from "@/types/proto/api/v1alpha/sheet_service";
import type { Worksheet } from "@/types/proto/api/v1alpha/worksheet_service";
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

export const extractSheetCommandByIndex = (sheet: Sheet, index: number) => {
  const commands = sheet.payload?.commands;
  if (!commands) return undefined;
  const command = commands[index];
  if (!command) return undefined;
  const subarray = sheet.content.subarray(command.start, command.end);
  return new TextDecoder().decode(subarray);
};
