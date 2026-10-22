import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorQueryParams } from "@/types";
import { Engine, ExportFormat } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  ExportRequest_Delivery,
  QueryResultSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { ResultView } from "./ResultView";

const { exportData, exportRequest } = vi.hoisted(() => ({
  exportData: vi.fn(),
  exportRequest: {
    delivery: 1,
    resolve: (_outcome: unknown) => {},
    reject: (_reason: unknown) => {},
  },
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
  RequestExportButton: () => null,
}));

vi.mock("@/modules/sql-editor/components/RequestQueryButton", () => ({
  RequestQueryButton: () => null,
}));

vi.mock("@/modules/sql-editor/components/useExportGrantBypass", () => ({
  useExportGrantBypass: () => ({
    loading: false,
    failedDatabases: [],
    unmatchedDatabases: [],
    retry: vi.fn(),
    grantName: undefined,
    tooltip: undefined,
  }),
}));

vi.mock("@/modules/sql-editor/components/useExportDelivery", () => ({
  useExportDelivery: () => ({
    deliveries: [2, 1],
    destination: { webUrl: "https://contoso.sharepoint.com/sites/x/Exports" },
  }),
}));

vi.mock("@/modules/sql-editor/hooks/useSQLEditorState", () => ({
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
}));

vi.mock("./EmptyView", () => ({
  EmptyView: () => <div data-testid="empty-view" />,
}));

vi.mock("./ErrorView", () => ({
  ErrorView: () => <div data-testid="error-view" />,
}));

vi.mock("./SingleResultView", () => ({
  SingleResultView: ({ onExport }: { onExport: (request: unknown) => void }) => (
    <button
      type="button"
      onClick={() =>
        onExport({
          options: {
            format: ExportFormat.CSV,
            limit: 100,
            password: "",
            delivery: exportRequest.delivery,
          },
          resolve: exportRequest.resolve,
          reject: exportRequest.reject,
          statement: "SELECT * FROM c",
        })
      }
    >
      export
    </button>
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

describe("ResultView", () => {
  beforeEach(() => {
    exportData.mockReset();
    exportData.mockResolvedValue({ content: new Uint8Array() });
    exportRequest.delivery = ExportRequest_Delivery.LOCAL;
    exportRequest.resolve = vi.fn();
    exportRequest.reject = vi.fn();
  });

  const singleResultSet = {
    error: "",
    results: [create(QueryResultSchema, { statement: "SELECT * FROM c" })],
  };

  test("blocks analytics recording for query results", () => {
    const { container } = render(
      <ResultView executeParams={executeParams} database={database} />
    );

    expect(container.firstElementChild).toHaveClass("ph-no-capture");
  });

  test("includes the selected CosmosDB container in export requests", async () => {
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

  test("resolves a SharePoint export with its delivered folder", async () => {
    exportRequest.delivery = ExportRequest_Delivery.SHAREPOINT;
    exportData.mockResolvedValue({
      content: new Uint8Array(),
      sharepointDelivery: { folderName: "20261022-main", webUrl: "https://f" },
    });
    render(
      <ResultView
        executeParams={executeParams}
        database={database}
        resultSet={singleResultSet}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: "export" }));

    await waitFor(() => {
      expect(exportRequest.resolve).toHaveBeenCalledWith({
        sharepoint: [
          {
            database: "main",
            status: "DELIVERED",
            folderName: "20261022-main",
            webUrl: "https://f",
          },
        ],
      });
    });
    expect(exportData).toHaveBeenCalledWith(
      expect.objectContaining({ delivery: ExportRequest_Delivery.SHAREPOINT })
    );
  });

  test("reports an unconfirmed SharePoint delivery instead of rejecting", async () => {
    exportRequest.delivery = ExportRequest_Delivery.SHAREPOINT;
    exportData.mockRejectedValue(
      new ConnectError("could not be confirmed", Code.Unavailable)
    );
    render(
      <ResultView
        executeParams={executeParams}
        database={database}
        resultSet={singleResultSet}
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
});
