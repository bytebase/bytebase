import { describe, expect, test, vi } from "vitest";
import { getRoleEnvironmentLimitationKind } from "./utils";

const fixtures: Record<string, string[]> = {
  "roles/sqlEditorUser": ["bb.sql.ddl", "bb.sql.dml"],
  "roles/sqlEditorDDLOnly": ["bb.sql.ddl"],
  "roles/sqlEditorDMLOnly": ["bb.sql.dml"],
  "roles/queryOnly": ["bb.sql.select"],
  "roles/projectViewer": [],
};
vi.mock("@/store", () => ({
  useRoleStore: () => ({
    getRoleByName: (role: string) =>
      fixtures[role] === undefined
        ? undefined
        : { name: role, permissions: fixtures[role] },
  }),
}));
vi.mock("@/utils", () => ({
  displayRoleTitle: (r: string) => r,
  checkRoleContainsAnyPermission: () => false,
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
