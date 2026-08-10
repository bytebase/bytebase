import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  fetchDatabases: vi.fn(),
  getOrFetchChangelogByName: vi.fn(async () => ({
    name: "instances/source/databases/source-db/changelogs/1",
    schema: "CREATE TABLE t(id INT);",
  })),
  routeQuery: {
    changelog: "instances/source/databases/source-db/changelogs/1",
  } as Record<string, string>,
  sourceDatabase: {
    name: "instances/source/databases/source-db",
    effectiveEnvironment: "",
    instanceResource: {
      name: "instances/source",
      title: "Source instance",
      engine: 1,
    },
  },
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/app/router", () => ({
  router: {
    beforeEach: () => () => {},
    currentRoute: {
      get value() {
        return { query: mocks.routeQuery };
      },
    },
    push: vi.fn(),
  },
}));

vi.mock("@/app/router/routeHelpers", () => ({
  buildPlanCreateRoute: vi.fn(),
}));

vi.mock("@/components/EngineIcon", () => ({
  EngineIcon: () => null,
}));

vi.mock("@/components/DatabaseSelect", () => ({
  DatabaseSelect: ({
    onChange,
  }: {
    onChange: (name: string, database: typeof mocks.sourceDatabase) => void;
  }) => (
    <button
      data-testid="select-source-database"
      type="button"
      onClick={() => onChange(mocks.sourceDatabase.name, mocks.sourceDatabase)}
    />
  ),
}));

vi.mock("@/components/EnvironmentSelect", () => ({
  EnvironmentSelect: () => null,
}));

vi.mock("@/components/LearnMoreLink", () => ({
  LearnMoreLink: () => null,
}));

vi.mock("@/components/ProjectPageLayout", () => ({
  ProjectPageContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  ProjectPageInfo: () => null,
  ProjectPageLayout: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock("@/components/ui/sheet", () => ({
  Sheet: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetBody: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  SheetFooter: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  SheetHeader: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  SheetTitle: ({ children }: { children: ReactNode }) => <h2>{children}</h2>,
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => ({ name: "projects/project", title: "Project" }),
}));

vi.mock("@/components/monaco/core", () => ({
  createMonacoDiffEditor: vi.fn(),
  createMonacoEditor: vi.fn(),
  loadMonacoEditor: vi.fn(),
  setMonacoModelLanguage: vi.fn(),
}));

vi.mock("@/stores/app", () => {
  const state = {
    environmentList: [],
    fetchPreviousChangelog: vi.fn(),
    getOrFetchChangelogByName: mocks.getOrFetchChangelogByName,
    listChangelogs: vi.fn(async () => ({
      changelogs: [
        {
          name: "instances/source/databases/source-db/changelogs/1",
          planTitle: "Source changelog",
        },
      ],
      nextPageToken: "",
    })),
    projectsByName: {},
  };
  const getState = () => ({
    ...state,
    fetchDatabaseSchema: vi.fn(async () => ({ schema: "" })),
    fetchDatabases: mocks.fetchDatabases,
    getDatabaseByName: () => mocks.sourceDatabase,
    getEnvironmentByName: vi.fn(),
    getOrFetchDatabaseByName: vi.fn(async () => mocks.sourceDatabase),
  });
  return {
    useAppStore: Object.assign(
      (selector: (appState: typeof state) => unknown) => selector(state),
      { getState }
    ),
  };
});

vi.mock("@/stores/modules/v1/common", async () => {
  const actual = await vi.importActual<
    typeof import("@/stores/modules/v1/common")
  >("@/stores/modules/v1/common");
  return { ...actual, projectNamePrefix: "projects/" };
});

vi.mock("@/utils", async () => {
  const actual = await vi.importActual<typeof import("@/utils")>("@/utils");
  return {
    ...actual,
    getDatabaseEnvironment: () => ({ title: "Test" }),
    getDefaultPagination: () => 50,
    getInstanceResource: (database: typeof mocks.sourceDatabase) =>
      database.instanceResource,
  };
});

import { ProjectSyncSchemaPage } from "./ProjectSyncSchemaPage";

const flush = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
};

const click = async (element: HTMLElement) => {
  act(() => {
    element.dispatchEvent(
      new MouseEvent("click", { bubbles: true, cancelable: true })
    );
  });
  await flush();
};

describe("ProjectSyncSchemaPage target database search", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  beforeEach(() => {
    mocks.routeQuery = {
      changelog: "instances/source/databases/source-db/changelogs/1",
    };
    mocks.fetchDatabases.mockReset();
    mocks.fetchDatabases.mockImplementation(
      async ({ filter }: { filter?: { query?: string } }) => ({
        databases: [],
        nextPageToken: filter?.query ? "filtered-page-2" : "",
      })
    );
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.useRealTimers();
  });

  test("debounces the database name query and reuses it for load more", async () => {
    act(() => {
      root.render(<ProjectSyncSchemaPage projectId="project" />);
    });
    await flush();

    const selectButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.select"
    );
    expect(selectButton).toBeDefined();
    await click(selectButton!);

    const searchInput = container.querySelector("input");
    expect(searchInput).not.toBeNull();
    const initialRequestCount = mocks.fetchDatabases.mock.calls.length;
    vi.useFakeTimers();
    act(() => {
      Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        "value"
      )?.set?.call(searchInput, "Payroll");
      searchInput!.dispatchEvent(new Event("input", { bubbles: true }));
    });

    expect(mocks.fetchDatabases).toHaveBeenCalledTimes(initialRequestCount);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(299);
    });
    expect(mocks.fetchDatabases).toHaveBeenCalledTimes(initialRequestCount);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });

    expect(mocks.fetchDatabases).toHaveBeenCalledWith({
      parent: "projects/project",
      pageSize: 50,
      filter: { engines: [1], query: "Payroll" },
    });
    vi.useRealTimers();

    const loadMoreButton = Array.from(
      container.querySelectorAll("button")
    ).find((button) => button.textContent === "common.load-more");
    expect(loadMoreButton).toBeDefined();
    await click(loadMoreButton!);

    expect(mocks.fetchDatabases).toHaveBeenCalledWith({
      parent: "projects/project",
      pageSize: 50,
      pageToken: "filtered-page-2",
      filter: { engines: [1], query: "Payroll" },
    });
  });

  test("allows an environmentless source database with a valid changelog", async () => {
    mocks.routeQuery = {};
    act(() => {
      root.render(<ProjectSyncSchemaPage projectId="project" />);
    });
    await flush();

    const selectSourceDatabase = container.querySelector<HTMLElement>(
      '[data-testid="select-source-database"]'
    );
    expect(selectSourceDatabase).not.toBeNull();
    await click(selectSourceDatabase!);

    const nextButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.next"
    );
    expect(nextButton).toBeDefined();
    expect(nextButton).not.toBeDisabled();
  });
});
