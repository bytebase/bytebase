import { create } from "@bufbuild/protobuf";
import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { useAppStore } from "@/stores/app";
import {
  type Expr,
  ExprSchema,
} from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  BindingSchema,
  IamPolicySchema,
} from "@/types/proto-es/v1/iam_policy_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { RoleSchema } from "@/types/proto-es/v1/role_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { useSQLEditorDatabasesWithExport } from "./useSQLEditorState";

const ident = (name: string) =>
  create(ExprSchema, { exprKind: { case: "identExpr", value: { name } } });
const databaseIs = (name: string): Expr =>
  create(ExprSchema, {
    exprKind: {
      case: "callExpr",
      value: {
        function: "_==_",
        args: [
          create(ExprSchema, {
            exprKind: {
              case: "selectExpr",
              value: { operand: ident("resource"), field: "database" },
            },
          }),
          create(ExprSchema, {
            exprKind: {
              case: "constExpr",
              value: { constantKind: { case: "stringValue", value: name } },
            },
          }),
        ],
      },
    },
  });

const project = "projects/p";
const databaseA = { name: "instances/i/databases/a", project } as Database;
const databaseB = { name: "instances/i/databases/b", project } as Database;
const roles = [
  create(RoleSchema, {
    name: "roles/sqlEditorExporter",
    permissions: ["bb.sql.select", "bb.sql.export"],
  }),
  create(RoleSchema, {
    name: "roles/sqlEditorReadUser",
    permissions: ["bb.sql.select"],
  }),
];

describe("useSQLEditorDatabasesWithExport", () => {
  beforeEach(() => {
    useAppStore.setState({
      currentUser: create(UserSchema, {
        name: "users/u@example.com",
        email: "u@example.com",
      }),
      roles,
      roleList: roles,
      workspacePolicy: create(IamPolicySchema, {}),
      projectsByName: { [project]: create(ProjectSchema, { name: project }) },
      projectPoliciesByName: {
        [project]: create(IamPolicySchema, {
          bindings: [
            create(BindingSchema, {
              role: "roles/sqlEditorExporter",
              members: ["user:u@example.com"],
              condition: {
                expression: `resource.database == "${databaseA.name}"`,
              },
              parsedExpr: databaseIs(databaseA.name),
            }),
            create(BindingSchema, {
              role: "roles/sqlEditorReadUser",
              members: ["user:u@example.com"],
            }),
          ],
        }),
      },
      fetchProject: vi.fn(),
      loadProjectIamPolicy: vi.fn(),
    } as never);
  });

  test("an Exporter binding scoped to one database grants export there only", () => {
    const databases = [databaseA, databaseB];
    const { result } = renderHook(() =>
      useSQLEditorDatabasesWithExport(databases)
    );

    expect([...result.current]).toEqual([databaseA.name]);
  });
});
