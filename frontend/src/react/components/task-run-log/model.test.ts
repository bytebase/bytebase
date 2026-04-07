import { create } from "@bufbuild/protobuf";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test } from "vitest";
import {
  type TaskRunLogEntry,
  TaskRunLogEntry_Type,
  TaskRunLogEntrySchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  buildReleaseFileGroups,
  buildSectionsFromEntries,
  groupEntriesByReleaseFile,
  hasReleaseFileMarkers,
} from "./model";
import {
  type UseTaskRunLogSectionsResult,
  useTaskRunLogSections,
} from "./useTaskRunLogSections";

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

  test("preserves orphan entries and marker-only file groups in release grouping", () => {
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
    ];

    const groupedEntries = groupEntriesByReleaseFile(entries);
    expect(groupedEntries).toHaveLength(3);
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

    const groups = buildReleaseFileGroups(entries, {
      getSectionLabel: (type) => String(type),
      includeOrphanGroup: true,
    });
    expect(groups).toHaveLength(3);
    expect(groups[0]).toMatchObject({
      id: "orphan",
      isOrphan: true,
      version: "",
      filePath: "",
      sections: [{ entryCount: 1 }],
    });
    expect(groups[1]).toMatchObject({
      id: "file-0",
      version: "v1",
      filePath: "001.sql",
    });
    expect(groups[1]?.sections).toHaveLength(0);
    expect(groups[2]).toMatchObject({
      id: "file-1",
      version: "v2",
      filePath: "002.sql",
      sections: [{ entryCount: 1 }],
    });
  });

  test("handles expand/collapse state for marker-only release-file groups", () => {
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
    ];

    const hook = createHookHarness({ entries, datasetKey: "marker-only" });

    expect(hook.getCurrent().releaseFileGroups).toHaveLength(2);
    expect(
      hook.getCurrent().releaseFileGroups.map((group) => group.id)
    ).toEqual(["file-0", "file-1"]);
    expect(hook.getCurrent().totalSections).toBe(0);

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
