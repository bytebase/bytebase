import type { SQLEditorTab } from "@/types";
import type { Worksheet } from "@/types/proto-es/v1/worksheet_service_pb";
import { extractWorksheetUID } from "@/utils";
import type { SheetViewMode } from "../../Sheet";

export type GroupType = SheetViewMode | "draft";

export const keyForWorksheet = (worksheet: Worksheet) => {
  return `bb-worksheet-list-worksheet-${extractWorksheetUID(worksheet.name)}`;
};
export const keyForDraft = (tab: SQLEditorTab) => {
  return `bb-worksheet-list-draft-${tab.id}`;
};
