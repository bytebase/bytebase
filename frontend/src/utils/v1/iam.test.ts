import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  type Expr,
  ExprSchema,
} from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { bindingScopesResources } from "./iam";

const binding = (expression?: string, parsedExpr?: Expr) =>
  ({
    condition: expression === undefined ? undefined : { expression },
    parsedExpr,
  }) as Binding;

const ident = (name: string) =>
  create(ExprSchema, { exprKind: { case: "identExpr", value: { name } } });
const select = (operand: Expr, field: string) =>
  create(ExprSchema, {
    exprKind: { case: "selectExpr", value: { operand, field } },
  });
const call = (fn: string, args: Expr[]) =>
  create(ExprSchema, {
    exprKind: { case: "callExpr", value: { function: fn, args } },
  });

// Mirrors the server's rule: any condition attribute other than request.time
// scopes the binding away from project-wide surfaces.
describe("bindingScopesResources", () => {
  test("no condition or expiry-only conditions are project-wide", () => {
    expect(bindingScopesResources(binding())).toBe(false);
    expect(bindingScopesResources(binding(""))).toBe(false);
    expect(
      bindingScopesResources(
        binding('request.time < timestamp("2099-01-01T00:00:00Z")')
      )
    ).toBe(false);
  });

  test("resource attributes scope the binding (string fallback)", () => {
    expect(
      bindingScopesResources(
        binding('resource.database == "instances/i/databases/d"')
      )
    ).toBe(true);
    expect(
      bindingScopesResources(
        binding(
          'resource.environment_id == "prod" && request.time < timestamp("2099-01-01T00:00:00Z")'
        )
      )
    ).toBe(true);
    expect(
      bindingScopesResources(binding('resource.schema_name == "public"'))
    ).toBe(true);
  });

  test("the parsed condition is authoritative when present", () => {
    const requestTimeOnly = call("_<_", [
      select(ident("request"), "time"),
      call("timestamp", []),
    ]);
    expect(
      bindingScopesResources(
        binding(
          'request.time < timestamp("2099-01-01T00:00:00Z")',
          requestTimeOnly
        )
      )
    ).toBe(false);

    const resourceScoped = call("_==_", [
      select(ident("resource"), "database"),
    ]);
    // Whitespace-formatted CEL ("resource . database") defeats the substring
    // fallback, but the parsed AST still names the reference — the server
    // walks the same tree.
    expect(
      bindingScopesResources(
        binding(
          'resource . database == "instances/i/databases/d"',
          resourceScoped
        )
      )
    ).toBe(true);

    const mixed = call("_&&_", [requestTimeOnly, resourceScoped]);
    expect(bindingScopesResources(binding("...", mixed))).toBe(true);
  });
});
