import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorDatabaseQueryContext } from "@/types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  setResultPanelMaximized: vi.fn(),
  state: {
    resultPanelMaximized: false,
    databaseQueryContexts: new Map<string, unknown[]>(),
  },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/lib/utils", () => ({
  cn: (...classes: Array<string | false | undefined>) =>
    classes.filter(Boolean).join(" "),
}));

vi.mock("@/components/ui/alert", () => ({ Alert: () => <div /> }));
vi.mock("@/components/ui/context-menu", () => ({
  ContextMenu: ({ children }: { children: ReactElement }) => children,
  ContextMenuContent: () => null,
  ContextMenuItem: () => null,
  ContextMenuTrigger: ({ render }: { render: ReactElement }) => render,
}));
vi.mock("@/components/ui/tabs", () => ({
  Tabs: ({ children }: { children: ReactElement | ReactElement[] }) => (
    <div>{children}</div>
  ),
  TabsList: ({ children }: { children: ReactElement | ReactElement[] }) => (
    <div>{children}</div>
  ),
  TabsPanel: ({ children }: { children: ReactElement | ReactElement[] }) => (
    <div>{children}</div>
  ),
  TabsTrigger: ({ children }: { children: ReactElement | ReactElement[] }) => (
    <div>{children}</div>
  ),
}));
vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactElement }) => children,
}));

// The real selector picks a database in an effect of its own; leaving it inert
// reproduces the window right after a remount, when nothing is selected yet.
vi.mock("./BatchQuerySelect", () => ({
  BatchQuerySelect: () => <div>batch-query-select</div>,
}));
vi.mock("./DatabaseQueryContext", () => ({
  DatabaseQueryContext: () => <div>database-query-context</div>,
}));
vi.mock("./MaximizeToggle", () => ({
  MaximizeToggle: () => <div>maximize-toggle</div>,
}));

vi.mock("@/modules/sql-editor/store", () => {
  const state = {
    get resultPanelMaximized() {
      return mocks.state.resultPanelMaximized;
    },
    setResultPanelMaximized: mocks.setResultPanelMaximized,
  };
  return {
    useSQLEditorStore: (selector: (s: typeof state) => unknown) =>
      selector(state),
  };
});

vi.mock("@/modules/sql-editor/store/tab", () => ({
  getSQLEditorTabsState: vi.fn(),
  useCurrentSQLEditorTab: () => ({ id: "tab-1" }),
  useIsInBatchMode: () => false,
  useSQLEditorTabState: (selector: (s: unknown) => unknown) =>
    selector({
      currentTabId: "tab-1",
      tabsById: new Map([
        ["tab-1", { databaseQueryContexts: mocks.state.databaseQueryContexts }],
      ]),
    }),
}));

vi.mock("@/types", () => ({ getDataSourceTypeI18n: () => "" }));
vi.mock("@/utils", () => ({
  formatAbsoluteDateTime: () => "now",
  getInstanceResource: () => ({ dataSources: [] }),
}));

import { ResultPanel } from "./ResultPanel";

const aContext = () =>
  ({
    id: "ctx-1",
    status: "DONE",
    params: { statement: "SELECT 1", connection: { dataSourceId: "ds" } },
  }) as unknown as SQLEditorDatabaseQueryContext;

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot> | undefined;

const render = () => {
  container = document.createElement("div");
  document.body.append(container);
  const mounted = createRoot(container);
  root = mounted;
  act(() => {
    mounted.render(<ResultPanel />);
  });
};

beforeEach(() => {
  document.body.innerHTML = "";
  mocks.setResultPanelMaximized.mockClear();
  mocks.state.resultPanelMaximized = false;
  mocks.state.databaseQueryContexts = new Map();
});

afterEach(() => {
  if (!root) return;
  const mounted = root;
  root = undefined;
  act(() => {
    mounted.unmount();
  });
});

describe("ResultPanel maximized handoff", () => {
  // A tab switch remounts this panel, and the database selection lands an
  // effect later. Reading the tab strip instead of the results would treat
  // that gap as "nothing to show" and drop the maximized pane every time.
  test("keeps a maximized pane while the database selection is pending", () => {
    mocks.state.resultPanelMaximized = true;
    mocks.state.databaseQueryContexts = new Map([["db-1", [aContext()]]]);
    render();

    expect(mocks.setResultPanelMaximized).not.toHaveBeenCalled();
  });

  test("restores the editor once the tab holds no results", () => {
    mocks.state.resultPanelMaximized = true;
    render();

    expect(mocks.setResultPanelMaximized).toHaveBeenCalledWith(false);
  });

  // Closing contexts one by one leaves the database key behind with an empty
  // array, so a key alone does not mean there is a strip to restore from.
  test("restores the editor when every database is emptied", () => {
    mocks.state.resultPanelMaximized = true;
    mocks.state.databaseQueryContexts = new Map([["db-1", []]]);
    render();

    expect(mocks.setResultPanelMaximized).toHaveBeenCalledWith(false);
  });

  test("leaves a docked pane alone", () => {
    render();

    expect(mocks.setResultPanelMaximized).not.toHaveBeenCalled();
  });
});
