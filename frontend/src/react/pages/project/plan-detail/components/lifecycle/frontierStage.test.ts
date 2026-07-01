import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  StageSchema,
  Task_Status,
  TaskSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { stageHasFailedOrCanceledTasks } from "./frontierStage";

const stage = (...statuses: Task_Status[]) =>
  create(StageSchema, {
    tasks: statuses.map((status) => create(TaskSchema, { status })),
  });

describe("stageHasFailedOrCanceledTasks", () => {
  test("false when all runnable tasks are fresh (not started)", () => {
    expect(stageHasFailedOrCanceledTasks(stage(Task_Status.NOT_STARTED))).toBe(
      false
    );
    expect(
      stageHasFailedOrCanceledTasks(
        stage(Task_Status.NOT_STARTED, Task_Status.DONE)
      )
    ).toBe(false);
  });

  test("true when any task previously ran (failed or canceled)", () => {
    expect(
      stageHasFailedOrCanceledTasks(
        stage(Task_Status.NOT_STARTED, Task_Status.FAILED)
      )
    ).toBe(true);
    expect(stageHasFailedOrCanceledTasks(stage(Task_Status.CANCELED))).toBe(
      true
    );
  });
});
