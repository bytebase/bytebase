import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
} from "react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { DatabaseCatalog } from "@/types/proto-es/v1/database_catalog_service_pb";
import { ObjectSchema_Type } from "@/types/proto-es/v1/database_catalog_service_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const makeColumn = (
  overrides: Partial<{
    name: string;
    semanticType: string;
    classification: string;
  }> = {}
) => ({
  name: overrides.name ?? "email",
  semanticType: overrides.semanticType ?? "EMAIL",
  classification: overrides.classification ?? "PII",
});

const makeObjectSchema = () => ({
  type: ObjectSchema_Type.OBJECT,
  semanticType: "",
  kind: {
    case: "structKind" as const,
    value: {
      properties: {
        contact: {
          type: ObjectSchema_Type.OBJECT,
          semanticType: "",
          kind: {
            case: "structKind" as const,
            value: {
              properties: {
                email: {
                  type: ObjectSchema_Type.STRING,
                  semanticType: "EMAIL",
                },
                phone: {
                  type: ObjectSchema_Type.STRING,
                  semanticType: "PHONE",
                },
              },
            },
          },
        },
      },
    },
  },
});

const makeDatabaseCatalog = (): DatabaseCatalog =>
  ({
    schemas: [
      {
        name: "public",
        tables: [
          {
            name: "book",
            kind: {
              case: "columns",
              value: {
                columns: [makeColumn()],
              },
            },
            classification: "",
          },
          {
            name: "profile",
            kind: {
              case: "objectSchema",
              value: makeObjectSchema(),
            },
            classification: "",
          },
        ],
      },
    ],
  }) as DatabaseCatalog;

const makeSimpleCatalog = (): DatabaseCatalog =>
  ({
    schemas: [
      {
        name: "public",
        tables: [
          {
            name: "book",
            kind: {
              case: "columns",
              value: {
                columns: [makeColumn()],
              },
            },
            classification: "",
          },
        ],
      },
    ],
  }) as DatabaseCatalog;

const makeDatabase = (): Database =>
  ({
    name: "instances/inst1/databases/db1",
    project: "projects/proj1",
  }) as Database;

const click = (element: HTMLElement) => {
  act(() => {
    element.dispatchEvent(
      new MouseEvent("click", { bubbles: true, cancelable: true })
    );
  });
};

const changeSelect = async (select: HTMLSelectElement, value: string) => {
  act(() => {
    const descriptor = Object.getOwnPropertyDescriptor(
      HTMLSelectElement.prototype,
      "value"
    );
    descriptor?.set?.call(select, value);
    select.dispatchEvent(new Event("change", { bubbles: true }));
  });

  await flush();
};

const flush = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
};

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

const mocks = vi.hoisted(() => {
  const localStorage = {
    clear: vi.fn(),
    getItem: vi.fn(() => null),
    removeItem: vi.fn(),
    setItem: vi.fn(),
  };

  vi.stubGlobal("localStorage", localStorage);

  const hasProjectPermissionV2 = vi.fn(
    (_project?: unknown, _permission?: string) => true
  );

  return {
    localStorage,
    featureToRef: vi.fn(() => ({ value: true })),
    useVueState: vi.fn((getter: () => unknown) => getter()),
    useDatabaseCatalog: vi.fn(() => ({
      value: makeDatabaseCatalog(),
    })),
    usePolicyV1Store: vi.fn(() => ({
      getOrFetchPolicyByParentAndType: vi.fn(),
      upsertPolicy: vi.fn(),
    })),
    useDatabaseCatalogV1Store: vi.fn(() => ({
      updateDatabaseCatalog: vi.fn(),
    })),
    useSettingV1Store: vi.fn(() => ({
      getOrFetchSettingByName: vi.fn(),
      getProjectClassification: vi.fn(() => ({
        id: "classification-config",
        classification: {
          PII: {
            id: "PII",
            title: "PII",
          },
          CONFIDENTIAL: {
            id: "CONFIDENTIAL",
            title: "Confidential",
          },
        },
      })),
      getSettingByName: vi.fn(() => ({
        value: {
          value: {
            case: "semanticType",
            value: {
              types: [
                {
                  id: "EMAIL",
                  title: "Email",
                },
                {
                  id: "PHONE",
                  title: "Phone",
                },
              ],
            },
          },
        },
      })),
    })),
    pushNotification: vi.fn(),
    getDatabaseProject: vi.fn((database: { project: string }) => ({
      name: database.project,
      dataClassificationConfigId: "classification-config",
    })),
    getInstanceResource: vi.fn(() => ({
      name: "instances/inst1",
      engine: 1,
    })),
    instanceV1MaskingForNoSQL: vi.fn(() => false),
    hasProjectPermissionV2,
    autoDatabaseRoute: vi.fn(() => ({ name: "database" })),
    routerResolve: vi.fn(() => ({ fullPath: "/database/route" })),
    PermissionGuard: vi.fn(
      ({
        project,
        permissions,
        children,
      }: {
        project?: { name: string };
        permissions: string[];
        children: (props: { disabled: boolean }) => ReactNode;
      }) => {
        const disabled =
          !project ||
          !permissions.every((permission) =>
            hasProjectPermissionV2(project, permission)
          );
        return children({ disabled });
      }
    ),
    FeatureAttention: vi.fn(() => <div data-testid="feature-attention" />),
    FeatureBadge: vi.fn(() => <div data-testid="feature-badge" />),
    Button: (props: ButtonHTMLAttributes<HTMLButtonElement>) => (
      <button {...props} />
    ),
    Input: (props: InputHTMLAttributes<HTMLInputElement>) => (
      <input {...props} />
    ),
    Dialog: ({ open, children }: { open: boolean; children: ReactNode }) =>
      open ? <div data-testid="dialog-root">{children}</div> : null,
    DialogContent: ({ children }: { children: ReactNode }) => (
      <div>{children}</div>
    ),
    DialogTitle: ({ children }: { children: ReactNode }) => <h1>{children}</h1>,
    AlertDialog: ({
      open,
      children,
    }: {
      open: boolean;
      children: ReactNode;
    }) => (open ? <div data-testid="alert-dialog-root">{children}</div> : null),
    AlertDialogContent: ({ children }: { children: ReactNode }) => (
      <div>{children}</div>
    ),
    AlertDialogClose: ({ children }: { children: ReactNode }) => (
      <>{children}</>
    ),
    AlertDialogDescription: ({ children }: { children: ReactNode }) => (
      <p>{children}</p>
    ),
    AlertDialogFooter: ({ children }: { children: ReactNode }) => (
      <div>{children}</div>
    ),
    AlertDialogTitle: ({ children }: { children: ReactNode }) => (
      <h1>{children}</h1>
    ),
    Select: ({
      value,
      disabled,
      onValueChange,
      children,
    }: {
      value?: string;
      disabled?: boolean;
      onValueChange?: (value: string) => void;
      children: ReactNode;
    }) => (
      <select
        value={value}
        disabled={disabled}
        onChange={(event) => onValueChange?.(event.target.value)}
      >
        {children}
      </select>
    ),
    SelectTrigger: ({ children }: { children: ReactNode }) => <>{children}</>,
    SelectValue: ({ children }: { children: ReactNode }) => (
      <option value="">{children}</option>
    ),
    SelectContent: ({ children }: { children: ReactNode }) => <>{children}</>,
    SelectItem: ({
      value,
      children,
    }: {
      value: string;
      children: ReactNode;
    }) => <option value={value}>{children}</option>,
    Tooltip: ({
      content,
      children,
    }: {
      content: ReactNode;
      children: ReactNode;
    }) => (
      <span data-testid="tooltip" data-content={String(content)}>
        {children}
      </span>
    ),
    DatabaseResourceSelector: vi.fn(() => (
      <div data-testid="resource-selector" />
    )),
    ExprEditor: vi.fn(() => <div data-testid="expr-editor" />),
    AccountMultiSelect: vi.fn(() => <div data-testid="account-multi-select" />),
    ExpirationPicker: vi.fn(() => <div data-testid="expiration-picker" />),
    GrantAccessDialog: vi.fn(
      ({
        open,
        columnList,
      }: {
        open: boolean;
        columnList: Array<{ maskData: { column: string } }>;
      }) => (
        <div
          data-testid="grant-access-dialog"
          data-open={open ? "true" : "false"}
          data-column-count={columnList.length}
          data-columns={JSON.stringify(
            columnList.map(({ maskData }) => maskData.column)
          )}
        />
      )
    ),
    useTranslation: vi.fn(() => ({
      t: (key: string) => key,
    })),
  };
});

let DatabaseCatalogPanel: typeof import("./DatabaseCatalogPanel").DatabaseCatalogPanel;

vi.stubGlobal("localStorage", mocks.localStorage);

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
  useSettingV1Store: mocks.useSettingV1Store,
}));

vi.mock("@/utils", () => ({
  autoDatabaseRoute: mocks.autoDatabaseRoute,
  getDatabaseProject: mocks.getDatabaseProject,
  getInstanceResource: mocks.getInstanceResource,
  hasProjectPermissionV2: mocks.hasProjectPermissionV2,
  instanceV1MaskingForNoSQL: mocks.instanceV1MaskingForNoSQL,
}));

vi.mock("@/router", () => ({
  router: {
    resolve: mocks.routerResolve,
  },
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

vi.mock("@/react/components/ui/alert-dialog", () => ({
  AlertDialog: mocks.AlertDialog,
  AlertDialogClose: mocks.AlertDialogClose,
  AlertDialogContent: mocks.AlertDialogContent,
  AlertDialogDescription: mocks.AlertDialogDescription,
  AlertDialogFooter: mocks.AlertDialogFooter,
  AlertDialogTitle: mocks.AlertDialogTitle,
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: mocks.Button,
}));

vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: mocks.Dialog,
  DialogContent: mocks.DialogContent,
  DialogTitle: mocks.DialogTitle,
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: mocks.Input,
}));

vi.mock("@/react/components/ui/select", () => ({
  Select: mocks.Select,
  SelectContent: mocks.SelectContent,
  SelectItem: mocks.SelectItem,
  SelectTrigger: mocks.SelectTrigger,
  SelectValue: mocks.SelectValue,
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: mocks.Tooltip,
}));

vi.mock("../catalog/GrantAccessDialog", () => ({
  GrantAccessDialog: mocks.GrantAccessDialog,
}));

const getButton = (container: HTMLElement, label: string) =>
  Array.from(container.querySelectorAll("button")).find(
    (button) => button.textContent === label
  );

const getCheckboxes = (container: HTMLElement) =>
  Array.from(
    container.querySelectorAll('input[type="checkbox"]')
  ) as HTMLInputElement[];

beforeEach(async () => {
  vi.stubGlobal("localStorage", mocks.localStorage);
  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });
  mocks.useVueState.mockReset();
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());
  mocks.useDatabaseCatalog.mockReset();
  mocks.useDatabaseCatalog.mockReturnValue({
    value: makeDatabaseCatalog(),
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
  mocks.useSettingV1Store.mockReset();
  mocks.useSettingV1Store.mockReturnValue({
    getOrFetchSettingByName: vi.fn(),
    getProjectClassification: vi.fn(() => ({
      id: "classification-config",
      classification: {
        PII: {
          id: "PII",
          title: "PII",
        },
        CONFIDENTIAL: {
          id: "CONFIDENTIAL",
          title: "Confidential",
        },
      },
    })),
    getSettingByName: vi.fn(() => ({
      value: {
        value: {
          case: "semanticType",
          value: {
            types: [
              {
                id: "EMAIL",
                title: "Email",
              },
              {
                id: "PHONE",
                title: "Phone",
              },
            ],
          },
        },
      },
    })),
  });
  mocks.pushNotification.mockReset();
  mocks.getDatabaseProject.mockReset();
  mocks.getDatabaseProject.mockImplementation(
    (database: { project: string }) => ({
      name: database.project,
      dataClassificationConfigId: "classification-config",
    })
  );
  mocks.getInstanceResource.mockReset();
  mocks.getInstanceResource.mockReturnValue({
    name: "instances/inst1",
    engine: 1,
  });
  mocks.instanceV1MaskingForNoSQL.mockReset();
  mocks.instanceV1MaskingForNoSQL.mockReturnValue(false);
  mocks.featureToRef.mockReset();
  mocks.featureToRef.mockReturnValue({ value: true });
  mocks.hasProjectPermissionV2.mockReset();
  mocks.hasProjectPermissionV2.mockReturnValue(true);
  mocks.PermissionGuard.mockClear();
  mocks.FeatureAttention.mockClear();
  mocks.FeatureBadge.mockClear();
  mocks.GrantAccessDialog.mockClear();
  mocks.DatabaseResourceSelector.mockClear();
  mocks.ExprEditor.mockClear();
  mocks.AccountMultiSelect.mockClear();
  mocks.ExpirationPicker.mockClear();
  mocks.autoDatabaseRoute.mockClear();
  mocks.routerResolve.mockClear();
  vi.resetModules();
  ({ DatabaseCatalogPanel } = await import("./DatabaseCatalogPanel"));
});

describe("DatabaseCatalogPanel", () => {
  test("renders flattened relational and object-schema rows", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );

    render();
    await flush();

    expect(container.textContent).toContain("public.book");
    expect(container.textContent).toContain("email");
    expect(container.textContent).toContain("public.profile");
    expect(container.textContent).toContain("contact.email");

    unmount();
  });

  test("enables grant access after selecting a row", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );

    render();
    await flush();

    const grantAccessButton = getButton(
      container,
      "settings.sensitive-data.grant-access"
    );
    expect(grantAccessButton?.hasAttribute("disabled")).toBe(true);

    const checkboxes = getCheckboxes(container);
    expect(checkboxes).toHaveLength(4);
    click(checkboxes[1]);
    await flush();

    const enabledGrantAccessButton = getButton(
      container,
      "settings.sensitive-data.grant-access"
    );
    expect(enabledGrantAccessButton?.hasAttribute("disabled")).toBe(false);

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

    const checkboxes = getCheckboxes(container);
    expect(checkboxes).toHaveLength(4);
    click(checkboxes[1]);
    await flush();

    click(
      getButton(
        container,
        "settings.sensitive-data.grant-access"
      ) as HTMLElement
    );
    await flush();

    expect(
      container.querySelector('[data-testid="dialog-root"]')
    ).not.toBeNull();
    expect(container.textContent).toContain("common.warning");
    expect(mocks.GrantAccessDialog).toHaveBeenLastCalledWith(
      expect.objectContaining({
        open: false,
      }),
      undefined
    );
    expect(
      container
        .querySelector('[data-testid="grant-access-dialog"]')
        ?.getAttribute("data-open")
    ).toBe("false");

    unmount();
  });

  test("passes permission inputs to the grant-access guard", async () => {
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

  test("clears a selected row from checked selection when deleted", async () => {
    const updateDatabaseCatalog = vi.fn().mockResolvedValue(undefined);
    mocks.useDatabaseCatalogV1Store.mockReturnValue({
      updateDatabaseCatalog,
    });

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );

    render();
    await flush();

    const checkboxes = getCheckboxes(container);
    expect(checkboxes).toHaveLength(4);
    click(checkboxes[1]);
    await flush();

    expect(
      getButton(
        container,
        "settings.sensitive-data.grant-access"
      )?.hasAttribute("disabled")
    ).toBe(false);

    const deleteButton = container.querySelector(
      'button[aria-label="settings.sensitive-data.remove-sensitive-column-tips"]'
    );
    expect(deleteButton).not.toBeNull();
    click(deleteButton as HTMLElement);
    await flush();

    const confirmButton = getButton(container, "common.delete");
    expect(confirmButton).not.toBeNull();
    click(confirmButton as HTMLElement);
    await flush();

    expect(updateDatabaseCatalog).toHaveBeenCalledTimes(1);
    expect(
      getButton(
        container,
        "settings.sensitive-data.grant-access"
      )?.hasAttribute("disabled")
    ).toBe(true);

    unmount();
  });

  test("updates semantic type and classification independently", async () => {
    const updateDatabaseCatalog = vi.fn().mockResolvedValue(undefined);
    const getOrFetchSettingByName = vi.fn();
    mocks.useDatabaseCatalogV1Store.mockReturnValue({
      updateDatabaseCatalog,
    });
    mocks.useSettingV1Store.mockReturnValue({
      getOrFetchSettingByName,
      getProjectClassification: vi.fn(() => ({
        id: "classification-config",
        classification: {
          PII: {
            id: "PII",
            title: "PII",
          },
          CONFIDENTIAL: {
            id: "CONFIDENTIAL",
            title: "Confidential",
          },
        },
      })),
      getSettingByName: vi.fn(() => ({
        value: {
          value: {
            case: "semanticType",
            value: {
              types: [
                {
                  id: "EMAIL",
                  title: "Email",
                },
                {
                  id: "PHONE",
                  title: "Phone",
                },
              ],
            },
          },
        },
      })),
    });
    const catalog = makeSimpleCatalog();
    mocks.useDatabaseCatalog.mockReturnValue({
      value: catalog,
    });

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );

    render();
    await flush();

    expect(getOrFetchSettingByName).toHaveBeenCalledWith(
      Setting_SettingName.SEMANTIC_TYPES,
      true
    );
    expect(getOrFetchSettingByName).toHaveBeenCalledWith(
      Setting_SettingName.DATA_CLASSIFICATION,
      true
    );

    const selects = Array.from(container.querySelectorAll("select"));
    expect(selects).toHaveLength(2);

    await changeSelect(selects[0] as HTMLSelectElement, "");
    expect(updateDatabaseCatalog).toHaveBeenCalledTimes(1);
    const table = catalog.schemas[0]?.tables[0];
    expect(table?.kind.case).toBe("columns");
    if (!table || table.kind.case !== "columns") {
      throw new Error("expected a column-backed table catalog");
    }
    expect(table.kind.value.columns[0]?.semanticType).toBe("");
    expect(table.kind.value.columns[0]?.classification).toBe("PII");

    await changeSelect(selects[1] as HTMLSelectElement, "");
    expect(updateDatabaseCatalog).toHaveBeenCalledTimes(2);
    expect(table.kind.value.columns[0]?.classification).toBe("");

    expect(mocks.pushNotification).toHaveBeenCalledWith(
      expect.objectContaining({
        style: "SUCCESS",
        title: "common.updated",
      })
    );

    unmount();
  });

  test("renders an empty state when catalog data is not loaded", async () => {
    mocks.useDatabaseCatalog.mockReturnValue({
      value: undefined as unknown as DatabaseCatalog,
    });

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );

    render();
    await flush();

    expect(container.textContent).toContain("common.no-data");
    expect(container.querySelector('input[type="checkbox"]')).toBeNull();

    unmount();
  });

  test("disables row selection in NoSQL mode and keeps object-schema rows", async () => {
    mocks.instanceV1MaskingForNoSQL.mockReturnValue(true);

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );

    render();
    await flush();

    expect(container.querySelector('input[type="checkbox"]')).toBeNull();
    expect(container.textContent).toContain("public.profile");
    expect(container.textContent).toContain("contact.email");

    unmount();
  });

  test("guards column-scoped grant-access selections in the dialog", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );

    render();
    await flush();

    const checkboxes = getCheckboxes(container);
    expect(checkboxes).toHaveLength(4);
    click(checkboxes[0]);
    await flush();

    click(
      getButton(
        container,
        "settings.sensitive-data.grant-access"
      ) as HTMLElement
    );
    await flush();

    const grantAccessDialog = container.querySelector(
      '[data-testid="grant-access-dialog"]'
    );
    expect(grantAccessDialog?.getAttribute("data-open")).toBe("true");
    expect(grantAccessDialog?.getAttribute("data-column-count")).toBe("3");
    expect(grantAccessDialog?.getAttribute("data-columns")).toContain("email");
    expect(grantAccessDialog?.getAttribute("data-columns")).toContain(
      "contact.email"
    );
    expect(grantAccessDialog?.getAttribute("data-columns")).toContain(
      "contact.phone"
    );

    unmount();
  });

  test("hides operation controls when catalog update permission is missing", async () => {
    mocks.hasProjectPermissionV2.mockImplementation(
      (_project?: unknown, permission?: string) =>
        permission !== "bb.databaseCatalogs.update"
    );

    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseCatalogPanel, {
        database: makeDatabase(),
      })
    );

    render();
    await flush();

    const select = container.querySelector(
      "select"
    ) as HTMLSelectElement | null;
    expect(select).not.toBeNull();
    expect(select?.disabled).toBe(true);
    expect(
      container.querySelector(
        'button[aria-label="settings.sensitive-data.remove-sensitive-column-tips"]'
      )
    ).toBeNull();

    unmount();
  });
});
