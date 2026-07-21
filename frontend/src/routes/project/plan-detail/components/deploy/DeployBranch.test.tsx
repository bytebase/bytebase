import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import type { Rollout, Stage } from "@/types/proto-es/v1/rollout_service_pb";
import type { PlanDetailPageState } from "../../shell/hooks/types";
import { PlanDetailProvider } from "../../shell/PlanDetailContext";
import { DeployBranch } from "./DeployBranch";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({ t: (key: string) => key }),
}));

const routerMocks = vi.hoisted(() => ({ push: vi.fn() }));
vi.mock("@/app/router", () => ({ router: { push: routerMocks.push } }));

// Render a button per stage that calls the CURRENT onSelectStage prop, so a
// click uses the latest render's handler (with up-to-date optimistic state).
vi.mock("./DeployStageCard", () => ({
  DeployStageList: ({
    onSelectStage,
    rollout,
  }: {
    onSelectStage: (stage: Stage) => void;
    rollout: Rollout;
  }) => (
    <div>
      {rollout.stages.map((stage) => (
        <button
          key={stage.name}
          type="button"
          data-testid={`select-${stage.name.split("/").pop()}`}
          onClick={() => onSelectStage(stage)}
        >
          {stage.name}
        </button>
      ))}
    </div>
  ),
}));
vi.mock("./DeployStageContentView", () => ({
  DeployStageContentView: () => null,
}));
vi.mock("./DeployPendingTasksSection", () => ({
  DeployPendingTasksSection: () => null,
}));
vi.mock("../../utils/rolloutPreview", () => ({
  generateRolloutPreview: () => Promise.resolve({ stages: [] }),
}));

const ROLLOUT = "projects/p/rollouts/r";
const stage = (id: string): Stage =>
  ({
    name: `${ROLLOUT}/stages/${id}`,
    environment: `environments/${id}`,
    tasks: [{ name: `${ROLLOUT}/stages/${id}/tasks/t1` }],
  }) as unknown as Stage;

const makePage = (routeStageId: string): PlanDetailPageState =>
  ({
    projectId: "p",
    routeStageId,
    selectedTaskName: undefined,
    readonly: false,
    projectCanCreateRollout: true,
    isEditing: false,
    bypassLeaveGuardOnce: vi.fn(),
    refreshState: () => Promise.resolve(),
    plan: { name: "projects/p/plans/1", specs: [] },
    rollout: { name: ROLLOUT, stages: [stage("a"), stage("b")] },
  }) as unknown as PlanDetailPageState;

describe("DeployBranch stage selection", () => {
  test("a quick click back to the route stage supersedes a pending optimistic switch", () => {
    routerMocks.push.mockClear();
    render(
      <PlanDetailProvider value={makePage("a")}>
        <DeployBranch />
      </PlanDetailProvider>
    );

    // Switch to b optimistically. routeStageId still reads "a" (its URL
    // re-render is pending), but the view now shows b.
    fireEvent.click(screen.getByTestId("select-b"));
    expect(routerMocks.push).toHaveBeenLastCalledWith({
      query: { phase: "deploy", stageId: "b" },
    });

    // Immediately click back to a. This must NOT be dropped as a no-op just
    // because routeStageId still reads "a" — the on-screen stage is b, so the
    // click has to push a and win over the pending b navigation.
    fireEvent.click(screen.getByTestId("select-a"));
    expect(routerMocks.push).toHaveBeenLastCalledWith({
      query: { phase: "deploy", stageId: "a" },
    });
  });
});
