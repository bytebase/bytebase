import { create } from "@bufbuild/protobuf";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
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
  buildSectionsFromEntries,
  groupEntriesByReleaseFile,
  hasReleaseFileMarkers,
  type TaskRunLogDetailText,
} from "./model";
import {
  buildReleaseSheetFetchResult,
  buildSheetFetchStateForMissingTask,
  getUnresolvedTaskMetadataStateKey,
} from "./useTaskRunLogData";
import {
  type UseTaskRunLogSectionsResult,
  useTaskRunLogSections,
} from "./useTaskRunLogSections";

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

const buildSections = (
  entries: TaskRunLogEntry[],
  options?: Partial<BuildSectionsOptions>
) =>
  buildSectionsFromEntries(entries, {
    getSectionLabel: (type) => String(type),
    entryKeys: assignEntryKeys(entries),
    ...options,
  });

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
(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

interface HookHarnessProps {
  entries: TaskRunLogEntry[];
  datasetKey?: string;
  taskRunStatus?: TaskRun_Status;
}

const createHookHarness = (initialProps: HookHarnessProps) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  let current: UseTaskRunLogSectionsResult | undefined;

  const Harness = (props: HookHarnessProps) => {
    current = useTaskRunLogSections({
      entries: props.entries,
      datasetKey: props.datasetKey,
      taskRunStatus: props.taskRunStatus,
      getSectionLabel: (type) => String(type),
    });
    return null;
  };

  const render = (props: HookHarnessProps) => {
    act(() => {
      root.render(createElement(Harness, props));
    });
  };

  const getCurrent = () => {
    if (!current) {
      throw new Error("hook result is unavailable");
    }
    return current;
  };

  render(initialProps);

  return {
    render,
    getCurrent,
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

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
      sections: [{ entryCount: 2 }],
    });
    expect(groups[1]).toMatchObject({
      id: "file-1",
      version: "v2",
      filePath: "002.sql",
      sections: [{ entryCount: 1 }],
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

  test("uses localized detail text for completed timed entries, prior backup completion, and retries", () => {
    const detailText = {
      completed: "Completed",
      backingUp: "Backing up...",
      runningByType: {
        [TaskRunLogEntry_Type.SCHEMA_DUMP]: "Dumping...",
      },
      backupCompleted: (count: number) => `Completed (${count} tables)`,
      retryAttempt: (current: number, max: number) =>
        `Attempt ${current}/${max}`,
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
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.RETRY_INFO,
        logTime: ts(5),
        retryInfo: {
          error: "",
          retryCount: 2,
          maximumRetries: 5,
        },
      }),
    ];

    const sections = buildSections(entries, { detailText });

    expect(sections[0]?.items[0]?.detail).toBe("Completed");
    expect(sections[1]?.items[0]?.detail).toBe("Completed (2 tables)");
    expect(sections[2]?.items[0]?.detail).toBe("Attempt 2/5");
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
      sections: [{ entryCount: 1 }],
    });
    expect(groups[1]).toMatchObject({
      id: "file-0",
      version: "v2",
      filePath: "002.sql",
      sections: [{ entryCount: 1 }],
    });
  });

  test("handles expand/collapse state for filtered release-file groups", () => {
    const entries = [
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE,
        logTime: ts(1),
        releaseFileExecute: { version: "v1", filePath: "001.sql" },
      }),
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.RELEASE_FILE_EXECUTE,
        logTime: ts(2),
        releaseFileExecute: { version: "v2", filePath: "002.sql" },
      }),
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
        logTime: ts(3),
        commandExecute: {
          statement: "SELECT 1;",
          response: { logTime: ts(4) },
        },
      }),
    ];

    const hook = createHookHarness({ entries, datasetKey: "marker-only" });

    expect(hook.getCurrent().releaseFileGroups).toHaveLength(1);
    expect(
      hook.getCurrent().releaseFileGroups.map((group) => group.id)
    ).toEqual(["file-0"]);
    expect(hook.getCurrent().totalSections).toBe(1);

    act(() => {
      hook.getCurrent().collapseAll();
    });
    expect(hook.getCurrent().areAllExpanded).toBe(false);

    act(() => {
      hook.getCurrent().expandAll();
    });
    expect(hook.getCurrent().areAllExpanded).toBe(true);

    act(() => {
      hook.getCurrent().toggleReleaseFile("file-0");
    });
    expect(hook.getCurrent().areAllExpanded).toBe(false);

    hook.unmount();
  });

  test("resets expansion state when dataset key changes", () => {
    const entries = [
      create(TaskRunLogEntrySchema, {
        type: TaskRunLogEntry_Type.COMMAND_EXECUTE,
        logTime: ts(1),
        commandExecute: {
          statement: "SELECT 1;",
          response: { logTime: ts(2) },
        },
      }),
    ];

    const hook = createHookHarness({ entries, datasetKey: "dataset-a" });
    const sectionId = hook.getCurrent().sections[0]?.id;
    expect(sectionId).toBe("section-0");

    act(() => {
      hook.getCurrent().toggleSection(sectionId!);
    });
    expect(hook.getCurrent().isSectionExpanded(sectionId!)).toBe(true);

    hook.render({ entries, datasetKey: "dataset-b" });
    expect(hook.getCurrent().isSectionExpanded(sectionId!)).toBe(false);

    hook.unmount();
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
    const sections = buildSections([transaction(1), retryMarker(2)]);

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

  test("one replica's success does not silence another replica's failure", () => {
    const hook = createHookHarness({
      entries: [
        command(1, { statement: "SELECT 1;", error: "boom", replicaId: "a" }),
        command(3, { statement: "SELECT 1;", error: "", replicaId: "b" }),
      ],
      datasetKey: "replicas",
    });

    const [first, second] = hook.getCurrent().replicaGroups;
    expect(markedDetails(first?.sections ?? [])).toEqual(["boom"]);
    expect(markedDetails(second?.sections ?? [])).toEqual([]);

    hook.unmount();
  });

  test("the hook forwards a DONE status into every builder it calls", () => {
    const failing = (replicaId: string) => [
      command(1, { statement: "SELECT 0;", error: "orphan", replicaId }),
      releaseFile(3, "v1", { replicaId }),
      command(4, { statement: "SELECT 1;", error: "in file", replicaId }),
    ];
    const entries = [...failing("a"), ...failing("b")];
    const allSections = (result: UseTaskRunLogSectionsResult) => [
      ...result.sections,
      ...result.releaseFileGroups.flatMap((group) => group.sections),
      ...result.replicaGroups.flatMap((group) => [
        ...group.sections,
        ...group.releaseFileGroups.flatMap((fileGroup) => fileGroup.sections),
      ]),
    ];

    const hook = createHookHarness({ entries, datasetKey: "done-guard" });
    // Flat, release-file, orphan, per-replica orphan and per-replica file.
    expect(
      markedDetails(allSections(hook.getCurrent())).length
    ).toBeGreaterThan(4);

    hook.render({
      entries,
      datasetKey: "done-guard",
      taskRunStatus: TaskRun_Status.DONE,
    });
    expect(markedDetails(allSections(hook.getCurrent()))).toEqual([]);

    hook.unmount();
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
      (group) => group.sections[0]?.items[0]?.key
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
      buildSectionsFromEntries([stranger], {
        getSectionLabel: (type) => String(type),
        entryKeys: assignEntryKeys([]),
      })
    ).toThrow();
  });

  test("a row's key survives a second replica appearing", () => {
    const first = () =>
      command(1, { statement: "SELECT 1;", error: "", replicaId: "a" });
    const hook = createHookHarness({ entries: [first()], datasetKey: "run" });
    const before = hook.getCurrent().sections[0]?.items[0]?.key;

    hook.render({
      entries: [
        first(),
        command(3, { statement: "SELECT 1;", error: "", replicaId: "b" }),
      ],
      datasetKey: "run",
    });

    expect(hook.getCurrent().hasMultipleReplicas).toBe(true);
    const after =
      hook.getCurrent().replicaGroups[0]?.sections[0]?.items[0]?.key;
    expect(before).toBeDefined();
    expect(after).toBe(before);

    hook.unmount();
  });

  test("a row's key survives a retry marker splitting its section", () => {
    const commands = () => [
      command(1, { statement: "SELECT 1;", error: "" }),
      command(5, { statement: "SELECT 2;", error: "" }),
    ];
    const before = buildSections(commands());
    expect(before).toHaveLength(1);

    const after = buildSections([...commands(), retryMarker(3)]);
    expect(after).toHaveLength(3);
    expect(after[2]?.items[0]?.key).toBe(before[0]?.items[1]?.key);
    expect(after[0]?.items[0]?.key).toBe(before[0]?.items[0]?.key);
  });
});
