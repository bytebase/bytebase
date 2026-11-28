// @vitest-environment node
import { describe, expect, it } from "vitest";
import { ExprType, type SimpleExpr } from "@/modules/cel";
import { CEL_ATTRIBUTE_ISSUE_LABELS } from "@/utils/cel-attributes";
import { validateSimpleExpr } from "./validate";

describe("validateSimpleExpr", () => {
  it.each([
    ["@contains", "groups/analysts@example.com"],
    ["@not_contains", "groups/42"],
  ] as const)("accepts group membership with %s", (operator, group) => {
    expect(
      validateSimpleExpr({
        type: ExprType.Condition,
        operator,
        args: ["principal.groups", group],
      })
    ).toBe(true);
  });

  it("rejects group membership before a group is selected", () => {
    expect(
      validateSimpleExpr({
        type: ExprType.Condition,
        operator: "@contains",
        args: ["principal.groups", ""],
      })
    ).toBe(false);
  });

  it("accepts valid list membership", () => {
    const expr: SimpleExpr = {
      type: ExprType.Condition,
      operator: "@contains",
      args: [CEL_ATTRIBUTE_ISSUE_LABELS, "prod"],
    };

    expect(validateSimpleExpr(expr)).toBe(true);
  });

  it("rejects list membership with empty string value", () => {
    const expr: SimpleExpr = {
      type: ExprType.Condition,
      operator: "@contains",
      args: [CEL_ATTRIBUTE_ISSUE_LABELS, ""],
    };

    expect(validateSimpleExpr(expr)).toBe(false);
  });
});
