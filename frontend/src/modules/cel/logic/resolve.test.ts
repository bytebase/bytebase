// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { ExprType, type SimpleExpr } from "@/modules/cel";
import {
  type Expr,
  ExprSchema,
} from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import { CEL_ATTRIBUTE_ISSUE_LABELS } from "@/utils/cel-attributes";
import { buildCELExpr } from "./build";
import { resolveCELExpr } from "./resolve";

const ident = (name: string) =>
  create(ExprSchema, { exprKind: { case: "identExpr", value: { name } } });
const string = (value: string) =>
  create(ExprSchema, {
    exprKind: {
      case: "constExpr",
      value: { constantKind: { case: "stringValue", value } },
    },
  });
const call = (fn: string, args: Expr[], target?: Expr) =>
  create(ExprSchema, {
    exprKind: { case: "callExpr", value: { function: fn, args, target } },
  });

describe("resolveCELExpr", () => {
  describe.each(["_&&_", "_||_"] as const)("%s groups", (operator) => {
    const comparisons = [
      ["resource.project_id", "example"],
      ["resource.instance_id", "primary"],
      ["resource.database_name", "app"],
      ["resource.column_name", "email"],
    ] as const;
    const leaves = comparisons.map(([factor, value]) =>
      call("_==_", [ident(factor), string(value)])
    );
    const conditions = comparisons.map(([factor, value]) => ({
      type: ExprType.Condition as const,
      operator: "_==_" as const,
      args: [factor, value] as [typeof factor, string],
    }));
    const flat: SimpleExpr = {
      type: ExprType.ConditionGroup,
      operator,
      args: conditions,
    };

    it.each([
      [
        "balanced",
        call(operator, [
          call(operator, [leaves[0], leaves[1]]),
          call(operator, [leaves[2], leaves[3]]),
        ]),
      ],
      [
        "left-associated",
        call(operator, [
          call(operator, [call(operator, [leaves[0], leaves[1]]), leaves[2]]),
          leaves[3],
        ]),
      ],
      [
        "right-associated",
        call(operator, [
          leaves[0],
          call(operator, [leaves[1], call(operator, [leaves[2], leaves[3]])]),
        ]),
      ],
    ])("flattens a %s tree into four conditions", (_, parsed) => {
      expect(resolveCELExpr(parsed)).toEqual(flat);
    });

    it.each(["left", "right"] as const)(
      "preserves a different logical operator on the %s",
      (side) => {
        const otherOperator = operator === "_&&_" ? "_||_" : "_&&_";
        const left = call(side === "left" ? otherOperator : operator, [
          leaves[0],
          leaves[1],
        ]);
        const right = call(side === "right" ? otherOperator : operator, [
          leaves[2],
          leaves[3],
        ]);
        const nested = {
          type: ExprType.ConditionGroup,
          operator: otherOperator,
          args: side === "left" ? conditions.slice(0, 2) : conditions.slice(2),
        };

        expect(resolveCELExpr(call(operator, [left, right]))).toEqual({
          type: ExprType.ConditionGroup,
          operator,
          args:
            side === "left"
              ? [nested, ...conditions.slice(2)]
              : [...conditions.slice(0, 2), nested],
        });
      }
    );

    it("round-trips a flat group through the CEL builder", async () => {
      const built = await buildCELExpr(flat);
      expect(built).toBeDefined();
      if (!built) throw new Error("expected built CEL expression");
      expect(resolveCELExpr(built)).toEqual(flat);
    });
  });

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
