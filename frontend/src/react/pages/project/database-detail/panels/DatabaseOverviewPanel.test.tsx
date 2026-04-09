import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
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
  schemaList: [{ name: "public" }, { name: "sales" }],
  routerReplace: vi.fn(),
  overviewInfoBridge: vi.fn(() => <div data-testid="overview-info-bridge" />),
  objectExplorerBridge: vi.fn(
    ({
      selectedSchemaName,
      tableSearchKeyword,
      onSelectedSchemaNameChange,
      onTableSearchKeywordChange,
    }: {
      selectedSchemaName: string;
      tableSearchKeyword: string;
      onSelectedSchemaNameChange: (value: string) => void;
      onTableSearchKeywordChange: (value: string) => void;
    }) => (
      <div data-testid="object-explorer-bridge">
        <select
          value={selectedSchemaName}
          onChange={(event) => onSelectedSchemaNameChange(event.target.value)}
        >
          {mocks.schemaList.map((schema) => (
            <option key={schema.name} value={schema.name}>
              {schema.name}
            </option>
          ))}
        </select>
        <input
          placeholder="common.filter-by-name"
          value={tableSearchKeyword}
          onChange={(event) => onTableSearchKeywordChange(event.target.value)}
        />
      </div>
    )
  ),
  useVueState: vi.fn(),
  useDBSchemaV1Store: vi.fn(),
}));

vi.stubGlobal("localStorage", mocks.localStorage);

let DatabaseOverviewPanel: typeof import("./DatabaseOverviewPanel").DatabaseOverviewPanel;

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

vi.mock("@/plugins/i18n", () => ({
  default: {
    install: vi.fn(),
  },
}));

vi.mock("@/plugins/naive-ui", () => ({
  default: {
    install: vi.fn(),
  },
}));

vi.mock("@/store", () => ({
  useDBSchemaV1Store: mocks.useDBSchemaV1Store,
}));

vi.mock("../legacy/DatabaseOverviewInfoBridge", () => ({
  DatabaseOverviewInfoBridge: mocks.overviewInfoBridge,
}));

vi.mock("../legacy/DatabaseObjectExplorerBridge", () => ({
  DatabaseObjectExplorerBridge: mocks.objectExplorerBridge,
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

const setInputValue = (input: HTMLInputElement, value: string) => {
  act(() => {
    const descriptor = Object.getOwnPropertyDescriptor(
      HTMLInputElement.prototype,
      "value"
    );
    descriptor?.set?.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
    input.dispatchEvent(new Event("change", { bubbles: true }));
  });
};

const setSelectValue = (select: HTMLSelectElement, value: string) => {
  act(() => {
    const descriptor = Object.getOwnPropertyDescriptor(
      HTMLSelectElement.prototype,
      "value"
    );
    descriptor?.set?.call(select, value);
    select.dispatchEvent(new Event("change", { bubbles: true }));
  });
};

const makeDatabase = (): Database =>
  ({
    name: "instances/inst1/databases/db1",
    project: "projects/proj1",
  }) as Database;

beforeEach(async () => {
  mocks.localStorage.clear.mockReset();
  mocks.localStorage.getItem.mockReset();
  mocks.localStorage.getItem.mockReturnValue(null);
  mocks.localStorage.removeItem.mockReset();
  mocks.localStorage.setItem.mockReset();
  mocks.routerReplace.mockReset();
  mocks.overviewInfoBridge.mockClear();
  mocks.objectExplorerBridge.mockClear();
  mocks.useDBSchemaV1Store.mockReset();
  mocks.useDBSchemaV1Store.mockReturnValue({
    getSchemaList: vi.fn(() => mocks.schemaList),
  });
  mocks.useVueState.mockReset();
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());

  vi.resetModules();
  ({ DatabaseOverviewPanel } = await import("./DatabaseOverviewPanel"));
});

describe("DatabaseOverviewPanel", () => {
  test("keeps schema selection and search state in React", async () => {
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
    setSelectValue(select as HTMLSelectElement, "sales");

    const input = container.querySelector("input") as HTMLInputElement | null;
    expect(input).not.toBeNull();
    setInputValue(input as HTMLInputElement, "book");

    await flush();

    expect((container.querySelector("select") as HTMLSelectElement).value).toBe(
      "sales"
    );
    expect((container.querySelector("input") as HTMLInputElement).value).toBe(
      "book"
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

    expect(
      container.querySelector('[data-testid="overview-info-bridge"]')
    ).not.toBeNull();
    expect(
      container.querySelector('[data-testid="object-explorer-bridge"]')
    ).toBeNull();

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
