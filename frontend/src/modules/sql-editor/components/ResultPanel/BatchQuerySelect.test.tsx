import { Code, ConnectError } from "@connectrpc/connect";
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Engine, ExportDestination } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { BatchQuerySelect } from "./BatchQuerySelect";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  databasesByName: {} as Record<string, Database>,
  databaseQueryContexts: new Map<string, unknown[]>(),
  environment: {
    name: "environments/prod",
    id: "prod",
    title: "Production",
    color: "#4f46e5",
    order: 0,
    tags: {},
  },
  deleteDatabaseQueryContext: vi.fn(),
  databasesWithExport: [] as string[],
  useExportGrantBypass: vi.fn(),
  destinations: [] as number[],
  exportData: vi.fn(),
  exportDestination: 2,
  notify: vi.fn(),
  resolveExport: vi.fn(),
  reportProgress: vi.fn(),
  exportController: new AbortController(),
  // The databases a retry asks for; undefined for a fresh export.
  retryDatabases: undefined as string[] | undefined,
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/DataExportButton", () => ({
  DataExportButton: ({
    formContent,
    text,
    onExport,
  }: {
    formContent: ReactNode;
    text: string;
    onExport: (request: unknown) => void;
  }) => (
    <div data-testid="batch-export">
      <span>{text}</span>
      {formContent}
      <button
        type="button"
        data-testid="batch-export-submit"
        onClick={() =>
          onExport({
            options: {
              format: 1,
              limit: 100,
              password: "",
              destination: mocks.exportDestination,
            },
            databases: mocks.retryDatabases,
            resolve: mocks.resolveExport,
            reject: vi.fn(),
            signal: mocks.exportController.signal,
            onProgress: mocks.reportProgress,
          })
        }
      />
    </div>
  ),
}));

vi.mock("@/components/database", () => ({
  DatabaseTableView: ({
    databases,
    onSelectedNamesChange,
  }: {
    databases: Database[];
    onSelectedNamesChange: (names: Set<string>) => void;
  }) => (
    <span data-testid="exportable">
      {databases.map((database) => database.name).join(",")}
      <button
        type="button"
        data-testid="select-all"
        onClick={() =>
          onSelectedNamesChange(new Set(databases.map((db) => db.name)))
        }
      />
    </span>
  ),
}));

vi.mock("@/components/EngineIcon", () => ({
  EngineIcon: ({ className }: { className?: string }) => (
    <span className={className} data-testid="engine-icon" />
  ),
}));

vi.mock("@/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({ environment }: { environment: { title: string } }) => (
    <span>{environment.title}</span>
  ),
}));

vi.mock("@/components/ui/ellipsis-text", () => ({
  EllipsisText: ({
    children,
    className,
    text,
  }: {
    children?: ReactNode;
    className?: string;
    text: string;
  }) => <span className={className}>{children ?? text}</span>,
}));

vi.mock("@/modules/sql-editor/components/RequestExportButton", () => ({
  RequestExportButton: ({ targets }: { targets: string[] }) => (
    <div data-testid="request-export">{targets.join(",")}</div>
  ),
}));

vi.mock("@/modules/sql-editor/components/useExportGrantBypass", () => ({
  useExportGrantBypass: mocks.useExportGrantBypass,
}));

vi.mock("@/modules/sql-editor/components/useExportDestination", () => ({
  useExportDestination: () => ({
    destinations: mocks.destinations,
    sharepointUrl: "https://contoso.sharepoint.com/sites/x/Exports",
  }),
}));

vi.mock("@/modules/sql-editor/hooks/useSQLEditorState", () => ({
  useSQLEditorDatabasesWithExport: (databases: Database[]) =>
    new Set(
      databases
        .map((database) => database.name)
        .filter((name) => mocks.databasesWithExport.includes(name))
    ),
  useSQLEditorQueryDataPolicy: () => ({
    maximumResultRows: 100,
  }),
}));

vi.mock("@/modules/sql-editor/store/editor", () => ({
  useSQLEditorEditorState: (
    selector: (state: { project: string }) => unknown
  ) => selector({ project: "projects/p" }),
}));

vi.mock("@/modules/sql-editor/store/tab", () => {
  const state = () => ({
    currentTabId: "tab-1",
    tabsById: new Map([
      [
        "tab-1",
        {
          databaseQueryContexts: mocks.databaseQueryContexts,
        },
      ],
    ]),
    deleteDatabaseQueryContext: mocks.deleteDatabaseQueryContext,
  });
  return {
    getSQLEditorTabsState: state,
    useCurrentSQLEditorTab: () =>
      state().tabsById.get(state().currentTabId),
    useSQLEditorTabState: (
      selector: (value: ReturnType<typeof state>) => unknown
    ) => selector(state()),
  };
});

vi.mock("@/stores/app", () => {
  const state = () => ({
    databasesByName: mocks.databasesByName,
    environmentList: [mocks.environment],
    exportData: mocks.exportData,
    getDatabaseByName: (name: string) => mocks.databasesByName[name],
    getEnvironmentByName: () => mocks.environment,
    notify: mocks.notify,
  });
  return {
    useAppStore: Object.assign(
      (selector: (value: ReturnType<typeof state>) => unknown) =>
        selector(state()),
      { getState: state }
    ),
  };
});

vi.mock("./ContextMenu", () => ({
  TabContextMenu: ({ children }: { children: ReactNode }) => children,
}));

describe("BatchQuerySelect", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;
  let cellA: Database;
  let cellB: Database;

  const grantLookup = (matched: string[], unmatched: string[]) =>
    mocks.useExportGrantBypass.mockReturnValue({
      loading: false,
      failedDatabases: [],
      matchedDatabases: matched,
      unmatchedDatabases: unmatched,
      retry: vi.fn(),
      grantName: "",
      tooltip: undefined,
    });

  beforeEach(() => {
    container = document.createElement("div");
    document.body.append(container);
    root = createRoot(container);
    mocks.databasesWithExport = [];
    mocks.destinations = [ExportDestination.DOWNLOAD];
    mocks.exportDestination = ExportDestination.SHAREPOINT;
    mocks.exportController = new AbortController();
    mocks.retryDatabases = undefined;
    grantLookup([], []);

    cellA = {
      name: "instances/cell-a/databases/linear",
      effectiveEnvironment: mocks.environment.name,
      instanceResource: {
        engine: Engine.POSTGRES,
        name: "instances/cell-a",
        title: "cell-a",
      },
    } as Database;
    cellB = {
      name: "instances/cell-b/databases/linear",
      effectiveEnvironment: mocks.environment.name,
      instanceResource: {
        engine: Engine.POSTGRES,
        name: "instances/cell-b",
        title: "cell-b",
      },
    } as Database;
    mocks.databasesByName = {
      [cellA.name]: cellA,
      [cellB.name]: cellB,
    };
    mocks.databaseQueryContexts = new Map([
      [
        cellA.name,
        [
          {
            id: "query-a",
            params: { connection: {}, statement: "SELECT 1" },
            status: "PENDING",
          },
        ],
      ],
      [
        cellB.name,
        [
          {
            id: "query-b",
            params: { connection: {}, statement: "SELECT 1" },
            status: "PENDING",
          },
        ],
      ],
    ]);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.clearAllMocks();
  });

  const render = () => {
    act(() => {
      root.render(
        <BatchQuerySelect
          selectedDatabase={cellA}
          onSelectedDatabaseChange={vi.fn()}
        />
      );
    });
    return Array.from(container.querySelectorAll("button"));
  };

  it("distinguishes same-named databases by instance and environment", () => {
    const buttons = render();

    expect(buttons).toHaveLength(2);
    expect(buttons[0].textContent).toContain("cell-a");
    expect(buttons[0].textContent).toContain("Production");
    expect(buttons[0].textContent).toContain("linear");
    expect(buttons[1].textContent).toContain("cell-b");
    expect(buttons[1].textContent).toContain("Production");
    expect(buttons[1].textContent).toContain("linear");
  });

  it("keeps each result tab from shrinking inside the horizontal scroller", () => {
    const buttons = render();

    expect(buttons[0].className).toContain("shrink-0");
    expect(buttons[1].className).toContain("shrink-0");
  });

  const testId = (id: string) =>
    container.querySelector(`[data-testid="${id}"]`);

  it("offers export of every database with bb.sql.export, without a grant lookup", () => {
    mocks.databasesWithExport = [cellA.name, cellB.name];
    render();

    expect(testId("batch-export")?.textContent).toContain(
      "sql-editor.batch-export.self"
    );
    expect(testId("exportable")?.textContent).toBe(
      `${cellA.name},${cellB.name}`
    );
    expect(testId("request-export")).toBeNull();
    expect(mocks.useExportGrantBypass).toHaveBeenLastCalledWith(
      expect.objectContaining({ enabled: false, targets: [] })
    );
  });

  it("exports the database with bb.sql.export and requests export for the other", () => {
    mocks.databasesWithExport = [cellA.name];
    grantLookup([], [cellB.name]);
    render();

    expect(testId("batch-export")?.textContent).toContain(
      "sql-editor.batch-export.partial"
    );
    expect(testId("exportable")?.textContent).toBe(cellA.name);
    expect(testId("request-export")?.textContent).toBe(cellB.name);
    expect(mocks.useExportGrantBypass).toHaveBeenLastCalledWith(
      expect.objectContaining({
        enabled: true,
        targets: [expect.objectContaining({ database: cellB.name })],
      })
    );
  });

  it("without bb.sql.export, exports granted databases and requests the rest", () => {
    grantLookup([cellA.name], [cellB.name]);
    render();

    expect(testId("batch-export")?.textContent).toContain(
      "sql-editor.batch-export.partial"
    );
    expect(testId("exportable")?.textContent).toBe(cellA.name);
    expect(testId("request-export")?.textContent).toBe(cellB.name);
    expect(mocks.useExportGrantBypass).toHaveBeenLastCalledWith(
      expect.objectContaining({ enabled: true, project: "projects/p" })
    );
  });

  it("without bb.sql.export or grants, offers only Request export", () => {
    grantLookup([], [cellA.name, cellB.name]);
    render();

    expect(testId("batch-export")).toBeNull();
    expect(testId("request-export")?.textContent).toBe(
      `${cellA.name},${cellB.name}`
    );
  });

  it("keeps Export but hides Request export when no destination is offered", () => {
    mocks.destinations = [];
    mocks.databasesWithExport = [cellA.name];
    grantLookup([], [cellB.name]);
    render();

    expect(testId("batch-export")).not.toBeNull();
    expect(testId("request-export")).toBeNull();
  });

  it("reports each database's SharePoint export separately", async () => {
    mocks.databasesWithExport = [cellA.name, cellB.name];
    mocks.destinations = [ExportDestination.SHAREPOINT];
    mocks.exportData
      .mockResolvedValueOnce({
        content: new Uint8Array(),
        uri: "https://contoso.sharepoint.com/sites/x/Exports/folder-a",
      })
      .mockRejectedValueOnce(
        new ConnectError("destination not found", Code.FailedPrecondition)
      );
    render();

    act(() => {
      container
        .querySelector<HTMLButtonElement>('[data-testid="select-all"]')
        ?.click();
    });
    await act(async () => {
      container
        .querySelector<HTMLButtonElement>('[data-testid="batch-export-submit"]')
        ?.click();
    });

    await vi.waitFor(() => {
      expect(mocks.resolveExport).toHaveBeenCalledWith({
        sharepoint: [
          {
            name: cellA.name,
            database: "cell-a/linear",
            status: "EXPORTED",
            uri: "https://contoso.sharepoint.com/sites/x/Exports/folder-a",
          },
          {
            name: cellB.name,
            database: "cell-b/linear",
            status: "FAILED",
            failure: { message: "destination not found" },
          },
        ],
      });
    });
    expect(mocks.exportData).toHaveBeenCalledWith(
      expect.objectContaining({ destination: ExportDestination.SHAREPOINT }),
      mocks.exportController.signal
    );
    expect(mocks.reportProgress.mock.calls).toEqual([
      [0, 2],
      [1, 2],
      [2, 2],
    ]);
  });

  const useDatabases = (count: number) => {
    const databases = Array.from(
      { length: count },
      (_, i) =>
        ({
          name: `instances/cell-${i}/databases/db${i}`,
          effectiveEnvironment: mocks.environment.name,
          instanceResource: {
            engine: Engine.POSTGRES,
            name: `instances/cell-${i}`,
            title: `cell-${i}`,
          },
        }) as Database
    );
    mocks.databasesByName = Object.fromEntries(
      databases.map((database) => [database.name, database])
    );
    mocks.databaseQueryContexts = new Map(
      databases.map((database, i) => [
        database.name,
        [
          {
            id: `query-${i}`,
            params: { connection: {}, statement: "SELECT 1" },
            status: "PENDING",
          },
        ],
      ])
    );
    mocks.databasesWithExport = databases.map((database) => database.name);
    return databases.map((database) => database.name);
  };

  // Each export stays pending until the test settles it or the export is
  // canceled, so the test decides the order in which exports finish.
  const deferExports = () => {
    const pending = new Map<
      string,
      { resolve: (uri: string) => void; reject: (error: Error) => void }
    >();
    let inFlight = 0;
    let peak = 0;
    mocks.exportData.mockImplementation(
      (request: { name: string }, signal: AbortSignal) =>
        new Promise((resolve, reject) => {
          inFlight++;
          peak = Math.max(peak, inFlight);
          signal.addEventListener("abort", () => {
            inFlight--;
            reject(new ConnectError("aborted", Code.Canceled));
          });
          pending.set(request.name, {
            resolve: (uri) => {
              inFlight--;
              resolve({ content: new TextEncoder().encode(uri), uri });
            },
            reject: (error) => {
              inFlight--;
              reject(error);
            },
          });
        })
    );
    const started = () =>
      mocks.exportData.mock.calls.map(([request]) => request.name);
    return { pending, started, peak: () => peak };
  };

  const flush = () =>
    act(() => new Promise<void>((resolve) => setTimeout(resolve, 0)));

  const exportAll = async () => {
    render();
    act(() => {
      testId("select-all")?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    act(() => {
      testId("batch-export-submit")?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    await flush();
  };

  it("exports at most four databases at a time and reports SharePoint results in selection order", async () => {
    mocks.destinations = [ExportDestination.SHAREPOINT];
    const names = useDatabases(6);
    const exports = deferExports();
    await exportAll();

    expect(exports.started()).toEqual(names.slice(0, 4));

    const settle = async (index: number, failure?: string) => {
      const entry = exports.pending.get(names[index]);
      if (failure) {
        entry?.reject(new ConnectError(failure, Code.FailedPrecondition));
      }
      else entry?.resolve(`https://example.com/exports/db${index}`);
      await flush();
    };
    await settle(3);
    expect(exports.started()).toEqual(names.slice(0, 5));
    await settle(0, "quota exceeded");
    expect(exports.started()).toEqual(names);
    await settle(5);
    await settle(2);
    await settle(4);
    expect(mocks.resolveExport).not.toHaveBeenCalled();
    await settle(1);

    expect(exports.peak()).toBe(4);
    expect(mocks.reportProgress.mock.calls).toEqual(
      [0, 1, 2, 3, 4, 5, 6].map((done) => [done, 6])
    );
    expect(mocks.resolveExport).toHaveBeenCalledTimes(1);
    const { sharepoint } = mocks.resolveExport.mock.calls[0][0];
    expect(sharepoint).toEqual([
      expect.objectContaining({
        name: names[0],
        database: "cell-0/db0",
        status: "FAILED",
        failure: { message: "quota exceeded" },
      }),
      ...[1, 2, 3, 4, 5].map((i) =>
        expect.objectContaining({
          name: names[i],
          database: `cell-${i}/db${i}`,
          status: "EXPORTED",
          uri: `https://example.com/exports/db${i}`,
        })
      ),
    ]);
  });

  it("downloads the successful exports in selection order when they finish out of order", async () => {
    mocks.exportDestination = ExportDestination.DOWNLOAD;
    const names = useDatabases(3);
    const exports = deferExports();
    await exportAll();

    expect(exports.started()).toEqual(names);
    exports.pending.get(names[2])?.resolve("db2-content");
    await flush();
    exports.pending.get(names[1])?.reject(new Error("connection refused"));
    await flush();
    exports.pending.get(names[0])?.resolve("db0-content");
    await flush();

    expect(mocks.reportProgress.mock.calls).toEqual(
      [0, 1, 2, 3].map((done) => [done, 3])
    );
    expect(mocks.notify).toHaveBeenCalledTimes(1);
    expect(mocks.notify).toHaveBeenCalledWith(
      expect.objectContaining({
        style: "CRITICAL",
        description: "Error: connection refused",
      })
    );
    expect(mocks.resolveExport).toHaveBeenCalledTimes(1);
    const files = mocks.resolveExport.mock.calls[0][0] as {
      content: Uint8Array;
      filename: string;
    }[];
    expect(
      files.map((file) => [
        new TextDecoder().decode(file.content),
        file.filename.split(".")[0],
      ])
    ).toEqual([
      ["db0-content", "db0"],
      ["db2-content", "db2"],
    ]);
  });

  it("stops starting databases once canceled and reports each database's SharePoint outcome", async () => {
    mocks.destinations = [ExportDestination.SHAREPOINT];
    const names = useDatabases(6);
    const exports = deferExports();
    await exportAll();
    exports.pending.get(names[3])?.resolve("https://example.com/exports/db3");
    await flush();
    expect(exports.started()).toEqual(names.slice(0, 5));

    act(() => mocks.exportController.abort());
    await flush();

    expect(exports.started()).toEqual(names.slice(0, 5));
    for (const call of mocks.exportData.mock.calls) {
      expect(call[1]).toBe(mocks.exportController.signal);
    }
    expect(mocks.resolveExport).toHaveBeenCalledWith({
      sharepoint: [
        { name: names[0], database: "cell-0/db0", status: "CANCELED" },
        { name: names[1], database: "cell-1/db1", status: "CANCELED" },
        { name: names[2], database: "cell-2/db2", status: "CANCELED" },
        {
          name: names[3],
          database: "cell-3/db3",
          status: "EXPORTED",
          uri: "https://example.com/exports/db3",
        },
        { name: names[4], database: "cell-4/db4", status: "CANCELED" },
        { name: names[5], database: "cell-5/db5", status: "NOT_STARTED" },
      ],
    });
  });

  it("retries only the requested databases and keeps the selection", async () => {
    mocks.destinations = [ExportDestination.SHAREPOINT];
    const names = useDatabases(3);
    const exports = deferExports();
    mocks.retryDatabases = [names[2], names[0]];
    await exportAll();

    expect(exports.started()).toEqual([names[2], names[0]]);
    exports.pending.get(names[2])?.resolve("https://example.com/exports/db2");
    exports.pending.get(names[0])?.resolve("https://example.com/exports/db0");
    await flush();
    expect(mocks.resolveExport).toHaveBeenCalledWith({
      sharepoint: [
        expect.objectContaining({ name: names[2], status: "EXPORTED" }),
        expect.objectContaining({ name: names[0], status: "EXPORTED" }),
      ],
    });

    mocks.retryDatabases = undefined;
    act(() => {
      testId("batch-export-submit")?.dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    await flush();
    expect(exports.started()).toEqual([names[2], names[0], ...names]);
  });

  it("reports no failures for a canceled download", async () => {
    mocks.exportDestination = ExportDestination.DOWNLOAD;
    useDatabases(3);
    deferExports();
    await exportAll();

    act(() => mocks.exportController.abort());
    await flush();

    expect(mocks.notify).not.toHaveBeenCalled();
    expect(mocks.resolveExport).toHaveBeenCalledWith([]);
  });
});
