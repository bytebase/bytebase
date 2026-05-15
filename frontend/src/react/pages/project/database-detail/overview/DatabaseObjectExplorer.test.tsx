import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

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
    currentRoute,
    routerReplace: vi.fn(() => Promise.resolve()),
    useVueState: vi.fn(),
    useDatabaseCatalog: vi.fn(),
    getColumnCatalog: vi.fn(() => ({
      semanticType: "",
      classification: "",
    })),
    getTableCatalog: vi.fn(),
    featureToRef: vi.fn(() => ({ value: true })),
    useDBSchemaV1Store: vi.fn(),
    useSettingV1Store: vi.fn(),
    getDatabaseProject: vi.fn((database: { project: string }) => ({
      name: database.project,
      dataClassificationConfigId: "classification-config",
    })),
    getDatabaseEngine: vi.fn(() => Engine.POSTGRES),
    hasIndexSizeProperty: vi.fn(() => true),
    hasSchemaProperty: vi.fn(() => true),
    hasTableEngineProperty: vi.fn(() => false),
    instanceV1HasCollationAndCharacterSet: vi.fn(() => true),
    instanceV1SupportsColumn: vi.fn(() => true),
    instanceV1SupportsIndex: vi.fn(() => true),
    instanceV1SupportsPackage: vi.fn(() => false),
    instanceV1SupportsSequence: vi.fn(() => false),
    instanceV1SupportsTrigger: vi.fn(() => false),
    bytesToString: vi.fn((size: number) => `${size} B`),
    hasProjectPermissionV2: vi.fn(() => true),
    dialogProps: [] as unknown[],
    useTranslation: vi.fn(() => ({
      t: (key: string) => key,
    })),
  };
});

let DatabaseObjectExplorer: typeof import("./DatabaseObjectExplorer").DatabaseObjectExplorer;

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
  getColumnCatalog: mocks.getColumnCatalog,
  getTableCatalog: mocks.getTableCatalog,
  featureToRef: mocks.featureToRef,
  useSettingV1Store: mocks.useSettingV1Store,
}));

vi.mock("@/utils", () => ({
  bytesToString: mocks.bytesToString,
  getDatabaseEngine: mocks.getDatabaseEngine,
  getDatabaseProject: mocks.getDatabaseProject,
  hasIndexSizeProperty: mocks.hasIndexSizeProperty,
  hasProjectPermissionV2: mocks.hasProjectPermissionV2,
  hasSchemaProperty: mocks.hasSchemaProperty,
  hasTableEngineProperty: mocks.hasTableEngineProperty,
  instanceV1HasCollationAndCharacterSet:
    mocks.instanceV1HasCollationAndCharacterSet,
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

vi.mock("./TableDetailDialog", () => ({
  EditableClassificationCell: () => null,
  TableDetailDialog: (props: unknown) => {
    mocks.dialogProps.push(props);
    return null;
  },
}));

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
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

const clickElement = (element: Element) => {
  act(() => {
    element.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
};

const makeDatabase = (): Database =>
  ({
    name: "instances/inst1/databases/db",
    project: "projects/proj1",
    instanceResource: {
      name: "instances/inst1",
      engine: Engine.POSTGRES,
    },
  }) as Database;

const makeTable = (name: string) =>
  ({
    name,
    rowCount: 7,
    dataSize: 64n,
    indexSize: 32n,
    comment: `${name} comment`,
    partitions: [
      {
        name: `${name}_2026`,
        type: 1,
        expression: "FOR VALUES FROM ('2026-01-01') TO ('2027-01-01')",
        subpartitions: [],
      },
    ],
    triggers: [
      {
        name: `${name}_before_insert`,
        event: "INSERT",
        timing: "BEFORE",
        body: "EXECUTE FUNCTION demo()",
        sqlMode: "",
      },
    ],
    columns: [
      {
        name: "id",
        type: "INT",
        comment: "",
      },
    ],
  }) as never;

beforeEach(async () => {
  mocks.currentRoute.value.query = {};
  mocks.routerReplace.mockReset();
  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });
  mocks.getDatabaseEngine.mockReset();
  mocks.getDatabaseEngine.mockReturnValue(Engine.POSTGRES);
  mocks.hasSchemaProperty.mockReset();
  mocks.hasSchemaProperty.mockReturnValue(true);
  mocks.instanceV1SupportsPackage.mockReset();
  mocks.instanceV1SupportsPackage.mockReturnValue(false);
  mocks.instanceV1SupportsSequence.mockReset();
  mocks.instanceV1SupportsSequence.mockReturnValue(false);
  mocks.instanceV1SupportsTrigger.mockReset();
  mocks.instanceV1SupportsTrigger.mockReturnValue(false);
  mocks.bytesToString.mockReset();
  mocks.bytesToString.mockImplementation((size: number) => `${size} B`);
  mocks.hasProjectPermissionV2.mockReset();
  mocks.hasProjectPermissionV2.mockReturnValue(true);
  mocks.useDBSchemaV1Store.mockReset();
  mocks.useDBSchemaV1Store.mockReturnValue({
    getSchemaList: vi.fn(() => [{ name: "public" }]),
    getTableList: vi.fn(() => [makeTable("orders")]),
    getViewList: vi.fn(() => []),
    getExtensionList: vi.fn(() => []),
    getExternalTableList: vi.fn(() => []),
    getFunctionList: vi.fn(() => []),
    getDatabaseMetadata: vi.fn(() => ({
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
  mocks.featureToRef.mockReset();
  mocks.featureToRef.mockReturnValue({ value: true });
  mocks.useSettingV1Store.mockReset();
  mocks.useSettingV1Store.mockReturnValue({
    getOrFetchSettingByName: vi.fn(),
    getProjectClassification: vi.fn(() => undefined),
  });
  mocks.getDatabaseProject.mockReset();
  mocks.getDatabaseProject.mockImplementation(
    (database: { project: string }) => ({
      name: database.project,
      dataClassificationConfigId: "classification-config",
    })
  );
  mocks.hasIndexSizeProperty.mockReset();
  mocks.hasIndexSizeProperty.mockReturnValue(true);
  mocks.useVueState.mockReset();
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());
  mocks.hasTableEngineProperty.mockReset();
  mocks.hasTableEngineProperty.mockReturnValue(false);
  mocks.instanceV1HasCollationAndCharacterSet.mockReset();
  mocks.instanceV1HasCollationAndCharacterSet.mockReturnValue(true);
  mocks.instanceV1SupportsColumn.mockReset();
  mocks.instanceV1SupportsColumn.mockReturnValue(true);
  mocks.instanceV1SupportsIndex.mockReset();
  mocks.instanceV1SupportsIndex.mockReturnValue(true);
  mocks.dialogProps.length = 0;

  vi.resetModules();
  ({ DatabaseObjectExplorer } = await import("./DatabaseObjectExplorer"));
});

describe("DatabaseObjectExplorer", () => {
  test("renders the default schema label when the schema name is empty", async () => {
    mocks.useDBSchemaV1Store.mockReturnValue({
      getSchemaList: vi.fn(() => [{ name: "" }]),
      getTableList: vi.fn(() => []),
      getViewList: vi.fn(() => []),
      getExtensionList: vi.fn(() => []),
      getExternalTableList: vi.fn(() => []),
      getFunctionList: vi.fn(() => []),
      getDatabaseMetadata: vi.fn(() => ({
        schemas: [],
      })),
    });

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseObjectExplorer, {
        database: makeDatabase(),
        loading: false,
        selectedSchemaName: "",
        tableSearchKeyword: "",
        externalTableSearchKeyword: "",
        onSelectedSchemaNameChange: vi.fn(),
        onTableSearchKeywordChange: vi.fn(),
        onExternalTableSearchKeywordChange: vi.fn(),
      })
    );

    render();
    await flush();

    expect(container.querySelector("#schema-select")?.textContent).toContain(
      "db.schema.default"
    );

    unmount();
  });

  test("passes serializable detail props after selecting a table row", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseObjectExplorer, {
        database: makeDatabase(),
        loading: false,
        selectedSchemaName: "public",
        tableSearchKeyword: "",
        externalTableSearchKeyword: "",
        onSelectedSchemaNameChange: vi.fn(),
        onTableSearchKeywordChange: vi.fn(),
        onExternalTableSearchKeywordChange: vi.fn(),
      })
    );

    render();
    await flush();

    const ordersRow = Array.from(container.querySelectorAll("td"))
      .find((cell) => cell.textContent === "orders")
      ?.closest("tr");
    expect(ordersRow).toBeTruthy();

    clickElement(ordersRow as HTMLTableRowElement);
    await flush();

    const latestDialogProps = mocks.dialogProps.at(-1);
    expect(() => JSON.stringify(latestDialogProps)).not.toThrow();
    expect(latestDialogProps).toEqual(
      expect.objectContaining({
        table: expect.objectContaining({
          partitions: [
            expect.objectContaining({
              name: "orders_2026",
            }),
          ],
          showPartitionTables: true,
          showTriggers: false,
          triggers: [
            expect.objectContaining({
              name: "orders_before_insert",
            }),
          ],
        }),
      })
    );

    unmount();
  });
});
