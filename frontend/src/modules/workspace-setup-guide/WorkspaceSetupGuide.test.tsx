import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { GuideContext } from "./types";
import { WorkspaceSetupGuide } from "./WorkspaceSetupGuide";

const guideContext = (
  overrides: Partial<GuideContext> = {}
): GuideContext => ({
  hasProject: false,
  hasInstance: false,
  hasExploredDatabase: false,
  hasFirstQuery: false,
  projectName: "",
  databaseProjectName: "",
  databaseName: "",
  route: { name: "workspace.home", params: {} },
  ...overrides,
});

const mocks = vi.hoisted(() => ({
  captureMetric: vi.fn(),
  currentRoute: {
    name: "workspace.home",
    params: {},
    query: {},
  } as {
    name?: string;
    params: Record<string, string | string[] | undefined>;
    query?: Record<string, string | string[] | undefined>;
  },
  guideContext: undefined as unknown as GuideContext,
  guideEnabled: true,
  introState: {} as Record<string, boolean>,
  loading: false,
  preCreateIssue: vi.fn(),
  productModelContent: "guide content" as string | undefined,
  routerPush: vi.fn(),
  saveIntroStateByKey: vi.fn(),
  workspaceName: "workspaces/default",
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string) => key,
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
  HowBytebaseWorksSheet: ({
    open,
    onOpenChange,
  }: {
    open: boolean;
    onOpenChange: (open: boolean) => void;
  }) =>
    open ? (
      <div data-testid="product-model-sheet">
        <button
          type="button"
          data-testid="close-product-model"
          onClick={() => onOpenChange(false)}
        >
          Close
        </button>
      </div>
    ) : null,
}));

vi.mock("@/components/SQLEditorButton", () => ({
  SQLEditorButton: ({
    className,
    label,
    ...props
  }: {
    className?: string;
    label?: ReactNode;
    database?: unknown;
    openInNewTab?: boolean;
    size?: string;
    "data-testid"?: string;
  }) => {
    const { database: _database, openInNewTab: _openInNewTab, size: _size, ...rest } =
      props;
    return (
      <a
        {...rest}
        className={`inline-flex h-7 gap-1 px-2 text-xs leading-4 ${className ?? ""}`}
      >
        {label}
      </a>
    );
  },
}));

vi.mock("@/hooks/useAppState", () => ({
  useIntroStateByKey: (key: string) => mocks.introState[key] ?? false,
}));

vi.mock("@/lib/plan/issue", () => ({
  preCreateIssue: mocks.preCreateIssue,
}));

vi.mock("@/stores/app", () => {
  const getState = () => ({
    workspaceResourceName: () => mocks.workspaceName,
    workspaceSetupGuideEnabled: () => mocks.guideEnabled,
  });
  const useAppStore = (selector: (state: ReturnType<typeof getState>) => unknown) =>
    selector(getState());
  useAppStore.getState = () => ({
    ...getState(),
    saveIntroStateByKey: mocks.saveIntroStateByKey,
  });
  return { useAppStore };
});

vi.mock("./useGuideContext", () => ({
  useGuideContext: () => ({
    context: mocks.guideContext,
    loading: mocks.loading,
  }),
}));

beforeEach(() => {
  vi.clearAllMocks();
  mocks.currentRoute = { name: "workspace.home", params: {}, query: {} };
  mocks.guideContext = guideContext();
  mocks.guideEnabled = true;
  mocks.introState = {};
  mocks.loading = false;
  mocks.productModelContent = "guide content";
  mocks.workspaceName = "workspaces/default";
  mocks.saveIntroStateByKey.mockImplementation(({ key, newState }) => {
    mocks.introState[key] = newState;
  });
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("WorkspaceSetupGuide", () => {
  test("renders the approved scenario order with legacy test IDs", () => {
    render(<WorkspaceSetupGuide />);
    expect(
      screen
        .getAllByTestId(/^setup-step-/)
        .map((element) => element.getAttribute("data-testid"))
    ).toEqual([
      "setup-step-hasProject",
      "setup-step-hasInstance",
      "setup-step-hasExploredDatabase",
      "setup-step-hasFirstQuery",
    ]);
  });

  test("blocks dependency steps and preserves their tooltip", async () => {
    vi.useFakeTimers();
    render(<WorkspaceSetupGuide />);
    const exploreStep = screen.getByTestId("setup-step-hasExploredDatabase");
    expect(exploreStep).toBeDisabled();
    expect(screen.getByTestId("setup-step-hasFirstQuery")).toBeDisabled();
    fireEvent.focusIn(exploreStep.parentElement!);
    await act(async () => {
      vi.advanceTimersByTime(100);
      await Promise.resolve();
    });
    expect(
      document.getElementById("bb-react-layer-overlay")?.textContent
    ).toContain("workspace-setup-guide.previous-step-required");
  });

  test("records and dispatches a selected step action", () => {
    render(<WorkspaceSetupGuide />);
    fireEvent.click(screen.getByTestId("setup-step-hasInstance"));
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "setup guide action clicked",
      properties: { step: "hasInstance" },
    });
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.instance",
      query: { intro: "create-instance" },
    });
  });

  test("renders query and change actions for the resolved query step", () => {
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });
    render(<WorkspaceSetupGuide />);
    expect(screen.getByTestId("active-action")).toBeVisible();
    expect(screen.getByTestId("secondary-action")).toBeVisible();
  });

  test("dismisses with the active step legacy analytics key", () => {
    render(<WorkspaceSetupGuide />);
    fireEvent.click(screen.getByTestId("dismiss-guide"));
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "setup guide dismissed",
      properties: { step: "hasProject" },
    });
    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: "workspace-setup-guide.dismissed",
      newState: true,
    });
  });

  test("opens the product model once for a new eligible guide", () => {
    render(<WorkspaceSetupGuide />);
    expect(screen.getByTestId("product-model-sheet")).toBeVisible();
  });

  test("does not auto-open the product model over a contextual intro", () => {
    mocks.currentRoute = {
      name: "workspace.project.database",
      params: {},
      query: { intro: "connect-database" },
    };
    render(<WorkspaceSetupGuide />);
    expect(screen.queryByTestId("product-model-sheet")).not.toBeInTheDocument();
  });

  test("does not auto-open a product model that was already seen", () => {
    mocks.introState["workspace-setup-guide.product-model-seen"] = true;
    render(<WorkspaceSetupGuide />);
    expect(screen.queryByTestId("product-model-sheet")).not.toBeInTheDocument();
  });

  test("disables the product model drawer when localized content is missing", () => {
    mocks.productModelContent = undefined;
    render(<WorkspaceSetupGuide />);
    expect(screen.queryByTestId("open-product-model")).not.toBeInTheDocument();
    expect(screen.queryByTestId("product-model-sheet")).not.toBeInTheDocument();
  });

  test("persists seen state when the product model closes", () => {
    render(<WorkspaceSetupGuide />);
    fireEvent.click(screen.getByTestId("close-product-model"));
    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: "workspace-setup-guide.product-model-seen",
      newState: true,
    });
  });

  test("uses the shared product model label for the guide control", () => {
    render(<WorkspaceSetupGuide />);
    expect(screen.getByTestId("open-product-model")).toHaveAttribute(
      "aria-label",
      "workspace-setup-guide.product-model"
    );
  });

  test("reopens the product model from the guide control", () => {
    mocks.introState["workspace-setup-guide.product-model-seen"] = true;
    render(<WorkspaceSetupGuide />);
    fireEvent.click(screen.getByTestId("open-product-model"));
    expect(screen.getByTestId("product-model-sheet")).toBeVisible();
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "setup guide action clicked",
      properties: {
        action: "product_model_open",
        source: "guide_bar",
      },
    });
  });

  test("uses caller-owned responsive button sizing", () => {
    const { container } = render(<WorkspaceSetupGuide />);
    expect(container.firstElementChild).toHaveClass("py-2", "2xl:py-4");
    expect(screen.getByTestId("setup-step-hasProject")).toHaveClass(
      "text-sm",
      "2xl:text-base",
      "py-1",
      "2xl:py-2"
    );
    expect(screen.getByTestId("dismiss-guide")).toHaveClass("h-7", "2xl:h-9");
  });

  test("shows setup step descriptions in tooltips", async () => {
    vi.useFakeTimers();
    const { container } = render(<WorkspaceSetupGuide />);
    expect(container).not.toHaveTextContent(
      "workspace-setup-guide.descriptions.project"
    );
    fireEvent.focusIn(screen.getByTestId("setup-step-hasProject"));
    await act(async () => {
      vi.advanceTimersByTime(100);
      await Promise.resolve();
    });
    expect(
      document.getElementById("bb-react-layer-overlay")?.textContent
    ).toContain("workspace-setup-guide.descriptions.project");
    expect(container).not.toHaveTextContent(
      "workspace-setup-guide.descriptions.project"
    );
  });

  test("shows previous-step guidance in tooltips for disabled setup steps", async () => {
    vi.useFakeTimers();
    render(<WorkspaceSetupGuide />);
    const exploreStep = screen.getByTestId("setup-step-hasExploredDatabase");
    expect(exploreStep).toBeDisabled();
    fireEvent.focusIn(exploreStep.parentElement!);
    await act(async () => {
      vi.advanceTimersByTime(100);
      await Promise.resolve();
    });
    expect(
      document.getElementById("bb-react-layer-overlay")?.textContent
    ).toContain("workspace-setup-guide.previous-step-required");
  });

  test("can be dismissed for the current workspace and user", () => {
    const { rerender } = render(<WorkspaceSetupGuide />);
    expect(screen.getByText("workspace-setup-guide.self")).toBeVisible();
    fireEvent.click(screen.getByTestId("dismiss-guide"));
    expect(mocks.saveIntroStateByKey).toHaveBeenCalledWith({
      key: "workspace-setup-guide.dismissed",
      newState: true,
    });
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "setup guide dismissed",
      properties: { step: "hasProject" },
    });
    rerender(<WorkspaceSetupGuide />);
    expect(screen.queryByText("workspace-setup-guide.self")).not.toBeInTheDocument();
  });

  test("captures the selected setup guide action", () => {
    render(<WorkspaceSetupGuide />);
    fireEvent.click(screen.getByTestId("setup-step-hasInstance"));
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "setup guide action clicked",
      properties: { step: "hasInstance" },
    });
  });

  test("stays hidden after it is dismissed", () => {
    mocks.introState["workspace-setup-guide.dismissed"] = true;
    render(<WorkspaceSetupGuide />);
    expect(screen.queryByText("workspace-setup-guide.self")).not.toBeInTheDocument();
  });

  test("does not replay create guidance from completed steps", () => {
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
    });
    render(<WorkspaceSetupGuide />);
    fireEvent.click(screen.getByTestId("setup-step-hasProject"));
    expect(mocks.routerPush).not.toHaveBeenCalled();
  });

  test("highlights the setup step matching the current route", () => {
    mocks.guideContext = guideContext({
      route: { name: "workspace.instance", params: {} },
    });
    mocks.currentRoute = { name: "workspace.instance", params: {}, query: {} };
    render(<WorkspaceSetupGuide />);
    expect(screen.getByTestId("setup-step-hasInstance")).toHaveClass(
      "bg-accent/10"
    );
    expect(screen.queryByTestId("active-action")).not.toBeInTheDocument();
  });

  test("does not show the active guide action when already on its route", () => {
    mocks.guideContext = guideContext({
      hasProject: true,
      projectName: "projects/app",
      route: { name: "workspace.instance.create", params: {} },
    });
    mocks.currentRoute = {
      name: "workspace.instance.create",
      params: {},
      query: {},
    };
    render(<WorkspaceSetupGuide />);
    expect(screen.queryByTestId("active-action")).not.toBeInTheDocument();
    expect(screen.getByTestId("dismiss-guide")).toBeVisible();
  });

  test("highlights the next incomplete step when the current route step is done", () => {
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      projectName: "projects/app",
      route: { name: "workspace.project", params: {} },
    });
    mocks.currentRoute = { name: "workspace.project", params: {}, query: {} };
    render(<WorkspaceSetupGuide />);
    expect(screen.getByTestId("setup-step-hasProject")).not.toHaveClass(
      "bg-accent/10"
    );
    expect(screen.getByTestId("setup-step-hasExploredDatabase")).toHaveClass(
      "bg-accent/10"
    );
    expect(screen.queryByTestId("active-action")).not.toBeInTheDocument();
  });

  test("does not highlight the next setup step on unrelated pages", () => {
    mocks.guideContext = guideContext({ hasProject: true });
    render(<WorkspaceSetupGuide />);
    for (const key of [
      "hasProject",
      "hasInstance",
      "hasExploredDatabase",
      "hasFirstQuery",
    ]) {
      expect(screen.getByTestId(`setup-step-${key}`)).not.toHaveClass(
        "bg-accent/10"
      );
    }
    expect(screen.queryByTestId("active-action")).not.toBeInTheDocument();
  });

  test("activates the query step when users click it after visiting another step", () => {
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      projectName: "projects/app",
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
      route: { name: "workspace.database", params: {} },
    });
    mocks.currentRoute = { name: "workspace.database", params: {}, query: {} };
    render(<WorkspaceSetupGuide />);
    fireEvent.click(screen.getByTestId("setup-step-hasInstance"));
    fireEvent.click(screen.getByTestId("setup-step-hasFirstQuery"));
    expect(screen.getByTestId("setup-step-hasFirstQuery")).toHaveClass(
      "bg-accent/10"
    );
    expect(screen.getByTestId("active-action")).toHaveTextContent(
      "workspace-setup-guide.actions.query"
    );
  });

  test("opens the first database change flow as a secondary action", () => {
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });
    render(<WorkspaceSetupGuide />);
    const changeButton = screen.getByTestId("secondary-action");
    expect(changeButton).toHaveTextContent(
      "workspace-setup-guide.actions.change"
    );
    expect(changeButton).toHaveClass("h-9");
    expect(screen.getByTestId("active-action")).toHaveClass("h-7", "2xl:h-9");
    expect(screen.getByTestId("dismiss-guide")).toHaveClass("h-7", "2xl:h-9");
    fireEvent.click(changeButton);
    expect(mocks.captureMetric).toHaveBeenCalledWith({
      event: "setup guide action clicked",
      properties: { step: "createFirstChange" },
    });
    expect(mocks.preCreateIssue).toHaveBeenCalledWith("projects/app", [
      "instances/sample/databases/employee",
    ]);
  });

  test("stays visible after the first query exists", () => {
    mocks.guideContext = guideContext({
      hasProject: true,
      hasInstance: true,
      hasExploredDatabase: true,
      hasFirstQuery: true,
      databaseProjectName: "projects/app",
      databaseName: "instances/sample/databases/employee",
    });
    render(<WorkspaceSetupGuide />);
    expect(screen.getByText("workspace-setup-guide.self")).toBeVisible();
    expect(screen.getByText("workspace-setup-guide.steps.query")).toBeVisible();
    expect(screen.getByTestId("active-action")).toHaveTextContent(
      "workspace-setup-guide.actions.query"
    );
  });
});
