import { describe, expect, it } from "vitest";
import { ExprType, type SimpleExpr } from "@/plugins/cel";
import {
  CEL_ATTRIBUTE_ISSUE_LABELS,
  CEL_ATTRIBUTE_STATEMENT_TEXT,
} from "@/utils/cel-attributes";
import { buildCELExpr } from "./build";

describe("buildCELExpr", () => {
  it("builds label membership as value in issue.labels", async () => {
    const expr: SimpleExpr = {
      type: ExprType.Condition,
      operator: "@contains",
      args: [CEL_ATTRIBUTE_ISSUE_LABELS, "prod"],
    };

    const built = await buildCELExpr(expr);

    expect(built?.exprKind.case).toBe("callExpr");
    if (built?.exprKind.case !== "callExpr") {
      throw new Error("expected callExpr");
    }
    const call = built.exprKind.value;
    expect(call.function).toBe("@in");

    const valueExpr = call.args[0];
    expect(valueExpr.exprKind.case).toBe("constExpr");
    if (valueExpr.exprKind.case !== "constExpr") {
      throw new Error("expected constExpr");
    }
    const constantKind = valueExpr.exprKind.value.constantKind;
    expect(constantKind.case).toBe("stringValue");
    if (constantKind.case !== "stringValue") {
      throw new Error("expected stringValue");
    }
    expect(constantKind.value).toBe("prod");

    const factorExpr = call.args[1];
    expect(factorExpr.exprKind.case).toBe("identExpr");
    if (factorExpr.exprKind.case !== "identExpr") {
      throw new Error("expected identExpr");
    }
    expect(factorExpr.exprKind.value.name).toBe(CEL_ATTRIBUTE_ISSUE_LABELS);
  });

  it("keeps string not contains as a target call", async () => {
    const expr: SimpleExpr = {
      type: ExprType.Condition,
      operator: "@not_contains",
      args: [CEL_ATTRIBUTE_STATEMENT_TEXT, "drop"],
    };

    const built = await buildCELExpr(expr);

    expect(built?.exprKind.case).toBe("callExpr");
    if (built?.exprKind.case !== "callExpr") {
      throw new Error("expected callExpr");
    }
    const negation = built.exprKind.value;
    expect(negation.function).toBe("!_");
    const containsExpr = negation.args[0];
    expect(containsExpr.exprKind.case).toBe("callExpr");
    if (containsExpr.exprKind.case !== "callExpr") {
      throw new Error("expected callExpr");
    }
    const contains = containsExpr.exprKind.value;
    expect(contains.function).toBe("contains");
    expect(contains.target?.exprKind.case).toBe("identExpr");
    if (contains.target?.exprKind.case !== "identExpr") {
      throw new Error("expected identExpr target");
    }
    expect(contains.target.exprKind.value.name).toBe(
      CEL_ATTRIBUTE_STATEMENT_TEXT
    );
  });
});
