import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
} from "react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const { mockPushNotification, mockUpdateProjectIamPolicy } = vi.hoisted(() => ({
  mockPushNotification: vi.fn(),
  mockUpdateProjectIamPolicy: vi.fn(),
}));

vi.mock("@/react/components/AccountMultiSelect", () => ({
  AccountMultiSelect: ({
    onChange,
  }: {
    onChange: (members: string[]) => void;
  }) =>
    createElement(
      "button",
      {
        "data-testid": "account-select",
        onClick: () => onChange(["user:dev1@example.com"]),
      },
      "select account"
    ),
}));

vi.mock("@/react/components/DatabaseResourceSelector", () => ({
  DatabaseResourceSelector: () =>
    createElement("div", { "data-testid": "database-resource-selector" }),
}));

vi.mock("@/react/components/EnvironmentSelect", () => ({
  EnvironmentSelect: () =>
    createElement("div", { "data-testid": "environment-multi-select" }),
}));

vi.mock("@/react/components/ExprEditor", () => ({
  ExprEditor: () => createElement("div", { "data-testid": "expr-editor" }),
}));

vi.mock("@/react/components/FeatureBadge", () => ({
  FeatureBadge: () => createElement("span", {}),
}));

vi.mock("@/react/components/LearnMoreLink", () => ({
  LearnMoreLink: () => createElement("a", {}),
}));

vi.mock("@/react/components/PermissionGuard", () => ({
  PermissionGuard: ({
    children,
  }: {
    children: (args: { disabled: boolean }) => ReactNode;
  }) => createElement("div", {}, children({ disabled: false })),
}));

vi.mock("@/react/components/RoleSelect", () => ({
  RoleSelect: ({
    value,
    onChange,
  }: {
    value: string[];
    onChange: (roles: string[]) => void;
  }) =>
    createElement("input", {
      "data-testid": "role-select",
      value: value[0] ?? "",
      onChange: (e: React.ChangeEvent<HTMLInputElement>) =>
        onChange(e.target.value ? [e.target.value] : []),
    }),
}));

vi.mock("@/react/components/role-grant/DDLWarningCallout", () => ({
  DDLWarningCallout: () => null,
}));

vi.mock("@/react/components/UserCell", () => ({
  UserCell: () => createElement("span", {}),
}));

vi.mock("@/react/components/ui/alert", () => ({
  Alert: ({
    children,
    description,
    title,
  }: {
    children?: ReactNode;
    description?: ReactNode;
    title?: ReactNode;
  }) => createElement("div", {}, title, description, children),
}));

vi.mock("@/react/components/ui/badge", () => ({
  Badge: ({ children }: { children?: ReactNode }) =>
    createElement("span", {}, children),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    disabled,
    onClick,
    variant: _variant,
    size: _size,
  }: ButtonHTMLAttributes<HTMLButtonElement> & {
    size?: string;
    variant?: string;
  }) => createElement("button", { disabled, onClick }, children),
}));

vi.mock("@/react/components/ui/checkbox", () => ({
  Checkbox: () => createElement("input", { type: "checkbox" }),
}));

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: InputHTMLAttributes<HTMLInputElement>) =>
    createElement("input", props),
}));

vi.mock("@/react/components/ui/search-input", () => ({
  SearchInput: (props: InputHTMLAttributes<HTMLInputElement>) =>
    createElement("input", props),
}));

vi.mock("@/react/components/ui/sheet", () => ({
  Sheet: ({ children, open }: { children: ReactNode; open: boolean }) =>
    open ? createElement("div", { "data-testid": "sheet" }, children) : null,
  SheetBody: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetContent: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetFooter: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetHeader: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetTitle: ({ children }: { children: ReactNode }) =>
    createElement("h2", {}, children),
}));

vi.mock("@/react/components/ui/table", () => ({
  Table: ({ children }: { children: ReactNode }) =>
    createElement("table", {}, children),
  TableBody: ({ children }: { children: ReactNode }) =>
    createElement("tbody", {}, children),
  TableCell: ({ children }: { children: ReactNode }) =>
    createElement("td", {}, children),
  TableHead: ({ children }: { children: ReactNode }) =>
    createElement("th", {}, children),
  TableHeader: ({ children }: { children: ReactNode }) =>
    createElement("thead", {}, children),
  TableRow: ({ children }: { children: ReactNode }) =>
    createElement("tr", {}, children),
}));

vi.mock("@/react/components/ui/tabs", () => ({
  Tabs: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  TabsList: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  TabsPanel: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  TabsTrigger: ({ children }: { children: ReactNode }) =>
    createElement("button", {}, children),
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) =>
    createElement("span", {}, children),
}));

vi.mock("@/react/hooks/useEscapeKey", () => ({
  useEscapeKey: () => {},
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: (getter: () => unknown) => getter(),
}));

vi.mock("@/react/lib/project-member/utils", () => ({
  getRoleEnvironmentLimitationKind: vi.fn(() => undefined),
  roleHasDatabaseLimitation: vi.fn((role: string) =>
    role.includes("sqlEditor")
  ),
}));

vi.mock("@/react/router", () => ({
  useNavigate: () => vi.fn(),
  WORKSPACE_ROUTE_GROUPS: "groups",
  WORKSPACE_ROUTE_USER_PROFILE: "user-profile",
}));

vi.mock("@/plugins/cel", () => ({
  buildCELExpr: vi.fn(),
  emptySimpleExpr: () => ({}),
  validateSimpleExpr: () => true,
  wrapAsGroup: (expr: unknown) => expr,
}));

vi.mock("@/types", () => ({
  ALL_USERS_USER_EMAIL: "allUsers",
  isDefaultProject: () => false,
  PresetRoleType: { PROJECT_OWNER: "roles/projectOwner" },
  userBindingPrefix: "user:",
}));

vi.mock("@/types/proto-es/google/type/expr_pb", () => ({
  ExprSchema: {},
}));

vi.mock("@/types/proto-es/v1/common_pb", () => ({
  State: { ACTIVE: 1, DELETED: 2 },
}));

vi.mock("@/types/proto-es/v1/iam_policy_pb", () => ({
  BindingSchema: {},
}));

vi.mock("@/types/proto-es/v1/setting_service_pb", () => ({
  Setting_SettingName: { EMAIL: 1 },
}));

vi.mock("@/types/proto-es/v1/subscription_service_pb", () => ({
  PlanFeature: { FEATURE_REQUEST_ROLE_WORKFLOW: 1 },
}));

vi.mock("@/types/v1/user", () => ({
  AccountType: { USER: "USER" },
  getAccountTypeByEmail: () => "USER",
}));

vi.mock("@/utils", () => ({
  batchConvertParsedExprToCELString: vi.fn(),
  displayRoleTitle: (role: string) => role,
  formatAbsoluteDateTime: () => "",
  getDatabaseNameOptionConfig: () => ({ options: [] }),
  hasProjectPermissionV2: () => true,
  hasWorkspacePermissionV2: () => true,
  isBindingPolicyExpired: () => false,
  sortRoles: (roles: string[]) => roles,
}));

vi.mock("@/utils/cel-attributes", () => ({
  CEL_ATTRIBUTE_RESOURCE_DATABASE: "resource.database",
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME: "resource.schema_name",
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME: "resource.table_name",
}));

vi.mock("@/utils/issue/cel", () => ({
  buildConditionExpr: (args: Record<string, unknown>) => ({ ...args }),
  convertFromExpr: () => ({}),
  stringifyConditionExpression: () => "",
}));

vi.mock("@/react/lib/memberBindings", () => ({
  getMemberBindings: () => [],
  groupProjectRoleBindings: () => [],
}));

vi.mock("@bufbuild/protobuf", () => ({
  create: (_schema: unknown, init?: Record<string, unknown>) => ({ ...init }),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

const projectIamPolicy = { bindings: [] };

vi.mock("@/store", () => ({
  pushNotification: mockPushNotification,
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => ({
    email: "me@example.com",
    name: "users/me@example.com",
  }),
}));

vi.mock("@/react/stores/app", () => {
  const buildState = () => ({
    batchGetOrFetchUsers: vi.fn(async () => []),
    roleList: [{ name: "roles/sqlEditorUser", permissions: [] }],
    workspacePolicy: { bindings: [] },
    patchWorkspaceIamPolicy: vi.fn(),
    findWorkspaceRolesByMember: () => [],
    fetchWorkspaceIamPolicy: vi.fn(async () => undefined),
    // MembersPage now subscribes to projectPoliciesByName directly so it
    // re-renders when loadProjectIamPolicy() resolves. The getter form is
    // still used inside async handlers.
    projectPoliciesByName: { "projects/sample-project": projectIamPolicy },
    getProjectIamPolicy: () => projectIamPolicy,
    updateProjectIamPolicy: mockUpdateProjectIamPolicy,
    loadProjectIamPolicy: vi.fn(async () => undefined),
    // Project store methods, migrated off the Pinia useProjectV1Store mock.
    projectsByName: {},
    getProjectByName: (name: string) => ({
      allowRequestRole: true,
      name,
      permissions: ["bb.projects.setIamPolicy"],
      state: 1,
    }),
    // Migrated off the Pinia actuator/setting/subscription store mocks.
    isSaaSMode: () => false,
    userCountInIam: () => 1,
    userCountLimit: () => 10,
    hasFeature: () => true,
    settingsByName: {},
    getOrFetchSettingByName: vi.fn(),
    getSettingByName: () => undefined,
    getWorkspaceProfile: () => ({}),
  });
  const useAppStore = (selector?: (state: unknown) => unknown) =>
    selector ? selector(buildState()) : buildState();
  useAppStore.getState = () => buildState();
  return { useAppStore };
});

vi.mock("./MemberBindingEnvironmentBanner", () => ({
  MemberBindingEnvironmentBanner: () => null,
}));

vi.mock("./MemberDatabaseResourceName", () => ({
  MemberDatabaseResourceName: () => null,
}));

vi.mock("./RequestRoleSheet", () => ({
  RequestRoleSheet: () => null,
}));

import { buildCELExpr } from "@/plugins/cel";
import { nativeChange } from "@/react/test-utils/nativeChange";
import { MembersPage } from "./MembersPage";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  vi.clearAllMocks();
  projectIamPolicy.bindings = [];
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  document.body.removeChild(container);
});

async function renderPage(): Promise<void> {
  await act(async () => {
    root.render(createElement(MembersPage, { projectId: "sample-project" }));
    await Promise.resolve();
  });
}

async function flush(): Promise<void> {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
}

describe("MembersPage project role grant drawer", () => {
  it("uses the graphical expression editor for database CEL scope", async () => {
    await renderPage();

    const grantButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "settings.members.grant-access"
    ) as HTMLButtonElement;
    await act(async () => {
      grantButton.click();
    });
    await flush();

    const accountButton = container.querySelector(
      "[data-testid='account-select']"
    ) as HTMLButtonElement;
    await act(async () => {
      accountButton.click();
    });

    const roleInput = container.querySelector(
      "[data-testid='role-select']"
    ) as HTMLInputElement;
    await act(async () => {
      nativeChange(roleInput, "roles/sqlEditorUser");
    });
    await flush();

    const expressionRadio = [...container.querySelectorAll("input")].find(
      (input) =>
        input instanceof HTMLInputElement &&
        input.type === "radio" &&
        input.parentElement?.textContent === "CEL Expression"
    ) as HTMLInputElement;
    await act(async () => {
      expressionRadio.click();
    });
    await flush();

    expect(
      container.querySelector("[data-testid='expr-editor']")
    ).not.toBeNull();
    expect(
      container.querySelector(
        "textarea[placeholder='e.g. resource.database_name.startsWith(\"employee_\")']"
      )
    ).toBeNull();
  });

  it("shows an error when CEL expression parsing fails", async () => {
    vi.mocked(buildCELExpr).mockRejectedValueOnce(new Error("parse failed"));

    await renderPage();

    const grantButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "settings.members.grant-access"
    ) as HTMLButtonElement;
    await act(async () => {
      grantButton.click();
    });
    await flush();

    const accountButton = container.querySelector(
      "[data-testid='account-select']"
    ) as HTMLButtonElement;
    await act(async () => {
      accountButton.click();
    });

    const roleInput = container.querySelector(
      "[data-testid='role-select']"
    ) as HTMLInputElement;
    await act(async () => {
      nativeChange(roleInput, "roles/sqlEditorUser");
    });
    await flush();

    const expressionRadio = [...container.querySelectorAll("input")].find(
      (input) =>
        input instanceof HTMLInputElement &&
        input.type === "radio" &&
        input.parentElement?.textContent === "CEL Expression"
    ) as HTMLInputElement;
    await act(async () => {
      expressionRadio.click();
    });
    await flush();

    const createButton = [...container.querySelectorAll("button")].find(
      (button) => button.textContent === "common.create"
    ) as HTMLButtonElement;
    await act(async () => {
      createButton.click();
    });
    await flush();

    expect(mockPushNotification).toHaveBeenCalledWith({
      module: "bytebase",
      style: "CRITICAL",
      title: "project.members.request-role.failed-to-build-expression",
    });
    expect(mockUpdateProjectIamPolicy).not.toHaveBeenCalled();
  });
});
