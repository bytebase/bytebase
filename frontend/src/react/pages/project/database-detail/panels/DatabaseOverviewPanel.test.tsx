import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import {
  type Database,
  SyncStatus,
} from "@/types/proto-es/v1/database_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => {
  const currentRoute = {
    value: {
      query: {},
    },
  };

  return {
    localStorage: {
      clear: vi.fn(),
      getItem: vi.fn(() => null),
      removeItem: vi.fn(),
      setItem: vi.fn(),
    },
    useTranslation: vi.fn(() => ({
      t: (key: string) => key,
    })),
    schemaList: [{ name: "public" }, { name: "sales" }],
    currentRoute,
    routerReplace: vi.fn(() => Promise.resolve()),
    useVueState: vi.fn(),
    useDBSchemaV1Store: vi.fn(),
    getDatabaseProject: vi.fn((database: { project: string }) => ({
      name: database.project,
    })),
    hasProjectPermissionV2: vi.fn(
      (_project?: unknown, _permission?: string) => true
    ),
    getDatabaseEngine: vi.fn(() => Engine.POSTGRES),
    hasSchemaProperty: vi.fn(() => true),
    instanceV1HasCollationAndCharacterSet: vi.fn(() => true),
    instanceV1SupportsPackage: vi.fn(() => false),
    instanceV1SupportsSequence: vi.fn(() => false),
    bytesToString: vi.fn((size: number) => `${size} B`),
    humanizeDate: vi.fn(() => "5 minutes ago"),
  };
});

vi.stubGlobal("localStorage", mocks.localStorage);

let DatabaseOverviewPanel: typeof import("./DatabaseOverviewPanel").DatabaseOverviewPanel;

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/router", () => ({
  router: {
    replace: mocks.routerReplace,
    currentRoute: mocks.currentRoute,
  },
}));

vi.mock("@/store", () => ({
  useDBSchemaV1Store: mocks.useDBSchemaV1Store,
}));

vi.mock("@/utils", () => ({
  bytesToString: mocks.bytesToString,
  getDatabaseEngine: mocks.getDatabaseEngine,
  getDatabaseProject: mocks.getDatabaseProject,
  humanizeDate: mocks.humanizeDate,
  hasProjectPermissionV2: mocks.hasProjectPermissionV2,
  hasSchemaProperty: mocks.hasSchemaProperty,
  instanceV1HasCollationAndCharacterSet:
    mocks.instanceV1HasCollationAndCharacterSet,
  instanceV1SupportsPackage: mocks.instanceV1SupportsPackage,
  instanceV1SupportsSequence: mocks.instanceV1SupportsSequence,
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => (
    <input {...props} />
  ),
}));

vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: ({
    open,
    onOpenChange,
    children,
  }: {
    open: boolean;
    onOpenChange?: (open: boolean) => void;
    children: React.ReactNode;
  }) =>
    open ? (
      <div data-testid="dialog-root">
        <button type="button" onClick={() => onOpenChange?.(false)}>
          close-dialog
        </button>
        {children}
      </div>
    ) : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DialogTitle: ({ children }: { children: React.ReactNode }) => (
    <h1>{children}</h1>
  ),
}));

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: (nextElement = element) => {
      act(() => {
        root.render(nextElement);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

const flush = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
};

const setSelectValue = (select: HTMLSelectElement, value: string) => {
  act(() => {
    const descriptor = Object.getOwnPropertyDescriptor(
      HTMLSelectElement.prototype,
      "value"
    );
    descriptor?.set?.call(select, value);
    select.dispatchEvent(new Event("input", { bubbles: true }));
    select.dispatchEvent(new Event("change", { bubbles: true }));
  });
};

const clickElement = (element: Element) => {
  act(() => {
    element.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
};

const makeTable = (name: string) =>
  ({
    name,
    rowCount: 7,
    dataSize: 64n,
    indexSize: 32n,
    comment: `${name} comment`,
    columns: [
      {
        name: "id",
        type: "INT",
        comment: "",
      },
    ],
  }) as never;

const makeDatabase = (): Database =>
  ({
    name: "instances/inst1/databases/db",
    project: "projects/proj1",
    effectiveEnvironment: "environments/prod",
    state: State.ACTIVE,
    syncStatus: SyncStatus.OK,
    syncError: "",
    successfulSyncTime: {
      seconds: 1n,
      nanos: 0,
    },
    instanceResource: {
      name: "instances/inst1",
      title: "Primary",
      engine: Engine.POSTGRES,
    },
  }) as Database;

beforeEach(async () => {
  mocks.schemaList = [{ name: "public" }, { name: "sales" }];
  mocks.localStorage.clear.mockReset();
  mocks.localStorage.getItem.mockReset();
  mocks.localStorage.getItem.mockReturnValue(null);
  mocks.localStorage.removeItem.mockReset();
  mocks.localStorage.setItem.mockReset();
  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });
  mocks.currentRoute.value.query = {};
  mocks.routerReplace.mockReset();
  mocks.routerReplace.mockImplementation(() => Promise.resolve());
  mocks.getDatabaseProject.mockReset();
  mocks.getDatabaseProject.mockImplementation(
    (database: { project: string }) => ({
      name: database.project,
    })
  );
  mocks.hasProjectPermissionV2.mockReset();
  mocks.hasProjectPermissionV2.mockReturnValue(true);
  mocks.getDatabaseEngine.mockReset();
  mocks.getDatabaseEngine.mockReturnValue(Engine.POSTGRES);
  mocks.hasSchemaProperty.mockReset();
  mocks.hasSchemaProperty.mockReturnValue(true);
  mocks.instanceV1HasCollationAndCharacterSet.mockReset();
  mocks.instanceV1HasCollationAndCharacterSet.mockReturnValue(true);
  mocks.instanceV1SupportsPackage.mockReset();
  mocks.instanceV1SupportsPackage.mockReturnValue(false);
  mocks.instanceV1SupportsSequence.mockReset();
  mocks.instanceV1SupportsSequence.mockReturnValue(false);
  mocks.bytesToString.mockReset();
  mocks.bytesToString.mockImplementation((size: number) => `${size} B`);
  mocks.humanizeDate.mockReset();
  mocks.humanizeDate.mockReturnValue("5 minutes ago");
  mocks.useDBSchemaV1Store.mockReset();
  mocks.useDBSchemaV1Store.mockReturnValue({
    getSchemaList: vi.fn(() => mocks.schemaList),
    getTableList: vi.fn(() => [makeTable("orders")]),
    getViewList: vi.fn(() => []),
    getExtensionList: vi.fn(() => []),
    getExternalTableList: vi.fn(() => []),
    getFunctionList: vi.fn(() => []),
    getDatabaseMetadata: vi.fn(() => ({
      characterSet: "UTF8",
      collation: "en_US.UTF-8",
      schemas: [],
    })),
  });
  mocks.useVueState.mockReset();
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());

  vi.resetModules();
  ({ DatabaseOverviewPanel } = await import("./DatabaseOverviewPanel"));
});

describe("DatabaseOverviewPanel", () => {
  test("renders legacy overview info fields and schema selector", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: true,
      })
    );

    render();
    await flush();

    expect(container.textContent).toContain("db.encoding");
    expect(container.textContent).toContain("UTF8");
    expect(container.textContent).toContain("db.collation");
    expect(container.textContent).toContain("en_US.UTF-8");
    expect(container.textContent).toContain("database.sync-status");
    expect(container.textContent).toContain("common.ok");
    expect(container.textContent).toContain("database.last-sync");
    expect(container.textContent).toContain("5 minutes ago");
    expect(container.querySelector("select")).not.toBeNull();
    expect(
      container.querySelector('input[placeholder="common.filter-by-name"]')
    ).not.toBeNull();

    unmount();
  });

  test("renders failed sync status and error from the legacy overview behavior", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: {
          ...makeDatabase(),
          syncStatus: SyncStatus.FAILED,
          syncError: "sync failed hard",
        },
        hasSchemaPermission: false,
      })
    );

    render();
    await flush();

    expect(container.textContent).toContain("database.sync-status-failed");
    expect(container.textContent).toContain("sync failed hard");

    unmount();
  });

  test("renders dash for last sync when the sync timestamp is missing", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: {
          ...makeDatabase(),
          successfulSyncTime: undefined,
        },
        hasSchemaPermission: false,
      })
    );

    render();
    await flush();

    expect(container.textContent).toContain("database.last-sync");
    expect(container.textContent).toContain("-");
    expect(mocks.humanizeDate).not.toHaveBeenCalled();

    unmount();
  });

  test("syncs selected schema back to the route query", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: true,
      })
    );

    render();
    await flush();

    const select = container.querySelector(
      "select"
    ) as HTMLSelectElement | null;
    expect(select).not.toBeNull();

    setSelectValue(select as HTMLSelectElement, "public");
    await flush();

    expect(mocks.routerReplace).toHaveBeenCalledWith(
      expect.objectContaining({
        query: expect.objectContaining({ schema: "public" }),
      })
    );

    unmount();
  });

  test("syncs selected table to the route query and restores it from the route", async () => {
    const { router } = await import("@/router");
    router.currentRoute.value.query = { schema: "public", table: "orders" };

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: true,
      })
    );

    render();
    await flush();

    expect(container.textContent).toContain("orders");
    expect(
      container.querySelector('[data-testid="dialog-root"]')
    ).not.toBeNull();

    const closeButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "close-dialog"
    );
    expect(closeButton).toBeTruthy();
    clickElement(closeButton as HTMLButtonElement);
    await flush();

    expect(mocks.routerReplace).toHaveBeenCalledWith(
      expect.objectContaining({
        query: expect.objectContaining({
          schema: "public",
          table: undefined,
        }),
      })
    );
    router.currentRoute.value.query = { schema: "public" };

    const ordersCell = Array.from(container.querySelectorAll("td")).find(
      (cell) => cell.textContent === "orders"
    );
    expect(ordersCell?.closest("tr")).toBeTruthy();
    clickElement(ordersCell?.closest("tr") as HTMLTableRowElement);
    await flush();

    expect(mocks.routerReplace).toHaveBeenCalledWith(
      expect.objectContaining({
        query: expect.objectContaining({
          schema: "public",
          table: "orders",
        }),
      })
    );

    unmount();
    router.currentRoute.value.query = {};
  });

  test("hides the schema explorer when schema permission is missing", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: false,
      })
    );

    render();
    await flush();

    expect(container.textContent).toContain("db");
    expect(container.querySelector("select")).toBeNull();

    unmount();
  });

  test("renders loading states for object sections before the schema selection resolves", async () => {
    mocks.schemaList = [{ name: "" }];
    mocks.useDBSchemaV1Store.mockReturnValue({
      getSchemaList: vi.fn(() => mocks.schemaList),
      getTableList: vi.fn(() => []),
      getViewList: vi.fn(() => []),
      getExtensionList: vi.fn(() => []),
      getExternalTableList: vi.fn(() => []),
      getFunctionList: vi.fn(() => []),
      getDatabaseMetadata: vi.fn(() => ({
        characterSet: "UTF8",
        collation: "en_US.UTF-8",
        schemas: [],
      })),
    });

    vi.resetModules();
    ({ DatabaseOverviewPanel } = await import("./DatabaseOverviewPanel"));

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: true,
      })
    );

    render();

    expect(container.textContent).toContain("common.loading");
    expect(container.textContent).not.toContain(
      "common.namecommon.definitioncommon.comment-"
    );

    unmount();
  });

  test("keeps an existing schema query instead of clearing it during initialization", async () => {
    const { router } = await import("@/router");
    router.currentRoute.value.query = { schema: "sales" };

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: true,
      })
    );

    render();
    await flush();

    expect((container.querySelector("select") as HTMLSelectElement).value).toBe(
      "sales"
    );
    expect(mocks.routerReplace).not.toHaveBeenCalledWith({
      query: expect.objectContaining({
        schema: undefined,
      }),
    });

    unmount();
    router.currentRoute.value.query = {};
  });

  test("flattens package objects across schemas when the engine has no schema property", async () => {
    mocks.getDatabaseEngine.mockReturnValue(Engine.ORACLE);
    mocks.hasSchemaProperty.mockReturnValue(false);
    mocks.instanceV1HasCollationAndCharacterSet.mockReturnValue(true);
    mocks.instanceV1SupportsPackage.mockReturnValue(true);
    mocks.useDBSchemaV1Store.mockReturnValue({
      getSchemaList: vi.fn(() => []),
      getTableList: vi.fn(() => []),
      getViewList: vi.fn(() => []),
      getExtensionList: vi.fn(() => []),
      getExternalTableList: vi.fn(() => []),
      getFunctionList: vi.fn(() => []),
      getDatabaseMetadata: vi.fn(() => ({
        characterSet: "AL32UTF8",
        collation: "",
        schemas: [
          {
            name: "alpha",
            packages: [{ name: "pkg_a", definition: "a" }],
            sequences: [],
            streams: [],
            tasks: [],
          },
          {
            name: "beta",
            packages: [{ name: "pkg_b", definition: "b" }],
            sequences: [],
            streams: [],
            tasks: [],
          },
        ],
      })),
    });

    vi.resetModules();
    ({ DatabaseOverviewPanel } = await import("./DatabaseOverviewPanel"));

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: true,
      })
    );

    render();
    await flush();

    expect(container.textContent).toContain("pkg_a");
    expect(container.textContent).toContain("pkg_b");

    unmount();
  });

  test("uses function signatures for overloaded function identity and display", async () => {
    mocks.useDBSchemaV1Store.mockReturnValue({
      getSchemaList: vi.fn(() => mocks.schemaList),
      getTableList: vi.fn(() => [makeTable("orders")]),
      getViewList: vi.fn(() => []),
      getExtensionList: vi.fn(() => []),
      getExternalTableList: vi.fn(() => []),
      getFunctionList: vi.fn(() => [
        { name: "compute", signature: "compute(int)", definition: "sql 1" },
        { name: "compute", signature: "compute(text)", definition: "sql 2" },
      ]),
      getDatabaseMetadata: vi.fn(() => ({
        characterSet: "UTF8",
        collation: "en_US.UTF-8",
        schemas: [],
      })),
    });

    vi.resetModules();
    ({ DatabaseOverviewPanel } = await import("./DatabaseOverviewPanel"));

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: true,
      })
    );

    render();
    await flush();

    expect(container.textContent).toContain("compute(int)");
    expect(container.textContent).toContain("compute(text)");

    unmount();
  });

  test("flattens sequences across schemas when schema properties are unsupported", async () => {
    mocks.hasSchemaProperty.mockReturnValue(false);
    mocks.instanceV1SupportsSequence.mockReturnValue(true);
    mocks.useDBSchemaV1Store.mockReturnValue({
      getSchemaList: vi.fn(() => []),
      getTableList: vi.fn(() => []),
      getViewList: vi.fn(() => []),
      getExtensionList: vi.fn(() => []),
      getExternalTableList: vi.fn(() => []),
      getFunctionList: vi.fn(() => []),
      getDatabaseMetadata: vi.fn(() => ({
        characterSet: "UTF8",
        collation: "",
        schemas: [
          { name: "alpha", sequences: [{ name: "seq_a", dataType: "BIGINT" }] },
          { name: "beta", sequences: [{ name: "seq_b", dataType: "BIGINT" }] },
        ],
      })),
    });

    vi.resetModules();
    ({ DatabaseOverviewPanel } = await import("./DatabaseOverviewPanel"));

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: true,
      })
    );

    render();
    await flush();

    expect(container.textContent).toContain("seq_a");
    expect(container.textContent).toContain("seq_b");

    unmount();
  });

  test("does not fall back to the first schema for schema-scoped sequences", async () => {
    const { router } = await import("@/router");
    router.currentRoute.value.query = { schema: "sales" };

    mocks.instanceV1SupportsSequence.mockReturnValue(true);
    mocks.useDBSchemaV1Store.mockReturnValue({
      getSchemaList: vi.fn(() => mocks.schemaList),
      getTableList: vi.fn(() => []),
      getViewList: vi.fn(() => []),
      getExtensionList: vi.fn(() => []),
      getExternalTableList: vi.fn(() => []),
      getFunctionList: vi.fn(() => []),
      getDatabaseMetadata: vi.fn(() => ({
        characterSet: "UTF8",
        collation: "",
        schemas: [
          { name: "alpha", sequences: [{ name: "seq_a", dataType: "BIGINT" }] },
        ],
      })),
    });

    vi.resetModules();
    ({ DatabaseOverviewPanel } = await import("./DatabaseOverviewPanel"));

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: true,
      })
    );

    render();
    await flush();

    expect(container.textContent).not.toContain("seq_a");

    unmount();
    router.currentRoute.value.query = {};
  });

  test("flattens streams and tasks across schemas when schema properties are unsupported", async () => {
    mocks.getDatabaseEngine.mockReturnValue(Engine.SNOWFLAKE);
    mocks.hasSchemaProperty.mockReturnValue(false);
    mocks.useDBSchemaV1Store.mockReturnValue({
      getSchemaList: vi.fn(() => []),
      getTableList: vi.fn(() => []),
      getViewList: vi.fn(() => []),
      getExtensionList: vi.fn(() => []),
      getExternalTableList: vi.fn(() => []),
      getFunctionList: vi.fn(() => []),
      getDatabaseMetadata: vi.fn(() => ({
        characterSet: "UTF8",
        collation: "",
        schemas: [
          {
            name: "alpha",
            streams: [{ name: "stream_a", tableName: "orders" }],
            tasks: [{ name: "task_a", schedule: "cron_a" }],
          },
          {
            name: "beta",
            streams: [{ name: "stream_b", tableName: "users" }],
            tasks: [{ name: "task_b", schedule: "cron_b" }],
          },
        ],
      })),
    });

    vi.resetModules();
    ({ DatabaseOverviewPanel } = await import("./DatabaseOverviewPanel"));

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: true,
      })
    );

    render();
    await flush();

    expect(container.textContent).toContain("stream_a");
    expect(container.textContent).toContain("stream_b");
    expect(container.textContent).toContain("task_a");
    expect(container.textContent).toContain("task_b");

    unmount();
  });

  test("does not fall back to the first schema for schema-scoped streams and tasks", async () => {
    const { router } = await import("@/router");
    router.currentRoute.value.query = { schema: "sales" };

    mocks.getDatabaseEngine.mockReturnValue(Engine.SNOWFLAKE);
    mocks.useDBSchemaV1Store.mockReturnValue({
      getSchemaList: vi.fn(() => mocks.schemaList),
      getTableList: vi.fn(() => []),
      getViewList: vi.fn(() => []),
      getExtensionList: vi.fn(() => []),
      getExternalTableList: vi.fn(() => []),
      getFunctionList: vi.fn(() => []),
      getDatabaseMetadata: vi.fn(() => ({
        characterSet: "UTF8",
        collation: "",
        schemas: [
          {
            name: "alpha",
            streams: [{ name: "stream_a", tableName: "orders" }],
            tasks: [{ name: "task_a", schedule: "cron_a" }],
          },
        ],
      })),
    });

    vi.resetModules();
    ({ DatabaseOverviewPanel } = await import("./DatabaseOverviewPanel"));

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: true,
      })
    );

    render();
    await flush();

    expect(container.textContent).not.toContain("stream_a");
    expect(container.textContent).not.toContain("task_a");

    unmount();
    router.currentRoute.value.query = {};
  });

  test("does not fall back to the first schema for schema-scoped package objects", async () => {
    const { router } = await import("@/router");
    router.currentRoute.value.query = { schema: "sales" };

    mocks.getDatabaseEngine.mockReturnValue(Engine.ORACLE);
    mocks.hasSchemaProperty.mockReturnValue(true);
    mocks.instanceV1SupportsPackage.mockReturnValue(true);
    mocks.useDBSchemaV1Store.mockReturnValue({
      getSchemaList: vi.fn(() => mocks.schemaList),
      getTableList: vi.fn(() => []),
      getViewList: vi.fn(() => []),
      getExtensionList: vi.fn(() => []),
      getExternalTableList: vi.fn(() => []),
      getFunctionList: vi.fn(() => []),
      getDatabaseMetadata: vi.fn(() => ({
        characterSet: "AL32UTF8",
        collation: "",
        schemas: [
          {
            name: "alpha",
            packages: [{ name: "pkg_a", definition: "a" }],
            sequences: [],
            streams: [],
            tasks: [],
          },
        ],
      })),
    });

    vi.resetModules();
    ({ DatabaseOverviewPanel } = await import("./DatabaseOverviewPanel"));

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: true,
      })
    );

    render();
    await flush();

    expect((container.querySelector("select") as HTMLSelectElement).value).toBe(
      "sales"
    );
    expect(container.textContent).not.toContain("pkg_a");

    unmount();
    router.currentRoute.value.query = {};
  });
});
