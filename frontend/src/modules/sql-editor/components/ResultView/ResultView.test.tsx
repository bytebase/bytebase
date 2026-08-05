import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SQLEditorQueryParams } from "@/types";
import { Engine, ExportFormat } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { QueryResultSchema } from "@/types/proto-es/v1/sql_service_pb";
import { ResultView } from "./ResultView";

const { exportData } = vi.hoisted(() => ({
  exportData: vi.fn(),
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
    grantName: undefined,
    tooltip: undefined,
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
          options: { format: ExportFormat.CSV, limit: 100, password: "" },
          resolve: vi.fn(),
          reject: vi.fn(),
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
    exportData.mockResolvedValue(new Uint8Array());
  });

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
});
