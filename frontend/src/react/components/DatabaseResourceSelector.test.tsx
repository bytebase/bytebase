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
  databaseStore: undefined as
    | { fetchDatabases: ReturnType<typeof vi.fn> }
    | undefined,
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

mocks.databaseStore = {
  fetchDatabases: mocks.fetchDatabases,
};
mocks.instanceStore = {
  fetchInstanceList: mocks.fetchInstanceList,
};

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: { count?: number }) =>
      options?.count === undefined ? key : key,
  }),
}));

vi.mock("@/plugins/i18n", () => ({
  default: {
    global: {
      t: (key: string) => key,
    },
  },
  t: (key: string) => key,
}));

vi.mock("@/store", () => ({
  useDatabaseV1Store: () => mocks.databaseStore,
  useEnvironmentV1Store: () => mocks.environmentStore,
  useInstanceV1Store: () => mocks.instanceStore,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: (getter: () => unknown) => getter(),
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

vi.mock("@/store/modules/v1/dbSchema", () => ({
  useDBSchemaV1Store: () => ({
    getOrFetchDatabaseMetadata: mocks.getOrFetchDatabaseMetadata,
  }),
}));

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
