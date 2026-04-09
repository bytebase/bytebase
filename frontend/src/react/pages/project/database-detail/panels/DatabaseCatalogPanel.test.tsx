import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
  })),
  useVueState: vi.fn((getter: () => unknown) => getter()),
  featureToRef: vi.fn(() => ({ value: true })),
  useDatabaseCatalog: vi.fn(() => ({
    value: { schemas: [] } as { schemas: unknown[] } | undefined,
  })),
  usePolicyV1Store: vi.fn(() => ({
    getOrFetchPolicyByParentAndType: vi.fn(),
    upsertPolicy: vi.fn(),
  })),
  hasProjectPermissionV2: vi.fn(
    (_project?: unknown, _permission?: string) => true
  ),
  getDatabaseProject: vi.fn((database: { project: string }) => ({
    name: database.project,
  })),
  getInstanceResource: vi.fn(() => ({ name: "instances/inst1", engine: 1 })),
  instanceV1MaskingForNoSQL: vi.fn(() => false),
  SensitiveColumnTableBridge: vi.fn(
    ({
      onCheckedColumnListChange,
    }: {
      onCheckedColumnListChange: (list: unknown[]) => void;
    }) => (
      <div data-testid="sensitive-column-table-bridge">
        <button
          type="button"
          data-testid="select-sensitive-row"
          onClick={() =>
            onCheckedColumnListChange([
              {
                schema: "public",
                table: "book",
                column: "email",
                semanticTypeId: "EMAIL",
                classificationId: "",
                target: {},
              },
            ])
          }
        >
          select
        </button>
      </div>
    )
  ),
  GrantAccessDrawerBridge: vi.fn(() => (
    <div data-testid="grant-access-drawer" />
  )),
  FeatureAttention: vi.fn(() => <div data-testid="feature-attention" />),
  FeatureBadge: vi.fn(() => <div data-testid="feature-badge" />),
  PermissionGuard: vi.fn(
    ({ children }: { children: React.ReactNode }) => children
  ),
}));

let DatabaseCatalogPanel: typeof import("./DatabaseCatalogPanel").DatabaseCatalogPanel;

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  featureToRef: mocks.featureToRef,
  useDatabaseCatalog: mocks.useDatabaseCatalog,
  usePolicyV1Store: mocks.usePolicyV1Store,
}));

vi.mock("@/utils", () => ({
  getDatabaseProject: mocks.getDatabaseProject,
  getInstanceResource: mocks.getInstanceResource,
  hasProjectPermissionV2: mocks.hasProjectPermissionV2,
  instanceV1MaskingForNoSQL: mocks.instanceV1MaskingForNoSQL,
}));

vi.mock("@/react/components/FeatureAttention", () => ({
  FeatureAttention: mocks.FeatureAttention,
}));

vi.mock("@/react/components/FeatureBadge", () => ({
  FeatureBadge: mocks.FeatureBadge,
}));

vi.mock("@/react/components/PermissionGuard", () => ({
  PermissionGuard: mocks.PermissionGuard,
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: (props: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button {...props} />
  ),
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => (
    <input {...props} />
  ),
}));

vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open: boolean; children: React.ReactNode }) =>
    open ? <div data-testid="dialog-root">{children}</div> : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DialogTitle: ({ children }: { children: React.ReactNode }) => (
    <h1>{children}</h1>
  ),
}));

vi.mock("../legacy/SensitiveColumnTableBridge", () => ({
  SensitiveColumnTableBridge: mocks.SensitiveColumnTableBridge,
}));

vi.mock("../legacy/GrantAccessDrawerBridge", () => ({
  GrantAccessDrawerBridge: mocks.GrantAccessDrawerBridge,
}));

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  return {
    container,
    render: (nextElement = element) => {
      act(() => {
        root.render(nextElement);
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
    name: "instances/inst1/databases/db1",
    project: "projects/proj1",
  }) as Database;

beforeEach(async () => {
  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });
  mocks.useVueState.mockReset();
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());
  mocks.featureToRef.mockReset();
  mocks.featureToRef.mockReturnValue({ value: true });
  mocks.useDatabaseCatalog.mockReset();
  mocks.useDatabaseCatalog.mockReturnValue({
    value: { schemas: [] } as { schemas: unknown[] } | undefined,
  });
  mocks.usePolicyV1Store.mockReset();
  mocks.usePolicyV1Store.mockReturnValue({
    getOrFetchPolicyByParentAndType: vi.fn(),
    upsertPolicy: vi.fn(),
  });
  mocks.hasProjectPermissionV2.mockReset();
  mocks.hasProjectPermissionV2.mockReturnValue(true);
  mocks.getDatabaseProject.mockReset();
  mocks.getDatabaseProject.mockImplementation(
    (database: { project: string }) => ({
      name: database.project,
    })
  );
  mocks.getInstanceResource.mockReset();
  mocks.getInstanceResource.mockReturnValue({
    name: "instances/inst1",
    engine: 1,
  });
  mocks.instanceV1MaskingForNoSQL.mockReset();
  mocks.instanceV1MaskingForNoSQL.mockReturnValue(false);
  mocks.SensitiveColumnTableBridge.mockClear();
  mocks.GrantAccessDrawerBridge.mockClear();
  mocks.FeatureAttention.mockClear();
  mocks.FeatureBadge.mockClear();
  mocks.PermissionGuard.mockClear();

  vi.resetModules();
  ({ DatabaseCatalogPanel } = await import("./DatabaseCatalogPanel"));
});

describe("DatabaseCatalogPanel", () => {
  test("keeps grant access disabled until at least one row is selected", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );
    render();
    await flush();

    const grantAccessButton = Array.from(
      container.querySelectorAll("button")
    ).find(
      (button) => button.textContent === "settings.sensitive-data.grant-access"
    );
    expect(grantAccessButton).toBeDefined();
    expect(grantAccessButton?.hasAttribute("disabled")).toBe(true);

    const selectRowButton = container.querySelector(
      '[data-testid="select-sensitive-row"]'
    );
    expect(selectRowButton).not.toBeNull();
    click(selectRowButton as HTMLElement);
    await flush();

    const updatedGrantAccessButton = Array.from(
      container.querySelectorAll("button")
    ).find(
      (button) => button.textContent === "settings.sensitive-data.grant-access"
    );
    expect(updatedGrantAccessButton?.hasAttribute("disabled")).toBe(false);

    unmount();
  });

  test("passes the database instance into feature attention", async () => {
    const { render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );
    render();
    await flush();

    expect(mocks.FeatureAttention).toHaveBeenCalledWith(
      expect.objectContaining({
        feature: expect.any(Number),
        instance: expect.objectContaining({
          name: "instances/inst1",
        }),
      }),
      undefined
    );

    unmount();
  });

  test("passes showOperation=false when catalog update permission is missing", async () => {
    mocks.hasProjectPermissionV2.mockImplementation(
      (_project?: unknown, permission?: string) =>
        permission !== "bb.databaseCatalogs.update"
    );

    const { render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );
    render();
    await flush();

    expect(mocks.SensitiveColumnTableBridge).toHaveBeenCalledWith(
      expect.objectContaining({
        showOperation: false,
      }),
      undefined
    );

    unmount();
  });

  test("passes the project-scoped grant-access guard inputs", async () => {
    const { render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );
    render();
    await flush();

    expect(mocks.PermissionGuard).toHaveBeenCalledWith(
      expect.objectContaining({
        permissions: [
          "bb.policies.createMaskingExemptionPolicy",
          "bb.policies.updateMaskingExemptionPolicy",
          "bb.databaseCatalogs.get",
        ],
        project: expect.objectContaining({
          name: "projects/proj1",
        }),
      }),
      undefined
    );

    unmount();
  });

  test("handles undefined catalog state before data is loaded", async () => {
    mocks.useDatabaseCatalog.mockReturnValue({ value: undefined });

    const { render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );
    render();
    await flush();

    expect(mocks.SensitiveColumnTableBridge).toHaveBeenCalledWith(
      expect.objectContaining({
        columnList: [],
      }),
      undefined
    );

    unmount();
  });

  test("renders feature badge as non-clickable icon when feature is missing", async () => {
    mocks.featureToRef.mockReturnValue({ value: false });

    const { render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );
    render();
    await flush();

    expect(mocks.FeatureBadge).toHaveBeenCalledWith(
      expect.objectContaining({
        clickable: false,
      }),
      undefined
    );

    unmount();
  });
});
