import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorQueryParams } from "@/types";
import {
  Engine,
  ExportDestination,
  ExportFormat,
} from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  QueryResult_QueryPlanSchema,
  QueryResultSchema,
  QueryRowSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { ResultView } from "./ResultView";

const {
  exportData,
  exportAccess,
  exportDestination,
  exportRequest,
  useExportGrantBypass,
} = vi.hoisted(() => ({
  exportData: vi.fn(),
  exportAccess: { databasesWithExport: [] as string[] },
  exportDestination: {
    destinations: [] as number[],
    sharepointUrl: undefined as string | undefined,
  },
  exportRequest: {
    destination: 1,
    resolve: (_outcome: unknown) => {},
    reject: (_reason: unknown) => {},
  },
  useExportGrantBypass: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: {
    init: vi.fn(),
    type: "3rdParty",
  },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/DataExportButton", () => ({
  DataExportButton: () => null,
}));

vi.mock("@/modules/sql-editor/components/RequestExportButton", () => ({
  RequestExportButton: ({ targets }: { targets: string[] }) => (
    <div data-testid="request-export">{targets.join(",")}</div>
  ),
}));

vi.mock("@/modules/sql-editor/components/RequestQueryButton", () => ({
  RequestQueryButton: () => null,
}));

vi.mock("@/modules/sql-editor/components/useExportGrantBypass", () => ({
  useExportGrantBypass,
}));

vi.mock("@/modules/sql-editor/components/useExportDestination", () => ({
  useExportDestination: () => exportDestination,
}));

vi.mock("@/modules/sql-editor/hooks/useSQLEditorState", () => ({
  useSQLEditorDatabasesWithExport: (databases: Database[]) =>
    new Set(
      databases
        .map((database) => database.name)
        .filter((name) => exportAccess.databasesWithExport.includes(name))
    ),
  useSQLEditorQueryDataPolicy: () => undefined,
}));

vi.mock("@/modules/sql-editor/store/editor", () => ({
  useSQLEditorEditorState: (
    selector: (state: { project: string }) => unknown
  ) => selector({ project: "projects/prod" }),
  useSQLEditorEditorStore: {},
}));

vi.mock("@/modules/sql-editor/store/tab", () => ({
  getSQLEditorTabsState: () => ({
    currentTabId: "",
    tabsById: new Map(),
  }),
  useSQLEditorTabsStore: {},
}));

vi.mock("@/stores/app", () => {
  const state = {
    exportData,
    getOrFetchDatabaseMetadata: vi.fn(),
    getOrFetchPolicyByParentAndType: vi.fn(),
    getQueryDataPolicyByParent: vi.fn(),
    notify: vi.fn(),
    syncDatabase: vi.fn(),
  };
  return {
    useAppStore: Object.assign(
      (selector: (value: typeof state) => unknown) => selector(state),
      { getState: () => state }
    ),
  };
});

vi.mock("@/utils/iam/permission", () => ({
  hasProjectPermissionV2: () => false,
}));

vi.mock("@/utils/v1/database", () => ({
  extractDatabaseResourceName: () => ({ databaseName: "main" }),
  getDatabaseProject: () => "projects/prod",
  getInstanceResource: () => ({ engine: Engine.POSTGRES }),
}));

vi.mock("./EmptyView", () => ({
  EmptyView: () => <div data-testid="empty-view" />,
}));

vi.mock("./ErrorView", () => ({
  ErrorView: () => <div data-testid="error-view" />,
}));

vi.mock("./SingleResultView", () => ({
  SingleResultView: ({
    onExport,
    requestExportSlot,
    showExport,
  }: {
    onExport: (request: unknown) => void;
    requestExportSlot?: React.ReactNode;
    showExport: boolean;
  }) =>
    showExport ? (
      <button
        type="button"
        onClick={() =>
          onExport({
            options: {
              format: ExportFormat.CSV,
              limit: 100,
              password: "",
              destination: exportRequest.destination,
            },
            resolve: exportRequest.resolve,
            reject: exportRequest.reject,
            statement: "SELECT * FROM c",
          })
        }
      >
        export
      </button>
    ) : (
      requestExportSlot
    ),
}));

const executeParams: SQLEditorQueryParams = {
  connection: {
    database: "instances/prod/databases/main",
    instance: "instances/prod",
  },
  engine: Engine.POSTGRES,
  explain: false,
  selection: null,
  statement: "select secret from customer",
};

const database = {
  effectiveEnvironment: "environments/prod",
  name: "instances/prod/databases/main",
  project: "projects/prod",
} as Database;

const grantLookup = ({
  grantName = "",
  unmatchedDatabases = [] as string[],
}) =>
  useExportGrantBypass.mockReturnValue({
    loading: false,
    failedDatabases: [],
    matchedDatabases: [],
    unmatchedDatabases,
    retry: vi.fn(),
    grantName,
    tooltip: undefined,
  });

describe("ResultView", () => {
  beforeEach(() => {
    exportData.mockReset();
    exportData.mockResolvedValue({ content: new Uint8Array(), uri: "" });
    exportAccess.databasesWithExport = [];
    exportDestination.destinations = [ExportDestination.SHAREPOINT, ExportDestination.DOWNLOAD];
    exportDestination.sharepointUrl =
      "https://contoso.sharepoint.com/sites/x/Exports";
    exportRequest.destination = ExportDestination.DOWNLOAD;
    exportRequest.resolve = vi.fn();
    exportRequest.reject = vi.fn();
    grantLookup({});
  });

  test("blocks analytics recording for query results", () => {
    const { container } = render(
      <ResultView executeParams={executeParams} database={database} />
    );

    expect(container.firstElementChild).toHaveClass("ph-no-capture");
  });

  test("includes the selected CosmosDB container in export requests", async () => {
    exportAccess.databasesWithExport = [database.name];
    render(
      <ResultView
        executeParams={{
          ...executeParams,
          engine: Engine.COSMOSDB,
          connection: {
            ...executeParams.connection,
            table: "customers",
          },
        }}
        database={database}
        resultSet={{
          error: "",
          results: [
            create(QueryResultSchema, {
              statement: "SELECT * FROM c",
            }),
          ],
        }}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: "export" }));

    await waitFor(() => {
      expect(exportData).toHaveBeenCalledWith(
        expect.objectContaining({ container: "customers" })
      );
    });
  });

  const queryResult = {
    error: "",
    results: [create(QueryResultSchema, { statement: "SELECT 1" })],
  };

  test("bb.sql.export offers Export without looking up a grant", () => {
    exportAccess.databasesWithExport = [database.name];
    render(
      <ResultView
        executeParams={executeParams}
        database={database}
        resultSet={queryResult}
      />
    );

    expect(screen.getByRole("button", { name: "export" })).toBeInTheDocument();
    expect(screen.queryByTestId("request-export")).not.toBeInTheDocument();
    expect(useExportGrantBypass).toHaveBeenLastCalledWith(
      expect.objectContaining({ enabled: false })
    );
  });

  test("a matching export grant offers Export without bb.sql.export", () => {
    grantLookup({ grantName: "projects/prod/accessGrants/g" });
    render(
      <ResultView
        executeParams={executeParams}
        database={database}
        resultSet={queryResult}
      />
    );

    expect(screen.getByRole("button", { name: "export" })).toBeInTheDocument();
    expect(useExportGrantBypass).toHaveBeenLastCalledWith(
      expect.objectContaining({ enabled: true, project: "projects/prod" })
    );
  });

  test("without bb.sql.export or a grant, offers Request export", () => {
    grantLookup({ unmatchedDatabases: [database.name] });
    render(
      <ResultView
        executeParams={executeParams}
        database={database}
        resultSet={queryResult}
      />
    );

    expect(
      screen.queryByRole("button", { name: "export" })
    ).not.toBeInTheDocument();
    expect(screen.getByTestId("request-export")).toHaveTextContent(
      database.name
    );
  });

  test("bb.sql.export scoped to one database offers Export there and Request export elsewhere", () => {
    const other = {
      ...database,
      name: "instances/prod/databases/other",
    } as Database;
    exportAccess.databasesWithExport = [database.name];
    grantLookup({ unmatchedDatabases: [other.name] });

    const { unmount } = render(
      <ResultView
        executeParams={executeParams}
        database={database}
        resultSet={queryResult}
      />
    );
    expect(screen.getByRole("button", { name: "export" })).toBeInTheDocument();
    expect(screen.queryByTestId("request-export")).not.toBeInTheDocument();
    unmount();

    render(
      <ResultView
        executeParams={executeParams}
        database={other}
        resultSet={queryResult}
      />
    );
    expect(
      screen.queryByRole("button", { name: "export" })
    ).not.toBeInTheDocument();
    expect(screen.getByTestId("request-export")).toHaveTextContent(other.name);
    expect(useExportGrantBypass).toHaveBeenLastCalledWith(
      expect.objectContaining({
        enabled: true,
        targets: [expect.objectContaining({ database: other.name })],
      })
    );
  });

  test("keeps Export but hides Request export when no destination is offered", () => {
    exportDestination.destinations = [];
    exportDestination.sharepointUrl = undefined;
    exportAccess.databasesWithExport = [database.name];
    const { unmount } = render(
      <ResultView
        executeParams={executeParams}
        database={database}
        resultSet={queryResult}
      />
    );
    expect(screen.getByRole("button", { name: "export" })).toBeInTheDocument();
    unmount();

    exportAccess.databasesWithExport = [];
    grantLookup({ unmatchedDatabases: [database.name] });
    render(
      <ResultView
        executeParams={executeParams}
        database={database}
        resultSet={queryResult}
      />
    );
    expect(screen.queryByTestId("request-export")).not.toBeInTheDocument();
  });

  test("offers Export when only SharePoint is offered", () => {
    exportDestination.destinations = [ExportDestination.SHAREPOINT];
    exportAccess.databasesWithExport = [database.name];
    render(
      <ResultView
        executeParams={executeParams}
        database={database}
        resultSet={queryResult}
      />
    );

    expect(screen.getByRole("button", { name: "export" })).toBeInTheDocument();
  });

  test("resolves a SharePoint export with its exported folder", async () => {
    exportAccess.databasesWithExport = [database.name];
    exportRequest.destination = ExportDestination.SHAREPOINT;
    exportData.mockResolvedValue({
      content: new Uint8Array(),
      uri: "https://contoso.sharepoint.com/sites/x/Exports/20261022-main",
    });
    render(
      <ResultView
        executeParams={executeParams}
        database={database}
        resultSet={queryResult}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: "export" }));

    await waitFor(() => {
      expect(exportRequest.resolve).toHaveBeenCalledWith({
        sharepoint: [
          {
            database: "main",
            status: "EXPORTED",
            uri: "https://contoso.sharepoint.com/sites/x/Exports/20261022-main",
          },
        ],
      });
    });
    expect(exportData).toHaveBeenCalledWith(
      expect.objectContaining({ destination: ExportDestination.SHAREPOINT })
    );
  });

  test("reports an unconfirmed SharePoint export instead of rejecting", async () => {
    exportAccess.databasesWithExport = [database.name];
    exportRequest.destination = ExportDestination.SHAREPOINT;
    exportData.mockRejectedValue(
      new ConnectError("could not be confirmed", Code.Unavailable)
    );
    render(
      <ResultView
        executeParams={executeParams}
        database={database}
        resultSet={queryResult}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: "export" }));

    await waitFor(() => {
      expect(exportRequest.resolve).toHaveBeenCalledWith({
        sharepoint: [
          {
            database: "main",
            status: "UNCONFIRMED",
            reason: "could not be confirmed",
          },
        ],
      });
    });
    expect(exportRequest.reject).not.toHaveBeenCalled();
  });

  /** A PostgreSQL plan result whose root costs `totalCost`. */
  const planResult = (totalCost: number) =>
    create(QueryResultSchema, {
      statement: "SELECT 1",
      columnNames: ["QUERY PLAN"],
      queryPlan: create(QueryResult_QueryPlanSchema, {}),
      rows: [
        create(QueryRowSchema, {
          values: [
            create(RowValueSchema, {
              kind: {
                case: "stringValue",
                value: JSON.stringify([
                  { Plan: { "Node Type": "Seq Scan", "Total Cost": totalCost } },
                ]),
              },
            }),
          ],
        }),
      ],
    });

  const tabs = () => screen.getAllByRole("tab").map((tab) => tab.textContent);

  test("says which of several plans is the one to read", () => {
    render(
      <ResultView
        executeParams={executeParams}
        database={database}
        resultSet={{ error: "", results: [planResult(90), planResult(10)] }}
      />
    );

    expect(tabs()[0]).toContain("90%");
    expect(tabs()[1]).toContain("10%");
  });

  test("leaves the tabs alone when the script's whole cost is unknown", () => {
    render(
      <ResultView
        executeParams={executeParams}
        database={database}
        resultSet={{
          error: "",
          results: [
            planResult(90),
            create(QueryResultSchema, { statement: "SELECT 2" }),
          ],
        }}
      />
    );

    expect(tabs().join()).not.toContain("%");
  });
});
