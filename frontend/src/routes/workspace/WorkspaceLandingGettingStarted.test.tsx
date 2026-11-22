import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type {
  GuideContext,
  ResolvedGuide,
  ResolvedGuideStep,
} from "@/modules/workspace-setup-guide/types";
import { WorkspaceLandingGettingStarted } from "./WorkspaceLandingGettingStarted";

const mocks = vi.hoisted(() => ({
  captureMetric: vi.fn(),
  completionAcknowledged: false,
  contextReady: true,
  dismissed: false,
  enabled: true,
  hasExploredDatabase: false,
  hasInstance: false,
  hasProject: true,
  projectName: "projects/orders",
  push: vi.fn(),
  saveIntroStateByKey: vi.fn(),
}));

vi.mock("@/app/analytics/behavior", () => ({
  createBehaviorMetric: (event: string, options: unknown) => ({
    event,
    options,
  }),
}));
vi.mock("@/app/analytics/provider", () => ({
  behaviorAnalytics: { captureMetric: mocks.captureMetric },
}));
vi.mock("@/app/router", () => ({
  router: { push: mocks.push },
  useCurrentRoute: () => ({ name: "workspace.landing", params: {}, query: {} }),
}));
vi.mock("@/hooks/useAppState", () => ({
  useIntroStateByKey: (key: string) =>
    key === "workspace-setup-guide.dismissed"
      ? mocks.dismissed
      : key.startsWith("workspace-setup-guide.completed.")
        ? mocks.completionAcknowledged
        : false,
}));
vi.mock("@/stores/app", () => {
  const storeState = {
    workspaceSetupGuideEnabled: () => mocks.enabled,
    saveIntroStateByKey: mocks.saveIntroStateByKey,
  };
  return {
    useAppStore: Object.assign(
      (selector: (state: typeof storeState) => unknown) => selector(storeState),
      { getState: () => storeState }
    ),
  };
});
vi.mock("@/modules/workspace-setup-guide/selection", () => ({
  readGuideWorkspaceUsage: () => undefined,
  readSelectedGuideScenarioId: () => undefined,
}));
vi.mock("@/modules/workspace-setup-guide/GuideContextProvider", () => ({
  useSharedGuideContext: () => ({
    context: {
      hasProject: mocks.hasProject,
      hasInstance: mocks.hasInstance,
      hasExploredDatabase: mocks.hasExploredDatabase,
      hasRunStatement: false,
      hasCreatedChangeIssue: false,
      hasMarkedSensitiveData: false,
      isSaaS: false,
      hasOtherHumanUser: false,
      hasOtherWorkspaceMember: false,
      projectName: mocks.projectName,
      instanceName: "",
      databaseProjectName: "projects/orders",
      databaseName: "instances/test/databases/app",
      route: { name: "workspace.landing", params: {} },
    } satisfies GuideContext,
    contextReady: mocks.contextReady,
  }),
}));
vi.mock("@/modules/workspace-setup-guide/WorkspaceHomeGettingStarted", () => ({
  WorkspaceHomeGettingStarted: ({
    guide,
    onSelectStep,
    onDismiss,
  }: {
    guide: ResolvedGuide;
    onSelectStep: (step: ResolvedGuideStep) => void;
    onDismiss: () => void;
  }): ReactNode => (
    <div data-testid="workspace-home-getting-started">
      <button
        type="button"
        onClick={() => guide.activeStep && onSelectStep(guide.activeStep)}
      >
        {guide.activeStep?.definition.id}
      </button>
      {guide.complete && (
        <button type="button" onClick={onDismiss}>
          Dismiss
        </button>
      )}
    </div>
  ),
}));

beforeEach(() => {
  vi.clearAllMocks();
  mocks.contextReady = true;
  mocks.completionAcknowledged = false;
  mocks.dismissed = false;
  mocks.enabled = true;
  mocks.hasExploredDatabase = false;
  mocks.hasInstance = false;
  mocks.hasProject = true;
  mocks.projectName = "projects/orders";
});
afterEach(cleanup);

describe("WorkspaceLandingGettingStarted", () => {
  test("renders the current setup step and opens its destination", () => {
    render(<WorkspaceLandingGettingStarted />);

    fireEvent.click(screen.getByRole("button", { name: "connect-instance" }));
    expect(mocks.push).toHaveBeenCalledWith(
      expect.objectContaining({ params: { projectId: "orders" } })
    );
    expect(mocks.captureMetric).toHaveBeenCalledWith(
      expect.objectContaining({
        event: "workspace setup guide step action selected",
      })
    );
  });

  test.each([
    { dismissed: true, enabled: true, contextReady: true },
    { dismissed: false, enabled: false, contextReady: true },
    { dismissed: false, enabled: true, contextReady: false },
  ])("hides the setup section when the guide is unavailable: %j", (state) => {
    Object.assign(mocks, state);
    render(<WorkspaceLandingGettingStarted />);

    expect(screen.queryByTestId("workspace-home-getting-started")).toBeNull();
  });

  test("acknowledges the completed guide through the shared progress key", () => {
    mocks.hasInstance = true;
    mocks.hasExploredDatabase = true;
    render(<WorkspaceLandingGettingStarted />);

    fireEvent.click(screen.getByRole("button", { name: "Dismiss" }));
    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: "workspace-setup-guide.completed.workspace-setup",
      newState: true,
    });
  });

  test("hides a completed guide after acknowledgment", () => {
    mocks.hasInstance = true;
    mocks.hasExploredDatabase = true;
    mocks.completionAcknowledged = true;
    render(<WorkspaceLandingGettingStarted />);

    expect(screen.queryByTestId("workspace-home-getting-started")).toBeNull();
  });
});
