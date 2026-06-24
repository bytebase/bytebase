import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import {
  type Stage,
  type Task,
  Task_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { PlanDetailPageState } from "../../shell/hooks/types";
import { PlanDetailProvider } from "../../shell/PlanDetailContext";
import { DeployStageContentView } from "./DeployStageContentView";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

// Stub the task list so the test can read exactly which tasks survive the
// stage's filter, without pulling in DeployTaskItem (Monaco, task actions…).
vi.mock("./DeployTaskList", () => ({
  DeployTaskList: ({ stage }: { stage: Stage }) => (
    <div>
      {stage.tasks.map((task) => (
        <span data-testid="visible-task" key={task.name}>
          {task.name}
        </span>
      ))}
    </div>
  ),
}));

// Stub the filter UI down to a single control that selects the RUNNING filter.
vi.mock("./DeployTaskFilter", () => ({
  DeployTaskFilter: ({
    onChange,
  }: {
    onChange: (statuses: Task_Status[]) => void;
  }) => (
    <button
      data-testid="select-running"
      onClick={() => onChange([Task_Status.RUNNING])}
      type="button"
    >
      running-filter
    </button>
  ),
}));

vi.mock("./DeployStageRollbackSection", () => ({
  DeployStageRollbackSection: () => null,
}));
vi.mock("../PlanDetailTaskRolloutActionPanel", () => ({
  PlanDetailTaskRolloutActionPanel: () => null,
}));
vi.mock("../../../issue-detail/utils/rollout", () => ({
  canRolloutTasks: () => false,
  preloadRolloutPermissionContext: () => Promise.resolve(),
  RUNNABLE_TASK_STATUSES: [],
}));
vi.mock("@/connect", () => ({ rolloutServiceClientConnect: {} }));
vi.mock("@/store", () => ({ pushNotification: () => undefined }));

const makeStage = (
  name: string,
  tasks: Array<{ name: string; status: Task_Status }>
): Stage =>
  ({
    name,
    environment: `environments/${name.split("/").pop()}`,
    tasks: tasks as unknown as Task[],
  }) as unknown as Stage;

const page = {
  currentUser: {},
  project: { name: "projects/p1" },
  issue: undefined,
  plan: { name: "projects/p1/plans/1" },
  refreshState: () => Promise.resolve(),
} as unknown as PlanDetailPageState;

const renderStage = (stage: Stage) => (
  <PlanDetailProvider value={page}>
    <DeployStageContentView stage={stage} />
  </PlanDetailProvider>
);

const visibleTaskNames = () =>
  screen.queryAllByTestId("visible-task").map((node) => node.textContent);

describe("DeployStageContentView filter scoping", () => {
  test("does not carry the task filter across stages (BYT-9762)", () => {
    const testStage = makeStage("projects/p1/rollouts/r1/stages/test", [
      {
        name: "projects/p1/rollouts/r1/stages/test/tasks/t1",
        status: Task_Status.RUNNING,
      },
      {
        name: "projects/p1/rollouts/r1/stages/test/tasks/t2",
        status: Task_Status.DONE,
      },
    ]);
    const prodStage = makeStage("projects/p1/rollouts/r1/stages/prod", [
      {
        name: "projects/p1/rollouts/r1/stages/prod/tasks/p1",
        status: Task_Status.NOT_STARTED,
      },
      {
        name: "projects/p1/rollouts/r1/stages/prod/tasks/p2",
        status: Task_Status.PENDING,
      },
    ]);

    const { rerender } = render(renderStage(testStage));

    // Both Test tasks visible before filtering.
    expect(visibleTaskNames()).toHaveLength(2);

    // Filter the Test stage down to RUNNING tasks.
    fireEvent.click(screen.getByTestId("select-running"));
    expect(visibleTaskNames()).toEqual([
      "projects/p1/rollouts/r1/stages/test/tasks/t1",
    ]);

    // Switch to the Prod stage (same instance, no `key`). Prod has no RUNNING
    // task, so a leaked filter would hide everything; the filter must reset.
    rerender(renderStage(prodStage));
    expect(visibleTaskNames()).toEqual([
      "projects/p1/rollouts/r1/stages/prod/tasks/p1",
      "projects/p1/rollouts/r1/stages/prod/tasks/p2",
    ]);
  });
});
