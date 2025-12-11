import type { Timestamp as PbTimestamp } from "@bufbuild/protobuf/wkt";
import { getDateForPbTimestampProtoEs } from "@/types";
import type { TaskRunLogEntry } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRunLogEntry_Type } from "@/types/proto-es/v1/rollout_service_pb";
import type { EntryGroup } from "./types";

// Timestamp utilities
export const getTimestampMs = (ts: PbTimestamp | undefined): number => {
  if (!ts) return 0;
  return Number(ts.seconds) * 1000 + ts.nanos / 1000000;
};

export const formatTime = (ts: PbTimestamp | undefined): string => {
  if (!ts) return "--:--:--.---";
  const date = getDateForPbTimestampProtoEs(ts);
  if (!date) return "--:--:--.---";
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
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  const mins = Math.floor(ms / 60000);
  const secs = ((ms % 60000) / 1000).toFixed(0);
  return `${mins}m ${secs}s`;
};

export const formatRelativeTime = (ms: number): string => {
  if (ms < 1000) return `+${ms.toFixed(0)}ms`;
  return `+${(ms / 1000).toFixed(2)}s`;
};

// Entry analysis
export const hasError = (entry: TaskRunLogEntry): boolean => {
  switch (entry.type) {
    case TaskRunLogEntry_Type.COMMAND_EXECUTE:
      return !!entry.commandExecute?.response?.error;
    case TaskRunLogEntry_Type.TRANSACTION_CONTROL:
      return !!entry.transactionControl?.error;
    case TaskRunLogEntry_Type.SCHEMA_DUMP:
      return !!entry.schemaDump?.error;
    case TaskRunLogEntry_Type.DATABASE_SYNC:
      return !!entry.databaseSync?.error;
    case TaskRunLogEntry_Type.PRIOR_BACKUP:
      return !!entry.priorBackup?.error;
    case TaskRunLogEntry_Type.COMPUTE_DIFF:
      return !!entry.computeDiff?.error;
    default:
      return false;
  }
};

export const isComplete = (entry: TaskRunLogEntry): boolean => {
  switch (entry.type) {
    case TaskRunLogEntry_Type.COMMAND_EXECUTE:
      return !!entry.commandExecute?.response;
    case TaskRunLogEntry_Type.SCHEMA_DUMP:
      return !!(entry.schemaDump?.startTime && entry.schemaDump?.endTime);
    case TaskRunLogEntry_Type.DATABASE_SYNC:
      return !!(entry.databaseSync?.startTime && entry.databaseSync?.endTime);
    case TaskRunLogEntry_Type.PRIOR_BACKUP:
      return !!(entry.priorBackup?.startTime && entry.priorBackup?.endTime);
    case TaskRunLogEntry_Type.COMPUTE_DIFF:
      return !!(entry.computeDiff?.startTime && entry.computeDiff?.endTime);
    case TaskRunLogEntry_Type.TRANSACTION_CONTROL:
    case TaskRunLogEntry_Type.TASK_RUN_STATUS_UPDATE:
    case TaskRunLogEntry_Type.RETRY_INFO:
      return true;
    default:
      return true;
  }
};

// Group consecutive entries of the same type
export const groupEntriesByType = (
  entries: TaskRunLogEntry[]
): EntryGroup[] => {
  if (!entries.length) return [];

  // Sort entries by time first
  const sortedEntries = [...entries].sort((a, b) => {
    return getTimestampMs(a.logTime) - getTimestampMs(b.logTime);
  });

  const groups: EntryGroup[] = [];
  let currentGroup: EntryGroup | null = null;

  for (const entry of sortedEntries) {
    if (currentGroup && currentGroup.type === entry.type) {
      currentGroup.entries.push(entry);
    } else {
      currentGroup = { type: entry.type, entries: [entry] };
      groups.push(currentGroup);
    }
  }

  return groups;
};

// Get unique deploy IDs from entries, preserving order of first appearance
export const getUniqueDeployIds = (entries: TaskRunLogEntry[]): string[] => {
  if (!entries.length) return [];

  const sortedEntries = [...entries].sort((a, b) => {
    return getTimestampMs(a.logTime) - getTimestampMs(b.logTime);
  });

  const seen = new Set<string>();
  const deployIds: string[] = [];

  for (const entry of sortedEntries) {
    const deployId = entry.deployId || "";
    if (deployId && !seen.has(deployId)) {
      seen.add(deployId);
      deployIds.push(deployId);
    }
  }

  return deployIds;
};

// Group entries by deploy ID, maintaining time order within each group
export const groupEntriesByDeploy = (
  entries: TaskRunLogEntry[]
): Map<string, TaskRunLogEntry[]> => {
  const result = new Map<string, TaskRunLogEntry[]>();

  for (const entry of entries) {
    const deployId = entry.deployId || "unknown";
    if (!result.has(deployId)) {
      result.set(deployId, []);
    }
    result.get(deployId)!.push(entry);
  }

  return result;
};
