import { Sheet_Visibility } from "@/types/proto/v1/sheet_service";

interface SheetVCSPayload {
  fileName: string;
  filePath: string;
  size: number;
  author: string;
  lastCommitId: string;
  lastSyncTs: number;
}

// eslint-disable-next-line @typescript-eslint/ban-types
type SheetEmptyPayload = {};

export type SheetPayload = SheetVCSPayload | SheetEmptyPayload;

export type AccessOption = {
  label: string;
  description: string;
  value: Sheet_Visibility;
};
