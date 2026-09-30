import type { LucideIcon } from "lucide-react";
import type {
  TaskRunLogEntry,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";

export type SectionStatus = "success" | "error" | "running" | "pending";

export interface DisplayItem {
  // The entry's identity, not its position: it must not change when a second
  // replica appears or a retry marker regroups the sections, because the
  // reader's folds are stored under it.
  key: string;
  /** Dense time-of-day; the date lives on the run header and in the tooltip. */
  time: string;
  /** Backs that tooltip; absent when the entry carries no timestamp. */
  timeMs: number | undefined;
  relativeTime: string;
  levelIndicator: string;
  levelClass: string;
  // The one line the row shows: the error when the command failed, otherwise
  // the statement collapsed to a single line.
  detail: string;
  detailClass: string;
  // The statement this row ran, verbatim. Absent when it could not be
  // recovered, and on entries that carry status words rather than a payload.
  statement?: string;
  // The error the command returned, verbatim.
  error?: string;
  // The failure that explains the outcome of its execution context: the view
  // renders it past the "Load more" window and brings it into view. It does not
  // open it. Picked across the whole context, so a view of one section cannot
  // re-derive it.
  marked?: boolean;
  affectedRows?: number;
  duration?: string;
}

export interface Section {
  kind: "section";
  id: string;
  type: TaskRunLogEntry_Type;
  label: string;
  status: SectionStatus;
  statusIcon: LucideIcon;
  statusClass: string;
  duration: string;
  entryCount: number;
  items: DisplayItem[];
}

// A superseded attempt: the entries between two retry markers (or before the
// first one). The marker that closed it supplies the reason.
export interface Attempt {
  id: string;
  number: number;
  reason: string;
  duration: string;
  sections: Section[];
}

// The "Previous attempts" umbrella: every superseded attempt of one retried
// execution scope. It is history, so it has no status of its own.
export interface AttemptGroup {
  kind: "attempts";
  id: string;
  attempts: Attempt[];
  duration: string;
  // A section of the final attempt is still running.
  retrying: boolean;
}

export type LogRow = Section | AttemptGroup;

export interface EntryGroup {
  type: TaskRunLogEntry_Type;
  entries: TaskRunLogEntry[];
}

export interface ReleaseFileInfo {
  version: string;
  filePath: string;
}

export interface ReleaseFileEntriesGroup {
  file: ReleaseFileInfo | null;
  entries: TaskRunLogEntry[];
}

export interface ReleaseFileGroup {
  id: string;
  version: string;
  filePath: string;
  isOrphan?: boolean;
  rows: LogRow[];
}

export interface ReplicaGroup {
  replicaId: string;
  releaseFileGroups: ReleaseFileGroup[];
  rows: LogRow[];
}
