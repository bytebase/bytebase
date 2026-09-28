import { create } from "@bufbuild/protobuf";
import { describe, expect, test, vi } from "vitest";
import {
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
  buildRowsFromEntries,
  collectLeafSections,
  groupEntriesByReleaseFile,
  hasReleaseFileMarkers,
  type TaskRunLogDetailText,
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
import type { AttemptGroup, LogRow, Section } from "./types";
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
