import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  localStorage: {
    clear: vi.fn(),
    getItem: vi.fn(() => null),
    removeItem: vi.fn(),
    setItem: vi.fn(),
  },
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
  })),
  useVueState: vi.fn((getter: () => unknown) => getter()),
  featureToRef: vi.fn(() => ({ value: true })),
  useDatabaseCatalog: vi.fn(() => ({
    value: {
      schemas: [
        {
          name: "public",
          tables: [
            {
              name: "book",
              kind: {
                case: "columns",
                value: {
                  columns: [
                    {
                      name: "email",
                      semanticType: "EMAIL",
                      classification: "",
                    },
                  ],
                },
              },
              classification: "",
            },
          ],
        },
      ],
    } as unknown,
  })),
  usePolicyV1Store: vi.fn(() => ({
    getOrFetchPolicyByParentAndType: vi.fn(),
    upsertPolicy: vi.fn(),
  })),
  useDatabaseCatalogV1Store: vi.fn(() => ({
    updateDatabaseCatalog: vi.fn(),
  })),
  pushNotification: vi.fn(),
  hasProjectPermissionV2: vi.fn(
    (_project?: unknown, _permission?: string) => true
  ),
  getDatabaseProject: vi.fn((database: { project: string }) => ({
    name: database.project,
  })),
  getInstanceResource: vi.fn(() => ({ name: "instances/inst1", engine: 1 })),
  instanceV1MaskingForNoSQL: vi.fn(() => false),
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
  pushNotification: mocks.pushNotification,
  useDatabaseCatalog: mocks.useDatabaseCatalog,
  useDatabaseCatalogV1Store: mocks.useDatabaseCatalogV1Store,
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

vi.mock("@/react/components/AccountMultiSelect", () => ({
  AccountMultiSelect: ({
    value,
    onChange,
  }: {
    value: string[];
    onChange: (value: string[]) => void;
  }) => (
    <button
      type="button"
      data-testid="member-select"
      onClick={() => onChange([...value, "user:test@example.com"])}
    >
      member-select
    </button>
  ),
}));

vi.mock("@/react/components/DatabaseResourceSelector", () => ({
  DatabaseResourceSelector: ({
    value,
    onChange,
  }: {
    value: unknown[];
    onChange: (value: unknown[]) => void;
  }) => (
    <button
      type="button"
      data-testid="resource-select"
      onClick={() => onChange(value)}
    >
      resource-select
    </button>
  ),
}));

vi.mock("@/react/components/ui/expiration-picker", () => ({
  ExpirationPicker: ({
    value,
    onChange,
  }: {
    value: string | undefined;
    onChange: (value: string | undefined) => void;
  }) => (
    <input
      data-testid="expiration-picker"
      value={value ?? ""}
      onChange={(event) => onChange(event.target.value || undefined)}
    />
  ),
}));

vi.mock("../legacy/SensitiveColumnTableBridge", () => ({
  SensitiveColumnTableBridge: () => null,
}));

vi.mock("../legacy/GrantAccessDrawerBridge", () => ({
  GrantAccessDrawerBridge: () => null,
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
  vi.stubGlobal("localStorage", mocks.localStorage);
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
    value: {
      schemas: [
        {
          name: "public",
          tables: [
            {
              name: "book",
              kind: {
                case: "columns",
                value: {
                  columns: [
                    {
                      name: "email",
                      semanticType: "EMAIL",
                      classification: "",
                    },
                  ],
                },
              },
              classification: "",
            },
          ],
        },
      ],
    } as unknown,
  });
  mocks.usePolicyV1Store.mockReset();
  mocks.usePolicyV1Store.mockReturnValue({
    getOrFetchPolicyByParentAndType: vi.fn(),
    upsertPolicy: vi.fn(),
  });
  mocks.useDatabaseCatalogV1Store.mockReset();
  mocks.useDatabaseCatalogV1Store.mockReturnValue({
    updateDatabaseCatalog: vi.fn(),
  });
  mocks.pushNotification.mockReset();
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
  mocks.FeatureAttention.mockClear();
  mocks.FeatureBadge.mockClear();
  mocks.PermissionGuard.mockClear();

  vi.resetModules();
  ({ DatabaseCatalogPanel } = await import("./DatabaseCatalogPanel"));
});

describe("DatabaseCatalogPanel", () => {
  test("renders catalog rows and opens the react grant-access dialog after selection", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );
    render();
    await flush();

    expect(container.textContent).toContain("public.book");
    expect(container.textContent).toContain("email");

    const grantAccessButton = Array.from(
      container.querySelectorAll("button")
    ).find(
      (button) => button.textContent === "settings.sensitive-data.grant-access"
    );
    expect(grantAccessButton).toBeDefined();
    expect(grantAccessButton?.hasAttribute("disabled")).toBe(true);

    const checkbox = container.querySelector('input[type="checkbox"]');
    expect(checkbox).not.toBeNull();
    click(checkbox as HTMLElement);
    await flush();

    const enabledGrantAccessButton = Array.from(
      container.querySelectorAll("button")
    ).find(
      (button) => button.textContent === "settings.sensitive-data.grant-access"
    );
    expect(enabledGrantAccessButton?.hasAttribute("disabled")).toBe(false);

    click(enabledGrantAccessButton as HTMLElement);
    await flush();

    expect(
      container.querySelector('[data-testid="dialog-root"]')
    ).not.toBeNull();
    expect(container.textContent).toContain(
      "settings.sensitive-data.grant-access"
    );

    unmount();
  });

  test("opens the feature dialog instead of grant access when masking feature is missing", async () => {
    mocks.featureToRef.mockReturnValue({ value: false });

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );
    render();
    await flush();

    const checkbox = container.querySelector('input[type="checkbox"]');
    expect(checkbox).not.toBeNull();
    click(checkbox as HTMLElement);
    await flush();

    const grantAccessButton = Array.from(
      container.querySelectorAll("button")
    ).find(
      (button) => button.textContent === "settings.sensitive-data.grant-access"
    );
    expect(grantAccessButton?.hasAttribute("disabled")).toBe(false);

    click(grantAccessButton as HTMLElement);
    await flush();

    expect(
      container.querySelector('[data-testid="dialog-root"]')
    ).not.toBeNull();
    expect(mocks.FeatureBadge).toHaveBeenCalledWith(
      expect.objectContaining({
        clickable: false,
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
    mocks.useDatabaseCatalog.mockReturnValue({ value: undefined as unknown });

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );
    render();
    await flush();

    expect(container.textContent).not.toContain("public.book");
    expect(container.textContent).not.toContain("email");

    unmount();
  });
});
