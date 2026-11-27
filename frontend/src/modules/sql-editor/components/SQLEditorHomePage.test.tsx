import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { useSharedGuideContext } from "@/modules/workspace-setup-guide/GuideContextProvider";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true;

const mocks = vi.hoisted(() => ({
  state: { resultPanelMaximized: false },
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: vi.fn() },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/app/router", () => ({
  useCurrentRoute: () => ({ name: "workspace.sql-editor", params: {} }),
  useNavigate: () => ({
    resolve: vi.fn(() => ({ fullPath: "/plans/create" })),
  }),
}));

vi.mock("@/components/IAMRemindDialog", () => ({
  IAMRemindDialog: () => <div />,
}));

vi.mock("@/modules/workspace-setup-guide/WorkspaceSetupGuide", () => ({
  WorkspaceSetupGuide: () => {
    const { contextReady } = useSharedGuideContext();
    return <div>{contextReady ? "unified-guide" : "loading-guide"}</div>;
  },
}));

vi.mock("@/hooks/useAppState", () => ({
  useIntroStateByKey: () => false,
}));

vi.mock("@/modules/workspace-setup-guide/selection", () => ({
  readGuideWorkspaceUsage: () => undefined,
  readSelectedGuideScenarioId: () => undefined,
}));

vi.mock("@/modules/workspace-setup-guide/useGuideContext", () => ({
  useGuideContext: () => ({ context: {}, contextReady: true }),
}));

vi.mock("@/components/ui/layer", () => ({
  getLayerRoot: () => document.body,
  LAYER_BACKDROP_CLASS: "layer-backdrop",
  LAYER_SURFACE_CLASS: "layer-surface",
}));

vi.mock("@/hooks/useAppProject", () => ({
  useAppProject: () => ({ name: "" }),
}));

vi.mock("@/lib/plan/issue", () => ({ preCreateIssue: vi.fn() }));
vi.mock("@/lib/plan/title", () => ({ applyPlanTitleToQuery: vi.fn() }));
vi.mock("@/lib/utils", () => ({
  cn: (...classes: Array<string | false | undefined>) =>
    classes.filter(Boolean).join(" "),
}));
vi.mock("@/modules/schema-editor/resize", () => ({
  resizeHandleClass: () => "resize-handle",
}));

vi.mock("@/modules/sql-editor/components/AsidePanel", () => ({
  AsidePanel: () => <div />,
}));

vi.mock("@/modules/sql-editor/components/ConnectionPanel", () => ({
  ConnectionPanel: () => <div />,
}));

vi.mock("@/modules/sql-editor/components/OpenAIButton", () => ({ OpenAIButton: () => <div /> }));

vi.mock("@/modules/sql-editor/components/SQLEditorHeader", () => ({
  SQLEditorHeader: () => <div />,
}));

vi.mock("@/modules/sql-editor/components/TabList", () => ({
  TabList: () => <div data-testid="tabs" />,
}));

vi.mock("@/modules/sql-editor/components/SQLEditorAgentHost", () => ({
  SQLEditorAgentHost: ({ children }: { children: ReactElement }) => (
    <div data-testid="agent-host">{children}</div>
  ),
}));

vi.mock("@/modules/sql-editor/components/Panels/Panels", () => ({
  Panels: () => <div data-testid="panels" />,
}));

vi.mock(
  "@/modules/sql-editor/components/theme/SQLEditorThemeScope",
  () => ({
    SQLEditorThemeScope: ({ children }: { children: ReactElement }) => children,
    useSQLEditorTheme: () => ({}),
  })
);

vi.mock("@/modules/sql-editor/model/events", () => ({
  sqlEditorEvents: { on: () => () => {} },
}));

vi.mock("@/modules/sql-editor/store", () => ({
  useSQLEditorStore: (selector: (state: unknown) => unknown) =>
    selector({
      resultPanelMaximized: mocks.state.resultPanelMaximized,
    }),
}));

vi.mock("@/modules/sql-editor/store/editor", () => ({
  useSQLEditorEditorState: (selector: (state: unknown) => unknown) =>
    selector({ projectContextReady: false, project: "" }),
}));

vi.mock("@/modules/sql-editor/store/tab", () => ({
  getSQLEditorTabsState: vi.fn(),
  useCurrentSQLEditorTab: () => undefined,
  useIsDisconnected: () => true,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: unknown) => unknown) =>
      selector({ workspaceSetupGuideEnabled: () => true }),
    { getState: vi.fn() }
  ),
}));

vi.mock("@/types", () => ({ unknownProject: () => ({ name: "" }) }));
vi.mock("@/utils", () => ({
  extractDatabaseResourceName: vi.fn(),
  extractProjectResourceName: vi.fn(),
}));

import { SQLEditorHomePage } from "./SQLEditorHomePage";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

const setWindowWidth = (width: number) => {
  Object.defineProperty(window, "innerWidth", {
    value: width,
    configurable: true,
  });
};

beforeEach(() => {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
  setWindowWidth(1440);
  mocks.state.resultPanelMaximized = false;
  vi.spyOn(HTMLElement.prototype, "offsetWidth", "get").mockReturnValue(1000);
  vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockReturnValue(800);
});

afterEach(() => {
  act(() => root.unmount());
  container.remove();
  vi.restoreAllMocks();
});

const render = () => {
  act(() => root.render(<SQLEditorHomePage />));
};

describe("SQLEditorHomePage guide", () => {
  test("hosts tabs and panels in one Agent main column without a connection", () => {
    render();
    const hosts = container.querySelectorAll('[data-testid="agent-host"]');
    expect(hosts).toHaveLength(1);
    expect(hosts[0].querySelector('[data-testid="tabs"]')).not.toBeNull();
    expect(hosts[0].querySelector('[data-testid="panels"]')).not.toBeNull();
  });

  test("renders the unified guide", () => {
    render();

    expect(container.textContent).toContain("unified-guide");
  });
});

// The sidebar goes away for two unrelated reasons, and only a narrow window
// wants the phone treatment: a floating toggle over a drawer. A maximized
// result pane must not drag that onto a desktop window.
describe("SQLEditorHomePage sidebar", () => {
  const sidebar = () => container.querySelector<HTMLElement>("[data-panel]")!;
  const separator = () => container.querySelector<HTMLElement>('[role="separator"]')!;
  const resize = (width: number) => {
    act(() => {
      setWindowWidth(width);
      window.dispatchEvent(new Event("resize"));
    });
  };
  const sidebarToggles = () =>
    document.querySelectorAll(
      'button[aria-label="Expand sidebar"], button[aria-label="Collapse sidebar"]'
    );
  const sidebarDrawers = () =>
    document.querySelectorAll('[role="dialog"][aria-label="Sidebar"]');

  test("leaves the phone drawer out of a maximized desktop window", () => {
    mocks.state.resultPanelMaximized = true;
    render();

    expect(sidebarToggles()).toHaveLength(0);
    expect(sidebarDrawers()).toHaveLength(0);
  });

  test("keeps the phone toggle in a narrow window", () => {
    setWindowWidth(600);
    render();

    expect(sidebarToggles()).toHaveLength(1);
  });

  // Dragging a collapsed sidebar open would leave the result pane short of the
  // width its control still claims.
  test("locks the sidebar separator while the pane is maximized", () => {
    mocks.state.resultPanelMaximized = true;
    render();

    expect(separator().getAttribute("aria-disabled")).toBe("true");
  });

  test("leaves the sidebar separator draggable while the pane is docked", () => {
    render();

    expect(separator().getAttribute("aria-disabled")).not.toBe("true");
  });

  test.each([false, true])("restores the desktop sidebar after repeated narrow resizes with maximized=%s", (maximized) => {
    setWindowWidth(600);
    mocks.state.resultPanelMaximized = maximized;
    render();
    const mainColumn = container.querySelector('[data-testid="agent-host"]');
    for (let i = 0; i < 3; i++) {
      resize(1440);
      expect(Number(sidebar().style.flexGrow)).toBe(maximized ? 0 : 25);
      expect(container.querySelector('[data-testid="agent-host"]')).toBe(mainColumn);
      resize(600);
      expect(sidebarToggles()).toHaveLength(1);
    }
  });

  test("preserves manual sidebar collapse and the dragged width through maximize and restore", () => {
    render();
    const press = (key: string) => act(() => {
      separator().dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true }));
    });
    press("ArrowLeft");
    expect(sidebar().style.flexGrow).toBe("20");
    press("Enter");
    expect(sidebar().style.flexGrow).toBe("0");
    press("Enter");
    expect(sidebar().style.flexGrow).toBe("20");
    mocks.state.resultPanelMaximized = true;
    render();
    expect(sidebar().style.flexGrow).toBe("0");
    mocks.state.resultPanelMaximized = false;
    render();
    expect(sidebar().style.flexGrow).toBe("20");
  });
});
