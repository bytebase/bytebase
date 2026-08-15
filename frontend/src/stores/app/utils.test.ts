import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import {
  BindingSchema,
  IamPolicySchema,
} from "@/types/proto-es/v1/iam_policy_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { buildDatabaseFilter, projectWideBindings } from "./utils";

describe("buildDatabaseFilter", () => {
  test("escapes the database name query as a CEL string literal", () => {
    expect(buildDatabaseFilter({ query: 'Payroll "Q3"\\West\nArchive' })).toBe(
      'name.contains("payroll \\"q3\\"\\\\west\\narchive")'
    );
  });
});

describe("projectWideBindings", () => {
  const me = create(UserSchema, {
    name: "users/me@example.com",
    groups: ["groups/eng"],
  });

  const policyWith = (bindings: MessageInitShape<typeof BindingSchema>[]) =>
    create(IamPolicySchema, {
      bindings: bindings.map((binding) => create(BindingSchema, binding)),
    });

  const roles = (policy: ReturnType<typeof policyWith>, skipAllUsers = false) =>
    projectWideBindings(policy, me, { skipAllUsers }).map((b) => b.role);

  test("matches the user directly, via group, and via allUsers", () => {
    const policy = policyWith([
      { role: "roles/direct", members: ["user:me@example.com"] },
      { role: "roles/viaGroup", members: ["group:eng"] },
      { role: "roles/viaAllUsers", members: ["allUsers"] },
      { role: "roles/somebodyElse", members: ["user:other@example.com"] },
    ]);
    expect(roles(policy)).toEqual([
      "roles/direct",
      "roles/viaGroup",
      "roles/viaAllUsers",
    ]);
  });

  test("skips allUsers when asked, as the server does for workspace policy in SaaS mode", () => {
    const policy = policyWith([
      { role: "roles/direct", members: ["user:me@example.com"] },
      { role: "roles/viaAllUsers", members: ["allUsers"] },
    ]);
    expect(roles(policy, true)).toEqual(["roles/direct"]);
  });

  test("a resource-scoped condition confers nothing", () => {
    const policy = policyWith([
      {
        role: "roles/scoped",
        members: ["user:me@example.com"],
        condition: {
          expression: 'resource.database == "instances/i/databases/d"',
        },
      },
    ]);
    expect(roles(policy)).toEqual([]);
  });

  test("honors the canonical expiry form in both directions", () => {
    const policy = policyWith([
      {
        role: "roles/live",
        members: ["user:me@example.com"],
        condition: {
          expression: 'request.time < timestamp("2099-01-01T00:00:00Z")',
        },
      },
      {
        role: "roles/expired",
        members: ["user:me@example.com"],
        condition: {
          expression: 'request.time < timestamp("2000-01-01T00:00:00Z")',
        },
      },
    ]);
    expect(roles(policy)).toEqual(["roles/live"]);
  });

  test("a time-only condition outside the canonical form hides the grant", () => {
    // The server evaluates the full CEL expression and would grant both of
    // these until 2099; the client cannot, so it hides the affordance
    // rather than showing one whose expiry it would never notice.
    const policy = policyWith([
      {
        role: "roles/reversed",
        members: ["user:me@example.com"],
        condition: {
          expression: 'timestamp("2099-01-01T00:00:00Z") > request.time',
        },
      },
      {
        role: "roles/window",
        members: ["user:me@example.com"],
        condition: {
          expression:
            'request.time > timestamp("2000-01-01T00:00:00Z") && request.time < timestamp("2099-01-01T00:00:00Z")',
        },
      },
    ]);
    expect(roles(policy)).toEqual([]);
  });
});
