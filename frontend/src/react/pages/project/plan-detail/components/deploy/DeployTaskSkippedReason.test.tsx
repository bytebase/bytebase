import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import { type Task, Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { DeployTaskSkippedReason } from "./DeployTaskSkippedReason";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
  })),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

const makeTask = (overrides: Partial<Task>): Task =>
  ({
    status: Task_Status.SKIPPED,
    skippedReason: "",
    ...overrides,
  }) as unknown as Task;

describe("DeployTaskSkippedReason", () => {
  test("shows the skip reason when present", () => {
    const { container, render, unmount } = renderIntoContainer(
      <DeployTaskSkippedReason
        task={makeTask({ skippedReason: "blocked by migration freeze" })}
      />
    );

    render();

    expect(container.textContent).toContain("blocked by migration freeze");
    expect(container.textContent).toContain("task.status.skipped");
    expect(container.textContent).not.toContain("task.skipped-no-reason");

    unmount();
  });

  test("shows a placeholder when the reason is empty", () => {
    const { container, render, unmount } = renderIntoContainer(
      <DeployTaskSkippedReason task={makeTask({ skippedReason: "" })} />
    );

    render();

    expect(container.textContent).toContain("task.skipped-no-reason");

    unmount();
  });

  test("renders nothing when the task is not skipped", () => {
    const { container, render, unmount } = renderIntoContainer(
      <DeployTaskSkippedReason
        task={makeTask({ status: Task_Status.DONE, skippedReason: "" })}
      />
    );

    render();

    expect(container.textContent).toBe("");

    unmount();
  });
});
