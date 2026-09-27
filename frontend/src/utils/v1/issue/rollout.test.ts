// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { describe, expect, test } from "vitest";
import { Plan_TaskStatusCountSchema } from "@/types/proto-es/v1/plan_service_pb";
import {
  RolloutSchema,
  StageSchema,
  Task_Status,
  TaskSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  getRolloutStatus,
  getStageStatus,
  getStageStatusFromCounts,
  isTaskActivelyTransitioning,
  scheduledRunTimeMs,
} from "./rollout";

const stage = (...statuses: Task_Status[]) =>
  create(StageSchema, {
    tasks: statuses.map((status) => create(TaskSchema, { status })),
  });

const rollout = (...stages: ReturnType<typeof stage>[]) =>
  create(RolloutSchema, { stages });

const counts = (...statuses: Task_Status[]) =>
  statuses.map((status) =>
    create(Plan_TaskStatusCountSchema, { status, count: 1 })
  );

describe("getStageStatus", () => {
  test("failed outranks running (BYT-9822)", () => {
    expect(
      getStageStatus(
        stage(Task_Status.RUNNING, Task_Status.RUNNING, Task_Status.FAILED)
      )
    ).toBe(Task_Status.FAILED);
    expect(getStageStatus(stage(Task_Status.FAILED, Task_Status.PENDING))).toBe(
      Task_Status.FAILED
    );
    expect(
      getStageStatus(stage(Task_Status.FAILED, Task_Status.CANCELED))
    ).toBe(Task_Status.FAILED);
  });

  test("active work outranks a canceled sibling", () => {
    expect(
      getStageStatus(stage(Task_Status.CANCELED, Task_Status.RUNNING))
    ).toBe(Task_Status.RUNNING);
    expect(
      getStageStatus(stage(Task_Status.CANCELED, Task_Status.PENDING))
    ).toBe(Task_Status.PENDING);
  });

  test("a cancel surfaces once nothing is active", () => {
    expect(
      getStageStatus(stage(Task_Status.CANCELED, Task_Status.NOT_STARTED))
    ).toBe(Task_Status.CANCELED);
    expect(getStageStatus(stage(Task_Status.CANCELED, Task_Status.DONE))).toBe(
      Task_Status.CANCELED
    );
    expect(getStageStatus(stage(Task_Status.CANCELED))).toBe(
      Task_Status.CANCELED
    );
  });

  test("plain progressions", () => {
    expect(getStageStatus(stage(Task_Status.RUNNING, Task_Status.DONE))).toBe(
      Task_Status.RUNNING
    );
    expect(
      getStageStatus(stage(Task_Status.PENDING, Task_Status.NOT_STARTED))
    ).toBe(Task_Status.PENDING);
    expect(getStageStatus(stage(Task_Status.DONE, Task_Status.SKIPPED))).toBe(
      Task_Status.DONE
    );
    expect(getStageStatus(stage(Task_Status.SKIPPED))).toBe(
      Task_Status.SKIPPED
    );
    expect(getStageStatus(stage())).toBe(Task_Status.NOT_STARTED);
  });
});

describe("getRolloutStatus", () => {
  test("a failure anywhere fails the rollout, even with later idle stages", () => {
    expect(
      getRolloutStatus(
        rollout(
          stage(Task_Status.DONE),
          stage(Task_Status.FAILED, Task_Status.RUNNING),
          stage(Task_Status.NOT_STARTED)
        )
      )
    ).toBe(Task_Status.FAILED);
  });

  test("partially deployed rollout aggregates to the idle frontier", () => {
    // The Deploy phase badge maps this to "In progress" via its
    // completed-tasks refinement; the raw aggregate stays NOT_STARTED.
    expect(
      getRolloutStatus(
        rollout(stage(Task_Status.DONE), stage(Task_Status.NOT_STARTED))
      )
    ).toBe(Task_Status.NOT_STARTED);
  });

  test("empty rollout is not started", () => {
    expect(getRolloutStatus(rollout())).toBe(Task_Status.NOT_STARTED);
  });
});

describe("getStageStatusFromCounts", () => {
  test("failed outranks running", () => {
    expect(
      getStageStatusFromCounts(counts(Task_Status.RUNNING, Task_Status.FAILED))
    ).toBe(Task_Status.FAILED);
  });

  test("empty counts fall back to unspecified", () => {
    expect(getStageStatusFromCounts([])).toBe(Task_Status.STATUS_UNSPECIFIED);
  });
});

const NOW_MS = 1_700_000_000_000;
const RUN_AT_MS = 1_772_452_800_000;

const taskAt = (status: Task_Status, runTimeMs?: number) =>
  create(TaskSchema, {
    status,
    runTime: runTimeMs === undefined ? undefined : timestampFromMs(runTimeMs),
  });

describe("isTaskActivelyTransitioning", () => {
  const at = (offsetSec: number) => NOW_MS + offsetSec * 1_000;

  test("RUNNING is always active, regardless of run_time", () => {
    expect(
      isTaskActivelyTransitioning(taskAt(Task_Status.RUNNING), NOW_MS)
    ).toBe(true);
    expect(
      isTaskActivelyTransitioning(taskAt(Task_Status.RUNNING, at(3600)), NOW_MS)
    ).toBe(true);
  });

  test("PENDING is active when unscheduled or already due", () => {
    expect(
      isTaskActivelyTransitioning(taskAt(Task_Status.PENDING), NOW_MS)
    ).toBe(true);
    expect(
      isTaskActivelyTransitioning(
        taskAt(Task_Status.PENDING, at(-3600)),
        NOW_MS
      )
    ).toBe(true);
  });

  test("PENDING scheduled for a future run_time is not active", () => {
    expect(
      isTaskActivelyTransitioning(taskAt(Task_Status.PENDING, at(3600)), NOW_MS)
    ).toBe(false);
  });

  test("settled and not-started tasks are never active", () => {
    for (const status of [
      Task_Status.NOT_STARTED,
      Task_Status.DONE,
      Task_Status.FAILED,
      Task_Status.CANCELED,
      Task_Status.SKIPPED,
    ]) {
      expect(isTaskActivelyTransitioning(taskAt(status), NOW_MS)).toBe(false);
    }
  });
});

describe("scheduledRunTimeMs", () => {
  test("gives the instant a waiting task is due to run", () => {
    expect(scheduledRunTimeMs(taskAt(Task_Status.PENDING, RUN_AT_MS))).toBe(
      RUN_AT_MS
    );
  });

  test("gives nothing for a waiting task with no time set", () => {
    // Such a task is due now, not scheduled -- the reading rollout.ts already
    // takes of an absent runTime.
    expect(scheduledRunTimeMs(taskAt(Task_Status.PENDING))).toBeUndefined();
  });

  test.each([
    ["running", Task_Status.RUNNING],
    ["done", Task_Status.DONE],
    ["failed", Task_Status.FAILED],
    ["skipped", Task_Status.SKIPPED],
    ["canceled", Task_Status.CANCELED],
  ])("gives nothing for a %s task that carries one", (_name, status) => {
    expect(scheduledRunTimeMs(taskAt(status, RUN_AT_MS))).toBeUndefined();
  });
});
