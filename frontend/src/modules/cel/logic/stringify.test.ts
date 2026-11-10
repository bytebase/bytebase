// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  type Expr,
  ExprSchema,
} from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import stringifyExpr from "./stringify";

const ident = (name: string) =>
  create(ExprSchema, { exprKind: { case: "identExpr", value: { name } } });
const call = (fn: string, args: Expr[]) =>
  create(ExprSchema, {
    exprKind: { case: "callExpr", value: { function: fn, args } },
  });

describe("stringifyExpr", () => {
  it("keeps unary negation", () => {
    expect(stringifyExpr(call("!_", [ident("request.export")]))).toBe(
      "!(request.export)"
    );
  });

  it("keeps unary arithmetic negation in computed arguments", () => {
    expect(
      stringifyExpr(call("-_", [ident("resource.classification_level")]))
    ).toBe("-(resource.classification_level)");
  });

  it("keeps global function names and argument boundaries", () => {
    expect(
      stringifyExpr(call("custom_function", [ident("a"), ident("b")]))
    ).toBe("custom_function(a, b)");
  });

  it("uses CEL membership syntax in raw expressions", () => {
    expect(stringifyExpr(call("@in", [ident("a"), ident("b")]))).toBe("a in b");
  });
});
