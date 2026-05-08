import type {
  ButtonHTMLAttributes,
  ReactNode,
  TextareaHTMLAttributes,
} from "react";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Permission } from "@/types";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

// ---------------------------------------------------------------------------
// Stub EnvironmentMultiSelect — the real component mounts Pinia-backed
// environment state that's not worth wiring up for these tests.
// ---------------------------------------------------------------------------

vi.mock("@/react/components/EnvironmentMultiSelect", () => ({
  EnvironmentMultiSelect: () =>
    createElement("div", { "data-testid": "env-multi-select" }),
}));

// ---------------------------------------------------------------------------
// UI primitive mocks — mirror the Task 7/8 test harness so the sheet renders
// as inert DOM and submit button text is the literal i18n key.
// ---------------------------------------------------------------------------

vi.mock("@/react/components/ui/sheet", () => ({
  Sheet: ({ children, open }: { children: ReactNode; open: boolean }) =>
    open ? createElement("div", { "data-testid": "sheet" }, children) : null,
  SheetContent: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetHeader: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetTitle: ({ children }: { children: ReactNode }) =>
    createElement("h2", {}, children),
  SheetBody: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
  SheetFooter: ({ children }: { children: ReactNode }) =>
    createElement("div", {}, children),
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    disabled,
    onClick,
    variant: _v,
  }: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: string }) =>
    createElement("button", { disabled, onClick }, children),
}));

vi.mock("@/react/components/ui/textarea", () => ({
  Textarea: ({
    className: _c,
    size: _s,
    ...props
  }: TextareaHTMLAttributes<HTMLTextAreaElement> & {
    size?: string;
  }) => createElement("textarea", props),
}));

vi.mock("@/react/components/ui/expiration-picker", () => ({
  ExpirationPicker: () => null,
}));

vi.mock("@/react/components/ui/alert", () => ({
  Alert: ({
    children,
    title,
    description,
  }: {
    children?: ReactNode;
    title?: ReactNode;
    description?: ReactNode;
  }) => createElement("div", {}, title, description, children),
}));

vi.mock("@/react/components/IssueLabelSelect", () => ({
  IssueLabelSelect: () => null,
}));

vi.mock("@/react/components/DatabaseResourceSelector", () => ({
  DatabaseResourceSelector: () => null,
}));

vi.mock("@/react/components/ExprEditor", () => ({
  ExprEditor: () => null,
}));

// RoleSelect: test-only stub that exposes a plain <select> so tests can pick a
// role by setting value. We drive role selection via a hidden input bound to
// onChange([value]).
vi.mock("@/react/components/RoleSelect", () => ({
  RoleSelect: ({
    value,
    onChange,
  }: {
    value: string[];
    onChange: (roles: string[]) => void;
    scope?: string;
    multiple?: boolean;
  }) =>
    createElement("input", {
      "data-testid": "role-select",
      value: value[0] ?? "",
      onChange: (e: React.ChangeEvent<HTMLInputElement>) =>
        onChange(e.target.value ? [e.target.value] : []),
    }),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: (getter: () => unknown) => getter(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

// ---------------------------------------------------------------------------
// Infra / cross-module mocks
// ---------------------------------------------------------------------------

// Preserve `create` from the real @bufbuild/protobuf so the production
// `create(IssueSchema, {...})` call still returns a plain object containing
// the `title` field we assert on.
vi.mock("@bufbuild/protobuf", () => ({
  create: (_schema: unknown, init?: Record<string, unknown>) => ({ ...init }),
}));

vi.mock("@bufbuild/protobuf/wkt", () => ({
  DurationSchema: {},
}));

vi.mock("@/types/proto-es/google/type/expr_pb", () => ({
  ExprSchema: {},
}));

vi.mock("@/types/proto-es/v1/issue_service_pb", () => ({
  CreateIssueRequestSchema: {},
  Issue_Type: { ROLE_GRANT: 1 },
  IssueSchema: {},
  RoleGrantSchema: {},
}));

vi.mock("@/types/proto-es/v1/project_service_pb", () => ({}));

vi.mock("@/router", () => ({
  router: {
    resolve: () => ({ fullPath: "/issues/1" }),
  },
}));

vi.mock("@/router/dashboard/projectV1", () => ({
  PROJECT_V1_ROUTE_ISSUE_DETAIL: "issue-detail",
}));

vi.mock("@/components/ProjectMember/utils", () => ({
  // PROJECT_OWNER is not a SQL-permission role — these return false, so the
  // database/environment scope sections stay hidden in the tests.
  roleHasDatabaseLimitation: () => false,
  roleHasEnvironmentLimitation: () => false,
}));

vi.mock("@/plugins/cel", () => ({
  buildCELExpr: vi.fn(),
  emptySimpleExpr: () => ({}),
  validateSimpleExpr: () => true,
  wrapAsGroup: (e: unknown) => e,
}));

vi.mock("@/utils/cel-attributes", () => ({
  CEL_ATTRIBUTE_RESOURCE_DATABASE: "resource.database",
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME: "resource.schema_name",
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME: "resource.table_name",
}));

vi.mock("@/utils/issue/cel", () => ({
  buildConditionExpr: (args: Record<string, unknown>) => ({ ...args }),
  stringifyConditionExpression: () => "",
}));

vi.mock("@/utils", () => ({
  batchConvertParsedExprToCELString: vi.fn(),
  displayRoleTitle: (role: string) => `TITLE(${role})`,
  extractIssueUID: (name: string) => name.split("/").pop() ?? "",
  extractProjectResourceName: (name: string) => name.split("/")[1] ?? name,
  formatIssueTitle: (title: string) => `FMT(${title})`,
  getDatabaseNameOptionConfig: () => ({ options: [] }),
  normalizeTitle: (s: string) => s.trim(),
}));

vi.mock("@/types", () => ({
  PresetRoleType: {
    PROJECT_OWNER: "roles/projectOwner",
  },
}));

// ---------------------------------------------------------------------------
// Store / connect mocks — stable singletons for Pinia-adjacent bridges.
// ---------------------------------------------------------------------------

const mocks = vi.hoisted(() => ({
  createIssue: vi.fn(),
  pushNotification: vi.fn(),
  currentUser: { name: "users/me@example.com", email: "me@example.com" },
}));

const stableSettingStore = {
  workspaceProfile: { maximumRoleExpiration: undefined },
};

vi.mock("@/connect", () => ({
  issueServiceClientConnect: {
    createIssue: (req: unknown) => mocks.createIssue(req),
  },
}));

vi.mock("@/store", () => ({
  useCurrentUserV1: () => ({ value: mocks.currentUser }),
  useRoleStore: () => ({
    getRoleByName: (name: string) => ({
      name,
      permissions:
        name === "roles/projectOwner"
          ? ["bb.projects.get", "bb.databases.get"]
          : [],
    }),
  }),
  useSettingV1Store: () => stableSettingStore,
  pushNotification: (...args: unknown[]) => mocks.pushNotification(...args),
}));

// ---------------------------------------------------------------------------
// Import SUT after mocks are registered.
// ---------------------------------------------------------------------------

import { nativeChange } from "@/react/test-utils/nativeChange";
import { RequestRoleSheet } from "./RequestRoleSheet";

const PROJECT_BASE = {
  name: "projects/foo",
  issueLabels: [],
  forceIssueLabels: false,
};

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  vi.clearAllMocks();
  mocks.createIssue.mockResolvedValue({
    name: "projects/foo/issues/1",
  });
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

async function flush(): Promise<void> {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
}

async function renderSheet(
  enforceIssueTitle: boolean,
  requiredPermissions?: Permission[]
): Promise<void> {
  const project = {
    ...PROJECT_BASE,
    enforceIssueTitle,
  } as never;
  await act(async () => {
    root.render(
      createElement(RequestRoleSheet, {
        open: true,
        project,
        requiredPermissions,
        onClose: () => {},
      })
    );
    await Promise.resolve();
    await Promise.resolve();
  });
}

function getRoleInput(): HTMLInputElement {
  return container.querySelector(
    "[data-testid='role-select']"
  ) as HTMLInputElement;
}

function getReasonTextarea(): HTMLTextAreaElement {
  return container.querySelector("textarea") as HTMLTextAreaElement;
}

function getSubmitButton(): HTMLButtonElement {
  return [...container.querySelectorAll("button")].find((b) =>
    b.textContent?.includes("common.submit")
  ) as HTMLButtonElement;
}

async function selectRole(role: string): Promise<void> {
  await act(async () => {
    nativeChange(getRoleInput(), role);
  });
}

async function typeReason(text: string): Promise<void> {
  await act(async () => {
    nativeChange(getReasonTextarea(), text);
  });
}

// Stub window.open — production handler opens the created issue in a new tab.
beforeEach(() => {
  vi.spyOn(window, "open").mockImplementation(() => null);
});

describe("RequestRoleSheet — enforceIssueTitle (BYT-9310)", () => {
  it("Submit is enabled with empty reason when enforceIssueTitle is false", async () => {
    await renderSheet(false);
    await selectRole("roles/projectOwner");
    await flush();

    expect(getReasonTextarea().value).toBe("");
    expect(getSubmitButton().disabled).toBe(false);
  });

  it("Submit is disabled until reason is typed when enforceIssueTitle is true", async () => {
    await renderSheet(true);
    await selectRole("roles/projectOwner");
    await flush();

    expect(getSubmitButton().disabled).toBe(true);

    await typeReason("need access for oncall");
    await flush();

    expect(getSubmitButton().disabled).toBe(false);
  });

  it("title is `[request-role] <reason>` when enforceIssueTitle is true", async () => {
    await renderSheet(true);
    await selectRole("roles/projectOwner");
    await typeReason("my reason");
    await flush();

    await act(async () => {
      getSubmitButton().click();
    });
    await flush();

    expect(mocks.createIssue).toHaveBeenCalledTimes(1);
    const req = mocks.createIssue.mock.calls[0][0] as {
      issue: { title: string };
    };
    expect(req.issue.title).toBe("[issue.title.request-role] my reason");
  });

  it("title uses request-specific-role auto-format when enforceIssueTitle is false", async () => {
    await renderSheet(false);
    await selectRole("roles/projectOwner");
    // No reason typed — auto-title path.
    await flush();

    await act(async () => {
      getSubmitButton().click();
    });
    await flush();

    expect(mocks.createIssue).toHaveBeenCalledTimes(1);
    const req = mocks.createIssue.mock.calls[0][0] as {
      issue: { title: string };
    };
    // formatIssueTitle sentinel: mock wraps in FMT(...) so if the production
    // code drops the formatIssueTitle() call, this assertion fails.
    expect(req.issue.title).toBe("FMT(issue.title.request-specific-role)");
  });

  it("blocks stale role submissions when the selected role misses required permissions", async () => {
    await renderSheet(false, ["bb.databases.get"]);
    await selectRole("roles/nonMatching");
    await flush();

    expect(getSubmitButton().disabled).toBe(true);

    await selectRole("roles/projectOwner");
    await flush();

    expect(getSubmitButton().disabled).toBe(false);
  });
});
