import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { IssueDetailRoleGrantDetails } from "./IssueDetailRoleGrantDetails";

// vi.mock factories are hoisted above imports — wrap mutable test state in
// vi.hoisted so the closure reads from a binding that's initialized at hoist time.
const { mockContextRef, mockParsedConditionRef } = vi.hoisted(() => ({
  mockContextRef: { current: undefined as unknown },
  mockParsedConditionRef: {
    current: undefined as
      | { environments?: string[]; expiredTime?: string }
      | undefined,
  },
}));

vi.mock("react-i18next", async () =>
  (await import("@/test-utils/i18n")).reactI18nextStub()
);

vi.mock("@/lib/project-member/utils", () => ({
  getRoleEnvironmentLimitationKind: (role: string) =>
    role === "roles/sqlEditorUser" ? "DDL/DML" : undefined,
}));

vi.mock("@/lib/role", () => ({
  displayRoleTitleFromList: (role: string) => `TITLE(${role})`,
  displayRoleDescriptionFromList: (role: string) => `DESC(${role})`,
}));

// What usePrincipal resolves: two people, a person whose record never arrived
// (title still empty), and a service account.
const PRINCIPALS: Record<string, { title: string; email: string }> = {
  "users/alex@example.com": {
    title: "Alex Kim",
    email: "alex@example.com",
  },
  "users/bot@example.com": {
    title: "CI Bot",
    email: "bot@example.com",
  },
  "users/new@example.com": {
    title: "",
    email: "new@example.com",
  },
  "users/deploy@service.bytebase.com": {
    title: "deploy",
    email: "deploy@service.bytebase.com",
  },
};

vi.mock("@/hooks/useAppState", () => ({
  useEnvironmentList: () => [
    { name: "environments/prod", title: "Prod", tags: {} },
    { name: "environments/test", title: "Test", tags: {} },
  ],
  usePlanFeature: () => true,
}));

vi.mock("@/hooks/usePrincipal", () => ({
  usePrincipal: (identifier?: string) =>
    identifier ? PRINCIPALS[identifier] : undefined,
}));

// Stub the badge — the real one pulls in theme tokens and router links.
vi.mock("@/components/EnvironmentLabel", async () =>
  (await import("@/test-utils/environmentSelectStub")).environmentBadgeStub()
);

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
  readableConditionFromCELString: async (expr: string) => {
    if (mockParsedConditionRef.current) return mockParsedConditionRef.current;
    if (expr === "unreadable") throw new Error("unreadable");
    // Mini parser just for tests: only recognizes "environment_id in [...]".
    const m = expr.match(/environment_id in \[([^\]]*)\]/);
    if (!m) return { environments: undefined };
    const ids = m[1]
      .split(",")
      .map((s) => s.trim().replace(/^"|"$/g, ""))
      .filter(Boolean);
    return { environments: ids.map((id) => `environments/${id}`) };
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
  mockParsedConditionRef.current = undefined;
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

  test("the grantee and what approving grants come before the permission list", async () => {
    mockContextRef.current = issueWith({
      expression: 'resource.environment_id in ["prod"]',
    });
    render(<IssueDetailRoleGrantDetails />);
    const execution = await screen.findByTestId("role-grant-direct-execution");
    const follows = (a: Element, b: Element) =>
      Boolean(a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING);
    expect(follows(screen.getByTestId("role-grant-grantee"), execution)).toBe(
      true
    );
    expect(follows(execution, screen.getByText(/^common\.permissions/))).toBe(
      true
    );
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
    expect(await screen.findByText(/direct-execution\.none/)).toBeTruthy();
    expect(screen.queryByTestId("env-label")).toBeNull();
  });

  test("no environment clause renders the unscoped warning", async () => {
    mockContextRef.current = issueWith({
      expression: 'request.time < timestamp("2026-10-01T00:00:00Z")',
    });
    render(<IssueDetailRoleGrantDetails />);
    expect(await screen.findByText(/direct-execution\.all/)).toBeTruthy();
  });

  test("shows a raw resource scope after parsing without losing expiry or environments", async () => {
    const expression =
      'resource.table_name.contains("archive") && resource.environment_id in ["prod"] && request.time < timestamp("2026-10-01T00:00:00Z")';
    mockContextRef.current = issueWith({ expression });
    mockParsedConditionRef.current = {
      environments: ["environments/prod"],
      expiredTime: "2026-10-01T00:00:00.000Z",
    };

    render(<IssueDetailRoleGrantDetails />);
    expect(screen.queryByText(expression)).toBeNull();

    const rawCondition = await screen.findByText(expression);
    expect(rawCondition.tagName).toBe("PRE");
    expect(screen.getByText("cel.condition.self")).toBeTruthy();
    expect(screen.queryByText("issue.role-grant.all-databases")).toBeNull();
    expect(screen.getByTestId("env-label").textContent).toBe("environments/prod");
    const expiry = screen.getByText("issue.role-grant.expired-at").parentElement;
    expect(expiry?.textContent).toContain("2026");
    expect(expiry?.textContent).not.toContain("project.members.never-expires");
  });

  test("an empty expression is unscoped too", () => {
    mockContextRef.current = issueWith({ expression: "" });
    render(<IssueDetailRoleGrantDetails />);
    expect(screen.getByText(/direct-execution\.all/)).toBeTruthy();
  });

  test("a condition the reader rejects renders the unscoped warning, not nothing", async () => {
    const consoleError = vi
      .spyOn(console, "error")
      .mockImplementation(() => {});
    try {
      mockContextRef.current = issueWith({ expression: "unreadable" });
      render(<IssueDetailRoleGrantDetails />);
      expect(await screen.findByText(/direct-execution\.all/)).toBeTruthy();
      expect(consoleError).toHaveBeenCalledWith(
        "Failed to parse CEL expression:",
        expect.any(Error)
      );
    } finally {
      consoleError.mockRestore();
    }
  });

  test("a role without DDL/DML has no execution row", () => {
    mockContextRef.current = issueWith({ role: "roles/queryOnly" });
    render(<IssueDetailRoleGrantDetails />);
    expect(screen.queryByTestId("role-grant-direct-execution")).toBeNull();
  });

  test("a service-account grantee is marked as one and named from its email until its record arrives", async () => {
    mockContextRef.current = issueWith({
      user: "users/deploy@service.bytebase.com",
      creator: "users/alex@example.com",
      expression: 'resource.environment_id in ["prod"]',
    });
    render(<IssueDetailRoleGrantDetails />);
    expect(screen.getByText("settings.members.service-account")).toBeTruthy();
    expect(screen.getByText("deploy")).toBeTruthy();
    expect(screen.getByText("deploy@service.bytebase.com")).toBeTruthy();
    expect((await screen.findByText(/lead-approver/)).textContent).toContain(
      '"grantee":"deploy"'
    );
  });

  test("a person as grantee carries no account badge", () => {
    mockContextRef.current = issueWith({});
    render(<IssueDetailRoleGrantDetails />);
    expect(screen.queryByText("settings.members.service-account")).toBeNull();
    expect(screen.queryByText("settings.members.workload-identity")).toBeNull();
  });

  test("a person whose record has not arrived is shown by email, not by resource name", () => {
    mockContextRef.current = issueWith({
      user: "users/new@example.com",
      creator: "users/bot@example.com",
    });
    render(<IssueDetailRoleGrantDetails />);
    expect(screen.getByText("new@example.com")).toBeTruthy();
    expect(screen.queryByText("users/new@example.com")).toBeNull();
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
