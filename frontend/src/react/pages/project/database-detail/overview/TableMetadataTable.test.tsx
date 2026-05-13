import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
} from "react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
  })),
  useVueState: vi.fn(),
  useDatabaseCatalog: vi.fn(),
  getTableCatalog: vi.fn(),
  featureToRef: vi.fn(() => ({ value: true })),
  useSettingV1Store: vi.fn(),
  updateTableCatalog: vi.fn(),
  getDatabaseEngine: vi.fn(() => Engine.POSTGRES),
  getDatabaseProject: vi.fn(() => ({
    name: "projects/proj1",
    dataClassificationConfigId: "classification-config",
  })),
  bytesToString: vi.fn((value: number) => `${value} B`),
  getInstanceResource: vi.fn(() => ({
    name: "instances/inst1",
    engine: Engine.POSTGRES,
  })),
  instanceV1MaskingForNoSQL: vi.fn(() => false),
  hasProjectPermissionV2: vi.fn(() => true),
  hasSchemaProperty: vi.fn(() => true),
  hasWorkspacePermissionV2: vi.fn(() => true),
  getOrFetchSettingByName: vi.fn(),
  getSettingByName: vi.fn(),
  getProjectClassification: vi.fn(),
}));

let TableMetadataTable: typeof import("./TableMetadataTable").TableMetadataTable;

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useDatabaseCatalog: mocks.useDatabaseCatalog,
  getTableCatalog: mocks.getTableCatalog,
  featureToRef: mocks.featureToRef,
  useSettingV1Store: mocks.useSettingV1Store,
}));

vi.mock("@/utils", () => ({
  bytesToString: mocks.bytesToString,
  getDatabaseEngine: mocks.getDatabaseEngine,
  getDatabaseProject: mocks.getDatabaseProject,
  getInstanceResource: mocks.getInstanceResource,
  instanceV1MaskingForNoSQL: mocks.instanceV1MaskingForNoSQL,
  hasProjectPermissionV2: mocks.hasProjectPermissionV2,
  hasSchemaProperty: mocks.hasSchemaProperty,
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

vi.mock("@/react/lib/column-data-table/utils", () => ({
  updateTableCatalog: mocks.updateTableCatalog,
}));

vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open: boolean; children: ReactNode }) =>
    open ? (
      <div
        onClick={(event) => event.stopPropagation()}
        onMouseDown={(event) => event.stopPropagation()}
      >
        {children}
      </div>
    ) : null,
  DialogContent: ({ children }: { children: ReactNode }) => (
    <div
      onClick={(event) => event.stopPropagation()}
      onMouseDown={(event) => event.stopPropagation()}
    >
      {children}
    </div>
  ),
  DialogTitle: ({ children }: { children: ReactNode }) => <h1>{children}</h1>,
}));

vi.mock("@/react/components/ui/search-input", () => ({
  SearchInput: (props: InputHTMLAttributes<HTMLInputElement>) => (
    <input {...props} />
  ),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({ children, ...props }: ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
}));

vi.mock("@/react/components/ui/table", () => ({
  Table: ({
    children,
    ...props
  }: React.TableHTMLAttributes<HTMLTableElement>) => (
    <table {...props}>{children}</table>
  ),
  TableBody: ({
    children,
    ...props
  }: React.HTMLAttributes<HTMLTableSectionElement>) => (
    <tbody {...props}>{children}</tbody>
  ),
  TableCell: ({
    children,
    ...props
  }: React.TdHTMLAttributes<HTMLTableCellElement>) => (
    <td {...props}>{children}</td>
  ),
  TableHead: ({
    children,
    ...props
  }: React.ThHTMLAttributes<HTMLTableCellElement>) => (
    <th {...props}>{children}</th>
  ),
  TableHeader: ({
    children,
    ...props
  }: React.HTMLAttributes<HTMLTableSectionElement>) => (
    <thead {...props}>{children}</thead>
  ),
  TableRow: ({
    children,
    ...props
  }: React.HTMLAttributes<HTMLTableRowElement>) => (
    <tr {...props}>{children}</tr>
  ),
}));

vi.mock("@/react/components/FeatureAttention", () => ({
  FeatureAttention: ({ feature }: { feature: PlanFeature }) => (
    <div>{PlanFeature[feature]}</div>
  ),
}));

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

const flush = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
};

const click = (element: HTMLElement) => {
  act(() => {
    element.dispatchEvent(
      new MouseEvent("click", { bubbles: true, cancelable: true })
    );
  });
};

const press = (element: HTMLElement) => {
  act(() => {
    element.dispatchEvent(
      new MouseEvent("mousedown", { bubbles: true, cancelable: true })
    );
    element.dispatchEvent(
      new MouseEvent("click", { bubbles: true, cancelable: true })
    );
  });
};

const keydown = (element: HTMLElement, key: string) => {
  act(() => {
    element.dispatchEvent(
      new KeyboardEvent("keydown", { bubbles: true, cancelable: true, key })
    );
  });
};

const makeDatabase = (): Database =>
  ({
    name: "instances/inst1/databases/db",
    project: "projects/proj1",
    instanceResource: {
      name: "instances/inst1",
      engine: Engine.POSTGRES,
    },
  }) as Database;

beforeEach(async () => {
  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });
  mocks.useVueState.mockReset();
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());
  mocks.useDatabaseCatalog.mockReset();
  mocks.useDatabaseCatalog.mockReturnValue({
    value: { schemas: [] },
  });
  mocks.getTableCatalog.mockReset();
  mocks.getTableCatalog.mockReturnValue({
    classification: "PII",
  });
  mocks.featureToRef.mockReset();
  mocks.featureToRef.mockReturnValue({ value: true });
  mocks.getDatabaseEngine.mockReset();
  mocks.getDatabaseEngine.mockReturnValue(Engine.POSTGRES);
  mocks.getDatabaseProject.mockReset();
  mocks.getDatabaseProject.mockReturnValue({
    name: "projects/proj1",
    dataClassificationConfigId: "classification-config",
  });
  mocks.bytesToString.mockReset();
  mocks.bytesToString.mockImplementation((value: number) => `${value} B`);
  mocks.getInstanceResource.mockReset();
  mocks.getInstanceResource.mockReturnValue({
    name: "instances/inst1",
    engine: Engine.POSTGRES,
  });
  mocks.instanceV1MaskingForNoSQL.mockReset();
  mocks.instanceV1MaskingForNoSQL.mockReturnValue(false);
  mocks.hasProjectPermissionV2.mockReset();
  mocks.hasProjectPermissionV2.mockReturnValue(true);
  mocks.hasSchemaProperty.mockReset();
  mocks.hasSchemaProperty.mockReturnValue(true);
  mocks.hasWorkspacePermissionV2.mockReset();
  mocks.hasWorkspacePermissionV2.mockReturnValue(true);
  mocks.getOrFetchSettingByName.mockReset();
  mocks.getSettingByName.mockReset();
  mocks.getProjectClassification.mockReset();
  mocks.getProjectClassification.mockReturnValue({
    id: "classification-config",
    levels: [
      { level: 1, title: "Level 1" },
      { level: 2, title: "Level 2" },
    ],
    classification: {
      PII: { id: "PII", title: "PII", level: 1 },
      CONFIDENTIAL: {
        id: "CONFIDENTIAL",
        title: "Confidential",
        level: 2,
      },
    },
  });
  mocks.useSettingV1Store.mockReset();
  mocks.useSettingV1Store.mockReturnValue({
    getOrFetchSettingByName: mocks.getOrFetchSettingByName,
    getSettingByName: mocks.getSettingByName,
    getProjectClassification: mocks.getProjectClassification,
  });
  mocks.updateTableCatalog.mockReset();
  mocks.updateTableCatalog.mockResolvedValue(undefined);

  vi.resetModules();
  ({ TableMetadataTable } = await import("./TableMetadataTable"));
});

describe("TableMetadataTable", () => {
  test("opens table detail when clicking a row", async () => {
    const onRowClick = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      createElement(TableMetadataTable, {
        database: makeDatabase(),
        schemaName: "public",
        rows: [
          {
            name: "audit",
            rowCount: 0,
            dataSize: 128n,
            indexSize: 64n,
            comment: "audit table",
            partitions: [],
          } as never,
        ],
        onRowClick,
      })
    );

    render();
    await flush();

    const row = Array.from(container.querySelectorAll("tr")).find((row) =>
      row.textContent?.includes("audit")
    );
    expect(row).not.toBeNull();

    click(row as HTMLElement);
    await flush();

    expect(onRowClick).toHaveBeenCalledTimes(1);

    unmount();
  });

  test("clicking the classification icon does not trigger row open", async () => {
    const onRowClick = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      createElement(TableMetadataTable, {
        database: makeDatabase(),
        schemaName: "public",
        rows: [
          {
            name: "audit",
            rowCount: 0,
            dataSize: 128n,
            indexSize: 64n,
            comment: "audit table",
            partitions: [],
          } as never,
        ],
        onRowClick,
      })
    );

    render();
    await flush();

    const editButton = container.querySelector(
      '[data-testid="table-row-classification-audit-edit"]'
    );
    expect(editButton).not.toBeNull();

    press(editButton as HTMLElement);
    await flush();
    click(
      container.querySelector(
        '[data-testid="classification-option-CONFIDENTIAL"]'
      ) as HTMLElement
    );
    await flush();

    expect(mocks.updateTableCatalog).toHaveBeenCalledWith(
      expect.objectContaining({
        database: "instances/inst1/databases/db",
        schema: "public",
        table: "audit",
        tableCatalog: { classification: "CONFIDENTIAL" },
      })
    );
    expect(onRowClick).not.toHaveBeenCalled();
    expect(mocks.getOrFetchSettingByName).toHaveBeenCalledWith(
      Setting_SettingName.DATA_CLASSIFICATION,
      true
    );

    unmount();
  });

  test("clicking the classification action region does not trigger row open", async () => {
    const onRowClick = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      createElement(TableMetadataTable, {
        database: makeDatabase(),
        schemaName: "public",
        rows: [
          {
            name: "audit",
            rowCount: 0,
            dataSize: 128n,
            indexSize: 64n,
            comment: "audit table",
            partitions: [],
          } as never,
        ],
        onRowClick,
      })
    );

    render();
    await flush();

    const actionRegion = container.querySelector(
      '[data-testid="table-row-classification-audit-action"]'
    );
    expect(actionRegion).not.toBeNull();

    press(actionRegion as HTMLElement);
    await flush();

    expect(onRowClick).not.toHaveBeenCalled();

    unmount();
  });

  test("pressing enter on the classification edit button does not trigger row open", async () => {
    const onRowClick = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      createElement(TableMetadataTable, {
        database: makeDatabase(),
        schemaName: "public",
        rows: [
          {
            name: "audit",
            rowCount: 0,
            dataSize: 128n,
            indexSize: 64n,
            comment: "audit table",
            partitions: [],
          } as never,
        ],
        onRowClick,
      })
    );

    render();
    await flush();

    const editButton = container.querySelector(
      '[data-testid="table-row-classification-audit-edit"]'
    );
    expect(editButton).not.toBeNull();

    keydown(editButton as HTMLElement, "Enter");
    await flush();

    expect(onRowClick).not.toHaveBeenCalled();

    unmount();
  });
});
