import { act, type ReactElement, useState } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { DatabaseResource } from "@/types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  fetchDatabases: vi.fn(),
  fetchInstanceList: vi.fn(),
  getOrFetchDatabaseMetadata: vi.fn(),
  instanceStore: undefined as
    | { fetchInstanceList: ReturnType<typeof vi.fn> }
    | undefined,
  environmentStore: {
    environmentList: [],
  },
  extractDatabaseResourceName: vi.fn((name: string) => {
    const parts = name.split("/");
    return {
      instance: parts[1] ?? "",
      instanceName: parts[1] ?? "",
      database: name,
      databaseName: parts[3] ?? name,
    };
  }),
  extractInstanceResourceName: vi.fn((name: string) => {
    const parts = name.split("/");
    return parts[1] ?? name;
  }),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: { count?: number }) =>
      options?.count === undefined ? key : key,
  }),
}));

vi.mock("@/react/i18n", () => ({
  default: {
    t: (key: string) => key,
  },
}));

vi.mock("@/react/components/AdvancedSearch", () => ({
  getValueFromScopes: (
    params: { scopes: { id: string; value: string }[] },
    id: string
  ) => params.scopes.find((scope) => scope.id === id)?.value ?? "",
  AdvancedSearch: ({
    params,
    onParamsChange,
  }: {
    params: { query: string; scopes: { id: string; value: string }[] };
    onParamsChange: (params: {
      query: string;
      scopes: { id: string; value: string }[];
    }) => void;
  }) => (
    <button
      type="button"
      onClick={() =>
        onParamsChange({
          query: "salary",
          scopes: [{ id: "table", value: "employee" }],
        })
      }
    >
      <span>advanced-search:{params.query}</span>
    </button>
  ),
}));

vi.mock("@/react/stores/app", () => {
  const appState = () => ({
    getOrFetchDatabaseMetadata: mocks.getOrFetchDatabaseMetadata,
    fetchInstanceList: mocks.fetchInstanceList,
    fetchDatabases: mocks.fetchDatabases,
    environmentList: mocks.environmentStore.environmentList,
  });
  return {
    useAppStore: Object.assign(
      (selector: (state: unknown) => unknown) => selector(appState()),
      { getState: appState }
    ),
  };
});

vi.mock("@/react/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({ environmentName }: { environmentName: string }) => (
    <span>{environmentName}</span>
  ),
}));

vi.mock("@/utils", () => ({
  engineNameV1: (engine: number) => `engine-${engine}`,
  extractDatabaseResourceName: mocks.extractDatabaseResourceName,
  extractInstanceResourceName: mocks.extractInstanceResourceName,
  getDefaultPagination: () => 100,
  supportedEngineV1List: () => [],
}));

let DatabaseResourceSelector: typeof import("./DatabaseResourceSelector").DatabaseResourceSelector;

const databaseName = "instances/prod/databases/hr";
const metadata = {
  schemas: [
    {
      name: "public",
      tables: [
        {
          name: "employee",
          columns: [{ name: "id" }, { name: "salary" }],
        },
      ],
    },
  ],
};
const schemaLessMetadata = {
  schemas: [
    {
      name: "",
      tables: [
        {
          name: "employee",
          columns: [{ name: "salary" }],
        },
      ],
    },
  ],
};

function Harness({
  includeColumns,
  initialValue = [],
  onValueChange,
}: {
  includeColumns?: boolean;
  initialValue?: DatabaseResource[];
  onValueChange?: (value: DatabaseResource[]) => void;
}) {
  const [value, setValue] = useState<DatabaseResource[]>(initialValue);

  return (
    <DatabaseResourceSelector
      projectName="projects/project"
      value={value}
      includeColumns={includeColumns}
      onChange={(next) => {
        setValue(next);
        onValueChange?.(next);
      }}
    />
  );
}

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  act(() => {
    root.render(element);
  });
  return {
    container,
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

const flushPromises = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
};

const rowForText = (container: HTMLElement, text: string): HTMLElement => {
  const matches = Array.from(container.querySelectorAll("span")).filter(
    (node) => node.textContent === text
  );
  const match = matches[0];
  if (!match) {
    throw new Error(`row text not found: ${text}`);
  }
  const row = match.parentElement;
  if (!row) {
    throw new Error(`row not found: ${text}`);
  }
  return row;
};

const clickFirstButtonInRow = (container: HTMLElement, text: string) => {
  const row = rowForText(container, text);
  const button =
    row instanceof HTMLButtonElement ? row : row.querySelector("button");
  if (!button) {
    throw new Error(`button not found: ${text}`);
  }
  act(() => {
    button.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
};

const clickCheckboxInRow = (container: HTMLElement, text: string) => {
  const row = rowForText(container, text);
  const checkbox = row.querySelector('[role="checkbox"]');
  if (!checkbox) {
    throw new Error(`checkbox not found: ${text}`);
  }
  act(() => {
    checkbox.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
};

const expandDatabase = async (container: HTMLElement) => {
  await waitForText(container, "hr");
  clickFirstButtonInRow(container, "hr");
  await flushPromises();
};

const waitForText = async (container: HTMLElement, text: string) => {
  for (let i = 0; i < 10; i++) {
    if (container.textContent?.includes(text)) {
      return;
    }
    await flushPromises();
  }
  throw new Error(`text not found: ${text}`);
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.fetchDatabases.mockResolvedValue({
    databases: [
      {
        name: databaseName,
        environment: "environments/prod",
        effectiveEnvironment: "environments/prod",
        instanceResource: { title: "Prod" },
      },
    ],
    nextPageToken: "",
  });
  mocks.fetchInstanceList.mockResolvedValue({
    instances: [],
    nextPageToken: "",
  });
  mocks.getOrFetchDatabaseMetadata.mockResolvedValue(metadata);
  ({ DatabaseResourceSelector } = await import("./DatabaseResourceSelector"));
});

describe("DatabaseResourceSelector", () => {
  test("passes AdvancedSearch query and table scope to database fetch", async () => {
    const { container, unmount } = renderIntoContainer(<Harness />);
    await flushPromises();
    await waitForText(container, "advanced-search:");

    mocks.fetchDatabases.mockClear();
    clickFirstButtonInRow(container, "advanced-search:");
    await flushPromises();

    expect(mocks.fetchDatabases).toHaveBeenLastCalledWith(
      expect.objectContaining({
        filter: expect.objectContaining({
          query: "salary",
          table: "employee",
        }),
      })
    );

    unmount();
  });

  test("does not expose column rows unless explicitly enabled", async () => {
    const { container, unmount } = renderIntoContainer(<Harness />);
    await flushPromises();
    await expandDatabase(container);
    clickFirstButtonInRow(container, "public");

    expect(container.textContent).toContain("employee");
    expect(container.textContent).not.toContain("salary");

    unmount();
  });

  test("selects multiple columns as one table-scoped DatabaseResource", async () => {
    const onValueChange = vi.fn();
    const { container, unmount } = renderIntoContainer(
      <Harness includeColumns onValueChange={onValueChange} />
    );
    await flushPromises();
    await expandDatabase(container);
    clickFirstButtonInRow(container, "public");
    clickFirstButtonInRow(container, "employee");

    clickCheckboxInRow(container, "id");
    clickCheckboxInRow(container, "salary");

    expect(onValueChange).toHaveBeenLastCalledWith([
      {
        databaseFullName: databaseName,
        schema: "public",
        table: "employee",
        columns: ["id", "salary"],
      },
    ]);
    expect(container.textContent).toContain("hr.public.employee: id, salary");

    unmount();
  });

  test("omits empty schema segment from column resource labels", async () => {
    mocks.getOrFetchDatabaseMetadata.mockResolvedValue(schemaLessMetadata);
    const { container, unmount } = renderIntoContainer(
      <Harness includeColumns />
    );
    await flushPromises();
    await expandDatabase(container);
    clickFirstButtonInRow(container, "employee");

    clickCheckboxInRow(container, "salary");

    expect(container.textContent).toContain("hr.employee: salary");
    expect(container.textContent).not.toContain("hr..employee: salary");

    unmount();
  });

  test("loads one page at a time instead of draining every page", async () => {
    mocks.fetchDatabases.mockReset();
    mocks.fetchDatabases
      .mockResolvedValueOnce({
        databases: [
          {
            name: "instances/prod/databases/db1",
            environment: "environments/prod",
            effectiveEnvironment: "environments/prod",
            instanceResource: { title: "Prod" },
          },
        ],
        nextPageToken: "page-2",
      })
      .mockResolvedValueOnce({
        databases: [
          {
            name: "instances/prod/databases/db2",
            environment: "environments/prod",
            effectiveEnvironment: "environments/prod",
            instanceResource: { title: "Prod" },
          },
        ],
        nextPageToken: "",
      });

    const { container, unmount } = renderIntoContainer(<Harness />);
    await flushPromises();
    await waitForText(container, "db1");

    // Opening the selector fetches exactly one bounded page, not the whole
    // list — this is the BYT-9785 freeze fix.
    expect(mocks.fetchDatabases).toHaveBeenCalledTimes(1);
    expect(mocks.fetchDatabases).toHaveBeenLastCalledWith(
      expect.objectContaining({ pageSize: 200, pageToken: "" })
    );
    expect(container.textContent).not.toContain("db2");

    const loadMoreButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("load-more"));
    expect(loadMoreButton).toBeTruthy();

    act(() => {
      loadMoreButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    await flushPromises();
    await waitForText(container, "db2");

    expect(mocks.fetchDatabases).toHaveBeenCalledTimes(2);
    expect(mocks.fetchDatabases).toHaveBeenLastCalledWith(
      expect.objectContaining({ pageSize: 200, pageToken: "page-2" })
    );

    unmount();
  });

  test("discards stale load-more results when the filter changes mid-flight", async () => {
    mocks.fetchDatabases.mockReset();
    let resolveStaleLoadMore: (value: unknown) => void = () => {};
    mocks.fetchDatabases
      // Initial first page.
      .mockResolvedValueOnce({
        databases: [
          {
            name: "instances/prod/databases/db1",
            instanceResource: { title: "Prod" },
          },
        ],
        nextPageToken: "page-2",
      })
      // Load-more request: stays pending until we resolve it by hand.
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveStaleLoadMore = resolve;
          })
      )
      // Refetch triggered by the filter change.
      .mockResolvedValueOnce({
        databases: [
          {
            name: "instances/prod/databases/db3",
            instanceResource: { title: "Prod" },
          },
        ],
        nextPageToken: "",
      });

    const { container, unmount } = renderIntoContainer(<Harness />);
    await flushPromises();
    await waitForText(container, "db1");

    // Kick off load more — the request hangs.
    const loadMoreButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent?.includes("load-more"));
    act(() => {
      loadMoreButton?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    await flushPromises();

    // Change the filter before load-more resolves -> new first page (db3).
    clickFirstButtonInRow(container, "advanced-search:");
    await flushPromises();
    await waitForText(container, "db3");

    // The stale load-more now resolves with a page from the old filter.
    act(() => {
      resolveStaleLoadMore({
        databases: [
          {
            name: "instances/prod/databases/db2",
            instanceResource: { title: "Prod" },
          },
        ],
        nextPageToken: "",
      });
    });
    await flushPromises();

    // Stale page must not leak into the filtered list.
    expect(container.textContent).not.toContain("db2");
    expect(container.textContent).toContain("db3");

    unmount();
  });

  test("hides Load more while a new filter's first page is refetching", async () => {
    mocks.fetchDatabases.mockReset();
    let resolveRefetch: (value: unknown) => void = () => {};
    mocks.fetchDatabases
      // Initial first page with more pages available.
      .mockResolvedValueOnce({
        databases: [
          {
            name: "instances/prod/databases/db1",
            instanceResource: { title: "Prod" },
          },
        ],
        nextPageToken: "page-2",
      })
      // First page after the filter change — stays pending so we can inspect
      // the in-between state.
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveRefetch = resolve;
          })
      );

    const { container, unmount } = renderIntoContainer(<Harness />);
    await flushPromises();
    await waitForText(container, "db1");

    const hasLoadMore = () =>
      Array.from(container.querySelectorAll("button")).some((button) =>
        button.textContent?.includes("load-more")
      );
    expect(hasLoadMore()).toBe(true);

    // Change the filter; the new first page is still in flight.
    clickFirstButtonInRow(container, "advanced-search:");
    await flushPromises();

    // The stale page token must be cleared so Load more cannot fire a request
    // pairing the old offset token with the new filter.
    expect(hasLoadMore()).toBe(false);

    // Once the new first page resolves, Load more reflects the new token.
    act(() => {
      resolveRefetch({
        databases: [
          {
            name: "instances/prod/databases/db3",
            instanceResource: { title: "Prod" },
          },
        ],
        nextPageToken: "new-page-2",
      });
    });
    await flushPromises();
    await waitForText(container, "db3");
    expect(hasLoadMore()).toBe(true);

    unmount();
  });

  test("selecting a table replaces child column selections", async () => {
    const onValueChange = vi.fn();
    const { container, unmount } = renderIntoContainer(
      <Harness
        includeColumns
        initialValue={[
          {
            databaseFullName: databaseName,
            schema: "public",
            table: "employee",
            columns: ["salary"],
          },
        ]}
        onValueChange={onValueChange}
      />
    );
    await flushPromises();
    await expandDatabase(container);
    clickFirstButtonInRow(container, "public");
    clickCheckboxInRow(container, "employee");

    expect(onValueChange).toHaveBeenLastCalledWith([
      {
        databaseFullName: databaseName,
        schema: "public",
        table: "employee",
      },
    ]);

    unmount();
  });
});
