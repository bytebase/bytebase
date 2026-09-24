import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { IssueDetailRoleGrantDetails } from "./IssueDetailRoleGrantDetails";

// vi.mock factories are hoisted above imports — wrap mutable test state in
// vi.hoisted so the closure reads from a binding that's initialized at hoist time.
const { mockContextRef } = vi.hoisted(() => ({
  mockContextRef: { current: undefined as unknown },
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars ? `${key} ${JSON.stringify(vars)}` : key,
    i18n: { language: "en-US" },
  }),
}));

vi.mock("@/lib/project-member/utils", () => ({
  getRoleEnvironmentLimitationKind: (role: string) =>
    role === "roles/sqlEditorUser" ? "DDL/DML" : undefined,
}));

vi.mock("@/lib/role", () => ({
  displayRoleTitleFromList: (role: string) => `TITLE(${role})`,
  displayRoleDescriptionFromList: (role: string) => `DESC(${role})`,
}));

const USERS: Record<string, { title: string; email: string }> = {
  "users/alex@example.com": { title: "Alex Kim", email: "alex@example.com" },
  "users/bot@example.com": { title: "CI Bot", email: "bot@example.com" },
};

vi.mock("@/hooks/useAppState", () => ({
  useEnvironmentList: () => [
    { name: "environments/prod", title: "Prod", tags: {} },
    { name: "environments/test", title: "Test", tags: {} },
  ],
  usePlanFeature: () => true,
  useUserByIdentifier: (identifier?: string) =>
    identifier ? USERS[identifier] : undefined,
}));

// Stub EnvironmentLabel — the real component pulls in app-store hooks and
// theme tokens.
vi.mock("@/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({ environmentName }: { environmentName: string }) => (
    <span data-testid="env-label">{environmentName}</span>
  ),
}));

vi.mock("@/types/v1/database", () => ({
  unknownDatabase: () => ({
    name: "instances/-/databases/-",
    effectiveEnvironment: "",
    instanceResource: undefined,
  }),
}));

vi.mock("@/stores/app", () => {
  const appState = () => ({
    roleList: [
      {
        name: "roles/sqlEditorUser",
        permissions: ["bb.sql.ddl", "bb.sql.dml"],
      },
      { name: "roles/queryOnly", permissions: ["bb.sql.select"] },
    ],
    getRoleByName: (role: string) =>
      role === "roles/sqlEditorUser"
        ? { name: role, permissions: ["bb.sql.ddl", "bb.sql.dml"] }
        : role === "roles/queryOnly"
          ? { name: role, permissions: ["bb.sql.select"] }
          : undefined,
    instancesByName: {} as Record<string, unknown>,
    databasesByName: {} as Record<string, unknown>,
    environmentList: [] as unknown[],
    getEnvironmentByName: () => ({ title: "" }),
    batchGetOrFetchDatabases: vi.fn(),
  });
  return {
    useAppStore: Object.assign(
      (selector: (state: unknown) => unknown) => selector(appState()),
      { getState: appState }
    ),
  };
});

vi.mock("@/utils/issue/cel", () => ({
  convertFromCELString: async (expr: string) => {
    // Mini parser just for tests: "environment_id in [...]", plus an OR
    // marker that the real decoder reports as unrecognized.
    const m = expr.match(/environment_id in \[([^\]]*)\]/);
    if (!m) return { environments: undefined };
    const ids = m[1]
      .split(",")
      .map((s) => s.trim().replace(/^"|"$/g, ""))
      .filter(Boolean);
    return {
      environments: ids.map((id) => `environments/${id}`),
      ...(expr.includes("||") ? { unrecognized: true as const } : {}),
    };
  },
}));

vi.mock("@/utils/v1/database", () => ({
  extractDatabaseResourceName: () => ({ databaseName: "", instanceName: "" }),
}));

vi.mock("../context/IssueDetailContext", () => ({
  useIssueDetailContext: () => mockContextRef.current,
}));

const issueWith = (
  overrides: Partial<{
    role: string;
    expression: string;
    user: string;
    creator: string;
  }>
) => ({
  issue: {
    creator: overrides.creator ?? "users/alex@example.com",
    roleGrant: {
      role: overrides.role ?? "roles/sqlEditorUser",
      user: overrides.user ?? "users/alex@example.com",
      condition: { expression: overrides.expression ?? "" },
    },
  },
});

beforeEach(() => {
  mockContextRef.current = undefined;
});

describe("IssueDetailRoleGrantDetails", () => {
  test("the grantee row shows the grant's user, not the creator, and no 'requested by' when they match", () => {
    mockContextRef.current = issueWith({});
    render(<IssueDetailRoleGrantDetails />);
    const row = screen.getByTestId("role-grant-grantee");
    expect(row.textContent).toContain("Alex Kim");
    expect(row.textContent).toContain("alex@example.com");
    expect(row.textContent).not.toContain("requested-by");
  });

  test("'requested by' appears only when the creator differs from the grantee", () => {
    mockContextRef.current = issueWith({ creator: "users/bot@example.com" });
    render(<IssueDetailRoleGrantDetails />);
    expect(
      screen.getByTestId("role-grant-grantee").textContent
    ).toContain('requested-by {"creator":"CI Bot"}');
  });

  test("a list renders the approver lead naming the grantee, with one chip per environment", async () => {
    mockContextRef.current = issueWith({
      expression: 'resource.environment_id in ["prod", "test"]',
    });
    render(<IssueDetailRoleGrantDetails />);
    const lead = await screen.findByText(/lead-approver/);
    expect(lead.textContent).toContain('"grantee":"Alex Kim"');
    expect(
      screen.getAllByTestId("env-label").map((el) => el.textContent)
    ).toEqual(["environments/prod", "environments/test"]);
  });

  test("the empty list renders the 'none' sentence rather than nothing", async () => {
    mockContextRef.current = issueWith({
      expression: "resource.environment_id in []",
    });
    render(<IssueDetailRoleGrantDetails />);
    expect(await screen.findByText(/none-grant/)).toBeTruthy();
    expect(screen.queryByTestId("env-label")).toBeNull();
  });

  test("no environment clause renders the unscoped warning", async () => {
    mockContextRef.current = issueWith({
      expression: 'request.time < timestamp("2026-10-01T00:00:00Z")',
    });
    render(<IssueDetailRoleGrantDetails />);
    expect(await screen.findByText(/direct-execution\.all/)).toBeTruthy();
  });

  test("an empty expression is unscoped too", () => {
    mockContextRef.current = issueWith({ expression: "" });
    render(<IssueDetailRoleGrantDetails />);
    expect(screen.getByText(/direct-execution\.all/)).toBeTruthy();
  });

  test("a condition the forms could not have written shows its raw text and no chip", async () => {
    const expression = 'resource.environment_id in ["prod"] || true';
    mockContextRef.current = issueWith({ expression });
    render(<IssueDetailRoleGrantDetails />);
    expect(await screen.findByText(expression)).toBeTruthy();
    expect(screen.getByText(/direct-execution\.custom/)).toBeTruthy();
    expect(screen.queryByTestId("env-label")).toBeNull();
  });

  test("a role without DDL/DML has no execution row", async () => {
    mockContextRef.current = issueWith({
      role: "roles/queryOnly",
      expression: 'resource.environment_id in ["prod"]',
    });
    render(<IssueDetailRoleGrantDetails />);
    await Promise.resolve();
    expect(screen.queryByTestId("role-grant-direct-execution")).toBeNull();
  });

  test("the role's description renders under its title", () => {
    mockContextRef.current = issueWith({});
    render(<IssueDetailRoleGrantDetails />);
    expect(screen.getByText("DESC(roles/sqlEditorUser)")).toBeTruthy();
  });

  test("clears stale environments when the issue prop changes", async () => {
    mockContextRef.current = issueWith({
      expression: 'resource.environment_id in ["prod"]',
    });
    const { rerender } = render(<IssueDetailRoleGrantDetails />);
    await screen.findByText(/lead-approver/);

    mockContextRef.current = issueWith({
      expression: 'resource.environment_id in ["test"]',
    });
    rerender(<IssueDetailRoleGrantDetails />);
    expect(
      (await screen.findAllByTestId("env-label")).map((el) => el.textContent)
    ).toEqual(["environments/test"]);
  });
});
