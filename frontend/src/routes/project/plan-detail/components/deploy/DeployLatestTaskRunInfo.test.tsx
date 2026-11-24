import { create } from "@bufbuild/protobuf";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import {
  TaskRun_Status,
  TaskRunSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { DeployLatestTaskRunInfo } from "./DeployLatestTaskRunInfo";

const mounts = vi.hoisted(() => vi.fn());
vi.mock("@/components/task-run-log", async () => {
  const { useEffect } = await import("react");
  return {
    TaskRunLogViewer: ({
      taskRunName,
      taskRunStatus,
      runError,
      agentHelp,
    }: {
      taskRunName: string;
      taskRunStatus: TaskRun_Status;
      runError: string;
      agentHelp?: { provenance: string };
    }) => {
      useEffect(() => {
        mounts();
      }, []);
      return (
        <div
          data-testid="log-viewer"
          data-task-run-name={taskRunName}
          data-status={taskRunStatus}
          data-provenance={agentHelp?.provenance}
        >
          {runError}
        </div>
      );
    },
  };
});
vi.mock("../PlanDetailTaskRunSession", () => ({
  PlanDetailTaskRunSession: () => null,
}));

afterEach(cleanup);

test("opts the displayed latest run in and remounts on a status or name change", () => {
  mounts.mockClear();
  const prefix = "projects/example/plans/42/rollout/stages/prod/tasks/2/taskRuns/";
  const latest = (id: number, status: TaskRun_Status, detail = "") => (
    <DeployLatestTaskRunInfo
      historyCount={1}
      onShowHistory={() => {}}
      taskRun={create(TaskRunSchema, { name: `${prefix}${id}`, status, detail })}
    />
  );
  const { rerender } = render(latest(3, TaskRun_Status.RUNNING));
  const initial = screen.getByTestId("log-viewer");
  expect(initial.dataset.provenance).toBe("latest");
  expect(initial.dataset.taskRunName).toBe(`${prefix}3`);

  rerender(latest(3, TaskRun_Status.FAILED, "Shown failure"));
  const failed = screen.getByTestId("log-viewer");
  expect(failed).not.toBe(initial);
  expect(failed.dataset.status).toBe(String(TaskRun_Status.FAILED));
  expect(failed.textContent).toBe("Shown failure");

  rerender(latest(4, TaskRun_Status.DONE));
  const replacement = screen.getByTestId("log-viewer");
  expect(replacement).not.toBe(failed);
  expect(replacement.dataset.taskRunName).toBe(`${prefix}4`);
  expect(replacement.textContent).toBe("");
  expect(mounts).toHaveBeenCalledTimes(3);
});
