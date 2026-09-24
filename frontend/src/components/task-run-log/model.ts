import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { LucideIcon } from "lucide-react";
import { CheckCircle2, Circle, LoaderCircle, XCircle } from "lucide-react";
import type { TaskRunLogEntry } from "@/types/proto-es/v1/rollout_service_pb";
import {
  TaskRun_Status,
  TaskRunLogEntry_TransactionControl_Type,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import type {
  DisplayItem,
  EntryGroup,
  ReleaseFileEntriesGroup,
  ReleaseFileGroup,
  Section,
  SectionStatus,
} from "./types";

const STATUS_CONFIG: Record<
  SectionStatus,
  { icon: LucideIcon; className: string }
> = {
  success: { icon: CheckCircle2, className: "text-success" },
  error: { icon: XCircle, className: "text-error" },
  running: { icon: LoaderCircle, className: "text-info" },
  pending: { icon: Circle, className: "text-control-placeholder" },
};

// Row keys for one run, from assignEntryKeys.
export type EntryKeys = ReadonlyMap<TaskRunLogEntry, string>;

export interface BuildSectionsOptions {
  getSectionLabel: (type: TaskRunLogEntry_Type) => string;
  // Assigned over the run's whole entry list. The builders only ever see a
  // filtered slice of it, so they cannot assign keys themselves.
  entryKeys: EntryKeys;
  sheet?: Sheet;
  sheetsMap?: Map<string, Sheet>;
  idPrefix?: string;
  forceError?: boolean;
  fileVersion?: string;
  detailText?: TaskRunLogDetailText;
  taskRunStatus?: TaskRun_Status;
}

export interface BuildReleaseFileGroupsOptions {
  getSectionLabel: (type: TaskRunLogEntry_Type) => string;
  entryKeys: EntryKeys;
  sheet?: Sheet;
  sheetsMap?: Map<string, Sheet>;
  idPrefix?: string;
  forceError?: boolean;
  detailText?: TaskRunLogDetailText;
  includeOrphanGroup?: boolean;
  taskRunStatus?: TaskRun_Status;
}

export interface TaskRunLogDetailText {
  completed?: string;
  backingUp?: string;
  runningByType?: Partial<Record<TaskRunLogEntry_Type, string>>;
  transactionError?: (typeLabel: string, error: string) => string;
  retryAttempt?: (current: number, max: number) => string;
  backupCompleted?: (count: number) => string;
}

export const getTimestampMs = (timestamp?: Timestamp): number => {
  if (!timestamp) return 0;
  return Number(timestamp.seconds) * 1000 + timestamp.nanos / 1000000;
};

// A row's key is the entry's identity: replica, log time at full precision,
// type, and an ordinal among the entries that tie on all three. task_run_log
// has no primary key and rows may share a created_at, so time alone ties; the
// API returns rows in append order, so a tie group's ordinals never shift.
// Call this once with the run's complete entry list, before any filtering: an
// ordinal counted over a slice restarts at zero and collides across slices.
export const assignEntryKeys = (entries: TaskRunLogEntry[]): EntryKeys => {
  const keys = new Map<TaskRunLogEntry, string>();
  const tieCounts = new Map<string, number>();
  for (const entry of entries) {
    const seconds = entry.logTime?.seconds.toString() ?? "";
    const nanos = entry.logTime?.nanos ?? 0;
    const tieGroup = `${entry.replicaId}:${seconds}.${nanos}:${entry.type}`;
    const ordinal = tieCounts.get(tieGroup) ?? 0;
    tieCounts.set(tieGroup, ordinal + 1);
    keys.set(entry, `${tieGroup}:${ordinal}`);
  }
  return keys;
};

const getEntryKey = (entryKeys: EntryKeys, entry: TaskRunLogEntry): string => {
  const key = entryKeys.get(entry);
  if (key === undefined) {
    throw new Error(
      "task-run log entry has no key; assignEntryKeys must cover every entry handed to a builder"
    );
  }
  return key;
};

export const formatTime = (timestamp?: Timestamp): string => {
  if (!timestamp) return "--:--:--.---";
  const date = new Date(getTimestampMs(timestamp));
  return (
    [
      date.getHours().toString().padStart(2, "0"),
      date.getMinutes().toString().padStart(2, "0"),
      date.getSeconds().toString().padStart(2, "0"),
    ].join(":") +
    "." +
    date.getMilliseconds().toString().padStart(3, "0")
  );
};

const formatDuration = (ms: number): string => {
  if (ms < 1) return "<1ms";
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  const minutes = Math.floor(ms / 60000);
  const seconds = ((ms % 60000) / 1000).toFixed(0);
  return `${minutes}m ${seconds}s`;
};

export const formatRelativeTime = (ms: number): string => {
  if (ms < 1000) return `+${ms.toFixed(0)}ms`;
  return `+${(ms / 1000).toFixed(2)}s`;
};

export const hasError = (entry: TaskRunLogEntry): boolean => {
  switch (entry.type) {
    case TaskRunLogEntry_Type.COMMAND_EXECUTE:
      return Boolean(entry.commandExecute?.response?.error);
    case TaskRunLogEntry_Type.TRANSACTION_CONTROL:
      return Boolean(entry.transactionControl?.error);
    case TaskRunLogEntry_Type.SCHEMA_DUMP:
      return Boolean(entry.schemaDump?.error);
    case TaskRunLogEntry_Type.DATABASE_SYNC:
      return Boolean(entry.databaseSync?.error);
    case TaskRunLogEntry_Type.PRIOR_BACKUP:
      return Boolean(entry.priorBackup?.error);
    case TaskRunLogEntry_Type.COMPUTE_DIFF:
      return Boolean(entry.computeDiff?.error);
    case TaskRunLogEntry_Type.GHOST_MIGRATION:
      return Boolean(entry.ghostMigration?.error);
    default:
      return false;
  }
};

export const isComplete = (entry: TaskRunLogEntry): boolean => {
  switch (entry.type) {
    case TaskRunLogEntry_Type.COMMAND_EXECUTE:
      return Boolean(entry.commandExecute?.response);
    case TaskRunLogEntry_Type.SCHEMA_DUMP:
      return Boolean(entry.schemaDump?.startTime && entry.schemaDump?.endTime);
    case TaskRunLogEntry_Type.DATABASE_SYNC:
      return Boolean(
        entry.databaseSync?.startTime && entry.databaseSync?.endTime
      );
    case TaskRunLogEntry_Type.PRIOR_BACKUP:
      return Boolean(
        entry.priorBackup?.startTime && entry.priorBackup?.endTime
      );
    case TaskRunLogEntry_Type.COMPUTE_DIFF:
      return Boolean(
        entry.computeDiff?.startTime && entry.computeDiff?.endTime
      );
    case TaskRunLogEntry_Type.GHOST_MIGRATION:
      return Boolean(
        entry.ghostMigration?.startTime && entry.ghostMigration?.endTime
      );
    case TaskRunLogEntry_Type.TRANSACTION_CONTROL:
    case TaskRunLogEntry_Type.RETRY_INFO:
      return true;
    default:
      return true;
  }
};

const sortEntriesByTime = (entries: TaskRunLogEntry[]): TaskRunLogEntry[] =>
  [...entries].sort(
    (a, b) => getTimestampMs(a.logTime) - getTimestampMs(b.logTime)
  );

export const groupEntriesByType = (
  entries: TaskRunLogEntry[]
): EntryGroup[] => {
  if (entries.length === 0) return [];
  const sorted = sortEntriesByTime(entries);
  const groups: EntryGroup[] = [];
  let current: EntryGroup | undefined;

  for (const entry of sorted) {
    if (
      current &&
      current.type === entry.type &&
      entry.type !== TaskRunLogEntry_Type.GHOST_MIGRATION
    ) {
      current.entries.push(entry);
      continue;
    }
    current = { type: entry.type, entries: [entry] };
    groups.push(current);
  }

  return groups;
};

export const hasReleaseFileMarkers = (entries: TaskRunLogEntry[]): boolean => {
  return entries.some(
    (entry) => entry.type === TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE
  );
};

export const groupEntriesByReleaseFile = (
  entries: TaskRunLogEntry[]
): ReleaseFileEntriesGroup[] => {
  if (entries.length === 0) return [];

  const sorted = [...entries].sort(
    (a, b) => getTimestampMs(a.logTime) - getTimestampMs(b.logTime)
  );
  const groups: ReleaseFileEntriesGroup[] = [];
  let current: ReleaseFileEntriesGroup = { file: null, entries: [] };

  for (const entry of sorted) {
    if (entry.type === TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE) {
      if (current.file !== null || current.entries.length > 0) {
        groups.push(current);
      }
      const releaseFile = entry.releaseFileExecute;
      current = {
        file: releaseFile
          ? {
              version: releaseFile.version,
              filePath: releaseFile.filePath || "",
            }
          : null,
        entries: [],
      };
      continue;
    }
    current.entries.push(entry);
  }

  if (current.file !== null || current.entries.length > 0) {
    groups.push(current);
  }

  return groups;
};

export const getUniqueReplicaIds = (entries: TaskRunLogEntry[]): string[] => {
  if (entries.length === 0) return [];
  const sorted = [...entries].sort(
    (a, b) => getTimestampMs(a.logTime) - getTimestampMs(b.logTime)
  );
  const replicaIds: string[] = [];
  const seen = new Set<string>();

  for (const entry of sorted) {
    const replicaId = entry.replicaId || "";
    if (!replicaId || seen.has(replicaId)) continue;
    seen.add(replicaId);
    replicaIds.push(replicaId);
  }

  return replicaIds;
};

export const groupEntriesByReplica = (
  entries: TaskRunLogEntry[]
): Map<string, TaskRunLogEntry[]> => {
  const grouped = new Map<string, TaskRunLogEntry[]>();

  for (const entry of entries) {
    const replicaId = entry.replicaId || "unknown";
    if (!grouped.has(replicaId)) {
      grouped.set(replicaId, []);
    }
    grouped.get(replicaId)!.push(entry);
  }

  return grouped;
};

const getEntryTimeRange = (
  entry: TaskRunLogEntry
): { start: number; end: number } => {
  switch (entry.type) {
    case TaskRunLogEntry_Type.COMMAND_EXECUTE: {
      const start = getTimestampMs(entry.commandExecute?.logTime);
      const end = getTimestampMs(entry.commandExecute?.response?.logTime);
      return { start: start || 0, end: end || start || 0 };
    }
    case TaskRunLogEntry_Type.SCHEMA_DUMP:
      return {
        start: getTimestampMs(entry.schemaDump?.startTime),
        end: getTimestampMs(entry.schemaDump?.endTime),
      };
    case TaskRunLogEntry_Type.DATABASE_SYNC:
      return {
        start: getTimestampMs(entry.databaseSync?.startTime),
        end: getTimestampMs(entry.databaseSync?.endTime),
      };
    case TaskRunLogEntry_Type.PRIOR_BACKUP:
      return {
        start: getTimestampMs(entry.priorBackup?.startTime),
        end: getTimestampMs(entry.priorBackup?.endTime),
      };
    case TaskRunLogEntry_Type.COMPUTE_DIFF:
      return {
        start: getTimestampMs(entry.computeDiff?.startTime),
        end: getTimestampMs(entry.computeDiff?.endTime),
      };
    case TaskRunLogEntry_Type.GHOST_MIGRATION:
      return {
        start: getTimestampMs(entry.ghostMigration?.startTime),
        end: getTimestampMs(entry.ghostMigration?.endTime),
      };
    default: {
      const time = getTimestampMs(entry.logTime);
      return { start: time, end: time };
    }
  }
};

const extractStatementFromRange = (
  range: { start: number; end: number },
  sheet: Sheet | undefined,
  sheetsMap: Map<string, Sheet> | undefined,
  fileVersion?: string
): string | undefined => {
  let content = sheet?.content;
  if (fileVersion && sheetsMap) {
    const releaseFileSheet = sheetsMap.get(fileVersion);
    if (releaseFileSheet?.content) {
      content = releaseFileSheet.content;
    }
  }
  if (!content) return undefined;
  // A range reaching past the content means the sheet came back partial.
  // subarray would clamp silently and pass a cut statement off as whole.
  if (range.end > content.byteLength) return undefined;
  const subarray = content.subarray(range.start, range.end);
  return new TextDecoder().decode(subarray);
};

interface EntryDetail {
  detail: string;
  statement?: string;
  error?: string;
}

// The one-line form of a statement. Clamping to the row's width is CSS's job:
// only the row knows how much fits.
const collapseStatement = (statement: string): string =>
  statement.trim().replace(/\s+/g, " ");

const getCommandExecuteDetail = (
  entry: TaskRunLogEntry,
  sheet: Sheet | undefined,
  sheetsMap: Map<string, Sheet> | undefined,
  fileVersion?: string
): EntryDetail => {
  const command = entry.commandExecute;
  if (!command) return { detail: "" };

  const recovered =
    command.statement ||
    (command.range
      ? extractStatementFromRange(command.range, sheet, sheetsMap, fileVersion)
      : undefined);
  const statement = recovered?.trim() ? recovered : undefined;
  const error = command.response?.error || undefined;

  return {
    detail: error ?? (statement ? collapseStatement(statement) : "-"),
    statement,
    error,
  };
};

const getTransactionControlDetail = (
  entry: TaskRunLogEntry,
  detailText: TaskRunLogDetailText | undefined
): string => {
  const transaction = entry.transactionControl;
  if (!transaction) return "";

  const typeLabels: Record<number, string> = {
    [TaskRunLogEntry_TransactionControl_Type.BEGIN]: "BEGIN",
    [TaskRunLogEntry_TransactionControl_Type.COMMIT]: "COMMIT",
    [TaskRunLogEntry_TransactionControl_Type.ROLLBACK]: "ROLLBACK",
  };

  const typeLabel = typeLabels[transaction.type] ?? "";
  if (!transaction.error) return typeLabel;

  if (detailText?.transactionError) {
    return detailText.transactionError(typeLabel, transaction.error);
  }
  return typeLabel ? `${typeLabel}: ${transaction.error}` : transaction.error;
};

const getTimedEntryDetail = (
  timedEntry:
    | {
        error?: string;
        startTime?: Timestamp;
        endTime?: Timestamp;
      }
    | undefined,
  runningLabel: string,
  completedLabel: string
): string => {
  if (!timedEntry) return "";
  if (timedEntry.error) return timedEntry.error;
  if (timedEntry.startTime && timedEntry.endTime) return completedLabel;
  return runningLabel;
};

const getPriorBackupDetail = (
  entry: TaskRunLogEntry,
  detailText: TaskRunLogDetailText | undefined
): string => {
  const priorBackup = entry.priorBackup;
  if (!priorBackup) return "";
  if (priorBackup.error) return priorBackup.error;
  const itemCount = priorBackup.priorBackupDetail?.items.length ?? 0;
  if (itemCount > 0) {
    if (detailText?.backupCompleted) {
      return detailText.backupCompleted(itemCount);
    }
    return String(itemCount);
  }
  if (priorBackup.startTime && priorBackup.endTime) {
    return detailText?.completed ?? "";
  }
  return detailText?.backingUp ?? "";
};

const getRetryInfoDetail = (
  entry: TaskRunLogEntry,
  detailText: TaskRunLogDetailText | undefined
): string => {
  const retryInfo = entry.retryInfo;
  if (!retryInfo) return "";
  const attempt = detailText?.retryAttempt
    ? detailText.retryAttempt(retryInfo.retryCount, retryInfo.maximumRetries)
    : `${retryInfo.retryCount}/${retryInfo.maximumRetries}`;
  return retryInfo.error ? `${attempt}: ${retryInfo.error}` : attempt;
};

const getReleaseFileExecuteDetail = (entry: TaskRunLogEntry): string => {
  const releaseFile = entry.releaseFileExecute;
  if (!releaseFile) return "";
  return releaseFile.filePath
    ? `${releaseFile.version}: ${releaseFile.filePath}`
    : releaseFile.version;
};

// Entries that carry status words rather than a payload.
const getStatusDetail = (
  entry: TaskRunLogEntry,
  detailText: TaskRunLogDetailText | undefined
): string => {
  const runningByType = detailText?.runningByType;
  const completed = detailText?.completed ?? "";

  switch (entry.type) {
    case TaskRunLogEntry_Type.TRANSACTION_CONTROL:
      return getTransactionControlDetail(entry, detailText);
    case TaskRunLogEntry_Type.SCHEMA_DUMP:
      return getTimedEntryDetail(
        entry.schemaDump,
        runningByType?.[TaskRunLogEntry_Type.SCHEMA_DUMP] ?? "",
        completed
      );
    case TaskRunLogEntry_Type.DATABASE_SYNC:
      return getTimedEntryDetail(
        entry.databaseSync,
        runningByType?.[TaskRunLogEntry_Type.DATABASE_SYNC] ?? "",
        completed
      );
    case TaskRunLogEntry_Type.PRIOR_BACKUP:
      return getPriorBackupDetail(entry, detailText);
    case TaskRunLogEntry_Type.RETRY_INFO:
      return getRetryInfoDetail(entry, detailText);
    case TaskRunLogEntry_Type.COMPUTE_DIFF:
      return getTimedEntryDetail(
        entry.computeDiff,
        runningByType?.[TaskRunLogEntry_Type.COMPUTE_DIFF] ?? "",
        completed
      );
    case TaskRunLogEntry_Type.GHOST_MIGRATION:
      return getTimedEntryDetail(
        entry.ghostMigration,
        runningByType?.[TaskRunLogEntry_Type.GHOST_MIGRATION] ?? "",
        completed
      );
    case TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE:
      return getReleaseFileExecuteDetail(entry);
    default:
      return "";
  }
};

const getEntryDetail = (
  entry: TaskRunLogEntry,
  sheet: Sheet | undefined,
  sheetsMap: Map<string, Sheet> | undefined,
  detailText: TaskRunLogDetailText | undefined,
  fileVersion?: string
): EntryDetail => {
  if (entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE) {
    return getCommandExecuteDetail(entry, sheet, sheetsMap, fileVersion);
  }
  return { detail: getStatusDetail(entry, detailText) };
};

const getCommandDuration = (entry: TaskRunLogEntry): string | undefined => {
  if (entry.type !== TaskRunLogEntry_Type.COMMAND_EXECUTE) return undefined;
  const startMs = getTimestampMs(entry.commandExecute?.logTime);
  const endMs = getTimestampMs(entry.commandExecute?.response?.logTime);
  if (startMs <= 0 || endMs <= 0) return undefined;
  return formatDuration(endMs - startMs);
};

const buildDisplayItems = (
  entries: TaskRunLogEntry[],
  startTime: number,
  markedEntry: TaskRunLogEntry | undefined,
  options: BuildSectionsOptions
): DisplayItem[] => {
  return entries.map((entry) => {
    const entryTime = getTimestampMs(entry.logTime);
    const relativeMs = startTime > 0 ? entryTime - startTime : 0;
    const entryHasError = hasError(entry);

    return {
      key: getEntryKey(options.entryKeys, entry),
      time: formatTime(entry.logTime),
      timeMs: entry.logTime ? entryTime : undefined,
      relativeTime: relativeMs > 0 ? formatRelativeTime(relativeMs) : "",
      levelIndicator: entryHasError ? "\u2717" : "\u2713",
      levelClass: entryHasError ? "text-error" : "text-success",
      ...getEntryDetail(
        entry,
        options.sheet,
        options.sheetsMap,
        options.detailText,
        options.fileVersion
      ),
      detailClass: entryHasError ? "text-error" : "text-control",
      marked: entry === markedEntry || undefined,
      affectedRows:
        entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE
          ? Number(entry.commandExecute?.response?.affectedRows ?? 0) ||
            undefined
          : undefined,
      duration: getCommandDuration(entry),
    };
  });
};

const commandFailed = (entry: TaskRunLogEntry): boolean =>
  entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE && hasError(entry);

const commandSucceeded = (entry: TaskRunLogEntry): boolean =>
  entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE &&
  isComplete(entry) &&
  !hasError(entry);

// The failure that explains the outcome of one execution context: the last
// failed command, unless a later command succeeded or a retry marker follows
// it. A lock timeout makes the driver re-run and re-log the whole command list,
// and it writes the marker before each new attempt, so either one means the
// failure was superseded. `sorted` must be the whole context, ungrouped:
// grouping by type would leave every retried failure last in its own section.
const pickMarkedEntry = (
  sorted: TaskRunLogEntry[],
  taskRunStatus: TaskRun_Status | undefined
): TaskRunLogEntry | undefined => {
  // Mitigation for a response the API drops (CockroachDB logs one per attempt
  // and the converter keeps the first): a run that finished DONE has no
  // failure to explain, whatever its entries claim.
  if (taskRunStatus === TaskRun_Status.DONE) return undefined;

  const lastFailedIndex = sorted.findLastIndex(commandFailed);
  if (lastFailedIndex < 0) return undefined;
  const superseded = sorted
    .slice(lastFailedIndex + 1)
    .some(
      (entry) =>
        entry.type === TaskRunLogEntry_Type.RETRY_INFO ||
        commandSucceeded(entry)
    );
  return superseded ? undefined : sorted[lastFailedIndex];
};

const calculateSectionStatus = (entries: TaskRunLogEntry[]): SectionStatus => {
  if (entries.some(hasError)) return "error";
  if (entries.every(isComplete)) return "success";
  return "running";
};

export const buildSectionsFromEntries = (
  entries: TaskRunLogEntry[],
  options: BuildSectionsOptions
): Section[] => {
  if (entries.length === 0) return [];

  const sorted = sortEntriesByTime(entries);
  const markedEntry = pickMarkedEntry(sorted, options.taskRunStatus);

  return groupEntriesByType(sorted).map((group, groupIndex) => {
    let status = calculateSectionStatus(group.entries);
    if (options.forceError && status === "running") {
      status = "error";
    }
    const statusConfig = STATUS_CONFIG[status];

    const timeRanges = group.entries.map(getEntryTimeRange);
    const startTimes = timeRanges
      .map((range) => range.start)
      .filter((time) => time > 0);
    const endTimes = timeRanges
      .map((range) => range.end)
      .filter((time) => time > 0);
    const startTime = startTimes.length > 0 ? Math.min(...startTimes) : 0;
    const endTime = endTimes.length > 0 ? Math.max(...endTimes) : 0;
    const durationMs = endTime - startTime;

    return {
      id: options.idPrefix
        ? `${options.idPrefix}-section-${groupIndex}`
        : `section-${groupIndex}`,
      type: group.type,
      label: options.getSectionLabel(group.type),
      status,
      statusIcon: statusConfig.icon,
      statusClass: statusConfig.className,
      duration:
        startTime > 0 && endTime > 0
          ? formatDuration(Math.max(durationMs, 0))
          : "",
      entryCount: group.entries.length,
      items: buildDisplayItems(group.entries, startTime, markedEntry, options),
    };
  });
};

export const buildReleaseFileGroups = (
  entries: TaskRunLogEntry[],
  options: BuildReleaseFileGroupsOptions
): ReleaseFileGroup[] => {
  const buildOptions: BuildReleaseFileGroupsOptions = {
    includeOrphanGroup: false,
    ...options,
  };

  const results: ReleaseFileGroup[] = [];
  let fileIndex = 0;

  for (const group of groupEntriesByReleaseFile(entries)) {
    if (group.file === null) {
      if (!buildOptions.includeOrphanGroup || group.entries.length === 0) {
        continue;
      }
      const orphanPrefix = buildOptions.idPrefix
        ? `${buildOptions.idPrefix}-orphan`
        : "orphan";
      results.push({
        id: orphanPrefix,
        version: "",
        filePath: "",
        isOrphan: true,
        sections: buildSectionsFromEntries(group.entries, {
          ...buildOptions,
          idPrefix: orphanPrefix,
        }),
      });
      continue;
    }

    if (group.entries.length === 0) {
      continue;
    }

    const filePrefix = buildOptions.idPrefix
      ? `${buildOptions.idPrefix}-file-${fileIndex}`
      : `file-${fileIndex}`;
    fileIndex++;
    results.push({
      id: filePrefix,
      version: group.file.version,
      filePath: group.file.filePath,
      sections: buildSectionsFromEntries(group.entries, {
        ...buildOptions,
        idPrefix: filePrefix,
        fileVersion: group.file.version,
      }),
    });
  }

  return results;
};
