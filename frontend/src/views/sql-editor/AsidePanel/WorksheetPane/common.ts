import type { SQLEditorTab } from "@/types";
import type { Worksheet } from "@/types/proto/api/v1alpha/worksheet_service";
import { extractWorksheetUID } from "@/utils";
import type { SheetViewMode } from "../../Sheet";

export type GroupType = SheetViewMode | "draft";

export const keyForWorksheet = (worksheet: Worksheet) => {
  return `bb-worksheet-list-worksheet-${extractWorksheetUID(worksheet.name)}`;
};
export const keyForDraft = (tab: SQLEditorTab) => {
  return `bb-worksheet-list-draft-${tab.id}`;
};
