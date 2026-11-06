import { create } from "@bufbuild/protobuf";
import { describe, expect, test, vi } from "vitest";
import {
  Task_ReleaseStateSchema,
  TaskRun_Status,
  type TaskRunLogEntry,
  TaskRunLogEntry_PriorBackup_PriorBackupDetail_ItemSchema,
  TaskRunLogEntry_Type,
  TaskRunLogEntrySchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { type Sheet, SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import {
  assignEntryKeys,
  type BuildSectionsOptions,
  buildReleaseFileGroups,
  buildReleaseSpine,
  buildRowsFromEntries,
  collectLeafSections,
  countSpine,
  groupEntriesByReleaseFile,
  hasReleaseFileMarkers,
  isSpineItemOpenByDefault,
  parseReleaseFileError,
  pendingVersionsOf,
  type TaskRunLogDetailText,
  windowSpine,
} from "./model";
import {
  begin,
  commit,
  databaseSync,
  failedAttempt,
  LOCK_TIMEOUT,
  passedAttempt,
  priorBackup,
  retry,
  rollback,
} from "./taskRunLogEntries";
import type {
  AttemptGroup,
  LogRow,
  Section,
  SpineItem,
  SpineWindowItem,
} from "./types";
import {
  buildReleaseSheetFetchResult,
  buildSheetFetchStateForMissingTask,
  getUnresolvedTaskMetadataStateKey,
} from "./useTaskRunLogData";

vi.mock("@/api", () => ({
  rolloutServiceClientConnect: {},
  sheetServiceClientConnect: {},
}));

vi.mock("@/stores/app", () => ({
  useAppStore: () => vi.fn(),
}));

vi.mock("@/utils", () => ({
  extractRolloutNameFromTaskRunName: () => "",
  extractTaskNameFromTaskRunName: () => "",
  isReleaseBasedTask: () => false,
  releaseNameOfTaskV1: () => "",
  sheetNameOfTaskV1: () => "",
}));

const ts = (seconds: number, nanos = 0) => ({
  seconds: BigInt(seconds),
  nanos,
});

const buildRows = (
  entries: TaskRunLogEntry[],
  options?: Partial<BuildSectionsOptions>
) =>
  buildRowsFromEntries(entries, {
    getSectionLabel: (type) => String(type),
    entryKeys: assignEntryKeys(entries),
    ...options,
  });

// Leaf sections in render order, superseded attempts included.
const buildSections = (
  entries: TaskRunLogEntry[],
  options?: Partial<BuildSectionsOptions>
) => collectLeafSections(buildRows(entries, options));

interface CommandOptions {
  statement?: string;
  range?: { start: number; end: number };
  // Omit for a command still running; "" for one that succeeded.
  error?: string;
  replicaId?: string;
  nanos?: number;
}

const command = (seconds: number, options: CommandOptions = {}) =>
  create(TaskRunLogEntrySchema, {
    type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
    logTime: ts(seconds, options.nanos),
    replicaId: options.replicaId ?? "",
    commandExecute: {
      logTime: ts(seconds, options.nanos),
      statement: options.statement ?? "",
      range: options.range,
      response:
        options.error === undefined
          ? undefined
          : { logTime: ts(seconds + 1), error: options.error },
    },
  });

const transaction = (seconds: number, replicaId = "") =>
  create(TaskRunLogEntrySchema, {
    type: TaskRunLogEntry_Type.TRANSACTION_CONTROL,
    logTime: ts(seconds),
    replicaId,
    transactionControl: {},
  });

const retryMarker = (seconds: number, replicaId = "") =>
  create(TaskRunLogEntrySchema, {
    type: TaskRunLogEntry_Type.RETRY_INFO,
    logTime: ts(seconds),
    replicaId,
    retryInfo: { error: "lock timeout", retryCount: 1, maximumRetries: 3 },
  });

const releaseFile = (
  seconds: number,
  version: string,
  options: { replicaId?: string; nanos?: number } = {}
) =>
  create(TaskRunLogEntrySchema, {
    type: TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE,
    logTime: ts(seconds, options.nanos),
    replicaId: options.replicaId ?? "",
    releaseFileExecute: { version, filePath: `${version}.sql` },
  });

const sheetOf = (content: string, contentSize?: number): Sheet => {
  const bytes = new TextEncoder().encode(content);
  return create(SheetSchema, {
    content: bytes,
    contentSize: BigInt(contentSize ?? bytes.byteLength),
  });
};

const markedDetails = (
  sections: { items: { marked?: boolean; detail: string }[] }[]
) =>
  sections.flatMap((section) =>
    section.items.filter((item) => item.marked).map((item) => item.detail)
  );

describe("task-run-log model", () => {
  test("keeps unresolved task sheet state non-error while metadata is pending", () => {
    const metadataIdleState = buildSheetFetchStateForMissingTask(
      "tasks/run-1",
      {
        status: "idle",
      }
    );
    expect(metadataIdleState).toMatchObject({
      status: "loading",
      source: "none",
    });

    const unresolvedTaskState = buildSheetFetchStateForMissingTask(
      "tasks/run-1",
      {
        status: "success",
      }
    );
    expect(unresolvedTaskState).toMatchObject({
      status: "error",
      source: "none",
      error: "Task cannot be resolved from rollout metadata",
    });

    const metadataLoadingState = buildSheetFetchStateForMissingTask(
      "tasks/run-1",
      {
        status: "loading",
      }
    );
    expect(metadataLoadingState).toMatchObject({
      status: "loading",
      source: "none",
    });
  });

  test("ignores metadata transitions for resolved task fetch dependencies", () => {
    const loadingKey = getUnresolvedTaskMetadataStateKey(true, {
      status: "loading",
    });
    const successKey = getUnresolvedTaskMetadataStateKey(true, {
      status: "success",
    });
    const errorKey = getUnresolvedTaskMetadataStateKey(true, {
      status: "error",
      error: "boom",
    });

    expect(loadingKey).toBe("resolved");
    expect(successKey).toBe("resolved");
    expect(errorKey).toBe("resolved");
  });

  test("marks release sheet fetch as partial when some versions fail", () => {
    const result = buildReleaseSheetFetchResult([
      { version: "v1", sheet: {} as Sheet },
      { version: "v2" },
    ]);

    expect(result.sheetsMap.size).toBe(1);
    expect(result.sheetsMap.has("v1")).toBe(true);
    expect(result.state).toMatchObject({
      status: "partial",
      source: "release",
      failedReleaseVersions: ["v2"],
    });
  });

  test("groups release-file entries by version marker", () => {
    const entries = [
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE,
        logTime: ts(1),
        releaseFileExecute: { version: "v1", filePath: "001.sql" },
      }),
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
        logTime: ts(2),
        commandExecute: {
          statement: "ALTER TABLE book ADD COLUMN title TEXT;",
          response: { logTime: ts(3) },
        },
      }),
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
        logTime: ts(4),
        commandExecute: {
          statement: "ALTER TABLE book ADD COLUMN author TEXT;",
          response: { logTime: ts(5) },
        },
      }),
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE,
        logTime: ts(6),
        releaseFileExecute: { version: "v2", filePath: "002.sql" },
      }),
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
        logTime: ts(7),
        commandExecute: {
          statement: "CREATE INDEX idx_book_author ON book(author);",
          response: { logTime: ts(8) },
        },
      }),
    ];

    expect(hasReleaseFileMarkers(entries)).toBe(true);
    const groups = buildReleaseFileGroups(entries, {
      getSectionLabel: (type) => String(type),
      entryKeys: assignEntryKeys(entries),
    });
    expect(groups).toHaveLength(2);
    expect(groups[0]).toMatchObject({
      id: "file-0",
      version: "v1",
      filePath: "001.sql",
      rows: [{ entryCount: 2 }],
    });
    expect(groups[1]).toMatchObject({
      id: "file-1",
      version: "v2",
      filePath: "002.sql",
      rows: [{ entryCount: 1 }],
    });
  });

  test("marks incomplete sections as running and errored sections as error", () => {
    const entries = [
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
        logTime: ts(10),
        commandExecute: { statement: "SELECT 1;" },
      }),
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.TRANSACTION_CONTROL,
        logTime: ts(11),
        transactionControl: { error: "rollback failed" },
      }),
    ];

    const sections = buildSections(entries);

    expect(sections[0]?.status).toBe("running");
    expect(sections[1]?.status).toBe("error");
  });

  test("gives a line with no log time no instant to show", () => {
    // The line still reads as a row -- its time column says so -- but there is
    // no instant behind it, and a zero would have been an instant.
    const sections = buildSections([
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
        logTime: ts(10),
        commandExecute: {},
      }),
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
        commandExecute: {},
      }),
    ]);

    // Which line got which instant, not how many of each: one undefined and
    // one number is also what swapping them produces.
    // The column a reader sees, paired with the instant behind it: both come
    // from one entry, and one undefined plus one number is also what swapping
    // them produces.
    expect(
      sections
        .flatMap((section) => section.items)
        .map((item) => [item.time, item.timeMs])
    ).toEqual([
      ["--:--:--.---", undefined],
      ["08:00:10.000", 10_000],
    ]);
  });

  test("renders gh-ost migration as a timed section", () => {
    const detailText = {
      completed: "Completed",
    } satisfies TaskRunLogDetailText;

    const entries = [
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.GHOST_MIGRATION,
        logTime: ts(10),
        ghostMigration: {
          startTime: ts(10),
          endTime: ts(13),
          error: "",
        },
      }),
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.GHOST_MIGRATION,
        logTime: ts(20),
        ghostMigration: {
          startTime: ts(20),
          error: "copy failed",
        },
      }),
    ];

    const sections = buildSections(entries, { detailText });

    expect(sections[0]?.status).toBe("success");
    expect(sections[0]?.duration).toBe("3.0s");
    expect(sections[0]?.items[0]?.detail).toBe("Completed");
    expect(sections[1]?.status).toBe("error");
    expect(sections[1]?.items[0]?.detail).toBe("copy failed");
  });

  test("uses localized detail text for completed timed entries and prior backup completion", () => {
    const detailText = {
      completed: "Completed",
      backingUp: "Backing up...",
      runningByType: {
        [TaskRunLogEntry_Type.SCHEMA_DUMP]: "Dumping...",
      },
      backupCompleted: (count: number) => `Completed (${count} tables)`,
    } satisfies TaskRunLogDetailText;

    const entries = [
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.SCHEMA_DUMP,
        logTime: ts(1),
        schemaDump: {
          startTime: ts(1),
          endTime: ts(2),
          error: "",
        },
      }),
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.PRIOR_BACKUP,
        logTime: ts(3),
        priorBackup: {
          startTime: ts(3),
          endTime: ts(4),
          priorBackupDetail: {
            items: [
              create(
                TaskRunLogEntry_PriorBackup_PriorBackupDetail_ItemSchema,
                {}
              ),
              create(
                TaskRunLogEntry_PriorBackup_PriorBackupDetail_ItemSchema,
                {}
              ),
            ],
          },
          error: "",
        },
      }),
    ];

    const sections = buildSections(entries, { detailText });

    expect(sections[0]?.items[0]?.detail).toBe("Completed");
    expect(sections[1]?.items[0]?.detail).toBe("Completed (2 tables)");
  });

  test("drops empty release-file groups created by consecutive and trailing markers", () => {
    const entries = [
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
        logTime: ts(1),
        commandExecute: {
          statement: "SELECT 1;",
          response: { logTime: ts(2) },
        },
      }),
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE,
        logTime: ts(3),
        releaseFileExecute: { version: "v1", filePath: "001.sql" },
      }),
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE,
        logTime: ts(4),
        releaseFileExecute: { version: "v2", filePath: "002.sql" },
      }),
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
        logTime: ts(5),
        commandExecute: {
          statement: "ALTER TABLE t ADD COLUMN c INT;",
          response: { logTime: ts(6) },
        },
      }),
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE,
        logTime: ts(7),
        releaseFileExecute: { version: "v3", filePath: "003.sql" },
      }),
    ];

    const groupedEntries = groupEntriesByReleaseFile(entries);
    expect(groupedEntries).toHaveLength(4);
    expect(groupedEntries[0]).toMatchObject({ file: null });
    expect(groupedEntries[0]?.entries).toHaveLength(1);
    expect(groupedEntries[1]).toMatchObject({
      file: { version: "v1", filePath: "001.sql" },
    });
    expect(groupedEntries[1]?.entries).toHaveLength(0);
    expect(groupedEntries[2]).toMatchObject({
      file: { version: "v2", filePath: "002.sql" },
    });
    expect(groupedEntries[2]?.entries).toHaveLength(1);
    expect(groupedEntries[3]).toMatchObject({
      file: { version: "v3", filePath: "003.sql" },
    });
    expect(groupedEntries[3]?.entries).toHaveLength(0);

    const groups = buildReleaseFileGroups(entries, {
      getSectionLabel: (type) => String(type),
      entryKeys: assignEntryKeys(entries),
      includeOrphanGroup: true,
    });
    expect(groups).toHaveLength(2);
    expect(groups[0]).toMatchObject({
      id: "orphan",
      isOrphan: true,
      version: "",
      filePath: "",
      rows: [{ entryCount: 1 }],
    });
    expect(groups[1]).toMatchObject({
      id: "file-0",
      version: "v2",
      filePath: "002.sql",
      rows: [{ entryCount: 1 }],
    });
  });
});

const DEADLOCK = "ERROR: deadlock detected";

describe("task-run-log attempt rows", () => {
  test("an empty log has no rows", () => {
    expect(buildRows([])).toEqual([]);
  });

  test("without a retry marker every row is a section with positional ids", () => {
    const rows = buildRows([
      begin(1),
      command(2, { error: "" }),
      commit(4),
      databaseSync(5, 7),
    ]);

    expect(rows.map((row) => row.kind)).toEqual([
      "section",
      "section",
      "section",
      "section",
    ]);
    expect(rows.map((row) => row.id)).toEqual([
      "section-0",
      "section-1",
      "section-2",
      "section-3",
    ]);
  });

  test("one retry marker folds the attempt before it into a Previous attempts row", () => {
    const rows = buildRows([
      begin(1),
      command(1, { error: LOCK_TIMEOUT }),
      rollback(2),
      retry(3, 1),
      begin(4),
      command(4, { error: "" }),
      commit(6),
      databaseSync(6, 8),
    ]);

    expect(rows.map((row) => row.kind)).toEqual([
      "attempts",
      "section",
      "section",
      "section",
      "section",
    ]);
    const umbrella = rows[0] as AttemptGroup;
    expect(umbrella).toMatchObject({
      id: "attempts-0",
      duration: "1.0s",
      retrying: false,
    });
    expect(umbrella.attempts).toHaveLength(1);
    expect(umbrella.attempts[0]).toMatchObject({
      id: "attempts-0-attempt-1",
      number: 1,
      reason: LOCK_TIMEOUT,
      duration: "1.0s",
    });
    expect(
      umbrella.attempts[0]?.sections.map((section) => [
        section.id,
        section.type,
        section.status,
      ])
    ).toEqual([
      [
        "attempts-0-attempt-1-section-0",
        TaskRunLogEntry_Type.TRANSACTION_CONTROL,
        "success",
      ],
      [
        "attempts-0-attempt-1-section-1",
        TaskRunLogEntry_Type.COMMAND_EXECUTE,
        "error",
      ],
      [
        "attempts-0-attempt-1-section-2",
        TaskRunLogEntry_Type.TRANSACTION_CONTROL,
        "success",
      ],
    ]);

    // The final attempt stays flat and green, ids continuing past the umbrella.
    expect(
      rows.slice(1).map((row) => [row.id, (row as Section).status])
    ).toEqual([
      ["section-1", "success"],
      ["section-2", "success"],
      ["section-3", "success"],
      ["section-4", "success"],
    ]);

    const leaves = collectLeafSections(rows);
    expect(leaves.some((s) => s.type === TaskRunLogEntry_Type.RETRY_INFO)).toBe(
      false
    );
    expect(leaves.reduce((sum, s) => sum + s.entryCount, 0)).toBe(7);
  });

  test("an auto-commit attempt, with no transaction entries, folds like any other", () => {
    const rows = buildRows([
      command(1, { error: "" }),
      command(2, { error: LOCK_TIMEOUT }),
      retry(4, 1),
      command(5, { error: "" }),
      command(6, { error: "" }),
    ]);

    expect(rows.map((row) => [row.id, row.kind])).toEqual([
      ["attempts-0", "attempts"],
      ["section-1", "section"],
    ]);
    expect(
      (rows[0] as AttemptGroup).attempts[0]?.sections.map((section) => [
        section.entryCount,
        section.status,
      ])
    ).toEqual([[2, "error"]]);
    expect((rows[1] as Section).status).toBe("success");
  });

  test("several markers number the attempts and label each with the marker that closes it", () => {
    const rows = buildRows([
      ...failedAttempt(1, 1),
      // Numbered by position, not by the marker's retryCount.
      ...failedAttempt(4, 9, { error: DEADLOCK }),
      ...passedAttempt(7),
    ]);

    expect(rows.map((row) => row.kind)).toEqual([
      "attempts",
      "section",
      "section",
      "section",
    ]);
    const umbrella = rows[0] as AttemptGroup;
    expect(umbrella.duration).toBe("2.0s");
    expect(
      umbrella.attempts.map((attempt) => [
        attempt.number,
        attempt.reason,
        attempt.duration,
        attempt.sections.length,
      ])
    ).toEqual([
      [1, LOCK_TIMEOUT, "1.0s", 3],
      [2, DEADLOCK, "1.0s", 3],
    ]);
  });

  test("prior backup and database sync stay outside the attempts, in place", () => {
    const rows = buildRows([
      priorBackup(1, 3),
      ...failedAttempt(4, 1),
      ...passedAttempt(7),
      databaseSync(10, 14),
    ]);

    expect(
      rows.map((row) => [
        row.kind,
        row.kind === "section" ? row.type : row.attempts.length,
        row.duration,
      ])
    ).toEqual([
      ["section", TaskRunLogEntry_Type.PRIOR_BACKUP, "2.0s"],
      ["attempts", 1, "1.0s"],
      ["section", TaskRunLogEntry_Type.TRANSACTION_CONTROL, "<1ms"],
      ["section", TaskRunLogEntry_Type.COMMAND_EXECUTE, "1.0s"],
      ["section", TaskRunLogEntry_Type.TRANSACTION_CONTROL, "<1ms"],
      ["section", TaskRunLogEntry_Type.DATABASE_SYNC, "4.0s"],
    ]);
  });

  test("a marker with nothing after it renders the umbrella alone, not retrying", () => {
    const rows = buildRows([priorBackup(1, 3), ...failedAttempt(4, 1)]);

    expect(rows.map((row) => row.kind)).toEqual(["section", "attempts"]);
    expect(rows[1]).toMatchObject({ retrying: false });
    expect((rows[1] as AttemptGroup).attempts).toHaveLength(1);
  });

  test("a final segment still running marks the umbrella as retrying", () => {
    const rows = buildRows([
      ...failedAttempt(1, 1),
      ...failedAttempt(4, 2),
      begin(7),
      command(7),
    ]);

    expect(rows.map((row) => row.kind)).toEqual([
      "attempts",
      "section",
      "section",
    ]);
    expect(rows[0]).toMatchObject({ retrying: true });
    expect((rows[0] as AttemptGroup).attempts).toHaveLength(2);
    expect((rows[2] as Section).status).toBe("running");
  });

  test("each release file that retried gets its own umbrella with its own count", () => {
    const entries = [
      releaseFile(1, "v1"),
      ...failedAttempt(2, 1),
      ...passedAttempt(5),
      releaseFile(8, "v2"),
      ...passedAttempt(9),
      releaseFile(12, "v3"),
      ...failedAttempt(13, 1),
      ...failedAttempt(16, 2),
      ...passedAttempt(19),
    ];
    const groups = buildReleaseFileGroups(entries, {
      getSectionLabel: (type) => String(type),
      entryKeys: assignEntryKeys(entries),
    });

    const shape = (row: LogRow) =>
      row.kind === "attempts" ? `attempts:${row.attempts.length}` : "section";
    expect(
      groups.map((group) => [group.version, group.rows.map(shape)])
    ).toEqual([
      ["v1", ["attempts:1", "section", "section", "section"]],
      ["v2", ["section", "section", "section"]],
      ["v3", ["attempts:2", "section", "section", "section"]],
    ]);
    // Ids nest under the file so two files' umbrellas never collide.
    expect(groups[0]?.rows[0]?.id).toBe("file-0-attempts-0");
    const thirdFileUmbrella = groups[2]?.rows[0] as AttemptGroup | undefined;
    expect(thirdFileUmbrella?.id).toBe("file-2-attempts-0");
    expect(thirdFileUmbrella?.attempts[1]?.id).toBe(
      "file-2-attempts-0-attempt-2"
    );
  });
});

describe("command row payloads", () => {
  const formatted =
    "\nCREATE TABLE public.loan_application (\n  id bigint PRIMARY KEY,\n  application_no text NOT NULL\n);\n";

  test("a multi-line statement yields a one-line detail and a verbatim statement", () => {
    const item = buildSections([
      command(1, { statement: formatted, error: "" }),
    ])[0]?.items[0];

    expect(item?.detail).toBe(
      "CREATE TABLE public.loan_application ( id bigint PRIMARY KEY, application_no text NOT NULL );"
    );
    expect(item?.statement).toBe(formatted);
    expect(item?.error).toBeUndefined();
  });

  test("a statement past 80 characters is not cut", () => {
    const long = `SELECT ${"column_name, ".repeat(20)}1;`;
    const item = buildSections([command(1, { statement: long, error: "" })])[0]
      ?.items[0];

    expect(long.length).toBeGreaterThan(80);
    expect(item?.detail).toBe(long);
  });

  test("a failed command keeps the error as its line and still carries the statement", () => {
    const item = buildSections([
      command(1, { statement: formatted, error: "ERROR: relation exists" }),
    ])[0]?.items[0];

    expect(item?.detail).toBe("ERROR: relation exists");
    expect(item?.error).toBe("ERROR: relation exists");
    expect(item?.statement).toBe(formatted);
  });

  test("a failed command recovers its statement from the sheet range", () => {
    const content = "SELECT 1;\nALTER TABLE t\n  ADD COLUMN c int;";
    const item = buildSections(
      [
        command(1, {
          range: { start: 9, end: content.length },
          error: "ERROR: column exists",
        }),
      ],
      { sheet: sheetOf(content) }
    )[0]?.items[0];

    expect(item?.detail).toBe("ERROR: column exists");
    expect(item?.statement).toBe("\nALTER TABLE t\n  ADD COLUMN c int;");
  });

  test("an entry with no statement yields a dash and no payload", () => {
    const item = buildSections([command(1, { error: "" })])[0]?.items[0];

    expect(item?.detail).toBe("-");
    expect(item?.statement).toBeUndefined();
    expect(item?.error).toBeUndefined();
  });

  test("a range reaching past a partial sheet recovers nothing", () => {
    const item = buildSections(
      [command(1, { range: { start: 0, end: 40 }, error: "" })],
      { sheet: sheetOf("SELECT 1;", 40) }
    )[0]?.items[0];

    expect(item?.detail).toBe("-");
    expect(item?.statement).toBeUndefined();
  });

  test("entries that carry status words return the detail alone", () => {
    const sections = buildSections([transaction(1), databaseSync(2, 3)]);

    for (const section of sections) {
      expect(section.items[0]?.statement).toBeUndefined();
      expect(section.items[0]?.error).toBeUndefined();
    }
  });
});

describe("the row that opens by default", () => {
  const attempt = (start: number, error: string) => [
    transaction(start),
    command(start + 1, { statement: "ALTER TABLE t ADD COLUMN c int;", error }),
    transaction(start + 3),
  ];

  test("a failure the driver retried successfully marks nothing", () => {
    const sections = buildSections([
      ...attempt(1, "lock timeout"),
      retryMarker(5),
      ...attempt(6, ""),
    ]);

    // The failure is the last entry of its own section, which is why the pick
    // cannot be section-local.
    expect(sections[1]?.items.at(-1)?.error).toBe("lock timeout");
    expect(markedDetails(sections)).toEqual([]);
  });

  test("a retry that fails again marks only the last failure", () => {
    const sections = buildSections([
      ...attempt(1, "lock timeout"),
      retryMarker(5),
      ...attempt(6, "lock timeout again"),
    ]);

    expect(markedDetails(sections)).toEqual(["lock timeout again"]);
  });

  test("a live failure already followed by the retry marker is not terminal", () => {
    const sections = buildSections([
      ...attempt(1, "lock timeout"),
      retryMarker(5),
    ]);

    expect(markedDetails(sections)).toEqual([]);
  });

  test("a failure followed by a command that succeeded marks nothing", () => {
    // No retry marker here: the later success alone says the run moved on.
    const sections = buildSections([
      command(1, { statement: "SELECT 1;", error: "boom" }),
      command(3, { statement: "SELECT 2;", error: "" }),
    ]);

    expect(markedDetails(sections)).toEqual([]);
  });

  test("a command still running after a failure does not supersede it", () => {
    const sections = buildSections([
      command(1, { statement: "SELECT 1;", error: "boom" }),
      command(3, { statement: "SELECT 2;" }),
    ]);

    expect(markedDetails(sections)).toEqual(["boom"]);
  });

  test("a failure with nothing after it is marked", () => {
    const sections = buildSections([
      command(1, { statement: "SELECT 1;", error: "" }),
      command(3, { statement: "SELECT 2;", error: "boom" }),
    ]);

    expect(markedDetails(sections)).toEqual(["boom"]);
  });

  test("a failure with no recoverable statement is still marked", () => {
    const sections = buildSections([command(1, { error: "boom" })]);

    expect(sections[0]?.items[0]).toMatchObject({
      detail: "boom",
      marked: true,
    });
    expect(sections[0]?.items[0]?.statement).toBeUndefined();
  });

  test("a DONE run marks nothing, whatever the entries say", () => {
    const sections = buildSections(
      [command(1, { statement: "SELECT 1;", error: "serialization failure" })],
      { taskRunStatus: TaskRun_Status.DONE }
    );

    expect(markedDetails(sections)).toEqual([]);
  });
});

describe("row identity", () => {
  // The SDL shape: the statement is logged instead of a range, so tied entries
  // have no byte offset to tell them apart.
  const tied = (statement: string) =>
    command(5, { statement, error: "", nanos: 123456789, replicaId: "a" });

  test("entries sharing a replica, a timestamp and a type get distinct keys", () => {
    const items = buildSections([tied("SELECT 1;"), tied("SELECT 2;")])[0]
      ?.items;

    expect(items).toHaveLength(2);
    expect(items?.[0]?.key).not.toBe(items?.[1]?.key);
  });

  test("tied entries in different release files get distinct keys", () => {
    const entries = [
      releaseFile(1, "v1"),
      tied("SELECT 1;"),
      releaseFile(5, "v2", { nanos: 123456789 }),
      tied("SELECT 2;"),
    ];
    const groups = buildReleaseFileGroups(entries, {
      getSectionLabel: (type) => String(type),
      entryKeys: assignEntryKeys(entries),
    });

    expect(groups).toHaveLength(2);
    const [first, second] = groups.map(
      (group) => collectLeafSections(group.rows)[0]?.items[0]?.key
    );
    expect(first).toBeDefined();
    expect(first).not.toBe(second);
  });

  test("sub-millisecond log times are told apart", () => {
    const items = buildSections([
      command(5, { statement: "SELECT 1;", error: "", nanos: 100 }),
      command(5, { statement: "SELECT 2;", error: "", nanos: 200 }),
    ])[0]?.items;

    expect(items?.[0]?.key).not.toBe(items?.[1]?.key);
    expect(items?.[0]?.key.endsWith(":0")).toBe(true);
    expect(items?.[1]?.key.endsWith(":0")).toBe(true);
  });

  test("a builder refuses entries that were never given a key", () => {
    const stranger = command(1, { statement: "SELECT 1;", error: "" });

    expect(() =>
      buildRowsFromEntries([stranger], {
        getSectionLabel: (type) => String(type),
        entryKeys: assignEntryKeys([]),
      })
    ).toThrow();
  });

  test("a row's key survives a retry marker splitting its section", () => {
    const commands = () => [
      command(1, { statement: "SELECT 1;", error: "" }),
      command(5, { statement: "SELECT 2;", error: "" }),
    ];
    const before = buildSections(commands());
    expect(before).toHaveLength(1);

    const after = buildSections([...commands(), retryMarker(3)]);
    expect(after).toHaveLength(2);
    expect(after[1]?.items[0]?.key).toBe(before[0]?.items[1]?.key);
    expect(after[0]?.items[0]?.key).toBe(before[0]?.items[0]?.key);
  });
});

describe("release spine grouping", () => {
  test("a file group ends at a database-sync entry, which becomes a sibling", () => {
    const sync = databaseSync(9, 10);
    const entries = [
      releaseFile(1, "001"),
      command(2, { statement: "CREATE TABLE a (id INT);", error: "" }),
      releaseFile(4, "002"),
      command(5, { statement: "CREATE TABLE b (id INT);", error: "" }),
      sync,
    ];
    const groups = groupEntriesByReleaseFile(entries);
    expect(
      groups.map((g) => [g.file?.version ?? null, g.entries.length])
    ).toEqual([
      ["001", 1],
      ["002", 1],
      [null, 1],
    ]);
    expect(groups[2]?.entries[0]).toBe(sync);
  });

  test("a prior backup is a sibling and a gh-ost migration stays inside its file", () => {
    const backup = priorBackup(1, 2);
    const ghost = create(TaskRunLogEntrySchema, {
      type: TaskRunLogEntry_Type.GHOST_MIGRATION,
      logTime: ts(5),
      ghostMigration: { startTime: ts(5), endTime: ts(6), error: "" },
    });
    const entries = [backup, releaseFile(3, "001"), ghost, databaseSync(7, 8)];
    const groups = groupEntriesByReleaseFile(entries);
    expect(
      groups.map((g) => [g.file?.version ?? null, g.entries.length])
    ).toEqual([
      [null, 1],
      ["001", 1],
      [null, 1],
    ]);
    expect(groups[1]?.entries[0]).toBe(ghost);
  });
});

const FILES = ["001", "002", "003", "004", "005"].map((version) => ({
  version,
  path: `${version}.sql`,
}));
const stateOf = (pending: string[], deleted: Record<string, string> = {}) =>
  create(Task_ReleaseStateSchema, {
    appliedCount: FILES.length - pending.length,
    pendingRanges: FILES.flatMap((file, i) =>
      pending.includes(file.version) ? [{ start: i, end: i + 1 }] : []
    ),
    deletedRevisions: pending
      .filter((version) => deleted[version])
      .map((version) => ({
        version,
        deleter: deleted[version],
        deleteTime: ts(100),
      })),
  });
// The spine as [kind, version or label, status] rows, top to bottom.
const shape = (spine: ReturnType<typeof buildReleaseSpine>) =>
  spine.items.map((item) => {
    switch (item.kind) {
      case "applied-earlier":
        return ["earlier", item.files.map((f) => f.version).join(",")];
      case "file":
        return ["file", item.version, item.status];
      case "step":
        return ["step", String(item.type)];
      case "run-failed":
        return ["run-failed"];
    }
  });
const SYNC = String(TaskRunLogEntry_Type.DATABASE_SYNC);
const FILE_ERROR = (version: string, message: string) =>
  `failed to execute release file ${version}.sql (version ${version}): ${message}`;

describe("release file error", () => {
  test("the path ends at the first version marker, so a message quoting SQL with one keeps its file", () => {
    expect(
      parseReleaseFileError(
        "failed to execute release file m/2_a.sql (version 2): Error 1064: near '-- (version 9): x' at line 1"
      )
    ).toEqual({
      path: "m/2_a.sql",
      version: "2",
      message: "Error 1064: near '-- (version 9): x' at line 1",
    });
  });
});

describe("stored release state", () => {
  test("ranges name positions in the release's files, and a range past the list stops at its end", () => {
    const state = create(Task_ReleaseStateSchema, {
      pendingRanges: [
        { start: 0, end: 1 },
        { start: 3, end: 9 },
      ],
    });
    expect([...pendingVersionsOf(state, FILES)]).toEqual(["001", "004", "005"]);
    expect(pendingVersionsOf(undefined, FILES).size).toBe(0);
  });
});

describe("release spine", () => {
  test("before the first run it lists the stored state: the files applied earlier, then what will apply", () => {
    const spine = buildReleaseSpine({
      files: FILES,
      entries: [],
      state: stateOf(["004", "005"], { "004": "users/alex@example.com" }),
    });
    expect(shape(spine)).toEqual([
      ["earlier", "001,002,003"],
      ["file", "004", "pending"],
      ["file", "005", "pending"],
    ]);
    const reapplied = spine.items[1];
    expect(
      reapplied?.kind === "file" && reapplied.revisionDeleted
    ).toMatchObject({
      deleter: "users/alex@example.com",
    });
  });

  test("a done run counts every file without a marker as applied earlier and applies every marked file", () => {
    const spine = buildReleaseSpine({
      files: FILES,
      entries: [
        releaseFile(1, "004"),
        command(2, { error: "" }),
        releaseFile(4, "005"),
        command(5, { error: "" }),
        databaseSync(7, 8),
      ],
      run: { status: TaskRun_Status.DONE, error: "" },
    });
    expect(shape(spine)).toEqual([
      ["earlier", "001,002,003"],
      ["file", "004", "applied"],
      ["file", "005", "applied"],
      ["step", SYNC],
    ]);
  });

  test("a done run with no marker has everything applied", () => {
    const spine = buildReleaseSpine({
      files: FILES,
      entries: [databaseSync(1, 2)],
      run: { status: TaskRun_Status.DONE, error: "" },
    });
    expect(shape(spine)).toEqual([
      ["earlier", "001,002,003,004,005"],
      ["step", SYNC],
    ]);
  });

  test("a failed run counts the unmarked files below its newest marker as applied earlier and lists nothing after the failed file", () => {
    const spine = buildReleaseSpine({
      files: FILES,
      entries: [
        releaseFile(1, "002"),
        command(2, { error: "" }),
        releaseFile(4, "003"),
        command(5, {
          statement: "ALTER TABLE t ADD c INT;",
          error: "column exists",
        }),
        databaseSync(7, 8),
      ],
      run: {
        status: TaskRun_Status.FAILED,
        error: FILE_ERROR("003", "column exists"),
      },
      state: stateOf(["002", "003", "004", "005"]),
    });
    expect(shape(spine)).toEqual([
      ["earlier", "001"],
      ["file", "002", "applied"],
      ["file", "003", "failed"],
      ["step", SYNC],
    ]);
    const failed = spine.items[2];
    expect(failed?.kind === "file" && failed.error).toBe("column exists");
    expect(failed?.kind === "file" && failed.noStatement).toBe(false);
  });

  test("a running run lists the files it has not reached as will apply, the boundary at the current file", () => {
    const spine = buildReleaseSpine({
      files: FILES,
      entries: [
        releaseFile(1, "002"),
        command(2, { error: "" }),
        releaseFile(4, "003"),
        command(5),
      ],
      run: { status: TaskRun_Status.RUNNING, error: "" },
      state: stateOf(["002", "003", "004", "005"]),
    });
    expect(shape(spine)).toEqual([
      ["earlier", "001"],
      ["file", "002", "applied"],
      ["file", "003", "running"],
      ["file", "004", "pending"],
      ["file", "005", "pending"],
    ]);
  });

  test("a stored will-apply version the run passed over joins the files applied earlier, and a stored applied version with a marker leaves them", () => {
    // The state said 002 and 004 would apply; 002 had been applied by another
    // plan, and 003's revision was deleted after the rollout was created.
    const spine = buildReleaseSpine({
      files: FILES,
      entries: [
        releaseFile(1, "003"),
        command(2, { error: "" }),
        releaseFile(4, "004"),
        command(5),
      ],
      run: { status: TaskRun_Status.RUNNING, error: "" },
      state: stateOf(["002", "004"]),
    });
    expect(shape(spine)).toEqual([
      ["earlier", "001,002,005"],
      ["file", "003", "applied"],
      ["file", "004", "running"],
    ]);
  });

  test("a run that failed before any marker takes the stored applied versions and ends with Run failed", () => {
    const error =
      'failed to get driver connection for instance "prod": connection refused';
    const withState = buildReleaseSpine({
      files: FILES,
      entries: [],
      run: { status: TaskRun_Status.FAILED, error },
      state: stateOf(["004", "005"]),
    });
    expect(shape(withState)).toEqual([
      ["earlier", "001,002,003"],
      ["run-failed"],
    ]);
    expect(withState.items.at(-1)).toMatchObject({ kind: "run-failed", error });

    const withoutState = buildReleaseSpine({
      files: FILES,
      entries: [],
      run: { status: TaskRun_Status.FAILED, error },
    });
    expect(shape(withoutState)).toEqual([["run-failed"]]);
  });

  test("a run that failed loading the second file's statement keeps the first file applied and ends with Run failed", () => {
    const run = {
      status: TaskRun_Status.FAILED,
      error: "failed to get sheet abc for version 005: not found",
    };
    const entries = [releaseFile(1, "004"), command(2, { error: "" })];
    expect(
      shape(
        buildReleaseSpine({
          files: FILES,
          entries,
          run,
          state: stateOf(["004", "005"]),
        })
      )
    ).toEqual([
      ["earlier", "001,002,003"],
      ["file", "004", "applied"],
      ["run-failed"],
    ]);
    // Without a stored state only the versions below the newest marker are known.
    expect(shape(buildReleaseSpine({ files: FILES, entries, run }))).toEqual([
      ["earlier", "001,002,003"],
      ["file", "004", "applied"],
      ["run-failed"],
    ]);
  });

  test("a revision-write failure after a file ran leaves the file applied and the error in Run failed", () => {
    const spine = buildReleaseSpine({
      files: FILES,
      entries: [releaseFile(1, "005"), command(2, { error: "" })],
      run: {
        status: TaskRun_Status.FAILED,
        error: "failed to create revision for version 005: duplicate key",
      },
      state: stateOf(["005"]),
    });
    expect(shape(spine)).toEqual([
      ["earlier", "001,002,003,004"],
      ["file", "005", "applied"],
      ["run-failed"],
    ]);
  });

  test("a file error with no failed command says no statement was identified", () => {
    const spine = buildReleaseSpine({
      files: FILES,
      entries: [releaseFile(1, "005")],
      run: {
        status: TaskRun_Status.FAILED,
        error: FILE_ERROR(
          "005",
          "ORA-00942: table or view does not exist\nmore"
        ),
      },
      state: stateOf(["005"]),
    });
    const failed = spine.items.at(-1);
    expect(failed).toMatchObject({
      kind: "file",
      status: "failed",
      noStatement: true,
      error: "ORA-00942: table or view does not exist\nmore",
    });
  });

  test("a failed prior backup owns the error, so there is no Run failed section", () => {
    const failedBackup = create(TaskRunLogEntrySchema, {
      type: TaskRunLogEntry_Type.PRIOR_BACKUP,
      logTime: ts(1),
      priorBackup: { startTime: ts(1), endTime: ts(2), error: "backup failed" },
    });
    const spine = buildReleaseSpine({
      files: FILES,
      entries: [failedBackup],
      run: {
        status: TaskRun_Status.FAILED,
        error: "failed to back up: backup failed",
      },
      state: stateOf(["005"]),
    });
    expect(shape(spine)).toEqual([
      ["earlier", "001,002,003,004"],
      ["step", String(TaskRunLogEntry_Type.PRIOR_BACKUP)],
    ]);
  });

  test("a canceled run marks the section it interrupted as canceled", () => {
    const spine = buildReleaseSpine({
      files: FILES,
      entries: [
        releaseFile(1, "004"),
        command(2, { error: "" }),
        releaseFile(4, "005"),
        command(5),
      ],
      run: { status: TaskRun_Status.CANCELED, error: "canceled" },
      state: stateOf(["004", "005"]),
    });
    expect(shape(spine)).toEqual([
      ["earlier", "001,002,003"],
      ["file", "004", "applied"],
      ["file", "005", "canceled"],
    ]);
  });

  test("a re-run version keeps the stored revision-deleted note on its section", () => {
    const spine = buildReleaseSpine({
      files: FILES,
      entries: [releaseFile(1, "005"), command(2, { error: "" })],
      run: { status: TaskRun_Status.DONE, error: "" },
      state: stateOf(["005"], { "005": "users/alex@example.com" }),
    });
    expect(spine.items.at(-1)).toMatchObject({
      kind: "file",
      version: "005",
      revisionDeleted: { deleter: "users/alex@example.com" },
    });
  });
});

describe("release spine limits and defaults", () => {
  const many = Array.from({ length: 40 }, (_, i) => ({
    version: String(i + 1).padStart(3, "0"),
    path: `${String(i + 1).padStart(3, "0")}.sql`,
  }));

  // The drawn spine as compact rows: "030:running", "load:applied:9:more",
  // "earlier:001".
  const windowShape = (items: SpineWindowItem[]) =>
    items.map((item) => {
      switch (item.kind) {
        case "file":
          return `${item.version}:${item.status}`;
        case "load":
          return `load:${item.key}:${item.count}${item.more ? ":more" : ""}`;
        case "applied-earlier":
          return `earlier:${item.files.length}`;
        default:
          return item.kind;
      }
    });
  const sixty = Array.from({ length: 60 }, (_, i) => ({
    version: String(i + 1).padStart(3, "0"),
    path: `${String(i + 1).padStart(3, "0")}.sql`,
  }));
  // A new database running the 30th of 60 files.
  const runningAt30 = () =>
    buildReleaseSpine({
      files: sixty,
      entries: sixty
        .slice(0, 30)
        .flatMap((file, i) => [
          releaseFile(i * 3 + 1, file.version),
          command(i * 3 + 2, i === 29 ? {} : { error: "" }),
        ]),
      run: { status: TaskRun_Status.RUNNING, error: "" },
      state: create(Task_ReleaseStateSchema, {
        pendingRanges: [{ start: 0, end: sixty.length }],
      }),
    });
  const range = (from: number, to: number, status: string) =>
    Array.from(
      { length: to - from + 1 },
      (_, i) => `${String(from + i).padStart(3, "0")}:${status}`
    );

  test("running at file 30 of 60: the first 20, a load row for the hidden applied, the running file, a load row for the pending", () => {
    expect(windowShape(windowSpine(runningAt30().items, {}))).toEqual([
      ...range(1, 20, "applied"),
      "load:applied:9:more",
      "030:running",
      "load:pending:30",
    ]);
  });

  test("a load row reveals its stretch in place, partly or all of it", () => {
    const spine = runningAt30();
    expect(
      windowShape(
        windowSpine(spine.items, {
          applied: Number.POSITIVE_INFINITY,
          pending: 20,
        })
      )
    ).toEqual([
      ...range(1, 29, "applied"),
      "030:running",
      ...range(31, 50, "pending"),
      "load:pending:10:more",
    ]);
  });

  test("before the first run the first 20 will apply and the rest load", () => {
    const spine = buildReleaseSpine({
      files: sixty,
      entries: [],
      state: create(Task_ReleaseStateSchema, {
        pendingRanges: [{ start: 0, end: sixty.length }],
      }),
    });
    expect(windowShape(windowSpine(spine.items, {}))).toEqual([
      ...range(1, 20, "pending"),
      "load:pending:40:more",
    ]);
  });

  test("a failed section past the cap is still shown, after the hidden applied files", () => {
    const entries = many.flatMap((file, i) => [
      releaseFile(i * 3 + 1, file.version),
      command(i * 3 + 2, { error: i === 39 ? "boom" : "" }),
    ]);
    const spine = buildReleaseSpine({
      files: many,
      entries,
      run: { status: TaskRun_Status.FAILED, error: FILE_ERROR("040", "boom") },
    });
    expect(windowShape(windowSpine(spine.items, {}))).toEqual([
      ...range(1, 20, "applied"),
      "load:applied:19:more",
      "040:failed",
    ]);
  });

  test("the files already applied pass through as one fold at the top, in version order, outside the cap", () => {
    const spine = buildReleaseSpine({
      files: sixty,
      entries: [],
      state: create(Task_ReleaseStateSchema, {
        appliedCount: 50,
        pendingRanges: [{ start: 50, end: sixty.length }],
      }),
    });
    const drawn = windowSpine(spine.items, {}, 5);
    expect(windowShape(drawn)).toEqual([
      "earlier:50",
      ...range(51, 55, "pending"),
      "load:pending:5:more",
    ]);
    const fold = drawn[0];
    expect(
      fold.kind === "applied-earlier" &&
        fold.files.map((f) => f.version).slice(0, 3)
    ).toEqual(["001", "002", "003"]);
  });

  test("a stretch that mixes states loads as files", () => {
    const items: SpineItem[] = [
      ...range(1, 3, "applied").map((row) => {
        const [version] = row.split(":");
        return {
          kind: "file" as const,
          id: `file-${version}`,
          version,
          path: "",
          status: "applied" as const,
          entries: [],
        };
      }),
      {
        kind: "file",
        id: "file-004",
        version: "004",
        path: "",
        status: "pending",
        entries: [],
      },
    ];
    expect(windowShape(windowSpine(items, {}, 2))).toEqual([
      "001:applied",
      "002:applied",
      "load:files:2:more",
    ]);
    // The fold above is not a section of the stretch's files.
    const folded: SpineItem[] = [
      { kind: "applied-earlier", files: sixty.slice(0, 1) },
      ...items,
    ];
    expect(windowShape(windowSpine(folded, {}, 0))).toEqual([
      "earlier:1",
      "load:files:4",
    ]);
  });

  test("the count line counts every file section by state and the files already applied", () => {
    expect(countSpine(runningAt30().items, 60)).toEqual({
      files: 60,
      applied: 29,
      running: 1,
      failed: 0,
      canceled: 0,
      pending: 30,
      appliedEarlier: 0,
    });
  });

  test("failed, running and Run failed sections open by default; applied, pending and steps do not", () => {
    const spine = buildReleaseSpine({
      files: FILES,
      entries: [
        releaseFile(1, "004"),
        command(2, { error: "" }),
        releaseFile(3, "005"),
        command(4, { error: "x" }),
        databaseSync(6, 7),
      ],
      run: { status: TaskRun_Status.FAILED, error: FILE_ERROR("005", "x") },
    });
    expect(
      spine.items.map((item) => [item.kind, isSpineItemOpenByDefault(item)])
    ).toEqual([
      ["applied-earlier", false],
      ["file", false],
      ["file", true],
      ["step", false],
    ]);
    expect(isSpineItemOpenByDefault({ kind: "run-failed", error: "e" })).toBe(
      true
    );
  });
});

describe("release spine review fixes", () => {
  test("a step that holds the run's error, such as a failed prior backup, opens by default", () => {
    const failedBackup = create(TaskRunLogEntrySchema, {
      type: TaskRunLogEntry_Type.PRIOR_BACKUP,
      logTime: ts(1),
      priorBackup: { startTime: ts(1), endTime: ts(2), error: "backup failed" },
    });
    const spine = buildReleaseSpine({
      files: FILES,
      entries: [failedBackup],
      run: {
        status: TaskRun_Status.FAILED,
        error: "failed to back up: backup failed",
      },
    });
    const step = spine.items.find((item) => item.kind === "step");
    expect(step && isSpineItemOpenByDefault(step)).toBe(true);
  });

  test("each group with no file, before and after the file loop, gets its own id", () => {
    const entries = [
      databaseSync(1, 2),
      releaseFile(3, "001"),
      command(4, { error: "" }),
      databaseSync(6, 7),
    ];
    const groups = buildReleaseFileGroups(entries, {
      getSectionLabel: (type) => String(type),
      entryKeys: assignEntryKeys(entries),
      includeOrphanGroup: true,
    });
    const ids = groups.map((group) => group.id);
    expect(ids).toHaveLength(3);
    expect(new Set(ids).size).toBe(3);
    expect(groups.filter((group) => group.isOrphan)).toHaveLength(2);
  });
});

describe("release spine error ownership", () => {
  test("a revision-write failure after a retried command stays in Run failed", () => {
    const error = "failed to create revision for version 005: duplicate key";
    const spine = buildReleaseSpine({
      files: FILES,
      entries: [
        releaseFile(1, "005"),
        command(2, { error: "lock timeout" }),
        retryMarker(4),
        command(5, { error: "" }),
      ],
      run: { status: TaskRun_Status.FAILED, error },
      state: stateOf(["005"]),
    });
    expect(spine.items.find((item) => item.kind === "file")).toMatchObject({
      version: "005",
      status: "applied",
    });
    expect(spine.items.at(-1)).toMatchObject({ kind: "run-failed", error });
  });
});
