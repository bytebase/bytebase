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
  useCurrentUserV1: vi.fn(),
  useProjectV1Store: vi.fn(),
  useRoleStore: vi.fn(),
  pushNotification: vi.fn(),
  issueServiceClientConnect: {
    createIssue: vi.fn().mockResolvedValue({ name: "projects/proj1/issues/1" }),
  },
  roleHasDatabaseLimitation: vi.fn(() => true),
}));

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: mocks.useVueState,
}));

vi.mock("@/store", () => ({
  useCurrentUserV1: mocks.useCurrentUserV1,
  useProjectV1Store: mocks.useProjectV1Store,
  useRoleStore: mocks.useRoleStore,
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/connect", () => ({
  issueServiceClientConnect: mocks.issueServiceClientConnect,
}));

vi.mock("@/router", () => ({
  router: {
    resolve: vi.fn(() => ({ fullPath: "/projects/proj1/issues/1" })),
  },
}));

vi.mock("@/router/dashboard/projectV1", () => ({
  PROJECT_V1_ROUTE_ISSUE_DETAIL: "project.issue-detail",
}));

vi.mock("@/utils", () => ({
  displayRoleTitle: vi.fn((r: string) => r),
  extractIssueUID: vi.fn(() => "1"),
  extractProjectResourceName: vi.fn(() => "proj1"),
  formatIssueTitle: vi.fn((title: string) => title),
}));

vi.mock("@/utils/issue/cel", () => ({
  buildConditionExpr: vi.fn(() => ({ expression: "", description: "" })),
}));

vi.mock("@/components/ProjectMember/utils", () => ({
  roleHasDatabaseLimitation: mocks.roleHasDatabaseLimitation,
}));

vi.mock("@/react/components/ui/sheet", () => ({
  Sheet: ({ children, open }: { children: React.ReactNode; open?: boolean }) =>
    open !== false ? <div data-testid="sheet">{children}</div> : null,
  SheetContent: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="sheet-content">{children}</div>
  ),
  SheetHeader: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  SheetTitle: ({ children }: { children: React.ReactNode }) => (
    <h2 data-testid="sheet-title">{children}</h2>
  ),
  SheetBody: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="sheet-body">{children}</div>
  ),
  SheetFooter: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="sheet-footer">{children}</div>
  ),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    disabled,
  }: {
    children: React.ReactNode;
    onClick?: () => void;
    disabled?: boolean;
  }) => (
    <button disabled={disabled} onClick={onClick}>
      {children}
    </button>
  ),
}));

vi.mock("@/react/components/ui/textarea", () => ({
  Textarea: ({
    value,
    onChange,
    placeholder,
  }: {
    value: string;
    onChange: (e: { target: { value: string } }) => void;
    placeholder?: string;
  }) => (
    <textarea
      data-testid="reason-textarea"
      value={value}
      placeholder={placeholder}
      onChange={onChange}
    />
  ),
}));

vi.mock("@/react/components/ui/expiration-picker", () => ({
  ExpirationPicker: () => <div data-testid="expiration-picker" />,
}));

vi.mock("@/react/components/IssueLabelSelect", () => ({
  IssueLabelSelect: () => <div data-testid="label-select" />,
}));

vi.mock("@/react/components/RoleSelect", () => ({
  RoleSelect: ({
    value,
    disabled,
    scope,
  }: {
    value: string[];
    onChange: (roles: string[]) => void;
    multiple?: boolean;
    disabled?: boolean;
    scope?: string;
  }) => (
    <div
      data-testid="role-select"
      data-value={value.join(",")}
      data-disabled={String(disabled ?? false)}
      data-scope={scope ?? ""}
    />
  ),
}));

vi.mock("@/react/components/DatabaseResourceSelector", () => ({
  DatabaseResourceSelector: ({
    value,
    onChange,
  }: {
    projectName: string;
    value: { databaseFullName: string }[];
    onChange: (resources: { databaseFullName: string }[]) => void;
  }) => (
    <div
      data-testid="database-resource-selector"
      data-count={value.length}
      onClick={() =>
        onChange([...value, { databaseFullName: "instances/i1/databases/db1" }])
      }
    />
  ),
}));

vi.mock("@/types/proto-es/v1/issue_service_pb", () => ({
  CreateIssueRequestSchema: {},
  Issue_Type: { ROLE_GRANT: 3 },
  IssueSchema: {},
  RoleGrantSchema: {},
}));

vi.mock("@bufbuild/protobuf", () => ({
  create: vi.fn((_, data) => data ?? {}),
}));

vi.mock("@bufbuild/protobuf/wkt", () => ({
  DurationSchema: {},
}));

// Stub ResizeObserver
globalThis.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

let RoleGrantPanel: typeof import("./RoleGrantPanel").RoleGrantPanel;

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

const setupDefaultMocks = () => {
  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });
  mocks.useCurrentUserV1.mockReturnValue({
    value: { email: "user@example.com" },
  });
  mocks.useProjectV1Store.mockReturnValue({
    getProjectByName: vi.fn(() => ({
      name: "projects/proj1",
      enforceIssueTitle: false,
      forceIssueLabels: false,
      issueLabels: [],
    })),
  });
  mocks.useRoleStore.mockReturnValue({
    getRoleByName: vi.fn(() => ({
      name: "roles/sqlEditorUser",
      permissions: ["bb.sql.select", "bb.sql.explain"],
    })),
  });
  mocks.roleHasDatabaseLimitation.mockReturnValue(true);
  mocks.useVueState.mockImplementation((getter: () => unknown) => getter());
};

beforeEach(async () => {
  vi.clearAllMocks();
  setupDefaultMocks();
  ({ RoleGrantPanel } = await import("./RoleGrantPanel"));
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("RoleGrantPanel", () => {
  test("renders with sheet title", () => {
    const onClose = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <RoleGrantPanel
        projectName="projects/proj1"
        databaseResources={[{ databaseFullName: "instances/i1/databases/db1" }]}
        role="roles/sqlEditorUser"
        requiredPermissions={[]}
        onClose={onClose}
      />
    );
    render();

    expect(
      container.querySelector("[data-testid='sheet-title']")?.textContent
    ).toBe("issue.title.request-role");
    unmount();
  });

  test("renders RoleSelect as disabled with the correct role value", () => {
    const onClose = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <RoleGrantPanel
        projectName="projects/proj1"
        databaseResources={[{ databaseFullName: "instances/i1/databases/db1" }]}
        role="roles/sqlEditorUser"
        requiredPermissions={[]}
        onClose={onClose}
      />
    );
    render();

    const roleSelect = container.querySelector("[data-testid='role-select']");
    expect(roleSelect).not.toBeNull();
    expect(roleSelect?.getAttribute("data-value")).toBe("roles/sqlEditorUser");
    expect(roleSelect?.getAttribute("data-disabled")).toBe("true");
    expect(roleSelect?.getAttribute("data-scope")).toBe("project");
    unmount();
  });

  test("shows required permissions when provided", () => {
    const onClose = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <RoleGrantPanel
        projectName="projects/proj1"
        databaseResources={[{ databaseFullName: "instances/i1/databases/db1" }]}
        role="roles/sqlEditorUser"
        requiredPermissions={["bb.sql.select", "bb.sql.explain"]}
        onClose={onClose}
      />
    );
    render();

    expect(container.textContent).toContain("bb.sql.select");
    expect(container.textContent).toContain("bb.sql.explain");
    unmount();
  });

  test("renders database resource selector when role has database limitation", () => {
    mocks.roleHasDatabaseLimitation.mockReturnValue(true);
    const onClose = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <RoleGrantPanel
        projectName="projects/proj1"
        databaseResources={[{ databaseFullName: "instances/i1/databases/db1" }]}
        role="roles/sqlEditorUser"
        requiredPermissions={[]}
        onClose={onClose}
      />
    );
    render();

    expect(
      container.querySelector("[data-testid='database-resource-selector']")
    ).not.toBeNull();
    unmount();
  });

  test("does not render database resource selector when role has no database limitation", () => {
    mocks.roleHasDatabaseLimitation.mockReturnValue(false);
    const onClose = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <RoleGrantPanel
        projectName="projects/proj1"
        databaseResources={[]}
        role="roles/projectOwner"
        requiredPermissions={[]}
        onClose={onClose}
      />
    );
    render();

    expect(
      container.querySelector("[data-testid='database-resource-selector']")
    ).toBeNull();
    unmount();
  });

  test("submit disabled when database resources empty and role has database limitation", () => {
    mocks.roleHasDatabaseLimitation.mockReturnValue(true);
    const onClose = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <RoleGrantPanel
        projectName="projects/proj1"
        databaseResources={[]}
        role="roles/sqlEditorUser"
        requiredPermissions={[]}
        onClose={onClose}
      />
    );
    render();

    const buttons = container.querySelectorAll("button");
    let submitBtn: HTMLButtonElement | null = null;
    for (const btn of Array.from(buttons)) {
      if (btn.textContent?.includes("common.submit")) {
        submitBtn = btn;
        break;
      }
    }
    expect(submitBtn).not.toBeNull();
    expect(submitBtn!.disabled).toBe(true);
    unmount();
  });

  test("cancel button calls onClose", () => {
    const onClose = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <RoleGrantPanel
        projectName="projects/proj1"
        databaseResources={[{ databaseFullName: "instances/i1/databases/db1" }]}
        role="roles/sqlEditorUser"
        requiredPermissions={[]}
        onClose={onClose}
      />
    );
    render();

    const buttons = container.querySelectorAll("button");
    let cancelBtn: HTMLButtonElement | null = null;
    for (const btn of Array.from(buttons)) {
      if (btn.textContent?.includes("common.cancel")) {
        cancelBtn = btn;
        break;
      }
    }
    expect(cancelBtn).not.toBeNull();
    act(() => {
      cancelBtn!.click();
    });
    expect(onClose).toHaveBeenCalledOnce();
    unmount();
  });

  test("submit button disabled when reason required but empty", () => {
    mocks.useProjectV1Store.mockReturnValue({
      getProjectByName: vi.fn(() => ({
        name: "projects/proj1",
        enforceIssueTitle: true,
        forceIssueLabels: false,
        issueLabels: [],
      })),
    });

    const onClose = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <RoleGrantPanel
        projectName="projects/proj1"
        databaseResources={[{ databaseFullName: "instances/i1/databases/db1" }]}
        role="roles/sqlEditorUser"
        requiredPermissions={[]}
        onClose={onClose}
      />
    );
    render();

    const buttons = container.querySelectorAll("button");
    let submitBtn: HTMLButtonElement | null = null;
    for (const btn of Array.from(buttons)) {
      if (btn.textContent?.includes("common.submit")) {
        submitBtn = btn;
        break;
      }
    }
    expect(submitBtn).not.toBeNull();
    expect(submitBtn!.disabled).toBe(true);
    unmount();
  });

  test("submit calls createIssue when reason not required and db resources provided", async () => {
    mocks.roleHasDatabaseLimitation.mockReturnValue(true);
    const onClose = vi.fn();
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);

    const { container, render, unmount } = renderIntoContainer(
      <RoleGrantPanel
        projectName="projects/proj1"
        databaseResources={[{ databaseFullName: "instances/i1/databases/db1" }]}
        role="roles/sqlEditorUser"
        requiredPermissions={[]}
        onClose={onClose}
      />
    );
    render();

    const buttons = container.querySelectorAll("button");
    let submitBtn: HTMLButtonElement | null = null;
    for (const btn of Array.from(buttons)) {
      if (btn.textContent?.includes("common.submit")) {
        submitBtn = btn;
        break;
      }
    }
    expect(submitBtn).not.toBeNull();
    expect(submitBtn!.disabled).toBe(false);

    await act(async () => {
      submitBtn!.click();
    });
    await vi.waitFor(() => {
      expect(mocks.issueServiceClientConnect.createIssue).toHaveBeenCalled();
    });
    openSpy.mockRestore();
    unmount();
  });
});
