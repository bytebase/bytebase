import type { MessageInitShape } from "@bufbuild/protobuf";
import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import type { TFunction } from "i18next";
import { describe, expect, test } from "vitest";
import {
  TaskRun_Status,
  TaskRunSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  executionDurationOfTaskRun,
  getTaskRunComment,
  getTaskRunWaitingMessage,
  sortTaskRunsNewestFirst,
} from "./taskRun";

const t = ((key: string, options?: Record<string, unknown>) =>
  options ? `${key}:${JSON.stringify(options)}` : key) as TFunction;

const ts = (seconds: number) =>
  create(TimestampSchema, { seconds: BigInt(seconds) });

const makeTaskRun = (overrides: MessageInitShape<typeof TaskRunSchema> = {}) =>
  create(TaskRunSchema, overrides);

describe("sortTaskRunsNewestFirst", () => {
  const taskName = "projects/p1/rollouts/r1/stages/s1/tasks/t1";

  test("sorts by createTime descending without mutating the input", () => {
    const oldest = makeTaskRun({
      name: `${taskName}/taskRuns/1`,
      createTime: ts(100),
    });
    const newest = makeTaskRun({
      name: `${taskName}/taskRuns/3`,
      createTime: ts(300),
    });
    const middle = makeTaskRun({
      name: `${taskName}/taskRuns/2`,
      createTime: ts(200),
    });
    const input = [oldest, newest, middle];

    const sorted = sortTaskRunsNewestFirst(input);

    expect(sorted.map((run) => run.name)).toEqual([
      newest.name,
      middle.name,
      oldest.name,
    ]);
    expect(input[0]).toBe(oldest);
  });

  test("orders runs created in the same second by sub-second precision", () => {
    const tsMs = (seconds: number, millis: number) =>
      create(TimestampSchema, {
        seconds: BigInt(seconds),
        nanos: millis * 1_000_000,
      });
    const earlier = makeTaskRun({ name: "r/a", createTime: tsMs(100, 100) });
    const later = makeTaskRun({ name: "r/b", createTime: tsMs(100, 800) });

    expect(
      sortTaskRunsNewestFirst([earlier, later]).map((run) => run.name)
    ).toEqual([later.name, earlier.name]);
  });
});

describe("executionDurationOfTaskRun", () => {
  test("returns undefined without start or update time", () => {
    expect(executionDurationOfTaskRun(makeTaskRun())).toBeUndefined();
    expect(
      executionDurationOfTaskRun(makeTaskRun({ startTime: ts(100) }))
    ).toBeUndefined();
    expect(
      executionDurationOfTaskRun(
        makeTaskRun({ startTime: ts(0), updateTime: ts(100) })
      )
    ).toBeUndefined();
  });

  test("computes duration from start to update time for finished runs", () => {
    const duration = executionDurationOfTaskRun(
      makeTaskRun({
        startTime: ts(100),
        status: TaskRun_Status.DONE,
        updateTime: ts(163),
      })
    );
    expect(duration?.seconds).toBe(63n);
  });

  test("computes elapsed-until-now duration for running runs", () => {
    const startSeconds = Math.floor(Date.now() / 1000) - 30;
    const duration = executionDurationOfTaskRun(
      makeTaskRun({
        startTime: ts(startSeconds),
        status: TaskRun_Status.RUNNING,
        updateTime: ts(startSeconds),
      })
    );
    expect(Number(duration?.seconds)).toBeGreaterThanOrEqual(29);
    expect(Number(duration?.seconds)).toBeLessThan(40);
  });

  test("a running run counts up even without an updateTime yet", () => {
    const startSeconds = Math.floor(Date.now() / 1000) - 30;
    const duration = executionDurationOfTaskRun(
      makeTaskRun({
        startTime: ts(startSeconds),
        status: TaskRun_Status.RUNNING,
      })
    );
    expect(Number(duration?.seconds)).toBeGreaterThanOrEqual(29);
  });

  test("clamps a future start time (clock skew) to zero, never negative", () => {
    const startSeconds = Math.floor(Date.now() / 1000) + 10_000;
    const duration = executionDurationOfTaskRun(
      makeTaskRun({
        startTime: ts(startSeconds),
        status: TaskRun_Status.RUNNING,
      })
    );
    expect(duration?.seconds).toBe(0n);
    expect(duration?.nanos).toBe(0);
  });
});

describe("getTaskRunComment", () => {
  test("pending without runTime reports enqueued", () => {
    const comment = getTaskRunComment(
      makeTaskRun({ status: TaskRun_Status.PENDING }),
      t
    );
    expect(comment).toBe("task-run.status.enqueued");
  });

  test("pending with runTime reports the scheduled time", () => {
    const comment = getTaskRunComment(
      makeTaskRun({ runTime: ts(1700000000), status: TaskRun_Status.PENDING }),
      t
    );
    expect(comment).toContain("task-run.status.enqueued-with-rollout-time");
  });

  test("running with parallel-tasks-limit waiting cause reports it", () => {
    const comment = getTaskRunComment(
      makeTaskRun({
        schedulerInfo: {
          reportTime: ts(1700000000),
          waitingCause: {
            cause: { case: "parallelTasksLimit", value: true },
          },
        },
        status: TaskRun_Status.RUNNING,
      }),
      t
    );
    expect(comment).toContain("task-run.status.waiting-max-tasks-per-rollout");
  });

  test("falls back to detail, then a dash", () => {
    expect(
      getTaskRunComment(
        makeTaskRun({ detail: "applied", status: TaskRun_Status.DONE }),
        t
      )
    ).toBe("applied");
    expect(
      getTaskRunComment(makeTaskRun({ status: TaskRun_Status.DONE }), t)
    ).toBe("-");
  });
});

describe("getTaskRunWaitingMessage", () => {
  test("reports waiting states and stays silent otherwise", () => {
    expect(
      getTaskRunWaitingMessage(
        makeTaskRun({ status: TaskRun_Status.PENDING }),
        t
      )
    ).toBe("task-run.status.enqueued");
    // Running without a recognized waiting cause is not "waiting" — even with
    // detail set, the waiting helper stays silent (the card shows no state
    // line rather than an italic dash).
    expect(
      getTaskRunWaitingMessage(
        makeTaskRun({ detail: "applying", status: TaskRun_Status.RUNNING }),
        t
      )
    ).toBeUndefined();
    expect(
      getTaskRunWaitingMessage(makeTaskRun({ status: TaskRun_Status.DONE }), t)
    ).toBeUndefined();
  });
});
