import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { BatchQuerySelect } from "./BatchQuerySelect";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  databasesByName: {} as Record<string, Database>,
  databaseQueryContexts: new Map<string, unknown[]>(),
  environment: {
    name: "environments/prod",
    id: "prod",
    title: "Production",
    color: "#4f46e5",
    order: 0,
    tags: {},
  },
  deleteDatabaseQueryContext: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/DataExportButton", () => ({
  DataExportButton: () => null,
}));

vi.mock("@/components/database", () => ({
  DatabaseTableView: () => null,
}));

vi.mock("@/components/EngineIcon", () => ({
  EngineIcon: ({ className }: { className?: string }) => (
    <span className={className} data-testid="engine-icon" />
  ),
}));

vi.mock("@/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({ environment }: { environment: { title: string } }) => (
    <span>{environment.title}</span>
  ),
}));

vi.mock("@/components/ui/ellipsis-text", () => ({
  EllipsisText: ({
    children,
    className,
    text,
  }: {
    children?: ReactNode;
    className?: string;
    text: string;
  }) => <span className={className}>{children ?? text}</span>,
}));

vi.mock("@/modules/sql-editor/components/RequestExportButton", () => ({
  RequestExportButton: () => null,
}));

vi.mock("@/modules/sql-editor/components/useExportGrantBypass", () => ({
  useExportGrantBypass: () => ({
    matchedDatabases: [],
    unmatchedDatabases: [],
    tooltip: undefined,
  }),
}));

vi.mock("@/modules/sql-editor/hooks/useSQLEditorState", () => ({
  useSQLEditorQueryDataPolicy: () => ({
    disableExport: true,
    maximumResultRows: 100,
  }),
}));

vi.mock("@/modules/sql-editor/store/editor", () => ({
  useSQLEditorEditorState: (
    selector: (state: { project: string }) => unknown
  ) => selector({ project: "projects/p" }),
}));

vi.mock("@/modules/sql-editor/store/tab", () => {
  const state = () => ({
    currentTabId: "tab-1",
    tabsById: new Map([
      [
        "tab-1",
        {
          databaseQueryContexts: mocks.databaseQueryContexts,
        },
      ],
    ]),
    deleteDatabaseQueryContext: mocks.deleteDatabaseQueryContext,
  });
  return {
    getSQLEditorTabsState: state,
    useCurrentSQLEditorTab: () =>
      state().tabsById.get(state().currentTabId),
    useSQLEditorTabState: (
      selector: (value: ReturnType<typeof state>) => unknown
    ) => selector(state()),
  };
});

vi.mock("@/stores/app", () => {
  const state = () => ({
    databasesByName: mocks.databasesByName,
    environmentList: [mocks.environment],
    exportData: vi.fn(),
    getDatabaseByName: (name: string) => mocks.databasesByName[name],
    getEnvironmentByName: () => mocks.environment,
  });
  return {
    useAppStore: Object.assign(
      (selector: (value: ReturnType<typeof state>) => unknown) =>
        selector(state()),
      { getState: state }
    ),
  };
});

vi.mock("./ContextMenu", () => ({
  TabContextMenu: ({ children }: { children: ReactNode }) => children,
}));

describe("BatchQuerySelect", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;
  let cellA: Database;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.append(container);
    root = createRoot(container);

    cellA = {
      name: "instances/cell-a/databases/linear",
      effectiveEnvironment: mocks.environment.name,
      instanceResource: {
        engine: Engine.POSTGRES,
        name: "instances/cell-a",
        title: "cell-a",
      },
    } as Database;
    const cellB = {
      name: "instances/cell-b/databases/linear",
      effectiveEnvironment: mocks.environment.name,
      instanceResource: {
        engine: Engine.POSTGRES,
        name: "instances/cell-b",
        title: "cell-b",
      },
    } as Database;
    mocks.databasesByName = {
      [cellA.name]: cellA,
      [cellB.name]: cellB,
    };
    mocks.databaseQueryContexts = new Map([
      [
        cellA.name,
        [
          {
            id: "query-a",
            params: { connection: {}, statement: "SELECT 1" },
            status: "PENDING",
          },
        ],
      ],
      [
        cellB.name,
        [
          {
            id: "query-b",
            params: { connection: {}, statement: "SELECT 1" },
            status: "PENDING",
          },
        ],
      ],
    ]);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.clearAllMocks();
  });

  const render = () => {
    act(() => {
      root.render(
        <BatchQuerySelect
          selectedDatabase={cellA}
          onSelectedDatabaseChange={vi.fn()}
        />
      );
    });
    return Array.from(container.querySelectorAll("button"));
  };

  it("distinguishes same-named databases by instance and environment", () => {
    const buttons = render();

    expect(buttons).toHaveLength(2);
    expect(buttons[0].textContent).toContain("cell-a");
    expect(buttons[0].textContent).toContain("Production");
    expect(buttons[0].textContent).toContain("linear");
    expect(buttons[1].textContent).toContain("cell-b");
    expect(buttons[1].textContent).toContain("Production");
    expect(buttons[1].textContent).toContain("linear");
  });

  it("keeps each result tab from shrinking inside the horizontal scroller", () => {
    const buttons = render();

    expect(buttons[0].className).toContain("shrink-0");
    expect(buttons[1].className).toContain("shrink-0");
  });
});
