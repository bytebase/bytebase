import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true;

const mocks = vi.hoisted(() => ({
  setPendingInsertAtCaret: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: vi.fn() },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("react-resizable-panels", () => ({
  Group: ({ children }: { children: ReactElement | ReactElement[] }) => (
    <div>{children}</div>
  ),
  Panel: ({ children }: { children: ReactElement | ReactElement[] }) => (
    <div>{children}</div>
  ),
  Separator: () => <div />,
}));

vi.mock("@/app/router", () => ({
  useNavigate: () => ({
    resolve: vi.fn(() => ({ fullPath: "/plans/create" })),
  }),
}));

vi.mock("@/components/IAMRemindDialog", () => ({
  IAMRemindDialog: () => <div />,
}));

vi.mock("@/components/WorkspaceSetupGuide", () => ({
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
    selector({ setPendingInsertAtCaret: mocks.setPendingInsertAtCaret }),
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

beforeEach(() => {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
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
