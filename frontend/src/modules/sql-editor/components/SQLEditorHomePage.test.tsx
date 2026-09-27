import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true;

const mocks = vi.hoisted(() => ({
  setPendingInsertAtCaret: vi.fn(),
  state: { resultPanelMaximized: false },
  collapse: vi.fn(),
  expand: vi.fn(),
  separatorDisabled: undefined as boolean | undefined,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: vi.fn() },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("react-resizable-panels", () => ({
  Group: ({ children }: { children: ReactElement | ReactElement[] }) => (
    <div>{children}</div>
  ),
  Panel: ({
    children,
    panelRef,
  }: {
    children: ReactElement | ReactElement[];
    panelRef?: { current: unknown };
  }) => {
    if (panelRef) {
      panelRef.current = { collapse: mocks.collapse, expand: mocks.expand };
    }
    return <div>{children}</div>;
  },
  Separator: ({ disabled }: { disabled?: boolean }) => {
    mocks.separatorDisabled = disabled;
    return <div />;
  },
}));

vi.mock("@/app/router", () => ({
  useNavigate: () => ({
    resolve: vi.fn(() => ({ fullPath: "/plans/create" })),
  }),
}));

vi.mock("@/components/IAMRemindDialog", () => ({
  IAMRemindDialog: () => <div />,
}));

vi.mock("@/modules/workspace-setup-guide/WorkspaceSetupGuide", () => ({
  WorkspaceSetupGuide: () => <div>unified-guide</div>,
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

vi.mock("@/modules/sql-editor/components/SQLEditorHeader", () => ({
  SQLEditorHeader: () => <div />,
}));

vi.mock("@/modules/sql-editor/components/TabList", () => ({
  TabList: () => <div />,
}));

vi.mock("@/modules/sql-editor/components/Panels/Panels", () => ({
  Panels: () => <div />,
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
      setPendingInsertAtCaret: mocks.setPendingInsertAtCaret,
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
    (selector: (state: unknown) => unknown) => selector({}),
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
  mocks.collapse.mockClear();
  mocks.expand.mockClear();
  mocks.separatorDisabled = undefined;
});

afterEach(() => {
  act(() => root.unmount());
  container.remove();
});

const render = () => {
  act(() => root.render(<SQLEditorHomePage />));
};

describe("SQLEditorHomePage guide", () => {
  test("renders the unified guide", () => {
    render();

    expect(container.textContent).toContain("unified-guide");
  });
});

// The sidebar goes away for two unrelated reasons, and only a narrow window
// wants the phone treatment: a floating toggle over a drawer. A maximized
// result pane must not drag that onto a desktop window.
describe("SQLEditorHomePage sidebar", () => {
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

    expect(mocks.separatorDisabled).toBe(true);
  });

  test("leaves the sidebar separator draggable while the pane is docked", () => {
    render();

    expect(mocks.separatorDisabled).toBe(false);
  });

  test("collapses the sidebar a maximized narrow window grows into", () => {
    setWindowWidth(600);
    mocks.state.resultPanelMaximized = true;
    render();
    // The desktop panel the collapse acts on does not exist while narrow.
    expect(mocks.collapse).not.toHaveBeenCalled();

    act(() => {
      setWindowWidth(1440);
      window.dispatchEvent(new Event("resize"));
    });

    expect(mocks.collapse).toHaveBeenCalled();
  });
});
