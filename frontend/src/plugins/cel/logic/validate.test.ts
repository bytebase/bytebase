import { describe, expect, it } from "vitest";
import { ExprType, type SimpleExpr } from "@/plugins/cel";
import { CEL_ATTRIBUTE_ISSUE_LABELS } from "@/utils/cel-attributes";
import { validateSimpleExpr } from "./validate";

describe("validateSimpleExpr", () => {
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
