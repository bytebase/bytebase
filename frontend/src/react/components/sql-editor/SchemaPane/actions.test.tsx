import { act, type ReactElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { NodeTarget, NodeType, TreeNode } from "./schemaTree";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => {
  return {
    getDatabaseByName: vi.fn(() => ({
      name: "instances/i/databases/db",
      project: "projects/p",
      effectiveEnvironment: "environments/prod",
      labels: {},
      // shape is enough for getInstanceResource → engine
      instanceResource: {
        engine: 1, // MYSQL
        engineVersion: "8.0",
      },
    })),
    getTableMetadata: vi.fn(() => ({
      columns: [{ name: "id" }, { name: "email" }],
    })),
    getInstanceResource: vi.fn(() => ({ engine: 1 })),
    instanceV1HasAlterSchema: vi.fn(() => true),
    supportGetStringSchema: vi.fn(() => true),
    sortByDictionary: vi.fn(
      <T,>(arr: T[], order: string[], keyFn: (i: T) => string) => {
        arr.sort((a, b) => {
          const ai = order.indexOf(keyFn(a));
          const bi = order.indexOf(keyFn(b));
          if (ai === -1 && bi === -1) return 0;
          if (ai === -1) return 1;
          if (bi === -1) return -1;
          return ai - bi;
        });
      }
    ),
  };
});

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (k: string) => k }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

vi.mock("@/react/i18n", () => ({
  default: { t: (k: string) => k },
}));

vi.mock("@/store", () => ({
  pushNotification: vi.fn(),
  useDatabaseV1Store: () => ({ getDatabaseByName: mocks.getDatabaseByName }),
  useDBSchemaV1Store: () => ({ getTableMetadata: mocks.getTableMetadata }),
  useSQLEditorTabStore: () => ({
    currentTab: { id: "t1", viewState: { schema: "" } },
    openTabList: [],
    addTab: vi.fn(),
    setCurrentTabId: vi.fn(),
    updateCurrentTab: vi.fn(),
    updateTab: vi.fn(),
  }),
}));

vi.mock("@/utils", () => ({
  // Stub everything actions.tsx imports. We don't `importOriginal` here
  // because the real `@/utils` transitively pulls in monaco / vue
  // surfaces that vitest can't load.
  defaultSQLEditorTab: () => ({
    id: "new",
    title: "Untitled",
    connection: { instance: "", database: "" },
    viewState: { view: "CODE" },
    treeState: {},
    editorState: { selection: null },
    mode: "WORKSHEET",
    status: "NEW",
  }),
  extractDatabaseResourceName: (n: string) => ({
    instance: "instances/i",
    databaseName: n.split("/").pop() ?? "",
  }),
  extractInstanceResourceName: (n: string) => n.split("/").pop() ?? "",
  extractProjectResourceName: (n: string) => n.split("/").pop() ?? "",
  generateSimpleDeleteStatement: () => "DELETE",
  generateSimpleInsertStatement: () => "INSERT",
  generateSimpleSelectAllStatement: () => "SELECT *",
  generateSimpleUpdateStatement: () => "UPDATE",
  getInstanceResource: mocks.getInstanceResource,
  instanceV1HasAlterSchema: mocks.instanceV1HasAlterSchema,
  isSameSQLEditorConnection: () => false,
  sortByDictionary: mocks.sortByDictionary,
  supportGetStringSchema: mocks.supportGetStringSchema,
  isDev: () => false,
}));

vi.mock("@/types", () => ({
  DEFAULT_SQL_EDITOR_TAB_MODE: "WORKSHEET",
  dialectOfEngineV1: () => "MYSQL",
  languageOfEngineV1: () => "sql",
  typeToView: (type: string) => type.toUpperCase(),
}));

vi.mock("@/types/proto-es/v1/common_pb", () => ({
  Engine: { REDIS: 7, MYSQL: 1 },
}));

vi.mock("@/types/proto-es/v1/database_service_pb", () => ({
  GetSchemaStringRequest_ObjectType: { TABLE: 1, VIEW: 2 },
}));

vi.mock("@/router", () => ({
  router: {
    resolve: () => ({ href: "/sql-editor/db" }),
  },
}));

vi.mock("@/router/sqlEditor", () => ({
  SQL_EDITOR_DATABASE_MODULE: "sql-editor.database",
}));

vi.mock("@/views/sql-editor/EditorCommon/utils", () => ({
  keyWithPosition: (k: string, p: number) => `${k}###${p}`,
}));

vi.mock("@/views/sql-editor/events", () => ({
  sqlEditorEvents: { emit: vi.fn() },
}));

vi.mock("@/composables/useExecuteSQL", () => ({
  useExecuteSQL: () => ({ execute: vi.fn() }),
}));

vi.mock("@/react/components/monaco/sqlFormatter", () => ({
  formatSQL: async (sql: string) => ({ data: sql, error: null }),
}));

vi.mock("@/types/sqlEditor/tabViewState", () => ({
  defaultViewState: () => ({ view: "CODE" }),
}));

let useSchemaPaneContextMenu: typeof import("./actions").useSchemaPaneContextMenu;
type SchemaMenuItem = import("./actions").SchemaMenuItem;

beforeEach(async () => {
  ({ useSchemaPaneContextMenu } = await import("./actions"));
});

const DATABASE = "instances/i/databases/db";

const makeNode = (type: NodeType, target: Partial<NodeTarget>): TreeNode =>
  ({
    key: `${type}/k`,
    meta: {
      type,
      target: { database: DATABASE, ...target },
    },
  }) as unknown as TreeNode;

function captureItems(
  node: TreeNode | null,
  deps: { availableActions: never[]; setSchemaViewer: () => void }
) {
  const captured: { items?: SchemaMenuItem[] } = {};
  function Probe() {
    captured.items = useSchemaPaneContextMenu(node, deps);
    return null;
  }
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => {
    root.render((<Probe />) as ReactElement);
  });
  act(() => {
    root.unmount();
  });
  container.remove();
  return captured.items ?? [];
}

const noopDeps = {
  availableActions: [] as never[],
  setSchemaViewer: () => {},
};

describe("useSchemaPaneContextMenu", () => {
  test("returns empty for a null node", () => {
    expect(captureItems(null, noopDeps)).toEqual([]);
  });

  test("returns empty for a disabled node", () => {
    const node = {
      ...makeNode("error", { id: "x", expandable: false, text: () => "" }),
      disabled: true,
    } as unknown as TreeNode;
    expect(captureItems(node, noopDeps)).toEqual([]);
  });

  test("table node populates copy-name / copy-all-columns / view-schema-text / edit-schema / copy-url / preview / generate-sql / view-detail", () => {
    const node = makeNode("table", {
      database: DATABASE,
      schema: "public",
      table: "users",
    });
    const items = captureItems(node, noopDeps);
    const keys = items.map((i) => i.key);
    expect(keys).toEqual([
      "copy-name",
      "copy-all-column-names",
      "preview-table-data",
      "generate-sql",
      "view-schema-text",
      "view-detail",
      "edit-schema",
      "copy-url",
    ]);
    const generate = items.find((i) => i.key === "generate-sql");
    expect(generate?.children?.map((c) => c.key)).toEqual([
      "generate-sql--select",
      "generate-sql--insert",
      "generate-sql--update",
      "generate-sql--delete",
    ]);
  });

  test("view node yields copy-name + view-schema-text + preview-table + generate-sql(SELECT only) + view-detail", () => {
    const node = makeNode("view", {
      database: DATABASE,
      schema: "public",
      view: "v_active",
    });
    const items = captureItems(node, noopDeps);
    const keys = items.map((i) => i.key);
    expect(keys).toEqual([
      "copy-name",
      "preview-table-data",
      "generate-sql",
      "view-schema-text",
      "view-detail",
    ]);
    const generate = items.find((i) => i.key === "generate-sql");
    expect(generate?.children?.map((c) => c.key)).toEqual([
      "generate-sql--select",
    ]);
  });

  test("procedure / function / trigger nodes only yield view-detail", () => {
    for (const type of ["procedure", "function", "trigger"] as const) {
      const node = makeNode(type, {
        database: DATABASE,
        schema: "public",
        ...(type === "procedure"
          ? { procedure: "p", position: 0 }
          : type === "function"
            ? { function: "f", position: 0 }
            : { trigger: "trg", position: 0, table: "t" }),
      });
      const items = captureItems(node, noopDeps);
      expect(items.map((i) => i.key)).toEqual(["view-detail"]);
    }
  });

  test("schema node renders one item per availableAction in given order", () => {
    const node = makeNode("schema", {
      database: DATABASE,
      schema: "public",
    });
    const deps = {
      availableActions: [
        { view: "TABLES", title: "Tables", icon: null },
        { view: "VIEWS", title: "Views", icon: null },
      ],
      setSchemaViewer: () => {},
    };
    const items = captureItems(node, deps as unknown as typeof noopDeps);
    expect(items.map((i) => i.label)).toEqual(["Tables", "Views"]);
  });

  test("REDIS engine drops generate-sql + preview", () => {
    // The hook calls getInstanceResource several times across schema /
    // generate-sql / view-schema-text checks, so we need a persistent
    // mock for this test rather than `mockReturnValueOnce`.
    mocks.getInstanceResource.mockReturnValue({ engine: 7 }); // REDIS
    mocks.supportGetStringSchema.mockReturnValue(false);
    mocks.instanceV1HasAlterSchema.mockReturnValue(false);
    try {
      const node = makeNode("table", {
        database: DATABASE,
        schema: "",
        table: "users",
      });
      const items = captureItems(node, noopDeps);
      const keys = items.map((i) => i.key);
      expect(keys).not.toContain("preview-table-data");
      expect(keys).not.toContain("generate-sql");
    } finally {
      mocks.getInstanceResource.mockReturnValue({ engine: 1 });
      mocks.supportGetStringSchema.mockReturnValue(true);
      mocks.instanceV1HasAlterSchema.mockReturnValue(true);
    }
  });
});
