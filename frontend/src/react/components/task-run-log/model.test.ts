import { create } from "@bufbuild/protobuf";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import {
  type TaskRunLogEntry,
  TaskRunLogEntry_PriorBackup_PriorBackupDetail_ItemSchema,
  TaskRunLogEntry_Type,
  TaskRunLogEntrySchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import {
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

vi.mock("@/connect", () => ({
  rolloutServiceClientConnect: {},
  sheetServiceClientConnect: {},
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: () => undefined,
}));

vi.mock("@/store", () => ({
  useReleaseStore: () => ({}),
  useRolloutStore: () => ({}),
}));

vi.mock("@/utils", () => ({
  extractRolloutNameFromTaskRunName: () => "",
  extractTaskNameFromTaskRunName: () => "",
  isReleaseBasedTask: () => false,
  releaseNameOfTaskV1: () => "",
  sheetNameOfTaskV1: () => "",
}));

const ts = (seconds: number) => ({ seconds: BigInt(seconds), nanos: 0 });
(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

interface HookHarnessProps {
  entries: TaskRunLogEntry[];
  datasetKey?: string;
}

const createHookHarness = (initialProps: HookHarnessProps) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  let current: UseTaskRunLogSectionsResult | undefined;

  const Harness = (props: HookHarnessProps) => {
    current = useTaskRunLogSections({
      entries: props.entries,
      datasetKey: props.datasetKey,
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
    const groups = buildReleaseFileGroups(entries);
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

    const sections = buildSectionsFromEntries(entries, {
      getSectionLabel: (type) => String(type),
    });

    expect(sections[0]?.status).toBe("running");
    expect(sections[1]?.status).toBe("error");
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

    const sections = buildSectionsFromEntries(entries, {
      getSectionLabel: (type) => String(type),
      detailText,
    });

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

    const sections = buildSectionsFromEntries(entries, {
      getSectionLabel: (type) => String(type),
      detailText,
    });

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
