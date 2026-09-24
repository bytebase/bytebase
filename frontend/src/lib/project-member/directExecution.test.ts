// @vitest-environment node
import { describe, expect, test, vi } from "vitest";
import {
  directExecutionEnvironments,
  directExecutionOf,
  EMPTY_DIRECT_EXECUTION,
  isDirectExecutionValid,
} from "./directExecution";

vi.mock("./utils", () => ({
  getRoleEnvironmentLimitationKind: (role: string) =>
    role === "roles/sqlEditorUser" ? "DDL/DML" : undefined,
}));

const scopeOf = (condition?: {
  environments?: string[];
  expiredTime?: string;
}) => directExecutionOf("roles/sqlEditorUser", condition)?.scope;

describe("directExecutionOf", () => {
  test("a role without DDL/DML has nothing to show", () => {
    expect(
      directExecutionOf("roles/viewer", { environments: [] })
    ).toBeUndefined();
  });

  test("no condition, or one without an environment clause, is unscoped", () => {
    expect(scopeOf(undefined)).toEqual({ type: "all" });
    expect(scopeOf({ expiredTime: "2026-10-01" })).toEqual({ type: "all" });
  });

  test("an empty list is the switch off; a list is the switch on, in order", () => {
    expect(scopeOf({ environments: [] })).toEqual({ type: "none" });
    expect(
      scopeOf({ environments: ["environments/staging", "environments/prod"] })
    ).toEqual({
      type: "some",
      environments: ["environments/staging", "environments/prod"],
    });
    expect(directExecutionOf("roles/sqlEditorUser", undefined)?.kind).toBe(
      "DDL/DML"
    );
  });
});

describe("the form value", () => {
  test("only on-with-nothing-picked is invalid", () => {
    expect(isDirectExecutionValid(EMPTY_DIRECT_EXECUTION)).toBe(true);
    expect(isDirectExecutionValid({ enabled: true, environments: [] })).toBe(
      false
    );
    expect(
      isDirectExecutionValid({
        enabled: true,
        environments: ["environments/prod"],
      })
    ).toBe(true);
  });

  test("off always submits the empty list; on submits the picked list", () => {
    expect(
      directExecutionEnvironments({
        enabled: false,
        environments: ["environments/prod"],
      })
    ).toEqual([]);
    expect(
      directExecutionEnvironments({
        enabled: true,
        environments: ["environments/prod"],
      })
    ).toEqual(["environments/prod"]);
  });
});
