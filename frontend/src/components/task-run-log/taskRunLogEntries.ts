import { create } from "@bufbuild/protobuf";
import {
  type TaskRunLogEntry,
  TaskRunLogEntry_TransactionControl_Type,
  TaskRunLogEntry_Type,
  TaskRunLogEntrySchema,
} from "@/types/proto-es/v1/rollout_service_pb";

// Task run log entries for tests of the log viewer. Times are whole seconds so
// durations read as literals ("2.0s") in assertions.

export const LOCK_TIMEOUT = "ERROR: canceling statement due to lock timeout";
const STATEMENT = "ALTER TABLE book ADD COLUMN title TEXT;";

export const ts = (seconds: number) => ({ seconds: BigInt(seconds), nanos: 0 });

interface EntryOptions {
  replicaId?: string;
}

interface OutcomeOptions extends EntryOptions {
  error?: string;
}

const transaction = (
  at: number,
  type: TaskRunLogEntry_TransactionControl_Type,
  { replicaId = "" }: EntryOptions
): TaskRunLogEntry =>
  create(TaskRunLogEntrySchema, {
    type: TaskRunLogEntry_Type.TRANSACTION_CONTROL,
    logTime: ts(at),
    replicaId,
    transactionControl: { type, error: "" },
  });

export const begin = (at: number, options: EntryOptions = {}) =>
  transaction(at, TaskRunLogEntry_TransactionControl_Type.BEGIN, options);
export const commit = (at: number, options: EntryOptions = {}) =>
  transaction(at, TaskRunLogEntry_TransactionControl_Type.COMMIT, options);
export const rollback = (at: number, options: EntryOptions = {}) =>
  transaction(at, TaskRunLogEntry_TransactionControl_Type.ROLLBACK, options);

export const command = (
  at: number,
  done: number,
  { error = "", replicaId = "" }: OutcomeOptions = {}
) =>
  create(TaskRunLogEntrySchema, {
    type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
    logTime: ts(at),
    replicaId,
    commandExecute: {
      logTime: ts(at),
      statement: STATEMENT,
      response: { logTime: ts(done), error },
    },
  });

// A command with no response yet.
export const runningCommand = (
  at: number,
  { replicaId = "" }: EntryOptions = {}
) =>
  create(TaskRunLogEntrySchema, {
    type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
    logTime: ts(at),
    replicaId,
    commandExecute: { logTime: ts(at), statement: STATEMENT },
  });

export const retry = (
  at: number,
  retryCount: number,
  { error = LOCK_TIMEOUT, replicaId = "" }: OutcomeOptions = {}
) =>
  create(TaskRunLogEntrySchema, {
    type: TaskRunLogEntry_Type.RETRY_INFO,
    logTime: ts(at),
    replicaId,
    retryInfo: { retryCount, maximumRetries: 5, error },
  });

export const priorBackup = (start: number, end: number) =>
  create(TaskRunLogEntrySchema, {
    type: TaskRunLogEntry_Type.PRIOR_BACKUP,
    logTime: ts(start),
    priorBackup: { startTime: ts(start), endTime: ts(end), error: "" },
  });

export const databaseSync = (
  start: number,
  end: number,
  { error = "" }: OutcomeOptions = {}
) =>
  create(TaskRunLogEntrySchema, {
    type: TaskRunLogEntry_Type.DATABASE_SYNC,
    logTime: ts(start),
    databaseSync: { startTime: ts(start), endTime: ts(end), error },
  });

export const releaseFile = (
  at: number,
  version: string,
  filePath: string,
  { replicaId = "" }: EntryOptions = {}
) =>
  create(TaskRunLogEntrySchema, {
    type: TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE,
    logTime: ts(at),
    replicaId,
    releaseFileExecute: { version, filePath },
  });

// One failed attempt spanning [at, at + 1], closed by a marker at at + 2.
export const failedAttempt = (
  at: number,
  retryCount: number,
  { error = LOCK_TIMEOUT, replicaId = "" }: OutcomeOptions = {}
) => [
  begin(at, { replicaId }),
  command(at, at + 1, { error, replicaId }),
  rollback(at + 1, { replicaId }),
  retry(at + 2, retryCount, { error, replicaId }),
];

export const passedAttempt = (
  at: number,
  { replicaId = "" }: EntryOptions = {}
) => [
  begin(at, { replicaId }),
  command(at, at + 1, { replicaId }),
  commit(at + 2, { replicaId }),
];
