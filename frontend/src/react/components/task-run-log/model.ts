import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { LucideIcon } from "lucide-react";
import { CheckCircle2, Circle, LoaderCircle, XCircle } from "lucide-react";
import type { TaskRunLogEntry } from "@/types/proto-es/v1/rollout_service_pb";
import {
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
  success: { icon: CheckCircle2, className: "text-green-600" },
  error: { icon: XCircle, className: "text-red-600" },
  running: { icon: LoaderCircle, className: "text-blue-600" },
  pending: { icon: Circle, className: "text-gray-400" },
};

export interface BuildSectionsOptions {
  getSectionLabel: (type: TaskRunLogEntry_Type) => string;
  sheet?: Sheet;
  sheetsMap?: Map<string, Sheet>;
  idPrefix?: string;
  forceError?: boolean;
  fileVersion?: string;
  detailText?: TaskRunLogDetailText;
}

export interface BuildReleaseFileGroupsOptions {
  getSectionLabel: (type: TaskRunLogEntry_Type) => string;
  sheet?: Sheet;
  sheetsMap?: Map<string, Sheet>;
  idPrefix?: string;
  forceError?: boolean;
  detailText?: TaskRunLogDetailText;
  includeOrphanGroup?: boolean;
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

export const formatDuration = (ms: number): string => {
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
    case TaskRunLogEntry_Type.TRANSACTION_CONTROL:
    case TaskRunLogEntry_Type.RETRY_INFO:
      return true;
    default:
      return true;
  }
};

export const groupEntriesByType = (
  entries: TaskRunLogEntry[]
): EntryGroup[] => {
  if (entries.length === 0) return [];
  const sorted = [...entries].sort(
    (a, b) => getTimestampMs(a.logTime) - getTimestampMs(b.logTime)
  );
  const groups: EntryGroup[] = [];
  let current: EntryGroup | undefined;

  for (const entry of sorted) {
    if (current && current.type === entry.type) {
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

export const getEntryTimeRange = (
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
  const subarray = content.subarray(range.start, range.end);
  return new TextDecoder().decode(subarray);
};

const getCommandExecuteDetail = (
  entry: TaskRunLogEntry,
  sheet: Sheet | undefined,
  sheetsMap: Map<string, Sheet> | undefined,
  fileVersion?: string
): string => {
  const command = entry.commandExecute;
  if (!command) return "";
  if (command.response?.error) return command.response.error;

  const statement =
    command.statement ||
    (command.range
      ? extractStatementFromRange(command.range, sheet, sheetsMap, fileVersion)
      : undefined);
  if (!statement) return "-";

  const normalized = statement.trim().replace(/\s+/g, " ");
  return normalized.length > 80
    ? `${normalized.substring(0, 80)}...`
    : normalized;
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

const getEntryDetail = (
  entry: TaskRunLogEntry,
  sheet: Sheet | undefined,
  sheetsMap: Map<string, Sheet> | undefined,
  detailText: TaskRunLogDetailText | undefined,
  fileVersion?: string
): string => {
  const runningByType = detailText?.runningByType;
  const completed = detailText?.completed ?? "";

  switch (entry.type) {
    case TaskRunLogEntry_Type.COMMAND_EXECUTE:
      return getCommandExecuteDetail(entry, sheet, sheetsMap, fileVersion);
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
    case TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE:
      return getReleaseFileExecuteDetail(entry);
    default:
      return "";
  }
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
  groupIndex: number,
  startTime: number,
  sheet: Sheet | undefined,
  sheetsMap: Map<string, Sheet> | undefined,
  idPrefix: string | undefined,
  detailText: TaskRunLogDetailText | undefined,
  fileVersion?: string
): DisplayItem[] => {
  return entries.map((entry, entryIndex) => {
    const entryTime = getTimestampMs(entry.logTime);
    const relativeMs = startTime > 0 ? entryTime - startTime : 0;
    const entryHasError = hasError(entry);

    return {
      key: `${idPrefix ?? "section"}-${groupIndex}-${entryIndex}`,
      time: formatTime(entry.logTime),
      relativeTime: relativeMs > 0 ? formatRelativeTime(relativeMs) : "",
      levelIndicator: entryHasError ? "\u2717" : "\u2713",
      levelClass: entryHasError ? "text-red-600" : "text-green-600",
      detail: getEntryDetail(entry, sheet, sheetsMap, detailText, fileVersion),
      detailClass: entryHasError ? "text-red-700" : "text-gray-600",
      affectedRows:
        entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE
          ? Number(entry.commandExecute?.response?.affectedRows ?? 0) ||
            undefined
          : undefined,
      duration: getCommandDuration(entry),
    };
  });
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

  return groupEntriesByType(entries).map((group, groupIndex) => {
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
      items: buildDisplayItems(
        group.entries,
        groupIndex,
        startTime,
        options.sheet,
        options.sheetsMap,
        options.idPrefix,
        options.detailText,
        options.fileVersion
      ),
    };
  });
};

export const buildReleaseFileGroups = (
  entries: TaskRunLogEntry[],
  options?: BuildReleaseFileGroupsOptions
): ReleaseFileGroup[] => {
  const buildOptions: BuildReleaseFileGroupsOptions = {
    getSectionLabel: (type) => String(type),
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
