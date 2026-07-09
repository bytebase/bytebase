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

const deepLinkedOf = (taskName: string) =>
  screen.getByText(taskName).getAttribute("data-deeplinked");

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

  test("expands only the deep-linked task, not the auto pick", () => {
    const stage = makeStage(stageName, [taskName(1), taskName(2), taskName(3)]);
    render(renderList(stage, taskName(3)));
    // An explicit ?taskId= selection is the focus — the auto pick (the first
    // task) must NOT also open, or a reloaded deep link shows two cards.
    expect(expandedOf(taskName(1))).toBe("false");
    expect(expandedOf(taskName(2))).toBe("false");
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

describe("DeployTaskList mirrors the focused task into the URL", () => {
  const stageName = "rollouts/r/stages/s";
  const taskName = (id: number) => `${stageName}/tasks/t${id}`;

  test("mounting an active stage writes its default task to the URL", () => {
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

    // The default focus (first running task) is expanded and mirrored so the
    // URL is a shareable link to the stage + task from the start.
    expect(expandedOf(taskName(2))).toBe("true");
    expect(routerMocks.replace).toHaveBeenCalledWith({
      query: { phase: "deploy", stageId: "s", taskId: "t2" },
    });
  });

  test("opening a card writes that task to the URL", () => {
    routerMocks.replace.mockClear();
    const stage = makeStage(stageName, [taskName(1), taskName(2)]);
    render(
      <PlanDetailProvider value={makePage()}>
        <DeployTaskList stage={stage} />
      </PlanDetailProvider>
    );

    // t1 is the default; opening t2 focuses it and mirrors it into the URL.
    fireEvent.click(screen.getByText(taskName(2)));
    expect(expandedOf(taskName(2))).toBe("true");
    expect(routerMocks.replace).toHaveBeenLastCalledWith({
      query: { phase: "deploy", stageId: "s", taskId: "t2" },
    });
  });

  test("preview (readonly) never writes the URL", () => {
    routerMocks.replace.mockClear();
    const stage = makeStage(stageName, [taskName(1), taskName(2)]);
    render(renderList(stage));

    fireEvent.click(screen.getByText(taskName(2)));
    expect(expandedOf(taskName(2))).toBe("true");
    expect(routerMocks.replace).not.toHaveBeenCalled();
  });

  test("landing with a ?taskId= honors it without re-writing", () => {
    routerMocks.replace.mockClear();
    const stage = makeStage(stageName, [taskName(1), taskName(2)]);
    render(
      <PlanDetailProvider value={makePage(taskName(2))}>
        <DeployTaskList stage={stage} />
      </PlanDetailProvider>
    );

    // The incoming ?taskId= already matches the focus — no redundant write.
    expect(expandedOf(taskName(2))).toBe("true");
    expect(routerMocks.replace).not.toHaveBeenCalled();
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

  test("first activation expands the default task and mirrors it", () => {
    routerMocks.replace.mockClear();
    const stage = makeStage(stageName, [taskName(1), taskName(2)]);
    const { rerender } = render(renderActive(stage, false));

    rerender(renderActive(stage, true));
    expect(expandedOf(taskName(1))).toBe("true");
    expect(routerMocks.replace).toHaveBeenCalledWith({
      query: { phase: "deploy", stageId: "s", taskId: "t1" },
    });
  });

  test("first activation reseeds focus to a task that went active while hidden", () => {
    routerMocks.replace.mockClear();
    // Mount hidden with everything not-started — the captured default focus is
    // the first task.
    const hidden = makeStage(stageName, [
      { name: taskName(1), status: Task_Status.NOT_STARTED },
      { name: taskName(2), status: Task_Status.NOT_STARTED },
    ]);
    const { rerender } = render(renderActive(hidden, false));

    // Still hidden, t2 starts running. Same task names, so the name-keyed reseed
    // doesn't fire — the captured focus (t1) is now stale.
    const running = makeStage(stageName, [
      { name: taskName(1), status: Task_Status.DONE },
      { name: taskName(2), status: Task_Status.RUNNING },
    ]);
    rerender(renderActive(running, false));

    // On first activation the default-open card AND the URL mirror must reflect
    // the current statuses (t2), not the stale mount-time default (t1).
    rerender(renderActive(running, true));
    expect(expandedOf(taskName(2))).toBe("true");
    expect(routerMocks.replace).toHaveBeenLastCalledWith({
      query: { phase: "deploy", stageId: "s", taskId: "t2" },
    });
  });

  test("bypasses the leave guard for the internal URL sync while editing", () => {
    routerMocks.replace.mockClear();
    const bypass = vi.fn();
    const stage = makeStage(stageName, [
      { name: taskName(1), status: Task_Status.DONE },
      { name: taskName(2), status: Task_Status.RUNNING },
    ]);
    const page = {
      ...makePage(),
      isEditing: true,
      bypassLeaveGuardOnce: bypass,
    } as unknown as PlanDetailPageState;
    render(
      <PlanDetailProvider value={page}>
        <DeployTaskList stage={stage} />
      </PlanDetailProvider>
    );
    // The internal same-path mirror still writes the URL, and bypasses the
    // unsaved-edits guard so no discard dialog interrupts it mid-edit.
    expect(routerMocks.replace).toHaveBeenCalledWith({
      query: { phase: "deploy", stageId: "s", taskId: "t2" },
    });
    expect(bypass).toHaveBeenCalled();
  });

  test("does not touch the leave guard when not editing", () => {
    routerMocks.replace.mockClear();
    const bypass = vi.fn();
    const stage = makeStage(stageName, [
      { name: taskName(1), status: Task_Status.RUNNING },
    ]);
    const page = {
      ...makePage(),
      isEditing: false,
      bypassLeaveGuardOnce: bypass,
    } as unknown as PlanDetailPageState;
    render(
      <PlanDetailProvider value={page}>
        <DeployTaskList stage={stage} />
      </PlanDetailProvider>
    );
    expect(routerMocks.replace).toHaveBeenCalled();
    expect(bypass).not.toHaveBeenCalled();
  });

  test("switching away and back preserves state without re-seeding", () => {
    const stage = makeStage(stageName, [taskName(1), taskName(2)]);
    const { rerender } = render(renderActive(stage, true));
    expect(expandedOf(taskName(1))).toBe("true");

    // User closes the default card, then switches to another stage and back;
    // the list keeps what the user left — no default re-expansion.
    fireEvent.click(screen.getByText(taskName(1)));
    expect(expandedOf(taskName(1))).toBe("false");
    rerender(renderActive(stage, false));
    rerender(renderActive(stage, true));
    expect(expandedOf(taskName(1))).toBe("false");
  });
});

describe("DeployTaskList arrival scroll (BYT-9765 offset jump)", () => {
  const stageName = "rollouts/r/stages/s";
  const taskName = (id: number) => `${stageName}/tasks/t${id}`;

  test("a genuine ?taskId= arrival deep-links its card", () => {
    const stage = makeStage(stageName, [taskName(1), taskName(2)]);
    // Arrived via a shared link to t2 (nothing self-written yet).
    render(
      <PlanDetailProvider value={makePage(taskName(2))}>
        <DeployTaskList stage={stage} />
      </PlanDetailProvider>
    );
    expect(deepLinkedOf(taskName(2))).toBe("true");
  });

  test("a task we wrote ourselves is not a deep-link once the route catches up", () => {
    const stage = makeStage(stageName, [taskName(1), taskName(2)]);
    const { rerender } = render(
      <PlanDetailProvider value={makePage()}>
        <DeployTaskList stage={stage} />
      </PlanDetailProvider>
    );

    // Open t2 — it is mirrored into the URL and marked self-written.
    fireEvent.click(screen.getByText(taskName(2)));

    // The route settles to t2 (as router.replace would deliver it). Because we
    // wrote it, it must NOT deep-link/scroll — nor may any other card, which is
    // the intermittent offset-jump the old per-render derivation caused.
    rerender(
      <PlanDetailProvider value={makePage(taskName(2))}>
        <DeployTaskList stage={stage} />
      </PlanDetailProvider>
    );
    expect(deepLinkedOf(taskName(1))).toBe("false");
    expect(deepLinkedOf(taskName(2))).toBe("false");
  });

  test("forward-navigation to a self-opened task still deep-links (scrolls)", () => {
    const stage = makeStage(stageName, [taskName(1), taskName(2)]);
    const at = (selected?: string) => (
      <PlanDetailProvider value={makePage(selected)}>
        <DeployTaskList stage={stage} />
      </PlanDetailProvider>
    );
    const { rerender } = render(at());

    // Open t2 ourselves, then let the route settle to it — not an arrival.
    fireEvent.click(screen.getByText(taskName(2)));
    rerender(at(taskName(2)));
    expect(deepLinkedOf(taskName(2))).toBe("false");

    // Back to t1, then FORWARD to t2. The forward navigation is a real arrival
    // and must deep-link, even though we opened t2 earlier — the self-write
    // marker is one-shot, not a persistent suppressor.
    rerender(at(taskName(1)));
    rerender(at(taskName(2)));
    expect(deepLinkedOf(taskName(2))).toBe("true");
  });
});
