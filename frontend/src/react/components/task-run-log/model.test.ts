import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  TaskRunLogEntry_Type,
  TaskRunLogEntrySchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  buildReleaseFileGroups,
  buildSectionsFromEntries,
  groupEntriesByReleaseFile,
  hasReleaseFileMarkers,
} from "./model";

const ts = (seconds: number) => ({ seconds: BigInt(seconds), nanos: 0 });

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
      version: "v1",
      filePath: "001.sql",
      sections: [{ entryCount: 2 }],
    });
    expect(groups[1]).toMatchObject({
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
      isOrphan: true,
      version: "",
      filePath: "",
      sections: [{ entryCount: 1 }],
    });
    expect(groups[1]).toMatchObject({
      version: "v1",
      filePath: "001.sql",
    });
    expect(groups[1]?.sections).toHaveLength(0);
    expect(groups[2]).toMatchObject({
      version: "v2",
      filePath: "002.sql",
      sections: [{ entryCount: 1 }],
    });
  });
});
