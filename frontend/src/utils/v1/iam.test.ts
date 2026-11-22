// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import * as projectAccess from "@/stores/app/projectAccess";
import {
  type Expr,
  ExprSchema,
} from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import {
  type Binding,
  BindingSchema,
  IamPolicySchema,
} from "@/types/proto-es/v1/iam_policy_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { RoleSchema } from "@/types/proto-es/v1/role_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import * as appStoreBridge from "@/utils/app-store-bridge";
import {
  bindingScopesResources,
  checkQuerierPermission,
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

describe("checkQuerierPermission", () => {
  const database = create(DatabaseSchema$, {
    name: "instances/i/databases/app",
    project: "projects/test",
  });
  let bindings: Binding[];

  const grant = (parsedExpr: Expr) => {
    bindings = [
      create(BindingSchema, {
        role: "roles/projectQuerier",
        members: ["user:alice@example.com"],
        parsedExpr,
      }),
    ];
  };

  beforeEach(() => {
    bindings = [];
    vi.spyOn(projectAccess, "getProjectByName").mockReturnValue(
      create(ProjectSchema, { name: database.project })
    );
    vi.spyOn(appStoreBridge, "appStoreUtilBridge").mockReturnValue({
      currentUser: () => create(UserSchema, { email: "alice@example.com" }),
      getProjectIamPolicy: () => create(IamPolicySchema, { bindings }),
      getRoleByName: () =>
        create(RoleSchema, {
          name: "roles/projectQuerier",
          permissions: ["bb.sql.select"],
        }),
    } as unknown as appStoreBridge.AppStoreUtilBridge);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  const pattern = (operator: string) =>
    create(ExprSchema, {
      exprKind: {
        case: "callExpr",
        value: {
          function: operator,
          target: resource("table_name"),
          args: [string("archive")],
        },
      },
    });

  test.each(["contains", "startsWith", "endsWith", "matches"])(
    "allows query entry for a potentially matching %s binding",
    (operator) => {
      grant(pattern(operator));

      expect(checkQuerierPermission(database)).toBe(true);
    }
  );

  test("denies a pattern binding with a guaranteed past expiration", () => {
    grant(
      call("_&&_", [pattern("contains"), timeBefore("2020-01-01T00:00:00Z")])
    );

    expect(checkQuerierPermission(database)).toBe(false);
  });

  test("does not treat an expiration in an OR branch as guaranteed", () => {
    grant(
      call("_||_", [pattern("contains"), timeBefore("2020-01-01T00:00:00Z")])
    );

    expect(checkQuerierPermission(database)).toBe(true);
  });

  test("denies a known scope on a different database", () => {
    grant(eq("database", "instances/i/databases/other"));

    expect(checkQuerierPermission(database)).toBe(false);
  });

  test.each(["_==_", "@in"])(
    "respects a table-only %s scope across databases",
    (operator) => {
      const value = string("orders");
      grant(
        call(operator, [
          resource("table_name"),
          operator === "@in"
            ? create(ExprSchema, {
                exprKind: { case: "listExpr", value: { elements: [value] } },
              })
            : value,
        ])
      );

      expect(checkQuerierPermission(database)).toBe(true);
      expect(
        checkQuerierPermission(database, ["bb.sql.select"], "public")
      ).toBe(true);
      expect(
        checkQuerierPermission(database, ["bb.sql.select"], "public", "orders")
      ).toBe(true);
      expect(
        checkQuerierPermission(database, ["bb.sql.select"], "public", "audit")
      ).toBe(false);
    }
  );

  test.each(["_==_", "@in"])(
    "respects a schema-only %s scope across databases",
    (operator) => {
      const value = string("public");
      grant(
        call(operator, [
          resource("schema_name"),
          operator === "@in"
            ? create(ExprSchema, {
                exprKind: { case: "listExpr", value: { elements: [value] } },
              })
            : value,
        ])
      );

      expect(checkQuerierPermission(database)).toBe(true);
      expect(
        checkQuerierPermission(database, ["bb.sql.select"], "public", "orders")
      ).toBe(true);
      expect(
        checkQuerierPermission(database, ["bb.sql.select"], "audit", "orders")
      ).toBe(false);
    }
  );

  test("respects a table constraint within a known database", () => {
    grant(
      call("_&&_", [eq("database", database.name), eq("table_name", "orders")])
    );

    expect(checkQuerierPermission(database)).toBe(true);
    expect(
      checkQuerierPermission(database, ["bb.sql.select"], "public", "orders")
    ).toBe(true);
    expect(
      checkQuerierPermission(database, ["bb.sql.select"], "public", "audit")
    ).toBe(false);
  });

  test("still requires the requested permission for unknown scopes", () => {
    grant(pattern("contains"));

    expect(checkQuerierPermission(database, ["bb.sql.export"])).toBe(false);
  });

  test("still requires binding membership for unknown scopes", () => {
    grant(pattern("contains"));
    bindings[0].members = ["user:bob@example.com"];

    expect(checkQuerierPermission(database)).toBe(false);
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
