import { Sheet_Visibility } from "@/types/proto/v1/sheet_service";
import { IssueId } from ".";

interface SheetVCSPayload {
  fileName: string;
  filePath: string;
  size: number;
  author: string;
  lastCommitId: string;
  lastSyncTs: number;
}

/**
 * Mark a link from sheets to issues if a sheet is created in an issue via
 * "Upload SQL"
 */
export type SheetIssueBacktracePayload = {
  type: "bb.sheet.issue-backtrace";
  issueId: IssueId;
  issueName: string;
};

// eslint-disable-next-line @typescript-eslint/ban-types
type SheetEmptyPayload = {};

export type SheetPayload =
  | SheetVCSPayload
  | SheetIssueBacktracePayload
  | SheetEmptyPayload;

export type AccessOption = {
  label: string;
  description: string;
  value: Sheet_Visibility;
};
