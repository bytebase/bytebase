import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { MatchedDatabaseView } from "./MatchedDatabaseView";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  batchGetOrFetchDatabases: vi.fn(),
  databasesByName: {} as Record<string, unknown>,
  fetchDatabases: vi.fn(),
  getDatabaseByName: vi.fn(),
  getEnvironmentByName: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params?.n !== undefined ? `${key}:${params.n}` : key,
  }),
}));

vi.mock("@/plugins/cel", () => ({
  validateSimpleExpr: () => true,
}));

vi.mock("@/react/components/EngineIcon", () => ({
  EngineIcon: ({
    className,
    engine,
  }: {
    className?: string;
    engine: Engine;
  }) => (
    <span className={className} data-testid="engine-icon">
      {Engine[engine]}
    </span>
  ),
}));

vi.mock("@/react/lib/utils", () => ({
  cn: (...classes: Array<string | false | null | undefined>) =>
    classes.filter(Boolean).join(" "),
}));

vi.mock("@/react/stores/app", () => {
  const getState = () => ({
    batchGetOrFetchDatabases: mocks.batchGetOrFetchDatabases,
    databasesByName: mocks.databasesByName,
    environmentList: [],
    fetchDatabases: mocks.fetchDatabases,
    getDatabaseByName: mocks.getDatabaseByName,
    getEnvironmentByName: mocks.getEnvironmentByName,
  });
  const useAppStore = <T,>(
    selector: (state: ReturnType<typeof getState>) => T
  ) => selector(getState());
  useAppStore.getState = getState;
  return { useAppStore };
});

vi.mock("@/types", () => ({
  DEBOUNCE_SEARCH_DELAY: 0,
  isValidDatabaseName: (name: string) => name.includes("/databases/"),
}));

vi.mock("@/types/v1/database", () => ({
  unknownDatabase: () => ({ name: "", instanceResource: { title: "" } }),
}));

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: (name: string) => ({
    databaseName: name.split("/databases/")[1] ?? name,
  }),
  getDefaultPagination: () => 2,
  getDatabaseEnvironment: (database: { effectiveEnvironment?: string }) => ({
    title: database.effectiveEnvironment?.includes("prod")
      ? "Production"
      : "Staging",
  }),
  getInstanceResource: (database: {
    instanceResource?: { engine: Engine; title: string };
  }) => database.instanceResource,
}));

const matchedDatabase = {
  name: "projects/p/instances/prod/databases/app",
  effectiveEnvironment: "environments/prod",
  instanceResource: {
    engine: Engine.POSTGRES,
    title: "prod-instance",
  },
};

const unmatchedDatabase = {
  name: "projects/p/instances/staging/databases/app_staging",
  effectiveEnvironment: "environments/staging",
  instanceResource: {
    engine: Engine.MYSQL,
    title: "staging-instance",
  },
};

async function flush() {
  await act(async () => {
    vi.runOnlyPendingTimers();
    await Promise.resolve();
    await Promise.resolve();
  });
}

describe("MatchedDatabaseView", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  beforeEach(() => {
    vi.useFakeTimers();
    container = document.createElement("div");
    document.body.append(container);
    root = createRoot(container);

    mocks.databasesByName = {
      [matchedDatabase.name]: matchedDatabase,
      [unmatchedDatabase.name]: unmatchedDatabase,
    };
    mocks.batchGetOrFetchDatabases.mockResolvedValue(undefined);
    mocks.getDatabaseByName.mockImplementation(
      (name: string) => mocks.databasesByName[name] ?? { name: "" }
    );
    mocks.fetchDatabases.mockResolvedValue({
      databases: [matchedDatabase, unmatchedDatabase],
      nextPageToken: "next",
    });
    mocks.getEnvironmentByName.mockImplementation((name: string) => ({
      name,
      title: name.includes("prod") ? "Production" : "Staging",
    }));
  });

  afterEach(() => {
    act(() => {
      root.unmount();
    });
    container.remove();
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it("renders matched and unmatched databases with engine, environment, instance, and database name", async () => {
    act(() => {
      root.render(
        <MatchedDatabaseView
          expr={{} as never}
          matchedDatabaseNames={[matchedDatabase.name]}
          project="projects/p"
        />
      );
    });
    await flush();

    expect(container.textContent).toContain("database-group.matched-databases");
    expect(container.textContent).toContain("1");
    expect(container.textContent).toContain("POSTGRES");
    expect(container.textContent).toContain("Production");
    expect(container.textContent).toContain("prod-instance");
    expect(container.textContent).toContain("app");
    expect(container.textContent).toContain(
      "database-group.unmatched-databases"
    );
    expect(container.textContent).toContain("MYSQL");
    expect(container.textContent).toContain("Staging");
    expect(container.textContent).toContain("staging-instance");
    expect(container.textContent).toContain("app_staging");
  });

  it("labels unmatched databases as a preview while more project pages remain", async () => {
    act(() => {
      root.render(
        <MatchedDatabaseView
          expr={{} as never}
          matchedDatabaseNames={[matchedDatabase.name]}
          project="projects/p"
        />
      );
    });
    await flush();

    expect(container.textContent).toContain(
      "database-group.unmatched-databases-preview"
    );
    expect(container.textContent).not.toContain("database-group.n-unmatched");
  });
});
