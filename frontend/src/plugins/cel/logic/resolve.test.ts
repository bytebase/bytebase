import { describe, expect, it } from "vitest";
import { ExprType, type SimpleExpr } from "@/plugins/cel";
import { CEL_ATTRIBUTE_ISSUE_LABELS } from "@/utils/cel-attributes";
import { buildCELExpr } from "./build";
import { resolveCELExpr } from "./resolve";

describe("resolveCELExpr", () => {
  it("round-trips list membership from built CEL", async () => {
    const expr: SimpleExpr = {
      type: ExprType.Condition,
      operator: "@contains",
      args: [CEL_ATTRIBUTE_ISSUE_LABELS, "prod"],
    };

    const built = await buildCELExpr(expr);

    expect(built).toBeDefined();
    if (!built) {
      throw new Error("expected built CEL expression");
    }
    expect(resolveCELExpr(built)).toEqual(expr);
  });

  it("round-trips negated list membership from built CEL", async () => {
    const expr: SimpleExpr = {
      type: ExprType.Condition,
      operator: "@not_contains",
      args: [CEL_ATTRIBUTE_ISSUE_LABELS, "prod"],
    };

    const built = await buildCELExpr(expr);

    expect(built).toBeDefined();
    if (!built) {
      throw new Error("expected built CEL expression");
    }
    expect(resolveCELExpr(built)).toEqual(expr);
  });
});
