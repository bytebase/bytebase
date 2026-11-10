import type { Timestamp } from "@bufbuild/protobuf/wkt";
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

export interface ReleaseSpineFile {
  version: string;
  path: string;
}

// "pending" is a file the task will apply; the rest describe the run.
export type SpineFileStatus =
  | "applied"
  | "running"
  | "failed"
  | "canceled"
  | "pending";

export interface SpineFileSection {
  kind: "file";
  id: string;
  version: string;
  path: string;
  status: SpineFileStatus;
  entries: TaskRunLogEntry[];
  // The run's error with the executor's file prefix removed, on the section it
  // belongs to.
  error?: string;
  // The failed section has no failed command row to show the error on.
  noStatement?: boolean;
  // The database had a revision of this version that was deleted, so the
  // version runs again. Noted at rollout creation with who and when; after it,
  // only the run tells, and neither is known.
  revisionDeleted?: {
    time?: Timestamp;
    deleter: string;
    afterRolloutCreation?: boolean;
  };
}

// A task-level step: prior backup, schema dump, database sync.
export interface SpineStepSection {
  kind: "step";
  id: string;
  type: TaskRunLogEntry_Type;
  entries: TaskRunLogEntry[];
}

// The release's versions the database already had, in version order. They are
// the database's state before the run, not part of it, so they are never
// windowed.
export interface SpineAppliedEarlier {
  kind: "applied-earlier";
  files: ReleaseSpineFile[];
}

// A run error that names no file and no step.
export interface SpineRunFailed {
  kind: "run-failed";
  error: string;
}

export type SpineItem =
  | SpineAppliedEarlier
  | SpineFileSection
  | SpineStepSection
  | SpineRunFailed;

export interface ReleaseSpine {
  items: SpineItem[];
}

// What a hidden stretch holds, which names its load row.
export type SpineLoadKey = "applied" | "pending" | "files";

// How many rows of each stretch the reader has loaded.
export type SpineLoads = Partial<Record<SpineLoadKey, number>>;

export type SpineWindowItem =
  | SpineAppliedEarlier
  | SpineFileSection
  | SpineStepSection
  | SpineRunFailed
  // `more`: a shown row above holds the same state.
  | { kind: "load"; key: SpineLoadKey; count: number; more: boolean };
