import { fireEvent, render, screen } from "@testing-library/react";
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
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({ t: (key: string) => key }),
}));

const routerMocks = vi.hoisted(() => ({
  replace: vi.fn(),
}));

vi.mock("@/react/router", () => ({
  router: { replace: routerMocks.replace },
}));

vi.mock("./DeployTaskItem", () => ({
  DeployTaskItem: ({
    task,
    deepLinked,
    isExpanded,
    onToggleExpand,
  }: {
    task: Task;
    deepLinked: boolean;
    isExpanded: boolean;
    onToggleExpand: (task: Task) => void;
  }) => (
    // One control per card: clicking the card toggles it open or closed.
    <div
      data-deeplinked={deepLinked ? "true" : "false"}
      data-expanded={isExpanded ? "true" : "false"}
      data-testid="task"
      onClick={() => onToggleExpand(task)}
    >
      {task.name}
    </div>
  ),
}));

vi.mock("./DeployTaskToolbar", () => ({ DeployTaskToolbar: () => null }));

const makePage = (selectedTaskName?: string) =>
  ({
    refreshState: () => Promise.resolve(),
    selectedTaskName,
    taskRunsByTaskName: new Map(),
  }) as unknown as PlanDetailPageState;

const makeStage = (
  name: string,
  tasks: Array<string | { name: string; status: Task_Status }>
): Stage =>
  ({
    name,
    environment: "environments/test",
    tasks: tasks.map((task) =>
      typeof task === "string"
        ? { name: task, status: Task_Status.NOT_STARTED }
        : task
    ) as unknown as Task[],
  }) as unknown as Stage;

const expandedOf = (taskName: string) =>
  screen.getByText(taskName).getAttribute("data-expanded");

const renderList = (stage: Stage, selectedTaskName?: string) => (
  <PlanDetailProvider value={makePage(selectedTaskName)}>
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

describe("DeployTaskList first-actionable auto-expand", () => {
  const stageName = "rollouts/r/stages/s";
  const taskName = (id: string) => `${stageName}/tasks/${id}`;

  test("expands the first running task over earlier finished ones", () => {
    const stage = makeStage(stageName, [
      { name: taskName("t1"), status: Task_Status.DONE },
      { name: taskName("t2"), status: Task_Status.RUNNING },
      { name: taskName("t3"), status: Task_Status.FAILED },
    ]);
    render(renderList(stage));
    expect(expandedOf(taskName("t1"))).toBe("false");
    expect(expandedOf(taskName("t2"))).toBe("true");
    expect(expandedOf(taskName("t3"))).toBe("false");
  });

  test("expands the first failed task when nothing is running", () => {
    const stage = makeStage(stageName, [
      { name: taskName("t1"), status: Task_Status.DONE },
      { name: taskName("t2"), status: Task_Status.FAILED },
    ]);
    render(renderList(stage));
    expect(expandedOf(taskName("t1"))).toBe("false");
    expect(expandedOf(taskName("t2"))).toBe("true");
  });

  test("falls back to the first task when nothing is actionable", () => {
    const stage = makeStage(stageName, [
      { name: taskName("t1"), status: Task_Status.DONE },
      { name: taskName("t2"), status: Task_Status.DONE },
    ]);
    render(renderList(stage));
    expect(expandedOf(taskName("t1"))).toBe("true");
    expect(expandedOf(taskName("t2"))).toBe("false");
  });
});

describe("DeployTaskList deep-linked task", () => {
  const stageName = "rollouts/r/stages/s";
  const taskName = (id: number) => `${stageName}/tasks/t${id}`;
  const manyTasks = Array.from({ length: 30 }, (_, i) => taskName(i + 1));

  test("expands the deep-linked task alongside the auto pick", () => {
    const stage = makeStage(stageName, [taskName(1), taskName(2), taskName(3)]);
    render(renderList(stage, taskName(3)));
    expect(expandedOf(taskName(1))).toBe("true");
    expect(expandedOf(taskName(3))).toBe("true");
  });

  test("reveals a deep-linked task beyond the first page", () => {
    const stage = makeStage(stageName, manyTasks);
    render(renderList(stage, taskName(26)));
    expect(expandedOf(taskName(26))).toBe("true");
  });

  test("expands and reveals when the deep link changes after mount", () => {
    const stage = makeStage(stageName, manyTasks);
    const { rerender } = render(renderList(stage));
    expect(screen.queryByText(taskName(26))).toBeNull();

    rerender(renderList(stage, taskName(26)));
    expect(expandedOf(taskName(26))).toBe("true");
    // The earlier auto pick stays expanded — deep links add, never collapse.
    expect(expandedOf(taskName(1))).toBe("true");
  });
});

describe("DeployTaskList toggle writes the opened task to the URL", () => {
  const stageName = "rollouts/r/stages/s";
  const taskName = (id: number) => `${stageName}/tasks/t${id}`;

  test("opening a card writes ?taskId=; closing leaves the URL alone", () => {
    routerMocks.replace.mockClear();
    const stage = makeStage(stageName, [taskName(1), taskName(2)]);
    render(
      <PlanDetailProvider value={makePage()}>
        <DeployTaskList stage={stage} />
      </PlanDetailProvider>
    );

    // t1 is the auto pick (first task) and starts open; t2 collapsed. Opening
    // t2 mirrors it into the URL.
    expect(expandedOf(taskName(2))).toBe("false");
    fireEvent.click(screen.getByText(taskName(2)));
    expect(expandedOf(taskName(2))).toBe("true");
    expect(routerMocks.replace).toHaveBeenLastCalledWith({
      query: { phase: "deploy", stageId: "s", taskId: "t2" },
    });

    // Closing it shuts the body and leaves the URL untouched.
    routerMocks.replace.mockClear();
    fireEvent.click(screen.getByText(taskName(2)));
    expect(expandedOf(taskName(2))).toBe("false");
    expect(routerMocks.replace).not.toHaveBeenCalled();
  });

  test("preview (readonly) toggles open cards but never touch the URL", () => {
    routerMocks.replace.mockClear();
    const stage = makeStage(stageName, [taskName(1), taskName(2)]);
    render(renderList(stage));

    // t2 starts collapsed (t1 is the auto pick). Toggling opens it, no URL.
    expect(expandedOf(taskName(2))).toBe("false");
    fireEvent.click(screen.getByText(taskName(2)));
    expect(expandedOf(taskName(2))).toBe("true");
    expect(routerMocks.replace).not.toHaveBeenCalled();
  });

  test("landing with no ?taskId= mirrors the auto-opened task into the URL", () => {
    routerMocks.replace.mockClear();
    const stage = makeStage(stageName, [
      { name: taskName(1), status: Task_Status.DONE },
      { name: taskName(2), status: Task_Status.RUNNING },
    ]);
    render(
      <PlanDetailProvider value={makePage()}>
        <DeployTaskList stage={stage} />
      </PlanDetailProvider>
    );

    // The auto-opened task (first running) is written to the URL on load.
    expect(expandedOf(taskName(2))).toBe("true");
    expect(routerMocks.replace).toHaveBeenCalledWith({
      query: { phase: "deploy", stageId: "s", taskId: "t2" },
    });
  });

  test("landing with a ?taskId= respects it and writes nothing", () => {
    routerMocks.replace.mockClear();
    const stage = makeStage(stageName, [taskName(1), taskName(2)]);
    render(
      <PlanDetailProvider value={makePage(taskName(2))}>
        <DeployTaskList stage={stage} />
      </PlanDetailProvider>
    );

    // The deep link is the source of truth — no default-select overwrite.
    expect(routerMocks.replace).not.toHaveBeenCalled();
  });

  test("a stale ?taskId= from another stage doesn't suppress the default write", () => {
    routerMocks.replace.mockClear();
    const stage = makeStage(stageName, [taskName(1), taskName(2)]);
    // Right after an optimistic stage switch the URL can still point at the
    // previous stage's task; this stage's default must still be written.
    render(
      <PlanDetailProvider value={makePage("rollouts/r/stages/other/tasks/x")}>
        <DeployTaskList stage={stage} />
      </PlanDetailProvider>
    );

    expect(routerMocks.replace).toHaveBeenCalledWith({
      query: { phase: "deploy", stageId: "s", taskId: "t1" },
    });
  });
});

describe("DeployTaskList keep-alive activation", () => {
  const stageName = "rollouts/r/stages/s";
  const taskName = (id: number) => `${stageName}/tasks/t${id}`;

  const renderActive = (stage: Stage, active: boolean) => (
    <PlanDetailProvider value={makePage()}>
      <DeployTaskList active={active} stage={stage} />
    </PlanDetailProvider>
  );

  test("a hidden list mounts collapsed and never writes the URL", () => {
    routerMocks.replace.mockClear();
    const stage = makeStage(stageName, [taskName(1), taskName(2)]);
    render(renderActive(stage, false));

    // No default expansion (that would mount Monaco + fetch logs offscreen)
    // and no ?taskId= mirror from a stage the user isn't looking at.
    expect(expandedOf(taskName(1))).toBe("false");
    expect(expandedOf(taskName(2))).toBe("false");
    expect(routerMocks.replace).not.toHaveBeenCalled();
  });

  test("first activation expands the default task and mirrors it into the URL", () => {
    routerMocks.replace.mockClear();
    const stage = makeStage(stageName, [taskName(1), taskName(2)]);
    const { rerender } = render(renderActive(stage, false));

    rerender(renderActive(stage, true));
    expect(expandedOf(taskName(1))).toBe("true");
    expect(routerMocks.replace).toHaveBeenCalledWith({
      query: { phase: "deploy", stageId: "s", taskId: "t1" },
    });
  });

  test("switching away and back preserves state without re-seeding", () => {
    routerMocks.replace.mockClear();
    const stage = makeStage(stageName, [taskName(1), taskName(2)]);
    const { rerender } = render(renderActive(stage, true));
    expect(expandedOf(taskName(1))).toBe("true");

    // User closes the default card, then switches to another stage and back.
    fireEvent.click(screen.getByText(taskName(1)));
    expect(expandedOf(taskName(1))).toBe("false");
    rerender(renderActive(stage, false));
    rerender(renderActive(stage, true));

    // The list keeps what the user left — no default re-expansion, and the
    // default ?taskId= mirror ran only once.
    expect(expandedOf(taskName(1))).toBe("false");
    expect(routerMocks.replace).toHaveBeenCalledTimes(1);
  });
});
