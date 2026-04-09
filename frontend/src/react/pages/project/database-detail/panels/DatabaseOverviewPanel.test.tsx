import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
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
  routerReplace: vi.fn(),
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
  instanceV1SupportsPackage: vi.fn(() => false),
  instanceV1SupportsSequence: vi.fn(() => false),
  bytesToString: vi.fn((size: number) => `${size} B`),
}));

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
    currentRoute: {
      value: {
        query: {},
      },
    },
  },
}));

vi.mock("@/store", () => ({
  useDBSchemaV1Store: mocks.useDBSchemaV1Store,
}));

vi.mock("@/utils", () => ({
  bytesToString: mocks.bytesToString,
  getDatabaseEngine: mocks.getDatabaseEngine,
  getDatabaseProject: mocks.getDatabaseProject,
  hasProjectPermissionV2: mocks.hasProjectPermissionV2,
  hasSchemaProperty: mocks.hasSchemaProperty,
  instanceV1SupportsPackage: mocks.instanceV1SupportsPackage,
  instanceV1SupportsSequence: mocks.instanceV1SupportsSequence,
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => (
    <input {...props} />
  ),
}));

vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open: boolean; children: React.ReactNode }) =>
    open ? <div data-testid="dialog-root">{children}</div> : null,
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

const makeDatabase = (): Database =>
  ({
    name: "instances/inst1/databases/db",
    project: "projects/proj1",
    effectiveEnvironment: "environments/prod",
    instanceResource: {
      name: "instances/inst1",
      title: "Primary",
      engine: Engine.POSTGRES,
    },
  }) as Database;

beforeEach(async () => {
  mocks.localStorage.clear.mockReset();
  mocks.localStorage.getItem.mockReset();
  mocks.localStorage.getItem.mockReturnValue(null);
  mocks.localStorage.removeItem.mockReset();
  mocks.localStorage.setItem.mockReset();
  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });
  mocks.routerReplace.mockReset();
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
  mocks.instanceV1SupportsPackage.mockReset();
  mocks.instanceV1SupportsPackage.mockReturnValue(false);
  mocks.instanceV1SupportsSequence.mockReset();
  mocks.instanceV1SupportsSequence.mockReturnValue(false);
  mocks.bytesToString.mockReset();
  mocks.bytesToString.mockImplementation((size: number) => `${size} B`);
  mocks.useDBSchemaV1Store.mockReset();
  mocks.useDBSchemaV1Store.mockReturnValue({
    getSchemaList: vi.fn(() => mocks.schemaList),
    getTableList: vi.fn(() => []),
    getViewList: vi.fn(() => []),
    getExtensionList: vi.fn(() => []),
    getExternalTableList: vi.fn(() => []),
    getFunctionList: vi.fn(() => []),
    getDatabaseMetadata: vi.fn(() => ({ schemas: [] })),
  });
  mocks.useVueState.mockReset();
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());

  vi.resetModules();
  ({ DatabaseOverviewPanel } = await import("./DatabaseOverviewPanel"));
});

describe("DatabaseOverviewPanel", () => {
  test("renders overview info and schema selector without legacy bridge mocks", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseOverviewPanel, {
        database: makeDatabase(),
        hasSchemaPermission: true,
      })
    );

    render();
    await flush();

    expect(container.textContent).toContain("db");
    expect(container.querySelector("select")).not.toBeNull();
    expect(
      container.querySelector('input[placeholder="common.filter-by-name"]')
    ).not.toBeNull();

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
});
