import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  useVueState: vi.fn<(getter: () => unknown) => unknown>(),
  useProjectV1Store: vi.fn(),
  useRoleStore: vi.fn(),
  useSQLEditorStore: vi.fn(),
  useSubscriptionV1Store: vi.fn(),
  hasFeature: vi.fn(() => true),
  parseStringToResource: vi.fn((s: string) => ({
    databaseFullName: s,
    databaseName: s,
  })),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useProjectV1Store: mocks.useProjectV1Store,
  useRoleStore: mocks.useRoleStore,
  useSQLEditorStore: mocks.useSQLEditorStore,
  useSubscriptionV1Store: mocks.useSubscriptionV1Store,
  hasFeature: mocks.hasFeature,
}));

vi.mock("@/components/RoleGrantPanel/DatabaseResourceForm/common", () => ({
  parseStringToResource: mocks.parseStringToResource,
}));

vi.mock("@/types", () => ({
  PRESET_ROLES: ["roles/sqlEditorReadUser", "roles/sqlEditorUser"],
  PresetRoleType: {
    SQL_EDITOR_READ_USER: "roles/sqlEditorReadUser",
    SQL_EDITOR_USER: "roles/sqlEditorUser",
  },
}));

vi.mock("@/types/proto-es/v1/subscription_service_pb", () => ({
  PlanFeature: {
    FEATURE_CUSTOM_ROLES: 47,
    FEATURE_JIT: 5,
    FEATURE_REQUEST_ROLE_WORKFLOW: 6,
  },
}));

vi.mock("@/react/components/FeatureBadge", () => ({
  FeatureBadge: () => <span data-testid="feature-badge" />,
}));

vi.mock("@/react/components/PermissionGuard", () => ({
  PermissionGuard: ({
    children,
  }: {
    children: (props: { disabled: boolean }) => React.ReactNode;
  }) => <>{children({ disabled: false })}</>,
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    disabled,
    ...rest
  }: {
    children: React.ReactNode;
    onClick?: (e: React.MouseEvent) => void;
    disabled?: boolean;
    [key: string]: unknown;
  }) => (
    <button disabled={disabled} onClick={onClick as () => void} {...rest}>
      {children}
    </button>
  ),
}));

vi.mock("./RoleGrantPanel", () => ({
  RoleGrantPanel: ({
    onClose,
    role,
  }: {
    onClose: () => void;
    role: string;
  }) => (
    <div data-testid="role-grant-panel" data-role={role}>
      <button data-close-btn onClick={onClose}>
        Close
      </button>
    </div>
  ),
}));

vi.mock("./AccessGrantRequestDrawer", () => ({
  AccessGrantRequestDrawer: ({ onClose }: { onClose: () => void }) => (
    <div data-testid="access-grant-drawer">
      <button data-close-btn onClick={onClose}>
        Close
      </button>
    </div>
  ),
}));

// Stub ResizeObserver
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

let RequestQueryButton: typeof import("./RequestQueryButton").RequestQueryButton;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

const makePermissionDeniedDetail = (overrides?: {
  allowJustInTimeAccess?: boolean;
  requiredPermissions?: string[];
  resources?: string[];
}) =>
  ({
    requiredPermissions: overrides?.requiredPermissions ?? ["bb.sql.select"],
    resources: overrides?.resources ?? ["instances/inst1/databases/db1"],
  }) as unknown as import("@/types/proto-es/v1/common_pb").PermissionDeniedDetail;

const setupDefaultMocks = (allowJIT = false, allowRequestRole = true) => {
  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });
  mocks.useProjectV1Store.mockReturnValue({
    getProjectByName: vi.fn(() => ({
      name: "projects/proj1",
      allowJustInTimeAccess: allowJIT,
      allowRequestRole,
    })),
  });
  mocks.useRoleStore.mockReturnValue({
    roleList: [
      {
        name: "roles/sqlEditorUser",
        permissions: ["bb.sql.select", "bb.sql.dml", "bb.sql.explain"],
      },
      {
        name: "roles/queryOnly",
        permissions: ["bb.sql.select"],
      },
      {
        name: "roles/sqlEditorReadUser",
        permissions: ["bb.sql.select", "bb.sql.explain"],
      },
    ],
  });
  mocks.useSQLEditorStore.mockReturnValue({ project: "projects/proj1" });
  mocks.useSubscriptionV1Store.mockReturnValue({
    hasInstanceFeature: vi.fn(() => false),
  });
  mocks.hasFeature.mockReturnValue(true);
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());
};

beforeEach(async () => {
  vi.clearAllMocks();
  setupDefaultMocks();
  ({ RequestQueryButton } = await import("./RequestQueryButton"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("RequestQueryButton", () => {
  test("renders nothing when allowRequestRole is false", () => {
    setupDefaultMocks(false, false);
    const { container, render, unmount } = renderIntoContainer(
      <RequestQueryButton
        text={false}
        permissionDeniedDetail={makePermissionDeniedDetail()}
      />
    );
    render();
    // Should not show a button
    expect(container.querySelector("button")).toBeNull();
    unmount();
  });

  test("renders request-query button in non-JIT mode", () => {
    setupDefaultMocks(false, true);
    const { container, render, unmount } = renderIntoContainer(
      <RequestQueryButton
        text={false}
        permissionDeniedDetail={makePermissionDeniedDetail({
          requiredPermissions: ["bb.sql.select"],
        })}
      />
    );
    render();
    expect(container.textContent).toContain("sql-editor.request-query");
    unmount();
  });

  test("renders request-jit button in JIT mode", () => {
    setupDefaultMocks(true, true);
    const { container, render, unmount } = renderIntoContainer(
      <RequestQueryButton
        text={false}
        permissionDeniedDetail={makePermissionDeniedDetail({
          requiredPermissions: ["bb.sql.select"],
        })}
      />
    );
    render();
    expect(container.textContent).toContain("sql-editor.request-jit");
    unmount();
  });

  test("click in non-JIT mode opens RoleGrantPanel", async () => {
    setupDefaultMocks(false, true);
    const { container, render, unmount } = renderIntoContainer(
      <RequestQueryButton
        text={false}
        permissionDeniedDetail={makePermissionDeniedDetail({
          requiredPermissions: ["bb.issues.create"],
        })}
      />
    );
    render();

    expect(
      container.querySelector("[data-testid='role-grant-panel']")
    ).toBeNull();

    const btn = container.querySelector("button") as HTMLButtonElement;
    await act(async () => {
      btn.click();
    });

    expect(
      container.querySelector("[data-testid='role-grant-panel']")
    ).not.toBeNull();
    unmount();
  });

  test("non-JIT role request defaults to least-permission SQL select role", async () => {
    setupDefaultMocks(false, true);
    const { container, render, unmount } = renderIntoContainer(
      <RequestQueryButton
        text={false}
        permissionDeniedDetail={makePermissionDeniedDetail({
          requiredPermissions: ["bb.sql.select"],
        })}
      />
    );
    render();

    const btn = container.querySelector("button") as HTMLButtonElement;
    await act(async () => {
      btn.click();
    });

    expect(
      container
        .querySelector("[data-testid='role-grant-panel']")
        ?.getAttribute("data-role")
    ).toBe("roles/sqlEditorReadUser");
    unmount();
  });

  test("non-JIT role request defaults to role covering denied permission", async () => {
    setupDefaultMocks(false, true);
    const { container, render, unmount } = renderIntoContainer(
      <RequestQueryButton
        text={false}
        permissionDeniedDetail={makePermissionDeniedDetail({
          requiredPermissions: ["bb.sql.dml"],
        })}
      />
    );
    render();

    const btn = container.querySelector("button") as HTMLButtonElement;
    await act(async () => {
      btn.click();
    });

    expect(
      container
        .querySelector("[data-testid='role-grant-panel']")
        ?.getAttribute("data-role")
    ).toBe("roles/sqlEditorUser");
    unmount();
  });

  test("non-JIT role request ignores custom roles when feature is disabled", async () => {
    setupDefaultMocks(false, true);
    const { container, render, unmount } = renderIntoContainer(
      <RequestQueryButton
        text={false}
        permissionDeniedDetail={makePermissionDeniedDetail({
          requiredPermissions: ["bb.sql.select"],
        })}
      />
    );
    render();

    const btn = container.querySelector("button") as HTMLButtonElement;
    await act(async () => {
      btn.click();
    });

    expect(
      container
        .querySelector("[data-testid='role-grant-panel']")
        ?.getAttribute("data-role")
    ).toBe("roles/sqlEditorReadUser");
    unmount();
  });

  test("non-JIT role request can default to custom role when feature is enabled", async () => {
    setupDefaultMocks(false, true);
    mocks.useSubscriptionV1Store.mockReturnValue({
      hasInstanceFeature: vi.fn(() => true),
    });
    const { container, render, unmount } = renderIntoContainer(
      <RequestQueryButton
        text={false}
        permissionDeniedDetail={makePermissionDeniedDetail({
          requiredPermissions: ["bb.sql.select"],
        })}
      />
    );
    render();

    const btn = container.querySelector("button") as HTMLButtonElement;
    await act(async () => {
      btn.click();
    });

    expect(
      container
        .querySelector("[data-testid='role-grant-panel']")
        ?.getAttribute("data-role")
    ).toBe("roles/queryOnly");
    unmount();
  });

  test("non-JIT role request works without Array.prototype.toSorted", async () => {
    setupDefaultMocks(false, true);
    const descriptor = Object.getOwnPropertyDescriptor(
      Array.prototype,
      "toSorted"
    );
    Object.defineProperty(Array.prototype, "toSorted", {
      configurable: true,
      value: undefined,
    });

    try {
      const { container, render, unmount } = renderIntoContainer(
        <RequestQueryButton
          text={false}
          permissionDeniedDetail={makePermissionDeniedDetail({
            requiredPermissions: ["bb.sql.select"],
          })}
        />
      );
      render();

      const btn = container.querySelector("button") as HTMLButtonElement;
      await act(async () => {
        btn.click();
      });

      expect(
        container
          .querySelector("[data-testid='role-grant-panel']")
          ?.getAttribute("data-role")
      ).toBe("roles/sqlEditorReadUser");
      unmount();
    } finally {
      if (descriptor) {
        Object.defineProperty(Array.prototype, "toSorted", descriptor);
      }
    }
  });

  test("click in JIT mode opens AccessGrantRequestDrawer", async () => {
    setupDefaultMocks(true, true);
    const { container, render, unmount } = renderIntoContainer(
      <RequestQueryButton
        text={false}
        statement="SELECT 1"
        permissionDeniedDetail={makePermissionDeniedDetail({
          requiredPermissions: ["bb.sql.select"],
        })}
      />
    );
    render();

    expect(
      container.querySelector("[data-testid='access-grant-drawer']")
    ).toBeNull();

    const btn = container.querySelector("button") as HTMLButtonElement;
    await act(async () => {
      btn.click();
    });

    expect(
      container.querySelector("[data-testid='access-grant-drawer']")
    ).not.toBeNull();
    unmount();
  });
});
