import { act, useMemo, useState } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import type { DatabaseFilter } from "@/react/lib/databaseFilter";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { DatabaseTable } from "./DatabaseTable";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  fetchDatabases: vi.fn(),
}));

vi.mock("@/react/stores/app", () => {
  const state = {
    fetchDatabases: mocks.fetchDatabases,
    workspaceResourceName: () => "workspaces/-",
  };
  const useAppStore = <T,>(selector: (s: typeof state) => T) => selector(state);
  useAppStore.getState = () => state;
  return { useAppStore };
});

vi.mock("@/react/router", () => ({
  router: {
    resolve: () => ({ fullPath: "/database" }),
    push: vi.fn(),
  },
}));

vi.mock("@/utils", () => ({
  autoDatabaseRoute: () => ({ name: "database" }),
}));

vi.mock("@/react/hooks/useSessionPageSize", () => ({
  getPageSizeOptions: () => [50],
  useSessionPageSize: () => [50, vi.fn()],
}));

vi.mock("@/react/hooks/usePagedData", () => ({
  PagedTableFooter: ({
    hasMore,
    isFetchingMore,
    onLoadMore,
  }: {
    hasMore: boolean;
    isFetchingMore: boolean;
    onLoadMore: () => void;
  }) =>
    hasMore ? (
      <button type="button" disabled={isFetchingMore} onClick={onLoadMore}>
        load more
      </button>
    ) : null,
}));

vi.mock("./DatabaseTableView", () => ({
  DatabaseTableView: ({ databases }: { databases: Database[] }) => (
    <div data-testid="database-names">
      {databases.map((database) => database.name).join(",")}
    </div>
  ),
}));

const db1 = { name: "instances/i/databases/db1" } as Database;
const db2 = { name: "instances/i/databases/db2" } as Database;

function ParentRerenderHarness() {
  const [, setVisibleDatabases] = useState<Database[]>([]);

  const selectedLabels: string[] = [];
  const filter = useMemo<DatabaseFilter>(
    () => ({
      labels: selectedLabels.length > 0 ? selectedLabels : undefined,
    }),
    [selectedLabels]
  );

  return (
    <DatabaseTable
      filter={filter}
      parent="instances/i"
      onDatabasesChange={setVisibleDatabases}
    />
  );
}

describe("DatabaseTable", () => {
  let root: Root | undefined;
  let container: HTMLDivElement | undefined;

  afterEach(() => {
    vi.useRealTimers();
    act(() => {
      root?.unmount();
    });
    root = undefined;
    container = undefined;
    document.body.innerHTML = "";
    mocks.fetchDatabases.mockReset();
  });

  test("keeps appended rows when the parent rerenders with an equivalent filter", async () => {
    vi.useFakeTimers();
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    mocks.fetchDatabases
      .mockResolvedValueOnce({ databases: [db1], nextPageToken: "next" })
      .mockResolvedValueOnce({ databases: [db2], nextPageToken: "" })
      .mockResolvedValueOnce({ databases: [db1], nextPageToken: "next" });

    await act(async () => {
      root!.render(<ParentRerenderHarness />);
      await Promise.resolve();
    });

    expect(
      container.querySelector("[data-testid='database-names']")?.textContent
    ).toBe(db1.name);

    await act(async () => {
      container!
        .querySelector("button")
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
      await Promise.resolve();
    });

    expect(
      container.querySelector("[data-testid='database-names']")?.textContent
    ).toBe(`${db1.name},${db2.name}`);

    await act(async () => {
      vi.advanceTimersByTime(300);
      await Promise.resolve();
    });

    expect(mocks.fetchDatabases).toHaveBeenCalledTimes(2);
    expect(
      container.querySelector("[data-testid='database-names']")?.textContent
    ).toBe(`${db1.name},${db2.name}`);
  });
});
