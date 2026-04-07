import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  TaskRunLogEntrySchema,
  TaskRunLogEntry_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  buildReleaseFileGroups,
  buildSectionsFromEntries,
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
    ];

    expect(hasReleaseFileMarkers(entries)).toBe(true);
    expect(buildReleaseFileGroups(entries)[0]).toMatchObject({
      version: "v1",
      filePath: "001.sql",
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
});
