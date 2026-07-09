import { create } from "@bufbuild/protobuf";
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

describe("isTaskActivelyTransitioning", () => {
  const NOW = 1_700_000_000_000; // fixed "now" in ms
  const at = (offsetSec: number) => ({
    seconds: BigInt(NOW / 1000 + offsetSec),
  });
  const task = (status: Task_Status, runTime?: { seconds: bigint }) =>
    create(TaskSchema, runTime ? { status, runTime } : { status });

  test("RUNNING is always active, regardless of run_time", () => {
    expect(isTaskActivelyTransitioning(task(Task_Status.RUNNING), NOW)).toBe(
      true
    );
    expect(
      isTaskActivelyTransitioning(task(Task_Status.RUNNING, at(3600)), NOW)
    ).toBe(true);
  });

  test("PENDING is active when unscheduled or already due", () => {
    expect(isTaskActivelyTransitioning(task(Task_Status.PENDING), NOW)).toBe(
      true
    );
    expect(
      isTaskActivelyTransitioning(task(Task_Status.PENDING, at(-3600)), NOW)
    ).toBe(true);
  });

  test("PENDING scheduled for a future run_time is not active", () => {
    expect(
      isTaskActivelyTransitioning(task(Task_Status.PENDING, at(3600)), NOW)
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
      expect(isTaskActivelyTransitioning(task(status), NOW)).toBe(false);
    }
  });
});
