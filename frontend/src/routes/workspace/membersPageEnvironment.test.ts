// @vitest-environment node
import { describe, expect, test, vi } from "vitest";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { getProjectRoleBindingDirectExecutionScope } from "./membersPageEnvironment";

vi.mock("@/lib/project-member/utils", () => ({
  getRoleEnvironmentLimitationKind: (role: string) =>
    role === "roles/sqlEditorUser" ? "DDL/DML" : undefined,
}));

vi.mock("@/utils/issue/cel", () => ({
  convertFromExpr: (expr: {
    environments?: string[];
    unrecognized?: true;
  }) => ({
    environments: expr.environments,
    unrecognized: expr.unrecognized,
  }),
}));

const binding = (
  role: string,
  parsed?: { environments?: string[]; unrecognized?: true },
  expression = ""
): Binding =>
  ({
    role,
    parsedExpr: parsed,
    condition: { expression },
  }) as unknown as Binding;

describe("getProjectRoleBindingDirectExecutionScope", () => {
  test("a role without DDL/DML has no scope to show", () => {
    expect(
      getProjectRoleBindingDirectExecutionScope(
        binding("roles/viewer", { environments: [] })
      )
    ).toBeUndefined();
  });

  test("no parsed condition is unscoped", () => {
    expect(
      getProjectRoleBindingDirectExecutionScope(binding("roles/sqlEditorUser"))
    ).toEqual({ type: "all" });
  });

  test("a condition without an environment clause is unscoped", () => {
    expect(
      getProjectRoleBindingDirectExecutionScope(
        binding("roles/sqlEditorUser", {})
      )
    ).toEqual({ type: "all" });
  });

  test("an explicit empty list is the switch off", () => {
    expect(
      getProjectRoleBindingDirectExecutionScope(
        binding("roles/sqlEditorUser", { environments: [] })
      )
    ).toEqual({ type: "none" });
  });

  test("a list is the switch on", () => {
    expect(
      getProjectRoleBindingDirectExecutionScope(
        binding("roles/sqlEditorUser", {
          environments: ["environments/staging"],
        })
      )
    ).toEqual({ type: "some", environments: ["environments/staging"] });
  });

  test("an unrecognized condition shows its raw expression", () => {
    const expression = 'resource.environment_id in ["staging"] || true';
    expect(
      getProjectRoleBindingDirectExecutionScope(
        binding(
          "roles/sqlEditorUser",
          { environments: ["environments/staging"], unrecognized: true },
          expression
        )
      )
    ).toEqual({ type: "custom", expression });
  });
});
