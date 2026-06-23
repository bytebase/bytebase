import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import {
  type Stage,
  type Task,
  Task_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { PlanDetailPageState } from "../../shell/hooks/types";
import { PlanDetailProvider } from "../../shell/PlanDetailContext";
import { DeployTaskList } from "./DeployTaskList";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("./DeployTaskItem", () => ({
  DeployTaskItem: ({
    task,
    isExpanded,
  }: {
    task: Task;
    isExpanded: boolean;
  }) => (
    <div data-expanded={isExpanded ? "true" : "false"} data-testid="task">
      {task.name}
    </div>
  ),
}));

vi.mock("./DeployTaskToolbar", () => ({ DeployTaskToolbar: () => null }));

const page = {
  refreshState: () => Promise.resolve(),
} as unknown as PlanDetailPageState;

const makeStage = (name: string, taskNames: string[]): Stage =>
  ({
    name,
    environment: "environments/test",
    tasks: taskNames.map((taskName) => ({
      name: taskName,
      status: Task_Status.NOT_STARTED,
    })) as unknown as Task[],
  }) as unknown as Stage;

const expandedOf = (taskName: string) =>
  screen.getByText(taskName).getAttribute("data-expanded");

const renderList = (stage: Stage) => (
  <PlanDetailProvider value={page}>
    <DeployTaskList readonly stage={stage} />
  </PlanDetailProvider>
);

describe("DeployTaskList stage switching (BYT-9763)", () => {
  test("auto-expands the new stage's first task on switch, without a stale frame", () => {
    const stageA = makeStage("rollouts/r/stages/a", [
      "rollouts/r/stages/a/tasks/a1",
      "rollouts/r/stages/a/tasks/a2",
    ]);
    const stageB = makeStage("rollouts/r/stages/b", [
      "rollouts/r/stages/b/tasks/b1",
      "rollouts/r/stages/b/tasks/b2",
    ]);

    const { rerender } = render(renderList(stageA));
    expect(expandedOf("rollouts/r/stages/a/tasks/a1")).toBe("true");
    expect(expandedOf("rollouts/r/stages/a/tasks/a2")).toBe("false");

    // Switch stage on the same instance. The first task of the new stage must
    // be expanded immediately — derived during render — instead of relying on
    // a post-paint effect that briefly leaves the list collapsed.
    rerender(renderList(stageB));
    expect(screen.queryByText("rollouts/r/stages/a/tasks/a1")).toBeNull();
    expect(expandedOf("rollouts/r/stages/b/tasks/b1")).toBe("true");
    expect(expandedOf("rollouts/r/stages/b/tasks/b2")).toBe("false");
  });
});
