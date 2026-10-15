import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  type Expr,
  ExprSchema,
} from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import {
  convertFromExpr,
  readableCondition,
  stringifyConditionExpression,
} from "./cel";

// Trees shaped as the server's parser returns them.
const ident = (name: string) =>
  create(ExprSchema, { exprKind: { case: "identExpr", value: { name } } });
const select = (operand: Expr, field: string) =>
  create(ExprSchema, {
    exprKind: { case: "selectExpr", value: { operand, field } },
  });
const call = (fn: string, ...args: Expr[]) =>
  create(ExprSchema, {
    exprKind: { case: "callExpr", value: { function: fn, args } },
  });
const constant = (
  constantKind:
    | { case: "stringValue"; value: string }
    | { case: "boolValue"; value: boolean }
    | { case: "int64Value"; value: bigint }
) =>
  create(ExprSchema, {
    exprKind: { case: "constExpr", value: { constantKind } },
  });
const str = (value: string) => constant({ case: "stringValue", value });
const bool = (value: boolean) => constant({ case: "boolValue", value });
const int = (value: number) =>
  constant({ case: "int64Value", value: BigInt(value) });
const list = (...elements: Expr[]) =>
  create(ExprSchema, { exprKind: { case: "listExpr", value: { elements } } });
const resource = (field: string) => select(ident("resource"), field);
const eq = (attribute: string, value: string) =>
  call("_==_", resource(attribute), str(value));
const envIn = (...ids: string[]) =>
  call("@in", resource("environment_id"), list(...ids.map(str)));
const timeBefore = (iso: string) =>
  call("_<_", select(ident("request"), "time"), call("timestamp", str(iso)));
const and = (...exprs: Expr[]) =>
  exprs.reduce((left, right) => call("_&&_", left, right));
const or = (...exprs: Expr[]) =>
  exprs.reduce((left, right) => call("_||_", left, right));

const environments = (expr: Expr) => readableCondition(expr).environments;
const A = "instances/i/databases/a";

describe("stringifyConditionExpression", () => {
  test("the switch off is byte-for-byte the empty picker's clause", () => {
    expect(stringifyConditionExpression({ environments: [] })).toBe(
      "resource.environment_id in []"
    );
    expect(stringifyConditionExpression({ environments: undefined })).toBe("");
    expect(
      stringifyConditionExpression({
        environments: ["environments/staging", "environments/prod"],
      })
    ).toBe('resource.environment_id in ["staging", "prod"]');
  });
});

describe("readableCondition proves the environments and keeps the rest best-effort", () => {
  test("the environment clause alone, and beside the other clauses the forms write", () => {
    expect(environments(envIn())).toEqual([]);
    expect(environments(envIn("test", "test"))).toEqual(["environments/test"]);
    expect(
      readableCondition(
        and(
          eq("database", A),
          timeBefore("2026-10-01T00:00:00Z"),
          envIn("prod")
        )
      )
    ).toEqual({
      databaseResources: [{ databaseFullName: A }],
      expiredTime: "2026-10-01T00:00:00.000Z",
      environments: ["environments/prod"],
    });
  });

  test("the frontend builder's dotted ident is read too", () => {
    expect(
      environments(
        call("@in", ident("resource.environment_id"), list(str("prod")))
      )
    ).toEqual(["environments/prod"]);
  });

  test("a clause under || or ! does not count: `in [] || true` is not the switch off", () => {
    expect(convertFromExpr(or(envIn(), bool(true))).environments).toEqual([]);
    expect(environments(or(envIn(), bool(true)))).toBeUndefined();
    expect(environments(or(envIn("prod"), envIn("dev")))).toBeUndefined();
    expect(
      environments(and(bool(true), or(envIn("prod"), bool(true))))
    ).toBeUndefined();
    expect(environments(call("!_", envIn()))).toBeUndefined();
  });

  test("a computed or non-string environment list does not count", () => {
    expect(
      environments(
        call(
          "@in",
          resource("environment_id"),
          call("_+_", list(str("prod")), list(str("dev")))
        )
      )
    ).toBeUndefined();
    expect(
      environments(call("@in", resource("environment_id"), list(int(1))))
    ).toBeUndefined();
  });

  test("two root environment clauses intersect", () => {
    expect(environments(and(envIn("prod", "dev"), envIn("dev")))).toEqual([
      "environments/dev",
    ]);
  });
});
