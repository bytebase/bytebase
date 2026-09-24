// @vitest-environment node
import { describe, expect, test } from "vitest";
import { directExecutionScopeFromCondition } from "./directExecutionScope";

describe("directExecutionScopeFromCondition", () => {
  test("no condition at all is unscoped", () => {
    expect(directExecutionScopeFromCondition(undefined, "")).toEqual({
      type: "all",
    });
  });

  test("a condition without an environment clause is unscoped", () => {
    expect(
      directExecutionScopeFromCondition(
        { databaseResources: [], expiredTime: "2026-10-01" },
        'request.time < timestamp("2026-10-01T00:00:00Z")'
      )
    ).toEqual({ type: "all" });
  });

  test("an empty list is the switch off", () => {
    expect(
      directExecutionScopeFromCondition(
        { environments: [] },
        "resource.environment_id in []"
      )
    ).toEqual({ type: "none" });
  });

  test("a list is the switch on, in order", () => {
    expect(
      directExecutionScopeFromCondition(
        { environments: ["environments/staging", "environments/prod"] },
        'resource.environment_id in ["staging", "prod"]'
      )
    ).toEqual({
      type: "some",
      environments: ["environments/staging", "environments/prod"],
    });
  });

  test("an unrecognized condition wins over whatever was decoded from it", () => {
    const expression = 'resource.environment_id in ["staging"] || true';
    expect(
      directExecutionScopeFromCondition(
        { environments: ["environments/staging"], unrecognized: true },
        expression
      )
    ).toEqual({ type: "custom", expression });
  });
});
