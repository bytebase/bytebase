import { create } from "@bufbuild/protobuf";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, test, vi } from "vitest";
import i18n from "@/lib/i18n";
import {
  type Task,
  Task_Status,
  TaskRun_Status,
  TaskRunSchema,
  TaskSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { PlanDetailTaskRolloutActionPanel } from "./PlanDetailTaskRolloutActionPanel";

const mocks = vi.hoisted(() => ({
  batchSkipTasks: vi.fn(),
  page: undefined as unknown,
}));

vi.mock("@/api", () => ({
  rolloutServiceClientConnect: {
    batchSkipTasks: mocks.batchSkipTasks,
    batchRunTasks: vi.fn(),
    batchCancelTaskRuns: vi.fn(),
  },
}));
vi.mock("@/api/taskRun", () => ({ listAllTaskRuns: vi.fn() }));
vi.mock("@/hooks/useAppState", () => {
  const user = { name: "users/owner@example.com" };
  return { useCurrentUser: () => user };
});
vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));
vi.mock("@/stores/app", () => {
  // One stable state: fresh objects per call would re-run the panel's
  // permission check on every render.
  const project = { name: "projects/p" };
  const STATE = {
    projectsByName: {},
    environmentList: [],
    getProjectByName: () => project,
    getEnvironmentByName: (name: string) => ({ name, title: "Staging" }),
    notify: vi.fn(),
  };
  const state = () => STATE;
  const useAppStore = (selector?: (s: unknown) => unknown) =>
    selector ? selector(state()) : state();
  useAppStore.getState = state;
  return { useAppStore };
});
vi.mock("../../issue-detail/utils/rollout", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../../issue-detail/utils/rollout")>()),
  canRolloutTasks: () => true,
  preloadRolloutPermissionContext: async () => {},
}));
vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => mocks.page,
}));
vi.mock("./PlanTargetDisplay", () => ({
  PlanTargetDisplay: ({ target }: { target: string }) => (
    <span data-testid="target">{target.split("/").at(-1)}</span>
  ),
}));

const STAGE = "projects/p/plans/1/rollout/stages/staging";
const releaseTask = (id: string, status: Task_Status) =>
  create(TaskSchema, {
    name: `${STAGE}/tasks/${id}`,
    status,
    target: `instances/i/databases/${id}`,
    payload: {
      case: "databaseUpdate",
      value: { source: { case: "release", value: "projects/p/releases/r1" } },
    },
  });
const sheetTask = (id: string, status: Task_Status) =>
  create(TaskSchema, {
    name: `${STAGE}/tasks/${id}`,
    status,
    target: `instances/i/databases/${id}`,
    payload: {
      case: "databaseUpdate",
      value: { source: { case: "sheet", value: "projects/p/sheets/abc" } },
    },
  });
const failedRun = (detail: string) =>
  create(TaskRunSchema, { status: TaskRun_Status.FAILED, detail });

const renderPanel = (tasks: Task[], runs: Record<string, string> = {}) => {
  mocks.page = {
    projectId: "p",
    plan: { name: "projects/p/plans/1", specs: [] },
    rollout: {
      name: "projects/p/plans/1/rollout",
      stages: [{ name: STAGE, environment: "environments/staging", tasks }],
    },
    taskRunsByTaskName: new Map(
      Object.entries(runs).map(([id, detail]) => [`${STAGE}/tasks/${id}`, [failedRun(detail)]])
    ),
  };
  return render(
    <PlanDetailTaskRolloutActionPanel
      action="SKIP"
      open
      onOpenChange={() => {}}
      target={{ type: "tasks", tasks }}
    />
  );
};
const confirmButton = () => screen.getByRole("button", { name: "Skip" });
const taskLine = (id: string) =>
  screen.getAllByTestId("rollout-action-task").find((line) =>
    within(line).queryByText(id)
  )!;

beforeAll(async () => {
  await i18n.changeLanguage("en-US");
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("Skip dialog", () => {
  test.each([
    ["release", releaseTask],
    ["UIOps", sheetTask],
  ])("each %s task line ends with its status chip, and a failed task carries its error's first line", async (_, makeTask) => {
    renderPanel(
      [makeTask("db1", Task_Status.FAILED), makeTask("db2", Task_Status.NOT_STARTED)],
      {
        db1: "failed to execute release file 2026/202610051530_orders.sql (version 202610051530): ERROR: relation \"orders\" already exists\nDETAIL: x",
      }
    );
    await waitFor(() => expect(taskLine("db1")).toBeDefined());
    expect(within(taskLine("db1")).getByTestId("task-status-chip").textContent).toBe("Failed");
    expect(within(taskLine("db2")).getByTestId("task-status-chip").textContent).toBe("Not started");
    const error = within(taskLine("db1")).getByTestId("task-error-line");
    expect(error.textContent).toContain('ERROR: relation "orders" already exists');
    expect(error.textContent).not.toContain("DETAIL");
    expect(within(taskLine("db2")).queryByTestId("task-error-line")).toBeNull();
  });

  test.each([
    ["one release task", [releaseTask("db1", Task_Status.FAILED)]],
    ["several release tasks", [releaseTask("db1", Task_Status.FAILED), releaseTask("db2", Task_Status.NOT_STARTED)]],
  ])("the reason is shown and required for %s", async (_, tasks) => {
    mocks.batchSkipTasks.mockResolvedValue({});
    renderPanel(tasks);
    const reason = await screen.findByRole("textbox");
    await waitFor(() => expect(confirmButton()).toHaveProperty("disabled", true));
    // The reason's error sits with the field, not only on the disabled button.
    expect(within(screen.getByTestId("skip-reason-field")).getByText("Enter why the task is skipped.")).toBeDefined();
    fireEvent.change(reason, { target: { value: "fixed in git" } });
    await waitFor(() => expect(confirmButton()).toHaveProperty("disabled", false));
    fireEvent.click(confirmButton());
    await waitFor(() => expect(mocks.batchSkipTasks).toHaveBeenCalled());
    expect(mocks.batchSkipTasks.mock.calls[0][0]).toMatchObject({
      reason: "fixed in git",
      tasks: tasks.map((task) => task.name),
    });
  });

  test("a UIOps task without an issue shows the reason field but does not require it", async () => {
    renderPanel([sheetTask("db1", Task_Status.FAILED)]);
    expect(await screen.findByRole("textbox")).toBeDefined();
    await waitFor(() => expect(confirmButton()).toHaveProperty("disabled", false));
  });
});
