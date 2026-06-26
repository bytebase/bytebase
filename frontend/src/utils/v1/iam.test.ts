import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { ExprSchema as ConditionExprSchema } from "@/types/proto-es/google/type/expr_pb";
import {
  BindingSchema,
  IamPolicySchema,
} from "@/types/proto-es/v1/iam_policy_pb";
import { revokeMemberFromBinding } from "./iam";

// A database-scoped grant with no expiration. Several users granted the same
// database via separate access requests end up as separate single-member
// bindings that share this exact role + condition expression.
const DB_SCOPED =
  '(resource.database in ["instances/nf-prod-jfapp/databases/ekb_app2"])';

const grant = (member: string) =>
  create(BindingSchema, {
    role: "roles/sqlEditorReadUser",
    members: [member],
    condition: create(ConditionExprSchema, { expression: DB_SCOPED }),
  });

describe("revokeMemberFromBinding", () => {
  test("revokes the clicked binding when several bindings share role and condition", () => {
    const grantA = grant("user:a@example.com");
    const grantB = grant("user:b@example.com");
    const policy = create(IamPolicySchema, { bindings: [grantA, grantB] });

    const result = revokeMemberFromBinding(
      policy,
      grantB,
      "user:b@example.com"
    );

    const members = result.bindings.flatMap((b) => b.members);
    expect(members).not.toContain("user:b@example.com");
    expect(members).toContain("user:a@example.com");
  });

  test("removes only the targeted member from a shared multi-member binding", () => {
    const binding = create(BindingSchema, {
      role: "roles/sqlEditorReadUser",
      members: [
        "user:a@example.com",
        "user:b@example.com",
        "user:c@example.com",
      ],
      condition: create(ConditionExprSchema, { expression: DB_SCOPED }),
    });
    const policy = create(IamPolicySchema, { bindings: [binding] });

    const result = revokeMemberFromBinding(
      policy,
      binding,
      "user:b@example.com"
    );

    expect(result.bindings).toHaveLength(1);
    expect(result.bindings[0].members).toEqual([
      "user:a@example.com",
      "user:c@example.com",
    ]);
  });

  test("drops the binding once its last member is revoked", () => {
    const binding = grant("user:a@example.com");
    const policy = create(IamPolicySchema, { bindings: [binding] });

    const result = revokeMemberFromBinding(
      policy,
      binding,
      "user:a@example.com"
    );

    expect(result.bindings).toHaveLength(0);
  });

  test("does not mutate the input policy", () => {
    const grantA = grant("user:a@example.com");
    const grantB = grant("user:b@example.com");
    const policy = create(IamPolicySchema, { bindings: [grantA, grantB] });

    revokeMemberFromBinding(policy, grantB, "user:b@example.com");

    expect(policy.bindings).toHaveLength(2);
    expect(policy.bindings[1].members).toEqual(["user:b@example.com"]);
  });

  test("falls back to role + condition + membership when the target is not the same reference", () => {
    const grantA = grant("user:a@example.com");
    const grantB = grant("user:b@example.com");
    const policy = create(IamPolicySchema, { bindings: [grantA, grantB] });
    // A structurally-equal copy with a different object identity, e.g. after a
    // round-trip through the cache.
    const targetCopy = grant("user:b@example.com");

    const result = revokeMemberFromBinding(
      policy,
      targetCopy,
      "user:b@example.com"
    );

    const members = result.bindings.flatMap((b) => b.members);
    expect(members).not.toContain("user:b@example.com");
    expect(members).toContain("user:a@example.com");
  });
});
