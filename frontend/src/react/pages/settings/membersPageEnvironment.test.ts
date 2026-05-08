import { describe, expect, test, vi } from "vitest";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { getProjectRoleBindingEnvironmentLimitationState } from "./membersPageEnvironment";

vi.mock("@/components/ProjectMember/utils", () => ({
  getRoleEnvironmentLimitationKind: (role: string) =>
    role === "roles/sqlEditorUser" ? "DDL/DML" : undefined,
}));

vi.mock("@/utils/issue/cel", () => ({
  convertFromExpr: (expr: { environments?: string[] }) => ({
    environments: expr.environments,
  }),
}));

const binding = (role: string, environments?: string[]): Binding =>
  ({
    role,
    parsedExpr: environments === undefined ? {} : { environments },
  }) as Binding;

const bindingWithoutParsedExpr = (role: string): Binding =>
  ({
    role,
  }) as Binding;

describe("getProjectRoleBindingEnvironmentLimitation", () => {
  test("returns undefined for roles without DDL or DML environment restrictions", () => {
    expect(
      getProjectRoleBindingEnvironmentLimitationState(
        binding("roles/viewer", [])
      )
    ).toBeUndefined();
  });

  test("returns unrestricted when a DDL/DML role has no environment condition", () => {
    expect(
      getProjectRoleBindingEnvironmentLimitationState(
        binding("roles/sqlEditorUser")
      )
    ).toEqual({
      type: "unrestricted",
    });
  });

  test("returns unrestricted when a DDL/DML role has no parsed condition", () => {
    expect(
      getProjectRoleBindingEnvironmentLimitationState(
        bindingWithoutParsedExpr("roles/sqlEditorUser")
      )
    ).toEqual({
      type: "unrestricted",
    });
  });

  test("preserves explicit empty environment restrictions", () => {
    expect(
      getProjectRoleBindingEnvironmentLimitationState(
        binding("roles/sqlEditorUser", [])
      )
    ).toEqual({
      environments: [],
      type: "restricted",
    });
  });
});
