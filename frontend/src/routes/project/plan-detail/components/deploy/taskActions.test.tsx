import { describe, expect, test } from "vitest";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { getDeployTaskActionState } from "./taskActionState";

describe("deploy task action visibility", () => {
  test("hides actions when permission is missing", () => {
    expect(
      getDeployTaskActionState({
        canPerformActions: false,
        status: Task_Status.NOT_STARTED,
      })
    ).toEqual({
      canCancel: false,
      canRun: false,
      canSkip: false,
      hasActions: false,
    });
  });

  test("shows run and skip for runnable task statuses", () => {
    expect(
      getDeployTaskActionState({
        canPerformActions: true,
        status: Task_Status.FAILED,
      })
    ).toEqual({
      canCancel: false,
      canRun: true,
      canSkip: true,
      hasActions: true,
    });
  });

  test("shows cancel for cancelable task statuses", () => {
    expect(
      getDeployTaskActionState({
        canPerformActions: true,
        status: Task_Status.RUNNING,
      })
    ).toEqual({
      canCancel: true,
      canRun: false,
      canSkip: false,
      hasActions: true,
    });
  });
});
