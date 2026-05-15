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
    useDatabaseCatalog: vi.fn(),
    useSubscriptionV1Store: vi.fn(),
    getColumnCatalog: vi.fn(() => ({
      semanticType: "",
      classification: "",
    })),
    getTableCatalog: vi.fn(),
    useSettingV1Store: vi.fn(),
    featureToRef: vi.fn(() => ({ value: true })),
    getDatabaseProject: vi.fn((database: { project: string }) => ({
      name: database.project,
      dataClassificationConfigId: "classification-config",
    })),
    getInstanceResource: vi.fn(
      (database: Database) => database.instanceResource
    ),
    hasProjectPermissionV2: vi.fn(
      (_project?: unknown, _permission?: string) => true
    ),
    getDatabaseEngine: vi.fn(() => Engine.POSTGRES),
    hasIndexSizeProperty: vi.fn(() => true),
    isDev: vi.fn(() => false),
    hasSchemaProperty: vi.fn(() => true),
    hasTableEngineProperty: vi.fn(() => false),
    instanceV1HasCollationAndCharacterSet: vi.fn(() => true),
    instanceV1MaskingForNoSQL: vi.fn(() => false),
    instanceV1SupportsColumn: vi.fn(() => true),
    instanceV1SupportsIndex: vi.fn(() => true),
    instanceV1SupportsPackage: vi.fn(() => false),
    instanceV1SupportsSequence: vi.fn(() => false),
    instanceV1SupportsTrigger: vi.fn(() => false),
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
  useDatabaseCatalog: mocks.useDatabaseCatalog,
  useSubscriptionV1Store: mocks.useSubscriptionV1Store,
  getColumnCatalog: mocks.getColumnCatalog,
  getTableCatalog: mocks.getTableCatalog,
  useSettingV1Store: mocks.useSettingV1Store,
  featureToRef: mocks.featureToRef,
}));

vi.mock("@/utils", () => ({
  bytesToString: mocks.bytesToString,
  getDatabaseEngine: mocks.getDatabaseEngine,
  getInstanceResource: mocks.getInstanceResource,
  getDatabaseProject: mocks.getDatabaseProject,
  humanizeDate: mocks.humanizeDate,
  hasIndexSizeProperty: mocks.hasIndexSizeProperty,
  isDev: mocks.isDev,
  hasProjectPermissionV2: mocks.hasProjectPermissionV2,
  hasSchemaProperty: mocks.hasSchemaProperty,
  hasTableEngineProperty: mocks.hasTableEngineProperty,
  instanceV1HasCollationAndCharacterSet:
    mocks.instanceV1HasCollationAndCharacterSet,
  instanceV1MaskingForNoSQL: mocks.instanceV1MaskingForNoSQL,
  instanceV1SupportsColumn: mocks.instanceV1SupportsColumn,
  instanceV1SupportsIndex: mocks.instanceV1SupportsIndex,
  instanceV1SupportsPackage: mocks.instanceV1SupportsPackage,
  instanceV1SupportsSequence: mocks.instanceV1SupportsSequence,
  instanceV1SupportsTrigger: mocks.instanceV1SupportsTrigger,
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => (
    <input {...props} />
  ),
}));

vi.mock("@/react/components/ui/select", () => ({
  Select: ({
    children,
    disabled,
    onValueChange,
    value,
  }: {
    children: React.ReactNode;
    disabled?: boolean;
    onValueChange?: (value: string) => void;
    value?: string;
  }) => (
    <select
      disabled={disabled}
      value={value}
      onChange={(event) => onValueChange?.(event.target.value)}
    >
      {children}
    </select>
  ),
  SelectContent: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  SelectItem: ({
    children,
    value,
  }: {
    children: React.ReactNode;
    value: string;
  }) => <option value={value}>{children}</option>,
  SelectTrigger: () => null,
  SelectValue: () => null,
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
      dataClassificationConfigId: "classification-config",
    })
  );
  mocks.getInstanceResource.mockReset();
  mocks.getInstanceResource.mockImplementation(
    (database: Database) => database.instanceResource
  );
  mocks.hasProjectPermissionV2.mockReset();
  mocks.hasProjectPermissionV2.mockReturnValue(true);
  mocks.getDatabaseEngine.mockReset();
  mocks.getDatabaseEngine.mockReturnValue(Engine.POSTGRES);
  mocks.isDev.mockReset();
  mocks.isDev.mockReturnValue(false);
  mocks.instanceV1MaskingForNoSQL.mockReset();
  mocks.instanceV1MaskingForNoSQL.mockReturnValue(false);
  mocks.hasSchemaProperty.mockReset();
  mocks.hasSchemaProperty.mockReturnValue(true);
  mocks.instanceV1HasCollationAndCharacterSet.mockReset();
  mocks.instanceV1HasCollationAndCharacterSet.mockReturnValue(true);
  mocks.instanceV1SupportsColumn.mockReset();
  mocks.instanceV1SupportsColumn.mockReturnValue(true);
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
  mocks.useDatabaseCatalog.mockReset();
  mocks.useDatabaseCatalog.mockReturnValue({
    value: {
      schemas: [],
    },
  });
  mocks.getColumnCatalog.mockReset();
  mocks.getColumnCatalog.mockReturnValue({
    semanticType: "",
    classification: "",
  });
  mocks.getTableCatalog.mockReset();
  mocks.getTableCatalog.mockReturnValue({
    classification: "",
  });
  mocks.useSettingV1Store.mockReset();
  mocks.useSettingV1Store.mockReturnValue({
    getOrFetchSettingByName: vi.fn(),
    getSettingByName: vi.fn(() => undefined),
    getProjectClassification: vi.fn(() => undefined),
  });
  mocks.useSubscriptionV1Store.mockReset();
  mocks.useSubscriptionV1Store.mockReturnValue({
    hasFeature: vi.fn(() => true),
    instanceMissingLicense: vi.fn(() => false),
  });
  mocks.featureToRef.mockReset();
  mocks.featureToRef.mockReturnValue({
    value: true,
  });
  mocks.hasIndexSizeProperty.mockReset();
  mocks.hasIndexSizeProperty.mockReturnValue(true);
  mocks.hasTableEngineProperty.mockReset();
  mocks.hasTableEngineProperty.mockReturnValue(false);
  mocks.instanceV1SupportsIndex.mockReset();
  mocks.instanceV1SupportsIndex.mockReturnValue(true);
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

  test("renders the legacy tables header instead of the generic object header", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: true,
      })
    );

    render();
    await flush();

    const tableHeaderText = Array.from(container.querySelectorAll("th")).map(
      (cell) => cell.textContent
    );

    expect(tableHeaderText).toEqual(
      expect.arrayContaining([
        "common.schema",
        "common.name",
        "database.classification.self",
        "database.partitioned",
        "database.row-count-est",
        "database.data-size",
        "database.index-size",
        "common.comment",
      ])
    );
    expect(tableHeaderText).not.toContain("common.definition");

    unmount();
  });

  test("restores the table engine column for MySQL overview rows", async () => {
    mocks.getDatabaseEngine.mockReturnValue(Engine.MYSQL);
    mocks.useDBSchemaV1Store.mockReturnValue({
      getSchemaList: vi.fn(() => mocks.schemaList),
      getTableList: vi.fn(
        () =>
          [
            {
              name: "orders",
              engine: "InnoDB",
              rowCount: 7,
              dataSize: 64n,
              indexSize: 32n,
              comment: "orders comment",
              columns: [
                {
                  name: "id",
                  type: "INT",
                  comment: "",
                },
              ],
            },
          ] as never
      ),
      getViewList: vi.fn(() => []),
      getExtensionList: vi.fn(() => []),
      getExternalTableList: vi.fn(() => []),
      getFunctionList: vi.fn(() => []),
      getDatabaseMetadata: vi.fn(() => ({
        characterSet: "utf8mb4",
        collation: "utf8mb4_0900_ai_ci",
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

    const tableHeaderText = Array.from(container.querySelectorAll("th")).map(
      (cell) => cell.textContent
    );
    expect(tableHeaderText).toContain("database.engine");
    expect(container.textContent).toContain("InnoDB");

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

  test("treats the empty-string schema as a valid resolved selection", async () => {
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
    await flush();

    expect(container.textContent).not.toContain("common.loading");
    expect(container.querySelector("select")?.getAttribute("disabled")).toBe(
      null
    );

    unmount();
  });

  test("clears a stale table query when the table does not exist", async () => {
    const { router } = await import("@/router");
    router.currentRoute.value.query = { schema: "public", table: "missing" };

    const { render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: true,
      })
    );

    render();
    await flush();

    expect(mocks.routerReplace).toHaveBeenCalledWith(
      expect.objectContaining({
        query: expect.objectContaining({
          schema: "public",
          table: undefined,
        }),
      })
    );

    unmount();
    router.currentRoute.value.query = {};
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
