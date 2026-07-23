import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";
import type { PlanDetailPageState } from "../../shell/hooks/types";
import { PlanDetailProvider } from "../../shell/PlanDetailContext";
import { RunStageAction } from "./DeployStageActions";

const mocks = vi.hoisted(() => ({
  bypassLeaveGuardOnce: vi.fn(),
  expandPhase: vi.fn(),
  push: vi.fn(),
  refreshState: vi.fn(async () => {}),
  resolve: vi.fn(() => ({ fullPath: "/projects/p/plans/1/rollout/stages/a" })),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/app/router", () => ({
  router: {
    currentRoute: {
      value: { fullPath: "/projects/p/plans/1/rollout/stages/b" },
    },
    push: mocks.push,
    resolve: mocks.resolve,
  },
}));

vi.mock("../../../issue-detail/utils/rollout", () => ({
  RUNNABLE_TASK_STATUSES: [],
  canRolloutTasks: () => true,
  preloadRolloutPermissionContext: () => Promise.resolve(),
}));

vi.mock("../../shell/focusPhase", () => ({
  focusPlanPhase: (
    phase: "deploy",
    expandPhase: (phase: "deploy") => void
  ) => expandPhase(phase),
}));

vi.mock("../PlanDetailTaskRolloutActionPanel", () => ({
  PlanDetailTaskRolloutActionPanel: ({
    onConfirm,
    open,
  }: {
    onConfirm?: () => Promise<void> | void;
    open: boolean;
  }) =>
    open ? (
      <button onClick={() => void onConfirm?.()} type="button">
        confirm-run
      </button>
    ) : null,
}));

vi.mock("./useStageTitle", () => ({
  useStageTitle: () => "Stage A",
}));

const stage = {
  name: "projects/p/plans/1/rollout/stages/a",
  environment: "environments/test",
  tasks: [],
} as unknown as Stage;

const makePage = (routeStageId: string): PlanDetailPageState =>
  ({
    bypassLeaveGuardOnce: mocks.bypassLeaveGuardOnce,
    currentUser: { name: "users/me@example.com" },
    expandPhase: mocks.expandPhase,
    isEditing: true,
    project: { name: "projects/p" },
    refreshState: mocks.refreshState,
    routeStageId,
  }) as unknown as PlanDetailPageState;

const confirmRun = async (routeStageId: string) => {
  render(
    <PlanDetailProvider value={makePage(routeStageId)}>
      <RunStageAction stage={stage} />
    </PlanDetailProvider>
  );

  fireEvent.click(await screen.findByRole("button"));
  fireEvent.click(screen.getByRole("button", { name: "confirm-run" }));
  await waitFor(() => expect(mocks.refreshState).toHaveBeenCalledTimes(1));
};

describe("RunStageAction", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("bypasses the leave guard before switching an edited page to the run stage", async () => {
    await confirmRun("b");

    expect(mocks.bypassLeaveGuardOnce).toHaveBeenCalledTimes(1);
    expect(mocks.push).toHaveBeenCalledWith(
      {
        name: "workspace.project.plan.detail.rollout.stage",
        params: { projectId: "p", planId: "1", stageId: "a" },
      },
      { preventScrollReset: true }
    );
    expect(mocks.bypassLeaveGuardOnce.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.push.mock.invocationCallOrder[0]
    );
  });

  test("does not arm the leave-guard bypass when the run stage is already selected", async () => {
    await confirmRun("a");

    expect(mocks.bypassLeaveGuardOnce).not.toHaveBeenCalled();
    expect(mocks.push).not.toHaveBeenCalled();
  });
});
