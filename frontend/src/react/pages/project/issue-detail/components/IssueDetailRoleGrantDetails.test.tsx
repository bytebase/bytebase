import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { IssueDetailRoleGrantDetails } from "./IssueDetailRoleGrantDetails";

// vi.mock factories are hoisted above imports — wrap mutable test state in
// vi.hoisted so the closure reads from a binding that's initialized at hoist time.
const { mockContextRef } = vi.hoisted(() => ({
  mockContextRef: { current: undefined as unknown },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) => {
      let s = key;
      if (vars) {
        for (const [k, v] of Object.entries(vars)) {
          s = s.replace(`{{${k}}}`, String(v));
        }
        // Append a JSON suffix so we can also assert on raw vars in messy cases.
        s += " " + JSON.stringify(vars);
      }
      return s;
    },
  }),
}));

vi.mock("@/components/ProjectMember/utils", () => ({
  getRoleEnvironmentLimitationKind: (role: string) =>
    role === "roles/sqlEditorUser" ? "DDL/DML" : undefined,
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useEnvironmentList: () => [
    { name: "environments/prod", title: "Prod" },
    { name: "environments/test", title: "Test" },
  ],
}));

// Stub EnvironmentLabel — the real component pulls in Pinia stores + theme tokens.
vi.mock("@/react/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({
    environmentName,
    className,
  }: {
    environmentName: string;
    className?: string;
  }) => (
    <span data-testid="env-label" className={className}>
      {environmentName}
    </span>
  ),
}));

// Stub other modules the component pulls in.
vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: <T,>(getter: () => T) => getter(),
}));

vi.mock("@/store", () => ({
  useDatabaseV1Store: () => ({
    batchGetOrFetchDatabases: vi.fn(),
    getDatabaseByName: () => ({
      effectiveEnvironment: "",
      instanceResource: undefined,
    }),
  }),
  useEnvironmentV1Store: () => ({
    getEnvironmentByName: () => ({ title: "" }),
  }),
  useInstanceV1Store: () => ({ getInstanceByName: () => ({ title: "" }) }),
  useRoleStore: () => ({
    getRoleByName: (role: string) =>
      role === "roles/sqlEditorUser"
        ? { name: role, permissions: ["bb.sql.ddl", "bb.sql.dml"] }
        : role === "roles/queryOnly"
          ? { name: role, permissions: ["bb.sql.select"] }
          : undefined,
  }),
}));

vi.mock("@/utils", () => ({
  displayRoleTitle: (r: string) => r,
}));

vi.mock("@/utils/issue/cel", () => ({
  convertFromCELString: async (expr: string) => {
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

beforeEach(() => {
  mockContextRef.current = undefined;
});

describe("IssueDetailRoleGrantDetails", () => {
  test("renders Environments row and warning when role has DDL/DML and env list is non-empty", async () => {
    mockContextRef.current = {
      issue: {
        roleGrant: {
          role: "roles/sqlEditorUser",
          condition: {
            expression: 'resource.environment_id in ["prod", "test"]',
          },
        },
      },
    };
    render(<IssueDetailRoleGrantDetails />);
    expect(
      await screen.findByText(/project.members.ddl-current-some/)
    ).toBeInTheDocument();
    expect(screen.getByText(/common.environments/)).toBeInTheDocument();
    // Env titles render as plain text inside the env section.
    expect(screen.getByText("Prod, Test")).toBeInTheDocument();
  });

  test("renders binding-all warning when condition has no environment clause (unrestricted)", async () => {
    // Expression with expiration but no environment_id clause — the grant
    // would apply to ALL environments. This is the highest-risk scenario;
    // the approver MUST see the warning.
    mockContextRef.current = {
      issue: {
        roleGrant: {
          role: "roles/sqlEditorUser",
          condition: {
            expression: 'request.time < timestamp("2026-12-31T00:00:00Z")',
          },
        },
      },
    };
    render(<IssueDetailRoleGrantDetails />);
    expect(
      await screen.findByText(/project.members.ddl-current-all/)
    ).toBeInTheDocument();
    // No env list rendered for binding-all.
    expect(screen.queryByText(/common.environments/)).not.toBeInTheDocument();
  });

  test("renders binding-all warning when expression is empty entirely", async () => {
    mockContextRef.current = {
      issue: {
        roleGrant: {
          role: "roles/sqlEditorUser",
          condition: { expression: "" },
        },
      },
    };
    render(<IssueDetailRoleGrantDetails />);
    expect(
      await screen.findByText(/project.members.ddl-current-all/)
    ).toBeInTheDocument();
    expect(screen.queryByText(/common.environments/)).not.toBeInTheDocument();
  });

  test("hides env section entirely for empty env clause (degenerate binding-none)", async () => {
    // A request submitted with no envs selected serializes to
    // `environment_id in []` — degenerate, grants no env access. We hide
    // the env section to avoid suggesting approval-relevant DDL/DML risk
    // when the binding wouldn't grant DDL/DML anywhere anyway.
    mockContextRef.current = {
      issue: {
        roleGrant: {
          role: "roles/sqlEditorUser",
          condition: { expression: "resource.environment_id in []" },
        },
      },
    };
    render(<IssueDetailRoleGrantDetails />);
    // Wait for the async CEL parse to complete by polling that the loading
    // guard has lifted (envScope === undefined during parse hides everything).
    // After parse, if any warning were going to show it would be present;
    // assert all three are absent.
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(
      screen.queryByText(/project.members.ddl-current-none/)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/project.members.ddl-current-all/)
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/project.members.ddl-current-some/)
    ).not.toBeInTheDocument();
    expect(screen.queryByText(/common.environments/)).not.toBeInTheDocument();
  });

  test("hides warning when role has been deleted (helper returns undefined)", async () => {
    mockContextRef.current = {
      issue: {
        roleGrant: {
          role: "roles/wasDeleted",
          condition: { expression: 'resource.environment_id in ["prod"]' },
        },
      },
    };
    render(<IssueDetailRoleGrantDetails />);
    expect(
      screen.queryByText(/project.members.ddl-current-some/)
    ).not.toBeInTheDocument();
  });

  test("hides warning when role lacks DDL/DML perms", async () => {
    mockContextRef.current = {
      issue: {
        roleGrant: {
          role: "roles/queryOnly",
          condition: { expression: 'resource.environment_id in ["prod"]' },
        },
      },
    };
    render(<IssueDetailRoleGrantDetails />);
    expect(
      screen.queryByText(/project.members.ddl-current-some/)
    ).not.toBeInTheDocument();
  });

  test("clears stale environments when the issue prop changes", async () => {
    mockContextRef.current = {
      issue: {
        roleGrant: {
          role: "roles/sqlEditorUser",
          condition: { expression: 'resource.environment_id in ["prod"]' },
        },
      },
    };
    const { rerender } = render(<IssueDetailRoleGrantDetails />);
    expect(await screen.findByText("Prod")).toBeInTheDocument();

    mockContextRef.current = {
      issue: {
        roleGrant: {
          role: "roles/sqlEditorUser",
          condition: { expression: 'resource.environment_id in ["test"]' },
        },
      },
    };
    rerender(<IssueDetailRoleGrantDetails />);

    // Without the synchronous setCondition(undefined), stale "Prod" leaks
    // into the env row until the async CEL parse for "test" resolves.
    expect(screen.queryByText("Prod")).not.toBeInTheDocument();

    // After the new parse, "Test" shows; "Prod" stays absent.
    expect(await screen.findByText("Test")).toBeInTheDocument();
    expect(screen.queryByText("Prod")).not.toBeInTheDocument();
  });
});
