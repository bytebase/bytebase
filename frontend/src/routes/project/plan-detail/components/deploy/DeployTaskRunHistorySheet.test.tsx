import { create } from "@bufbuild/protobuf";
import { TimestampSchema, timestampFromMs } from "@bufbuild/protobuf/wkt";
import type { ReactElement, ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import {
  TaskRun_Status,
  TaskRunSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import { DeployTaskRunHistorySheet } from "./DeployTaskRunHistorySheet";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) =>
      options ? `${key}:${JSON.stringify(options)}` : key,
  }),
}));

vi.mock("@/components/ui/sheet", () => ({
  Sheet: ({ children, open }: { children: ReactNode; open: boolean }) =>
    open ? <div>{children}</div> : null,
  SheetBody: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  SheetHeader: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetTitle: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

const viewerMounts = vi.hoisted(() => ({ count: 0 }));
vi.mock("@/components/task-run-log", async () => {
  const { useEffect } = await import("react");
  return {
    TaskRunLogViewer: ({ taskRunName }: { taskRunName: string }) => {
      // Counts fresh mounts so a test can assert the viewer remounts (rather
      // than re-renders in place) when its key changes.
      useEffect(() => {
        viewerMounts.count += 1;
      }, []);
      return <div data-task-run-name={taskRunName} data-testid="log-viewer" />;
    },
  };
});

vi.mock("@/components/HumanizeTs", async () => ({
  ...(await import("@/test-utils/humanizeTs")).humanizeTsStub(),
}));

const taskName = "projects/p1/rollouts/r1/stages/s1/tasks/t1";

const makeTaskRuns = (count: number) =>
  // Newest first, matching the component contract.
  Array.from({ length: count }, (_, index) =>
    create(TaskRunSchema, {
      createTime: create(TimestampSchema, {
        seconds: BigInt(1000 * (count - index)),
      }),
      creator: "users/runner@example.com",
      name: `${taskName}/taskRuns/${count - index}`,
      status: TaskRun_Status.FAILED,
    })
  );

const renderSheet = (taskRuns: ReturnType<typeof makeTaskRuns>) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  const element: ReactElement = (
    <DeployTaskRunHistorySheet
      onOpenChange={() => {}}
      open={true}
      taskRuns={taskRuns}
    />
  );
  act(() => {
    root.render(element);
  });
  return {
    container,
    cleanup: () => {
      act(() => root.unmount());
      container.remove();
    },
  };
};

const shownInstants = (container: HTMLElement) =>
  Array.from(
    container.querySelectorAll<HTMLElement>("[data-testid=humanize-ts]")
  ).map((node) => node.textContent);

const timestampModes = (container: HTMLElement) =>
  Array.from(
    container.querySelectorAll<HTMLElement>("[data-testid=humanize-ts]")
  ).map((node) => node.dataset.mode);

const mountedLogNames = (container: HTMLElement) =>
  Array.from(
    container.querySelectorAll<HTMLElement>("[data-testid=log-viewer]")
  ).map((node) => node.dataset.taskRunName);

describe("DeployTaskRunHistorySheet", () => {
  test("dates each run by when it ran, in the embedded history form", () => {
    // A history row orients; the record itself testifies elsewhere. Both the
    // form and the instant are decisions this surface makes, and a run dated
    // from the wrong one reads as plausibly as the right one.
    const { container, cleanup } = renderSheet(makeTaskRuns(1));
    expect(timestampModes(container)).toEqual(["compact"]);
    expect(shownInstants(container)).toEqual([String(1000 * 1000)]);
    cleanup();
  });

  test("dates a run that started by its start, not by its creation", () => {
    // A run is created when it is queued and starts when it is picked up, so
    // the queue time would date it earlier than it ran.
    const startedMs = 5_000_000;
    const [queued] = makeTaskRuns(1);
    const { container, cleanup } = renderSheet([
      create(TaskRunSchema, {
        ...queued,
        startTime: timestampFromMs(startedMs),
      }),
    ]);

    expect(shownInstants(container)).toEqual([String(startedMs)]);
    cleanup();
  });

  test("renders runs newest-first with descending run numbers", () => {
    const { container, cleanup } = renderSheet(makeTaskRuns(3));
    const labels = Array.from(container.querySelectorAll("button")).map(
      (button) => button.textContent
    );
    expect(labels[0]).toContain('task-run.run-number:{"number":3}');
    expect(labels[2]).toContain('task-run.run-number:{"number":1}');
    cleanup();
  });

  test("expands every run when there are three or fewer", () => {
    const { container, cleanup } = renderSheet(makeTaskRuns(3));
    expect(mountedLogNames(container)).toEqual([
      `${taskName}/taskRuns/3`,
      `${taskName}/taskRuns/2`,
      `${taskName}/taskRuns/1`,
    ]);
    cleanup();
  });

  test("remounts the log viewer when a run flips RUNNING -> DONE", () => {
    viewerMounts.count = 0;
    const run = (status: TaskRun_Status) =>
      create(TaskRunSchema, {
        createTime: create(TimestampSchema, { seconds: 1000n }),
        creator: "users/runner@example.com",
        name: `${taskName}/taskRuns/1`,
        status,
      });
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(
        <DeployTaskRunHistorySheet
          onOpenChange={() => {}}
          open={true}
          taskRuns={[run(TaskRun_Status.RUNNING)]}
        />
      );
    });
    expect(viewerMounts.count).toBe(1);

    // Same run, now DONE. Status is part of the viewer key, so it must remount
    // (not re-render in place) — that's what triggers useTaskRunLogData's
    // unmount cleanup to invalidate any in-flight RUNNING log request.
    act(() => {
      root.render(
        <DeployTaskRunHistorySheet
          onOpenChange={() => {}}
          open={true}
          taskRuns={[run(TaskRun_Status.DONE)]}
        />
      );
    });
    expect(viewerMounts.count).toBe(2);

    act(() => root.unmount());
    container.remove();
  });

  test("collapses older runs at four or more, expanding on click", () => {
    const { container, cleanup } = renderSheet(makeTaskRuns(4));
    expect(mountedLogNames(container)).toEqual([`${taskName}/taskRuns/4`]);

    const buttons = container.querySelectorAll("button");
    expect(buttons[0].getAttribute("aria-expanded")).toBe("true");
    expect(buttons[1].getAttribute("aria-expanded")).toBe("false");

    act(() => {
      buttons[1].dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(mountedLogNames(container)).toEqual([
      `${taskName}/taskRuns/4`,
      `${taskName}/taskRuns/3`,
    ]);
    expect(buttons[1].getAttribute("aria-expanded")).toBe("true");
    cleanup();
  });
});
