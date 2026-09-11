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
  contextReady: true,
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
    size,
    className,
    "data-testid": testId,
  }: {
    label?: ReactNode;
    size?: string;
    className?: string;
    "data-testid"?: string;
  }) => (
    <button
      className={className}
      data-testid={testId ?? "sql-editor-action"}
      data-size={size}
    >
      {label}
    </button>
  ),
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
    contextReady: mocks.contextReady,
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
  mocks.contextReady = true;
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
    expect(screen.getByTestId("active-action")).toHaveAttribute(
      "data-size",
      "sm"
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
      "setup-step-add-member",
    ]);
    expect(screen.getByTestId("setup-step-add-member")).toBeEnabled();
  });

  test("acknowledges a completed multi-member team journey when closed", () => {
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
    expect(screen.queryByTestId("complete-guide")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("dismiss-guide"));
    first.unmount();

    render(<WorkspaceSetupGuide />);
    expect(screen.queryByTestId("dismiss-guide")).not.toBeInTheDocument();
  });

  test("opens Users for a self-host team journey without another user", () => {
    mocks.workspaceUsage = "team";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });
    render(<WorkspaceSetupGuide />);

    fireEvent.click(screen.getByTestId("setup-step-add-member"));

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.users",
      query: { intro: "create-user" },
    });
  });

  test("opens Members for a self-host team journey with an existing user", () => {
    mocks.workspaceUsage = "team";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      hasOtherHumanUser: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });
    render(<WorkspaceSetupGuide />);

    fireEvent.click(screen.getByTestId("setup-step-add-member"));

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.members",
      query: { intro: "grant-access" },
    });
  });

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

    fireEvent.click(screen.getByTestId("setup-step-add-member"));

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

  test("sizes the active action independently of step overflow", () => {
    mocks.scenarioId = "query-data";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });
    render(<WorkspaceSetupGuide />);

    expect(screen.getByTestId("active-action")).toHaveAttribute(
      "data-size",
      "sm"
    );
    expect(screen.getByTestId("active-action")).toHaveClass(
      "2xl:h-9",
      "2xl:px-3",
      "2xl:text-sm"
    );

    const viewport = screen.getByTestId("guide-step-viewport");
    const measurement = screen.getByTestId("guide-step-measurement");
    Object.defineProperty(viewport, "clientWidth", {
      configurable: true,
      value: 500,
    });
    Object.defineProperty(measurement, "scrollWidth", {
      configurable: true,
      value: 900,
    });

    act(() => {
      for (const callback of resizeObserverCallbacks) {
        callback([], {} as ResizeObserver);
      }
    });
    expect(screen.getByTestId("active-action")).toHaveAttribute(
      "data-size",
      "sm"
    );

    expect(screen.getByTestId("active-action")).toHaveAttribute(
      "data-size",
      "sm"
    );
  });

  test("records a selected guide step action", () => {
    render(<WorkspaceSetupGuide />);
    mocks.captureMetric.mockClear();
    fireEvent.click(screen.getByTestId("setup-step-create-project"));

    expect(mocks.routerPush).toHaveBeenCalled();
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "workspace setup guide step action selected",
      properties: {
        journey: "workspace-setup",
        scenario: "unselected",
        collaboration_type: "unselected",
        completed_steps: [],
        completed_step_count: 0,
        total_step_count: 3,
        next_step: "create-project",
        step: "create-project",
        action_type: "navigate",
      },
    });
  });

  test("records the active SQL Editor action", () => {
    mocks.scenarioId = "query-data";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });
    render(<WorkspaceSetupGuide />);
    mocks.captureMetric.mockClear();

    fireEvent.click(screen.getByTestId("active-action"));

    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "workspace setup guide step action selected",
      properties: {
        journey: "query-data",
        scenario: "query-data",
        collaboration_type: "unselected",
        completed_steps: [
          "create-project",
          "connect-instance",
          "explore-database",
        ],
        completed_step_count: 3,
        total_step_count: 4,
        next_step: "query-data",
        step: "query-data",
        action_type: "open-sql-editor",
      },
    });
  });

  test("records final progress when selected guide dismissed", () => {
    mocks.scenarioId = "create-database-change";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });
    render(<WorkspaceSetupGuide />);
    mocks.captureMetric.mockClear();

    fireEvent.click(screen.getByTestId("dismiss-guide"));

    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: GUIDE_PROGRESS_KEYS.dismissed,
      newState: true,
    });
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "workspace setup guide dismissed",
      properties: {
        journey: "create-database-change",
        scenario: "create-database-change",
        collaboration_type: "unselected",
        completed_steps: [
          "create-project",
          "connect-instance",
          "explore-database",
        ],
        completed_step_count: 3,
        total_step_count: 4,
        next_step: "create-database-change",
      },
    });
  });

  test("records a step completion only when guide progress changes", () => {
    const { rerender } = render(<WorkspaceSetupGuide />);
    mocks.captureMetric.mockClear();

    mocks.guideContext = guideContext({ hasProject: true });
    rerender(<WorkspaceSetupGuide />);
    rerender(<WorkspaceSetupGuide />);

    expect(mocks.captureMetric).toHaveBeenCalledTimes(1);
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "workspace setup guide step completed",
      properties: {
        journey: "workspace-setup",
        scenario: "unselected",
        collaboration_type: "unselected",
        completed_steps: ["create-project"],
        completed_step_count: 1,
        total_step_count: 3,
        next_step: "connect-instance",
        step: "create-project",
      },
    });
  });

  test("records journey completion after visible guide becomes complete", () => {
    mocks.scenarioId = "query-data";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });
    const { rerender } = render(<WorkspaceSetupGuide />);
    mocks.captureMetric.mockClear();

    mocks.guideContext = guideContext({
      ...mocks.guideContext,
      hasRunStatement: true,
    });
    rerender(<WorkspaceSetupGuide />);

    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "workspace setup guide completed",
      properties: {
        journey: "query-data",
        scenario: "query-data",
        collaboration_type: "unselected",
        completed_steps: [
          "create-project",
          "connect-instance",
          "explore-database",
          "query-data",
        ],
        completed_step_count: 4,
        total_step_count: 4,
      },
    });
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
    expect(screen.queryByTestId("complete-guide")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("dismiss-guide"));
    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: guideCompletionAcknowledgedKey("workspace-setup"),
      newState: true,
    });
  });

  test("compacts completion actions only when the completion row overflows", () => {
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });

    render(<WorkspaceSetupGuide />);

    const guideBar = screen.getByTestId("workspace-setup-guide");
    const completionTitle = screen.getByTestId("completion-title");
    Object.defineProperty(guideBar, "clientWidth", {
      configurable: true,
      value: 800,
    });
    Object.defineProperty(completionTitle, "clientWidth", {
      configurable: true,
      value: 160,
    });
    Object.defineProperty(completionTitle, "scrollWidth", {
      configurable: true,
      value: 260,
    });

    act(() => {
      for (const callback of resizeObserverCallbacks) {
        callback([], {} as ResizeObserver);
      }
    });

    expect(
      screen.queryByText("workspace-setup-guide.generic.completion-description")
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText("workspace-setup-guide.actions.change")
    ).not.toBeInTheDocument();
    expect(screen.getByTestId("sql-editor-action")).toHaveAttribute(
      "data-size",
      "sm"
    );

    Object.defineProperty(guideBar, "clientWidth", {
      configurable: true,
      value: 900,
    });
    Object.defineProperty(completionTitle, "clientWidth", {
      configurable: true,
      value: 260,
    });
    act(() => {
      for (const callback of resizeObserverCallbacks) {
        callback([], {} as ResizeObserver);
      }
    });

    expect(
      screen.getByText("workspace-setup-guide.generic.completion-description")
    ).toBeVisible();
    expect(screen.getByText("workspace-setup-guide.actions.change")).toBeVisible();
    expect(screen.getByTestId("sql-editor-action")).not.toHaveAttribute(
      "data-size"
    );
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
    expect(screen.getByTestId("sql-editor-action")).not.toHaveAttribute(
      "data-size"
    );
    expect(screen.queryByText("workspace-setup-guide.actions.change")).not.toBeInTheDocument();
  });

  test("records the initial guide progress once", () => {
    mocks.scenarioId = "create-database-change";
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      hasCreatedChangeIssue: true,
    });

    render(<WorkspaceSetupGuide />);

    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: "workspace-setup-guide.progress-observed.create-database-change.v1",
      newState: true,
    });
    expect(mocks.captureMetric).toHaveBeenCalledTimes(1);
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "workspace setup guide progress observed",
      properties: {
        journey: "create-database-change",
        scenario: "create-database-change",
        collaboration_type: "unselected",
        completed_steps: [
          "create-project",
          "connect-instance",
          "create-database-change",
        ],
        completed_step_count: 3,
        total_step_count: 4,
        next_step: "explore-database",
        observation: "initial",
      },
    });
  });

  test("does not record initial guide progress after it was observed", () => {
    mocks.scenarioId = "create-database-change";
    mocks.introState[
      "workspace-setup-guide.progress-observed.create-database-change.v1"
    ] = true;

    render(<WorkspaceSetupGuide />);

    expect(mocks.captureMetric).not.toHaveBeenCalled();
  });

  test("records initial progress when its marker cannot be saved", () => {
    mocks.saveIntroStateByKey.mockImplementation(() => {
      throw new Error("localStorage unavailable");
    });

    expect(() => render(<WorkspaceSetupGuide />)).not.toThrow();
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "workspace setup guide progress observed",
      properties: expect.objectContaining({ observation: "initial" }),
    });
  });
});
