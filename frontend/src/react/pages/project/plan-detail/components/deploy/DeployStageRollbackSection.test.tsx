import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test, vi } from "vitest";
import {
  type Stage,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { PlanDetailPageState } from "../../shell/hooks/types";
import { PlanDetailProvider } from "../../shell/PlanDetailContext";
import { DeployStageRollbackSection } from "./DeployStageRollbackSection";

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

vi.mock("../PlanDetailRollbackSheet", () => ({
  PlanDetailRollbackSheet: () => null,
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

const taskName = "projects/p1/rollouts/r1/stages/s1/tasks/t1";

const makePageState = (): PlanDetailPageState =>
  ({
    pageKey: "projects/p1/plans/1",
    plan: { name: "projects/p1/plans/1" },
    projectId: "p1",
    rollout: { name: "projects/p1/rollouts/r1" },
    taskRuns: [
      {
        creator: "users/runner@example.com",
        hasPriorBackup: true,
        name: `${taskName}/taskRuns/1`,
        status: TaskRun_Status.DONE,
      },
    ],
  }) as unknown as PlanDetailPageState;

describe("DeployStageRollbackSection", () => {
  test("hides stage run history while keeping rollback action", () => {
    const { container, render, unmount } = renderIntoContainer(
      <PlanDetailProvider value={makePageState()}>
        <DeployStageRollbackSection
          stage={
            {
              tasks: [
                {
                  name: taskName,
                  target: "instances/prod/databases/app",
                },
              ],
            } as unknown as Stage
          }
        />
      </PlanDetailProvider>
    );

    render();

    expect(container.textContent).toContain("task-run.rollback.available");
    expect(container.textContent).not.toContain("task.run-history");

    unmount();
  });
});
