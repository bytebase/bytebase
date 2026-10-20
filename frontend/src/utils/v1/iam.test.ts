// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  type Expr,
  ExprSchema,
} from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import {
  bindingScopesResources,
  convertFullnameToMember,
  convertMemberToFullname,
  isBindingPolicyExpired,
} from "./iam";

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
const string = (value: string) =>
  create(ExprSchema, {
    exprKind: {
      case: "constExpr",
      value: { constantKind: { case: "stringValue", value } },
    },
  });
const resource = (field: string) => select(ident("resource"), field);
const eq = (field: string, value: string) =>
  call("_==_", [resource(field), string(value)]);
const timeBefore = (value: string) =>
  call("_<_", [
    select(ident("request"), "time"),
    call("timestamp", [string(value)]),
  ]);

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

describe("isBindingPolicyExpired", () => {
  test("keeps checking expiry when resource branches cannot be summarized", () => {
    const expr = call("_&&_", [
      call("_||_", [
        eq("database", "instances/i/databases/a"),
        eq("database", "instances/i/databases/b"),
      ]),
      call("_||_", [eq("schema_name", "public"), eq("schema_name", "audit")]),
      timeBefore("2020-01-01T00:00:00Z"),
    ]);

    expect(isBindingPolicyExpired(binding(undefined, expr))).toBe(true);
  });
});

describe("member fullname conversion", () => {
  test("round-trips supported IAM member identifiers", () => {
    const members = [
      "allUsers",
      "user:alice@example.com",
      "group:engineering@example.com",
      "serviceAccount:deployer@service.bytebase.com",
      "workloadIdentity:ci@workload.bytebase.com",
    ];

    expect(members.map(convertMemberToFullname)).toEqual([
      "users/allUsers",
      "users/alice@example.com",
      "groups/engineering@example.com",
      "serviceAccounts/deployer@service.bytebase.com",
      "workloadIdentities/ci@workload.bytebase.com",
    ]);
    expect(
      members.map(convertMemberToFullname).map(convertFullnameToMember)
    ).toEqual(members);
  });
});
