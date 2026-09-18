// @vitest-environment node

import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { describe, expect, test } from "vitest";
import {
  Task_Status,
  TaskSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { scheduledRunTimeMs } from "./scheduledRunTime";

const RUN_AT_MS = 1_772_452_800_000;

const task = (status: Task_Status, runTimeMs?: number) =>
  create(TaskSchema, {
    status,
    runTime: runTimeMs === undefined ? undefined : timestampFromMs(runTimeMs),
  });

describe("scheduledRunTimeMs", () => {
  test("gives the instant a waiting task is due to run", () => {
    expect(scheduledRunTimeMs(task(Task_Status.PENDING, RUN_AT_MS))).toBe(
      RUN_AT_MS
    );
  });

  test("gives nothing for a waiting task with no time set", () => {
    // Zero is an instant, and an instant reads as a schedule: a task waiting
    // for nothing in particular must not show one.
    expect(scheduledRunTimeMs(task(Task_Status.PENDING))).toBeUndefined();
  });

  test.each([
    ["running", Task_Status.RUNNING],
    ["done", Task_Status.DONE],
    ["failed", Task_Status.FAILED],
    ["skipped", Task_Status.SKIPPED],
    ["canceled", Task_Status.CANCELED],
  ])("gives nothing for a %s task that carries one", (_name, status) => {
    expect(scheduledRunTimeMs(task(status, RUN_AT_MS))).toBeUndefined();
  });
});
