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
vi.mock("@/api", () => ({ rolloutServiceClientConnect: {} }));
vi.mock("@/stores", () => ({ pushNotification: () => undefined }));

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

const visibleTaskNames = () =>
  screen.queryAllByTestId("visible-task").map((node) => node.textContent);

describe("DeployStageContentView filter scoping", () => {
  test("each stage instance keeps its own filter (BYT-9762)", () => {
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

    // All stages stay mounted side by side (the active one is visible, the
    // rest are hidden by the parent); each instance owns its stage's filter.
    render(
      <PlanDetailProvider value={page}>
        <DeployStageContentView stage={testStage} />
        <DeployStageContentView active={false} stage={prodStage} />
      </PlanDetailProvider>
    );

    // All four tasks rendered before filtering.
    expect(visibleTaskNames()).toHaveLength(4);

    // Filter the Test stage down to RUNNING tasks. Prod has no RUNNING task —
    // a filter leaking across stages (BYT-9762) would hide its tasks too.
    fireEvent.click(screen.getAllByTestId("select-running")[0]);
    expect(visibleTaskNames()).toEqual([
      "projects/p1/rollouts/r1/stages/test/tasks/t1",
      "projects/p1/rollouts/r1/stages/prod/tasks/p1",
      "projects/p1/rollouts/r1/stages/prod/tasks/p2",
    ]);
  });
});
