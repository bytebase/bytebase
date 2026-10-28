import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorQueryParams } from "@/types";
import { Engine, ExportFormat } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  QueryResult_QueryPlanSchema,
  QueryResultSchema,
  QueryRowSchema,
  RowValueSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { ResultView } from "./ResultView";

const { exportData, exportAccess, useExportGrantBypass } = vi.hoisted(() => ({
  exportData: vi.fn(),
  exportAccess: { databasesWithExport: [] as string[] },
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
            options: { format: ExportFormat.CSV, limit: 100, password: "" },
            resolve: vi.fn(),
            reject: vi.fn(),
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
    exportData.mockResolvedValue(new Uint8Array());
    exportAccess.databasesWithExport = [];
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
