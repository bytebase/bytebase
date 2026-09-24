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
// Stub EnvironmentSelect — the real component loads the environment list
// from the app store, which is not worth wiring up for these tests.
// ---------------------------------------------------------------------------

vi.mock("@/components/EnvironmentSelect", () => ({
  EnvironmentSelect: ({ onChange }: { onChange: (next: string[]) => void }) =>
    createElement(
      "div",
      { "data-testid": "env-multi-select" },
      createElement("button", {
        type: "button",
        "data-testid": "pick-staging",
        onClick: () => onChange(["environments/staging"]),
      }),
      createElement("button", {
        type: "button",
        "data-testid": "clear-envs",
        onClick: () => onChange([]),
      })
    ),
}));

vi.mock("@/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({ environmentName }: { environmentName: string }) =>
    createElement("span", { "data-testid": "env-label" }, environmentName),
}));

// ---------------------------------------------------------------------------
// UI primitive mocks — mirror the Task 7/8 test harness so the sheet renders
// as inert DOM and submit button text is the literal i18n key.
// ---------------------------------------------------------------------------

vi.mock("@/components/ui/sheet", () => ({
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

vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    disabled,
    onClick,
    variant: _v,
  }: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: string }) =>
    createElement("button", { disabled, onClick }, children),
}));

vi.mock("@/components/ui/textarea", () => ({
  Textarea: ({
    className: _c,
    size: _s,
    ...props
  }: TextareaHTMLAttributes<HTMLTextAreaElement> & {
    size?: string;
  }) => createElement("textarea", props),
}));

vi.mock("@/components/ui/expiration-picker", () => ({
  ExpirationPicker: () => null,
}));

vi.mock("@/components/ui/alert", () => ({
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

vi.mock("@/components/IssueLabelSelect", () => ({
  IssueLabelSelect: () => null,
}));

vi.mock("@/components/DatabaseResourceSelector", () => ({
  DatabaseResourceSelector: () => null,
}));

vi.mock("@/components/ExprEditor", () => ({
  ExprEditor: () => null,
}));

// RoleSelect: test-only stub that exposes a plain <select> so tests can pick a
// role by setting value. We drive role selection via a hidden input bound to
// onChange([value]).
vi.mock("@/components/RoleSelect", () => ({
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

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en-US" },
  }),
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
vi.mock("@/types/proto-es/v1/subscription_service_pb", () => ({
  PlanFeature: { FEATURE_ENVIRONMENT_TIERS: "FEATURE_ENVIRONMENT_TIERS" },
}));

vi.mock("@/app/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/app/router")>()),
  router: {
    resolve: () => ({ fullPath: "/issues/1" }),
  },
}));

vi.mock("@/lib/project-member/utils", () => ({
  // Wrap in vi.fn() so per-test overrides (vi.mocked(x).mockReturnValue(...))
  // work — plain arrow functions can't be re-mocked at runtime.
  // Default: PROJECT_OWNER is not a SQL-permission role — both return
  // false / undefined, so the scope sections stay hidden by default.
  roleHasDatabaseLimitation: vi.fn(() => false),
  getRoleEnvironmentLimitationKind: vi.fn(() => undefined),
}));

vi.mock("@/modules/cel", () => ({
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
// Store / connect mocks — stable singletons.
// ---------------------------------------------------------------------------

const mocks = vi.hoisted(() => ({
  createIssue: vi.fn(),
  pushNotification: vi.fn(),
  currentUser: { name: "users/me@example.com", email: "me@example.com" },
  maximumRoleExpirationSeconds: undefined as number | undefined,
  maximumRequestExpirationSeconds: undefined as number | undefined,
}));

vi.mock("@/api", () => ({
  issueServiceClientConnect: {
    createIssue: (req: unknown) => mocks.createIssue(req),
  },
}));

vi.mock("@/stores", () => ({
  pushNotification: (...args: unknown[]) => mocks.pushNotification(...args),
}));

vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: () => mocks.currentUser,
  useEnvironmentList: () => [
    { name: "environments/staging", title: "Staging", tags: {} },
  ],
  usePlanFeature: () => true,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      roleList: [],
      getRoleByName: (name: string) => ({
        name,
        permissions:
          name === "roles/projectOwner"
            ? ["bb.projects.get", "bb.databases.get"]
            : [],
      }),
      getWorkspaceProfile: () => ({
        maximumRoleExpiration:
          mocks.maximumRoleExpirationSeconds === undefined
            ? undefined
            : { seconds: BigInt(mocks.maximumRoleExpirationSeconds) },
        maximumRequestExpiration:
          mocks.maximumRequestExpirationSeconds === undefined
            ? undefined
            : { seconds: BigInt(mocks.maximumRequestExpirationSeconds) },
      }),
    }),
}));

// ---------------------------------------------------------------------------
// Import SUT after mocks are registered.
// ---------------------------------------------------------------------------

import { nativeChange } from "@/test-utils/nativeChange";
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
  mocks.maximumRoleExpirationSeconds = undefined;
  mocks.maximumRequestExpirationSeconds = undefined;
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
beforeEach(async () => {
  vi.spyOn(window, "open").mockImplementation(() => null);
  // Reset env-kind mock between tests so per-test overrides don't leak.
  const utilsMock = await import("@/lib/project-member/utils");
  vi.mocked(utilsMock.getRoleEnvironmentLimitationKind).mockReturnValue(
    undefined
  );
});

describe("RequestRoleSheet — enforceIssueTitle (BYT-9310)", () => {
  it("shows the workspace maximum expiration hint when role requests are capped", async () => {
    mocks.maximumRoleExpirationSeconds = 30 * 24 * 60 * 60;

    await renderSheet(false);

    expect(container.textContent).toContain(
      "common.expiration-max-hint"
    );
  });

  it("does not cap role requests with the request access expiration", async () => {
    mocks.maximumRequestExpirationSeconds = 30 * 24 * 60 * 60;

    await renderSheet(false);

    expect(container.textContent).not.toContain(
      "common.expiration-max-hint"
    );
  });

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

  describe("direct DDL/DML execution", () => {
    async function useDdlRole(): Promise<void> {
      const utilsMock = await import("@/lib/project-member/utils");
      vi.mocked(utilsMock.getRoleEnvironmentLimitationKind).mockImplementation(
        (role: string) => (role === "roles/sqlEditorUser" ? "DDL/DML" : undefined)
      );
    }
    function getSwitch(): HTMLElement {
      return container.querySelector("[role='switch']") as HTMLElement;
    }
    async function toggleSwitch(): Promise<void> {
      await act(async () => {
        getSwitch().click();
      });
      await flush();
    }
    async function pickStaging(): Promise<void> {
      await act(async () => {
        (
          container.querySelector(
            "[data-testid='pick-staging']"
          ) as HTMLButtonElement
        ).click();
      });
      await flush();
    }
    function submittedRequest(): {
      issue: {
        title: string;
        roleGrant: { condition: { environments?: string[] } };
      };
    } {
      expect(mocks.createIssue).toHaveBeenCalledTimes(1);
      return mocks.createIssue.mock.calls[0][0] as never;
    }

    it("is off by default: caption, no picker, and the empty clause on submit", async () => {
      await useDdlRole();
      await renderSheet(false);
      await selectRole("roles/sqlEditorUser");
      await flush();

      expect(getSwitch().getAttribute("aria-checked")).toBe("false");
      expect(container.textContent).toContain(
        "project.members.direct-execution.off-caption"
      );
      expect(container.textContent).toContain(
        "project.members.direct-execution.role-pointer"
      );
      expect(
        container.querySelector("[data-testid='env-multi-select']")
      ).toBeNull();
      expect(getSubmitButton().disabled).toBe(false);

      await act(async () => {
        getSubmitButton().click();
      });
      await flush();

      const req = submittedRequest();
      expect(req.issue.roleGrant.condition.environments).toEqual([]);
      expect(req.issue.title).not.toContain("direct-execution-suffix");
    });

    it("blocks submit while on with nothing picked, with the error beside the picker", async () => {
      await useDdlRole();
      await renderSheet(false);
      await selectRole("roles/sqlEditorUser");
      await toggleSwitch();

      expect(getSwitch().getAttribute("aria-checked")).toBe("true");
      expect(
        container.querySelector("[data-testid='env-multi-select']")
      ).not.toBeNull();
      expect(container.textContent).toContain(
        "project.members.direct-execution.pick-or-off"
      );
      expect(container.textContent).not.toContain(
        "project.members.direct-execution.lead-request"
      );
      expect(getSubmitButton().disabled).toBe(true);
    });

    it("submits the picked list, shows the requester's lead, and suffixes the title", async () => {
      await useDdlRole();
      await renderSheet(false);
      await selectRole("roles/sqlEditorUser");
      await toggleSwitch();
      await pickStaging();

      expect(container.textContent).toContain(
        "project.members.direct-execution.lead-request"
      );
      expect(container.textContent).not.toContain(
        "project.members.direct-execution.pick-or-off"
      );
      expect(getSubmitButton().disabled).toBe(false);

      await act(async () => {
        getSubmitButton().click();
      });
      await flush();

      const req = submittedRequest();
      expect(req.issue.roleGrant.condition.environments).toEqual([
        "environments/staging",
      ]);
      expect(req.issue.title).toBe(
        "FMT(issue.title.request-specific-role) · issue.role-grant.direct-execution-suffix"
      );
    });

    it("suffixes an enforced title too", async () => {
      await useDdlRole();
      await renderSheet(true);
      await selectRole("roles/sqlEditorUser");
      await typeReason("fix the backfill");
      await toggleSwitch();
      await pickStaging();

      await act(async () => {
        getSubmitButton().click();
      });
      await flush();

      expect(submittedRequest().issue.title).toBe(
        "[issue.title.request-role] fix the backfill · issue.role-grant.direct-execution-suffix"
      );
    });

    it("a role change turns the switch off and clears the error", async () => {
      await useDdlRole();
      await renderSheet(false);
      await selectRole("roles/sqlEditorUser");
      await toggleSwitch();
      expect(container.textContent).toContain(
        "project.members.direct-execution.pick-or-off"
      );

      await selectRole("roles/projectOwner");
      await flush();
      expect(getSwitch()).toBeNull();

      await selectRole("roles/sqlEditorUser");
      await flush();
      expect(getSwitch().getAttribute("aria-checked")).toBe("false");
      expect(container.textContent).not.toContain(
        "project.members.direct-execution.pick-or-off"
      );
    });

    it("renders no field for a role without DDL/DML", async () => {
      await renderSheet(false);
      await selectRole("roles/projectOwner");
      await flush();

      expect(getSwitch()).toBeNull();
      expect(container.textContent).not.toContain(
        "project.members.direct-execution"
      );
    });
  });
});
