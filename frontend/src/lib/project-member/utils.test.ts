// @vitest-environment node
import { describe, expect, test, vi } from "vitest";
import {
  getRoleEnvironmentLimitationKind,
  getRolesEnvironmentLimitationKind,
} from "./utils";

const fixtures: Record<string, string[]> = {
  "roles/sqlEditorUser": ["bb.sql.ddl", "bb.sql.dml"],
  "roles/sqlEditorDDLOnly": ["bb.sql.ddl"],
  "roles/sqlEditorDMLOnly": ["bb.sql.dml"],
  "roles/queryOnly": ["bb.sql.select"],
  "roles/projectViewer": [],
};
vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => ({ roleList: [] }) },
}));
vi.mock("@/utils", () => ({
  displayRoleTitle: (r: string) => r,
  checkRoleContainsAnyPermission: (role: string, ...permissions: string[]) =>
    permissions.some((permission) => fixtures[role]?.includes(permission)),
}));

describe("getRoleEnvironmentLimitationKind", () => {
  test("returns 'DDL/DML' when role has both ddl and dml", () => {
    expect(getRoleEnvironmentLimitationKind("roles/sqlEditorUser")).toBe(
      "DDL/DML"
    );
  });

  test("returns 'DDL' when role has only ddl", () => {
    expect(getRoleEnvironmentLimitationKind("roles/sqlEditorDDLOnly")).toBe(
      "DDL"
    );
  });

  test("returns 'DML' when role has only dml", () => {
    expect(getRoleEnvironmentLimitationKind("roles/sqlEditorDMLOnly")).toBe(
      "DML"
    );
  });

  test("returns undefined when role has neither ddl nor dml", () => {
    expect(getRoleEnvironmentLimitationKind("roles/queryOnly")).toBeUndefined();
  });

  test("returns undefined for an unknown role", () => {
    expect(
      getRoleEnvironmentLimitationKind("roles/doesNotExist")
    ).toBeUndefined();
  });
});

describe("getRolesEnvironmentLimitationKind", () => {
  test("unions the roles' permissions", () => {
    expect(
      getRolesEnvironmentLimitationKind([
        "roles/sqlEditorDDLOnly",
        "roles/sqlEditorDMLOnly",
      ])
    ).toBe("DDL/DML");
    expect(getRolesEnvironmentLimitationKind(["roles/sqlEditorDDLOnly"])).toBe(
      "DDL"
    );
  });

  test("ignores roles without the permissions and unknown roles", () => {
    expect(
      getRolesEnvironmentLimitationKind(["roles/queryOnly", "roles/missing"])
    ).toBeUndefined();
    expect(getRolesEnvironmentLimitationKind([])).toBeUndefined();
    expect(
      getRolesEnvironmentLimitationKind([
        "roles/missing",
        "roles/sqlEditorUser",
      ])
    ).toBe("DDL/DML");
  });
});
