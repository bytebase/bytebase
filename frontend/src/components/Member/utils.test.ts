import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { ExprSchema as ConditionExprSchema } from "@/types/proto-es/google/type/expr_pb";
import { BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import { getUniqueProjectRoleBindings } from "./projectRoleBindings";

describe("getUniqueProjectRoleBindings", () => {
  test("dedupes repeated roles and prefers non-expired bindings", () => {
    const activeProjectDeveloper = create(BindingSchema, {
      role: "roles/projectDeveloper",
      members: ["user:test@example.com"],
    });
    const expiredProjectDeveloper = create(BindingSchema, {
      role: "roles/projectDeveloper",
      members: ["user:test@example.com"],
      condition: create(ConditionExprSchema, {
        expression: 'request.time < timestamp("2000-01-01T00:00:00Z")',
      }),
    });
    const sqlEditorUser = create(BindingSchema, {
      role: "roles/sqlEditorUser",
      members: ["user:test@example.com"],
    });

    const result = getUniqueProjectRoleBindings([
      expiredProjectDeveloper,
      activeProjectDeveloper,
      sqlEditorUser,
    ]);

    expect(result.map((binding) => binding.role)).toEqual([
      "roles/projectDeveloper",
      "roles/sqlEditorUser",
    ]);
    expect(result[0].condition?.expression).toBeUndefined();
  });
});
