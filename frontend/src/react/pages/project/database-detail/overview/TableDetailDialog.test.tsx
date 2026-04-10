import type {
  ButtonHTMLAttributes,
  ElementType,
  InputHTMLAttributes,
  ReactNode,
} from "react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { DataClassificationSetting_DataClassificationConfig } from "@/types/proto-es/v1/setting_service_pb";
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
  useSettingV1Store: vi.fn(),
  useSubscriptionV1Store: vi.fn(),
  getOrFetchSettingByName: vi.fn(),
  getSettingByName: vi.fn(),
  getProjectClassification: vi.fn(),
  updateColumnCatalog: vi.fn(),
  updateTableCatalog: vi.fn(),
  getDatabaseProject: vi.fn(),
  getInstanceResource: vi.fn(),
  hasProjectPermissionV2: vi.fn(),
  hasWorkspacePermissionV2: vi.fn(),
}));

let TableDetailDialog: typeof import("./TableDetailDialog").TableDetailDialog;

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open: boolean; children: ReactNode }) =>
    open ? <div data-testid="dialog-root">{children}</div> : null,
  DialogContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  DialogTitle: ({ children }: { children: ReactNode }) => <h1>{children}</h1>,
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: InputHTMLAttributes<HTMLInputElement>) => <input {...props} />,
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

vi.mock("@/react/components/FeatureAttention", () => ({
  FeatureAttention: ({ feature }: { feature: PlanFeature }) => (
    <div>{PlanFeature[feature]}</div>
  ),
}));

vi.mock("@/store", () => ({
  useSettingV1Store: mocks.useSettingV1Store,
  useSubscriptionV1Store: mocks.useSubscriptionV1Store,
}));

vi.mock("@/utils", () => ({
  getDatabaseProject: mocks.getDatabaseProject,
  getInstanceResource: mocks.getInstanceResource,
  hasProjectPermissionV2: mocks.hasProjectPermissionV2,
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

vi.mock("@/components/ColumnDataTable/utils", () => ({
  updateColumnCatalog: mocks.updateColumnCatalog,
  updateTableCatalog: mocks.updateTableCatalog,
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
      PII: {
        id: "PII",
        title: "PII",
        level: 1,
      },
      CONFIDENTIAL: {
        id: "CONFIDENTIAL",
        title: "Confidential",
        level: 2,
      },
    },
  });
  mocks.getSettingByName.mockImplementation((name: Setting_SettingName) => {
    if (name !== Setting_SettingName.SEMANTIC_TYPES) {
      return undefined;
    }
    return {
      value: {
        value: {
          case: "semanticType",
          value: {
            types: [
              { id: "EMAIL", title: "Email" },
              { id: "PHONE", title: "Phone" },
            ],
          },
        },
      },
    };
  });
  mocks.useSettingV1Store.mockReset();
  mocks.useSettingV1Store.mockReturnValue({
    getOrFetchSettingByName: mocks.getOrFetchSettingByName,
    getSettingByName: mocks.getSettingByName,
    getProjectClassification: mocks.getProjectClassification,
  });
  mocks.useSubscriptionV1Store.mockReset();
  mocks.useSubscriptionV1Store.mockReturnValue({
    hasFeature: vi.fn(() => true),
    instanceMissingLicense: vi.fn(() => false),
  });
  mocks.updateColumnCatalog.mockReset();
  mocks.updateColumnCatalog.mockResolvedValue(undefined);
  mocks.updateTableCatalog.mockReset();
  mocks.updateTableCatalog.mockResolvedValue(undefined);
  mocks.getDatabaseProject.mockReset();
  mocks.getDatabaseProject.mockReturnValue({
    name: "projects/proj1",
    dataClassificationConfigId: "classification-config",
  });
  mocks.getInstanceResource.mockReset();
  mocks.getInstanceResource.mockReturnValue({
    name: "instances/inst1",
    engine: Engine.POSTGRES,
  });
  mocks.hasProjectPermissionV2.mockReset();
  mocks.hasProjectPermissionV2.mockReturnValue(true);
  mocks.hasWorkspacePermissionV2.mockReset();
  mocks.hasWorkspacePermissionV2.mockReturnValue(true);

  vi.resetModules();
  ({ TableDetailDialog } = await import("./TableDetailDialog"));
});

describe("TableDetailDialog", () => {
  test("restores the legacy table detail sections for columns and indexes", async () => {
    const classificationConfig = {
      id: "classification-config",
      levels: [{ level: 1, title: "L1" }],
      classification: {
        PII: {
          id: "PII",
          title: "PII",
          level: 1,
        },
      },
    } as unknown as DataClassificationSetting_DataClassificationConfig;

    const { container, render, unmount } = renderIntoContainer(
      createElement(TableDetailDialog, {
        open: true,
        onOpenChange: vi.fn(),
        table: {
          database: makeDatabase(),
          editable: true,
          name: '"public"."audit"',
          schema: "public",
          tableName: "audit",
          classification: "PII",
          classificationConfig,
          columns: [
            {
              name: "id",
              semanticType: "",
              classification: "",
              type: "integer",
              defaultValue: "nextval('public.audit_id_seq'::regclass)",
              nullable: false,
              collation: "",
              comment: "",
            },
            {
              name: "query",
              semanticType: "SQL",
              classification: "PII",
              type: "text",
              defaultValue: "No default",
              nullable: true,
              collation: "en_US",
              comment: "query text",
            },
          ],
          rowCount: "0",
          dataSize: "8 KB",
          indexSize: "32 KB",
          collation: "en_US",
          indexes: [
            {
              name: "audit_pkey",
              expressions: ["id"],
              unique: true,
              visible: true,
              comment: "",
            },
          ],
          showColumnClassification: true,
          showColumnCollation: true,
          showCollation: true,
          showIndexComment: true,
          showIndexes: true,
          showIndexSize: true,
          showIndexVisible: false,
          showSemanticType: true,
        },
      })
    );

    render();
    await flush();

    expect(container.textContent).toContain("database.classification.self");
    expect(container.textContent).toContain("PII");
    expect(container.textContent).toContain(
      "settings.sensitive-data.semantic-types.table.semantic-type"
    );
    expect(container.textContent).toContain("common.default");
    expect(container.textContent).toContain("database.nullable");
    expect(container.textContent).toContain("db.collation");
    expect(container.textContent).toContain("database.indexes");
    expect(container.textContent).toContain("audit_pkey");
    expect(container.textContent).toContain("database.expression");
    expect(container.textContent).toContain("database.unique");
    expect(container.textContent).toContain(
      "nextval('public.audit_id_seq'::regclass)"
    );
    expect(container.textContent).toContain("query text");

    unmount();
  });

  test("restores edit actions for table classification and column semantic metadata", async () => {
    const classificationConfig = {
      id: "classification-config",
      levels: [
        { level: 1, title: "Level 1" },
        { level: 2, title: "Level 2" },
      ],
      classification: {
        PII: {
          id: "PII",
          title: "PII",
          level: 1,
        },
        CONFIDENTIAL: {
          id: "CONFIDENTIAL",
          title: "Confidential",
          level: 2,
        },
      },
    } as unknown as DataClassificationSetting_DataClassificationConfig;

    const { container, render, unmount } = renderIntoContainer(
      createElement(TableDetailDialog as unknown as ElementType, {
        open: true,
        onOpenChange: vi.fn(),
        table: {
          database: makeDatabase(),
          editable: true,
          name: '"public"."audit"',
          schema: "public",
          tableName: "audit",
          classification: "PII",
          classificationConfig,
          columns: [
            {
              name: "query",
              semanticType: "EMAIL",
              classification: "PII",
              type: "text",
              defaultValue: "No default",
              nullable: true,
            },
          ],
          rowCount: "0",
          dataSize: "8 KB",
          indexSize: "32 KB",
          indexes: [],
          showColumnClassification: true,
          showIndexes: false,
          showSemanticType: true,
        },
      })
    );

    render();
    await flush();

    const tableEditButton = container.querySelector(
      '[data-testid="table-classification-edit"]'
    );
    const semanticTypeEditButton = container.querySelector(
      '[data-testid="column-semantic-type-query-edit"]'
    );
    const semanticTypeRemoveButton = container.querySelector(
      '[data-testid="column-semantic-type-query-remove"]'
    );
    const classificationEditButton = container.querySelector(
      '[data-testid="column-classification-query-edit"]'
    );
    const classificationRemoveButton = container.querySelector(
      '[data-testid="column-classification-query-remove"]'
    );

    expect(tableEditButton).not.toBeNull();
    expect(semanticTypeEditButton).not.toBeNull();
    expect(semanticTypeRemoveButton).not.toBeNull();
    expect(classificationEditButton).not.toBeNull();
    expect(classificationRemoveButton).not.toBeNull();

    click(tableEditButton as HTMLElement);
    await flush();
    click(
      container.querySelector(
        '[data-testid="classification-option-CONFIDENTIAL"]'
      ) as HTMLElement
    );
    await flush();

    click(semanticTypeEditButton as HTMLElement);
    await flush();
    click(
      container.querySelector(
        '[data-testid="semantic-type-option-PHONE"]'
      ) as HTMLElement
    );
    await flush();

    click(classificationRemoveButton as HTMLElement);
    await flush();

    expect(mocks.updateTableCatalog).toHaveBeenCalledWith(
      expect.objectContaining({
        database: "instances/inst1/databases/db",
        schema: "public",
        table: "audit",
        tableCatalog: { classification: "CONFIDENTIAL" },
      })
    );
    expect(mocks.updateColumnCatalog).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({
        database: "instances/inst1/databases/db",
        schema: "public",
        table: "audit",
        column: "query",
        columnCatalog: { semanticType: "PHONE" },
      })
    );
    expect(mocks.updateColumnCatalog).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        database: "instances/inst1/databases/db",
        schema: "public",
        table: "audit",
        column: "query",
        columnCatalog: { classification: "" },
      })
    );

    unmount();
  });

  test("loads classification config from settings when the dialog prop is unavailable", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(TableDetailDialog as unknown as ElementType, {
        open: true,
        onOpenChange: vi.fn(),
        table: {
          database: makeDatabase(),
          editable: true,
          name: '"public"."audit"',
          schema: "public",
          tableName: "audit",
          classification: "PII",
          columns: [
            {
              name: "query",
              semanticType: "EMAIL",
              classification: "PII",
              type: "text",
              defaultValue: "No default",
              nullable: true,
            },
          ],
          rowCount: "0",
          dataSize: "8 KB",
          indexSize: "32 KB",
          indexes: [],
          showColumnClassification: true,
          showIndexes: false,
          showSemanticType: true,
        },
      })
    );

    render();
    await flush();

    expect(
      container.querySelector('[data-testid="table-classification-edit"]')
    ).not.toBeNull();
    expect(
      container.querySelector(
        '[data-testid="column-classification-query-edit"]'
      )
    ).not.toBeNull();

    click(
      container.querySelector(
        '[data-testid="column-classification-query-edit"]'
      ) as HTMLElement
    );
    await flush();
    click(
      container.querySelector(
        '[data-testid="classification-option-PII"]'
      ) as HTMLElement
    );
    await flush();

    expect(mocks.getOrFetchSettingByName).toHaveBeenCalledWith(
      Setting_SettingName.DATA_CLASSIFICATION,
      true
    );

    unmount();
  });
});
