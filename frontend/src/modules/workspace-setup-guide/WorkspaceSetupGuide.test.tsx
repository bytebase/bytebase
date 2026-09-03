import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  GUIDE_PROGRESS_KEYS,
  guideCompletionAcknowledgedKey,
} from "./progress";
import type {
  GuideContext,
  GuideScenarioId,
  GuideWorkspaceUsage,
} from "./types";
import { WorkspaceSetupGuide } from "./WorkspaceSetupGuide";

const guideContext = (
  overrides: Partial<GuideContext> = {}
): GuideContext => ({
  hasProject: false,
  hasInstance: false,
  hasExploredDatabase: false,
  hasRunStatement: false,
  hasCreatedChangeIssue: false,
  isSaaS: false,
  hasOtherHumanUser: false,
  hasOtherWorkspaceMember: false,
  projectName: "",
  instanceName: "",
  databaseProjectName: "",
  databaseName: "",
  route: { name: "workspace.home", params: {} },
  ...overrides,
});

const mocks = vi.hoisted(() => ({
  captureMetric: vi.fn(),
  currentRoute: { name: "workspace.home", params: {}, query: {} },
  guideContext: undefined as unknown as GuideContext,
  guideEnabled: true,
  guideUserCount: 1,
  introState: {} as Record<string, boolean>,
  isSaaS: false,
  loading: false,
  preCreateIssue: vi.fn(),
  productModelContent: "guide content" as string | undefined,
  routerPush: vi.fn(),
  saveIntroStateByKey: vi.fn(),
  scenarioId: undefined as GuideScenarioId | undefined,
  workspaceUsage: undefined as GuideWorkspaceUsage | undefined,
}));

const resizeObserverCallbacks: ResizeObserverCallback[] = [];

globalThis.ResizeObserver = class ResizeObserver {
  constructor(callback: ResizeObserverCallback) {
    resizeObserverCallbacks.push(callback);
  }
  observe() {}
  unobserve() {}
  disconnect() {}
} as typeof ResizeObserver;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => undefined },
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) =>
      key === "workspace-setup-guide.step-progress"
        ? `${options?.current} of ${options?.total}`
        : key,
    i18n: { resolvedLanguage: "en-US" },
  }),
}));

vi.mock("@/app/router", () => ({
  router: { push: mocks.routerPush },
  useCurrentRoute: () => mocks.currentRoute,
}));

vi.mock("@/app/analytics/provider", () => ({
  behaviorAnalytics: { captureMetric: mocks.captureMetric },
}));

vi.mock("@/components/HowBytebaseWorksSheet", () => ({
  getHowBytebaseWorksGuideContent: () => mocks.productModelContent,
  HowBytebaseWorksSheet: ({ open }: { open: boolean }) =>
    open ? <div data-testid="product-model-sheet" /> : null,
}));

vi.mock("@/components/SQLEditorButton", () => ({
  SQLEditorButton: ({
    label,
    "data-testid": testId,
  }: {
    label?: ReactNode;
    "data-testid"?: string;
  }) => <button data-testid={testId ?? "sql-editor-action"}>{label}</button>,
}));

vi.mock("@/hooks/useAppState", () => ({
  useIntroStateByKey: (key: string) => mocks.introState[key] ?? false,
}));

vi.mock("@/lib/plan/issue", () => ({ preCreateIssue: mocks.preCreateIssue }));

vi.mock("@/stores/app", () => {
  const state = () => ({
    isSaaSMode: () => mocks.isSaaS,
    workspaceSetupGuideEnabled: (allowMultipleMembers = false) =>
      mocks.guideEnabled &&
      (mocks.guideUserCount === 1 || allowMultipleMembers),
  });
  const useAppStore = Object.assign(
    (selector: (value: ReturnType<typeof state>) => unknown) =>
      selector(state()),
    {
      getState: () => ({
        ...state(),
        saveIntroStateByKey: mocks.saveIntroStateByKey,
        getIntroStateByKey: (key: string) => mocks.introState[key] ?? false,
      }),
    }
  );
  return { useAppStore };
});

vi.mock("./selection", () => ({
  readGuideWorkspaceUsage: () => mocks.workspaceUsage,
  readSelectedGuideScenarioId: () => mocks.scenarioId,
}));

vi.mock("./useGuideContext", () => ({
  useGuideContext: () => ({
    context: mocks.guideContext,
    loading: mocks.loading,
  }),
}));

beforeEach(() => {
  vi.clearAllMocks();
  resizeObserverCallbacks.length = 0;
  mocks.currentRoute = { name: "workspace.home", params: {}, query: {} };
  mocks.guideContext = guideContext();
  mocks.guideEnabled = true;
  mocks.guideUserCount = 1;
  mocks.introState = {};
  mocks.isSaaS = false;
  mocks.loading = false;
  mocks.productModelContent = "guide content";
  mocks.scenarioId = undefined;
  mocks.workspaceUsage = undefined;
  mocks.saveIntroStateByKey.mockImplementation(({ key, newState }) => {
    mocks.introState[key] = newState;
  });
});

afterEach(cleanup);

describe("WorkspaceSetupGuide", () => {
  test("uses the same guide title for generic and scenario journeys", () => {
    const first = render(<WorkspaceSetupGuide />);
    expect(
      screen.getByText("workspace-setup-guide.getting-started")
    ).toBeVisible();
    first.unmount();

    mocks.scenarioId = "create-database-change";
    render(<WorkspaceSetupGuide />);

    expect(
      screen.getByText("workspace-setup-guide.getting-started")
    ).toBeVisible();
    expect(
      screen.queryByText(
        "workspace-setup-guide.scenarios.create-database-change.title"
      )
    ).not.toBeInTheDocument();
  });

  test("renders generic resource setup when no scenario was selected", () => {
    render(<WorkspaceSetupGuide />);

    expect(
      screen.getAllByTestId(/^setup-step-/).map((node) => node.dataset.testid)
    ).toEqual([
      "setup-step-create-project",
      "setup-step-connect-instance",
      "setup-step-explore-database",
    ]);
    expect(screen.queryByTestId("product-model-sheet")).not.toBeInTheDocument();
    expect(screen.getByTestId("open-product-model")).toBeVisible();
  });

  test("keeps Query Data prerequisites visible when they are complete", () => {
    mocks.scenarioId = "query-data";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });

    render(<WorkspaceSetupGuide />);

    expect(
      screen.getAllByTestId(/^setup-step-/).map((node) => node.dataset.testid)
    ).toEqual([
      "setup-step-create-project",
      "setup-step-connect-instance",
      "setup-step-explore-database",
      "setup-step-query-data",
    ]);
    expect(screen.getByTestId("active-action")).toHaveTextContent(
      "workspace-setup-guide.actions.query"
    );
    expect(screen.getByTestId("open-product-model")).toBeVisible();
  });

  test("shows the full Query Data chain when setup has no resources", () => {
    mocks.scenarioId = "query-data";

    render(<WorkspaceSetupGuide />);

    expect(screen.getByTestId("setup-step-create-project")).toBeEnabled();
    expect(screen.getByTestId("setup-step-connect-instance")).toBeDisabled();
    expect(screen.getByTestId("setup-step-explore-database")).toBeDisabled();
    expect(screen.getByTestId("setup-step-query-data")).toBeDisabled();
  });

  test("appends Add teammate after the selected outcome for team usage", () => {
    mocks.scenarioId = "query-data";
    mocks.workspaceUsage = "team";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      hasRunStatement: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });

    render(<WorkspaceSetupGuide />);

    expect(
      screen.getAllByTestId(/^setup-step-/).map((node) => node.dataset.testid)
    ).toEqual([
      "setup-step-create-project",
      "setup-step-connect-instance",
      "setup-step-explore-database",
      "setup-step-query-data",
      "setup-step-add-teammate",
    ]);
    expect(screen.getByTestId("setup-step-add-teammate")).toBeEnabled();
  });

  test("keeps a completed multi-member team journey until Done", () => {
    mocks.scenarioId = "query-data";
    mocks.workspaceUsage = "team";
    mocks.guideUserCount = 2;
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      hasRunStatement: true,
      hasOtherHumanUser: true,
      hasOtherWorkspaceMember: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });

    const first = render(<WorkspaceSetupGuide />);
    expect(screen.getByTestId("complete-guide")).toBeVisible();
    fireEvent.click(screen.getByTestId("complete-guide"));
    first.unmount();

    render(<WorkspaceSetupGuide />);
    expect(screen.queryByTestId("complete-guide")).not.toBeInTheDocument();
  });

  test.each([false, true])(
    "keeps the self-host teammate action fixed when a user exists: %s",
    (hasOtherHumanUser) => {
    mocks.workspaceUsage = "team";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      hasOtherHumanUser,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });
    render(<WorkspaceSetupGuide />);

      fireEvent.click(screen.getByTestId("setup-step-add-teammate"));

    expect(mocks.routerPush).toHaveBeenCalledWith({
        name: "workspace.users",
        query: { intro: "create-user" },
    });
    }
  );

  test("uses the fixed SaaS grant-access teammate action", () => {
    mocks.isSaaS = true;
    mocks.workspaceUsage = "team";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      isSaaS: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });
    render(<WorkspaceSetupGuide />);

    fireEvent.click(screen.getByTestId("setup-step-add-teammate"));

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.members",
      query: { intro: "grant-access" },
    });
  });

  test("starts a database change for the discovered database", () => {
    mocks.scenarioId = "create-database-change";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });

    render(<WorkspaceSetupGuide />);
    fireEvent.click(screen.getByTestId("setup-step-create-database-change"));

    expect(mocks.preCreateIssue).toHaveBeenCalledWith("projects/app", [
      "instances/sample/databases/employee",
    ]);
  });

  test("keeps a completed step selected after its navigation finishes", () => {
    mocks.scenarioId = "create-database-change";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
      route: { name: "workspace.project.database", params: {} },
    });
    const { rerender } = render(<WorkspaceSetupGuide />);

    fireEvent.click(screen.getByTestId("setup-step-create-project"));
    mocks.guideContext = {
      ...mocks.guideContext,
      route: { name: "workspace.project.database", params: {} },
    };
    rerender(<WorkspaceSetupGuide />);

    mocks.currentRoute = {
      name: "workspace.project",
      params: {},
      query: { intro: "create-project" },
    };
    mocks.guideContext = {
      ...mocks.guideContext,
      route: { name: "workspace.project", params: {} },
    };
    rerender(<WorkspaceSetupGuide />);

    expect(screen.getByTestId("setup-step-create-project")).toHaveClass(
      "bg-accent/10"
    );
    expect(
      screen.getByTestId("setup-step-create-database-change")
    ).not.toHaveClass("bg-accent/10");
  });

  test.each<GuideScenarioId | undefined>([
    undefined,
    "query-data",
    "create-database-change",
  ])("opens optional product help for the %s guide", (scenarioId) => {
    mocks.scenarioId = scenarioId;
    render(<WorkspaceSetupGuide />);

    const productModelButton = screen.getByTestId("open-product-model");
    expect(productModelButton).toBeVisible();
    expect(
      screen.getByText("workspace-setup-guide.getting-started").parentElement
    ).toContainElement(productModelButton);
    fireEvent.click(productModelButton);
    expect(screen.getByTestId("product-model-sheet")).toBeVisible();
  });

  test("uses medium density below the 2xl breakpoint", () => {
    render(<WorkspaceSetupGuide />);

    const title = screen.getByText("workspace-setup-guide.getting-started");
    const guideBar = title.parentElement?.parentElement?.parentElement;
    expect(guideBar).toHaveClass("px-4", "py-3", "2xl:px-5", "2xl:py-4");

    const firstStep = screen.getByTestId("setup-step-create-project");
    expect(firstStep).toHaveClass(
      "px-2.5",
      "py-1.5",
      "2xl:px-3",
      "2xl:py-2"
    );
    expect(firstStep).toHaveClass("text-sm", "2xl:text-base");
  });

  test("uses the compact step navigator only when the step list overflows", async () => {
    mocks.scenarioId = "query-data";
    mocks.workspaceUsage = "team";
    render(<WorkspaceSetupGuide />);

    const viewport = screen.getByTestId("guide-step-viewport");
    const measurement = screen.getByTestId("guide-step-measurement");
    Object.defineProperty(viewport, "clientWidth", {
      configurable: true,
      value: 500,
    });
    Object.defineProperty(measurement, "scrollWidth", {
      configurable: true,
      value: 400,
    });

    act(() => {
      for (const callback of resizeObserverCallbacks) {
        callback([], {} as ResizeObserver);
      }
    });
    expect(screen.getByTestId("guide-step-list")).toBeVisible();
    expect(
      screen.queryByTestId("compact-step-navigator")
    ).not.toBeInTheDocument();

    Object.defineProperty(measurement, "scrollWidth", {
      configurable: true,
      value: 900,
    });
    act(() => {
      for (const callback of resizeObserverCallbacks) {
        callback([], {} as ResizeObserver);
      }
    });

    expect(screen.getByTestId("compact-step-navigator")).toBeVisible();
    expect(screen.queryByTestId("guide-step-list")).not.toBeInTheDocument();
    expect(screen.getByText("1 of 5")).toBeVisible();
    expect(screen.getByTestId("compact-active-step")).toHaveTextContent(
      "workspace-setup-guide.steps.project"
    );

    fireEvent.click(screen.getByTestId("open-step-list"));
    expect(await screen.findAllByRole("menuitem")).toHaveLength(5);
  });

  test("routes generic actions without recording guide analytics", () => {
    render(<WorkspaceSetupGuide />);
    fireEvent.click(screen.getByTestId("setup-step-create-project"));

    expect(mocks.routerPush).toHaveBeenCalled();
    expect(mocks.captureMetric).not.toHaveBeenCalled();
  });

  test("dismisses the selected guide without recording analytics", () => {
    mocks.scenarioId = "create-database-change";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
    });
    render(<WorkspaceSetupGuide />);

    fireEvent.click(screen.getByTestId("dismiss-guide"));

    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: GUIDE_PROGRESS_KEYS.dismissed,
      newState: true,
    });
    expect(mocks.captureMetric).not.toHaveBeenCalled();
  });

  test("shows generic completion with both next actions", () => {
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });

    render(<WorkspaceSetupGuide />);

    expect(
      screen.getByText("workspace-setup-guide.generic.completion-title")
    ).toBeVisible();
    expect(screen.getByText("workspace-setup-guide.actions.change")).toBeVisible();
    expect(screen.getByText("workspace-setup-guide.actions.query")).toBeVisible();
    fireEvent.click(screen.getByTestId("complete-guide"));
    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: guideCompletionAcknowledgedKey("workspace-setup"),
      newState: true,
    });
  });

  test("Query completion offers a database change", () => {
    mocks.scenarioId = "query-data";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      hasRunStatement: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });

    render(<WorkspaceSetupGuide />);

    expect(screen.getByText("workspace-setup-guide.actions.change")).toBeVisible();
    expect(screen.queryByText("workspace-setup-guide.actions.query")).not.toBeInTheDocument();
  });

  test("change completion offers SQL Editor", () => {
    mocks.scenarioId = "create-database-change";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      hasCreatedChangeIssue: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });

    render(<WorkspaceSetupGuide />);

    expect(screen.getByText("workspace-setup-guide.actions.query")).toBeVisible();
    expect(screen.queryByText("workspace-setup-guide.actions.change")).not.toBeInTheDocument();
  });

  test("does not record scenario guide lifecycle analytics", () => {
    mocks.scenarioId = "create-database-change";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      hasCreatedChangeIssue: true,
    });

    render(<WorkspaceSetupGuide />);

    expect(mocks.captureMetric).not.toHaveBeenCalled();
  });
});
