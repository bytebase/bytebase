import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { ExprSchema as ConditionExprSchema } from "@/types/proto-es/google/type/expr_pb";
import { BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import { getProjectRoleBindingKey, groupProjectRoleBindings } from "./member";

describe("groupProjectRoleBindings", () => {
  test("groups repeated roles without dropping distinct bindings", () => {
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

    const result = groupProjectRoleBindings([
      expiredProjectDeveloper,
      activeProjectDeveloper,
      sqlEditorUser,
    ]);

    expect(result.map((group) => group.role)).toEqual([
      "roles/projectDeveloper",
      "roles/sqlEditorUser",
    ]);
    expect(result[0].bindings).toEqual([
      expiredProjectDeveloper,
      activeProjectDeveloper,
    ]);
  });
});

describe("getProjectRoleBindingKey", () => {
  test("keeps repeated same-role bindings unique", () => {
    const binding = create(BindingSchema, {
      role: "roles/projectDeveloper",
      members: ["user:test@example.com"],
    });

    expect(getProjectRoleBindingKey(binding, 0)).not.toEqual(
      getProjectRoleBindingKey(binding, 1)
    );
  });
});
