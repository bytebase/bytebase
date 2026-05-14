import { act, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorTreeNode } from "@/types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  allowAdmin: false,
  sqlEditorEventsEmit: vi.fn().mockResolvedValue(undefined),
  setShowConnectionPanel: vi.fn(),
  setAsidePanelTab: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useSQLEditorStore: () => ({ allowAdmin: mocks.allowAdmin }),
  useSQLEditorTabStore: () => ({ currentTab: null }),
  useSQLEditorWorksheetStore: () => ({
    createWorksheet: vi.fn().mockResolvedValue(undefined),
    maybeUpdateWorksheet: vi.fn().mockResolvedValue(undefined),
  }),
}));

vi.mock("@/react/stores/sqlEditor", () => ({
  useSQLEditorStore: Object.assign(
    (
      selector: (s: { setShowConnectionPanel: (v: boolean) => void }) => unknown
    ) =>
      selector({
        setShowConnectionPanel: mocks.setShowConnectionPanel,
      }),
    {
      getState: () => ({
        setAsidePanelTab: mocks.setAsidePanelTab,
      }),
    }
  ),
}));

vi.mock("@/router", () => ({
  router: { resolve: vi.fn(() => ({ href: "/x" })) },
}));

vi.mock("@/router/dashboard/projectV1", () => ({
  PROJECT_V1_ROUTE_DATABASE_DETAIL: "project.database.detail",
}));

vi.mock("@/views/sql-editor/events", () => ({
  sqlEditorEvents: { emit: mocks.sqlEditorEventsEmit },
}));

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: (name: string) => ({
    instance: "instances/prod",
    databaseName: name.split("/").pop() ?? "",
  }),
  extractInstanceResourceName: () => "prod",
  extractProjectResourceName: () => "p",
  getInstanceResource: () => ({ engine: "MYSQL" }),
  instanceV1HasAlterSchema: () => true,
  instanceV1HasReadonlyMode: () => true,
}));

vi.mock("@/types", async () => {
  return {
    instanceOfSQLEditorTreeNode: () => ({ engine: "MYSQL" }),
    isConnectableSQLEditorTreeNode: () => true,
  };
});

let useConnectionMenu: typeof import("./actions").useConnectionMenu;

const renderHook = <T,>(hookFn: () => T) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  let value: T;
  function Host() {
    value = hookFn();
    return null;
  }
  act(() => {
    root.render(<Host />);
  });
  return {
    get value() {
      return value;
    },
    unmount: () => act(() => root.unmount()),
  };
};

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

const makeDatabaseNode = (
  overrides?: Partial<{ disabled: boolean }>
): SQLEditorTreeNode =>
  ({
    key: "databases/bb",
    disabled: overrides?.disabled ?? false,
    meta: {
      type: "database",
      target: {
        name: "instances/prod/databases/bb",
        project: "projects/p",
      },
    },
  }) as unknown as SQLEditorTreeNode;

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useVueState.mockImplementation((getter) => getter());
  mocks.allowAdmin = false;
  ({ useConnectionMenu } = await import("./actions"));
});

describe("useConnectionMenu", () => {
  test("returns empty array when node is null", () => {
    const hook = renderHook(() => useConnectionMenu(null));
    expect(hook.value.items).toEqual([]);
    hook.unmount();
  });

  test("returns empty array when node is disabled", () => {
    const hook = renderHook(() =>
      useConnectionMenu(makeDatabaseNode({ disabled: true }))
    );
    expect(hook.value.items).toEqual([]);
    hook.unmount();
  });

  test("includes connect + new-tab + view-detail + alter-schema for a plain database node", () => {
    const hook = renderHook(() => useConnectionMenu(makeDatabaseNode()));
    const keys = hook.value.items.map((i) => i.key);
    expect(keys).toContain("connect");
    expect(keys).toContain("connect-in-new-tab");
    expect(keys).toContain("view-database-detail");
    expect(keys).toContain("alter-schema");
    expect(keys).not.toContain("connect-in-admin-mode");
    hook.unmount();
  });

  test("adds connect-in-admin-mode when editorStore.allowAdmin is true", () => {
    mocks.allowAdmin = true;
    const hook = renderHook(() => useConnectionMenu(makeDatabaseNode()));
    const keys = hook.value.items.map((i) => i.key);
    expect(keys).toContain("connect-in-admin-mode");
    hook.unmount();
  });

  test("alter-schema item emits sqlEditorEvents.alter-schema with the database name", () => {
    // Render via a React component so the menu item's handler runs in act
    // scope.
    function Harness() {
      const { items } = useConnectionMenu(makeDatabaseNode());
      return (
        <button
          type="button"
          data-testid="alter"
          onClick={() =>
            items.find((i) => i.key === "alter-schema")?.onSelect()
          }
        />
      );
    }
    const { container, render, unmount } = renderIntoContainer(<Harness />);
    render();
    act(() => {
      container
        .querySelector("[data-testid='alter']")
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(mocks.sqlEditorEventsEmit).toHaveBeenCalledWith(
      "alter-schema",
      expect.objectContaining({ databaseName: "instances/prod/databases/bb" })
    );
    unmount();
  });
});
