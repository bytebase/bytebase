// @vitest-environment node
import { describe, expect, test } from "vitest";
import {
  type ConditionGroupExpr,
  ExprType,
  emptySimpleExpr,
  wrapAsGroup,
} from "@/modules/cel";
import type { DatabaseResource } from "@/types";
import {
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
  CEL_ATTRIBUTE_RESOURCE_DATABASE,
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
} from "@/utils/cel-attributes";
import {
  buildMaskingExemption,
  rewriteMaskingExemptionDatabaseConditions,
} from "./maskingExemption";

const expr = wrapAsGroup(emptySimpleExpr());

describe("buildMaskingExemption", () => {
  test("builds an all-database exemption without condition expression", async () => {
    const exemption = await buildMaskingExemption({
      radioValue: "ALL",
      expr,
      databaseResources: [],
      memberList: ["user:test@example.com"],
      description: "reason",
    });

    expect(exemption.members).toEqual(["user:test@example.com"]);
    expect(exemption.condition?.description).toBe("reason");
    expect(exemption.condition?.expression).toBe("");
  });

  test("rejects expression exemption when CEL parsing fails", async () => {
    await expect(
      buildMaskingExemption({
        radioValue: "EXPRESSION",
        expr,
        databaseResources: [],
        memberList: ["user:test@example.com"],
        description: "reason",
      })
    ).rejects.toThrow("Invalid masking exemption expression");
  });

  test("builds a selected single-resource condition with expiration", async () => {
    const exemption = await buildMaskingExemption({
      radioValue: "SELECT",
      expr,
      databaseResources: [
        {
          databaseFullName: "instances/prod/databases/hr",
          schema: "public",
          table: "employee",
          columns: ["email"],
        },
      ],
      memberList: ["user:test@example.com"],
      description: "",
      expirationTimestamp: "2026-04-15T00:00:00.000Z",
    });

    expect(exemption.condition?.expression).toBe(
      'request.time < timestamp("2026-04-15T00:00:00.000Z") && (resource.instance_id == "prod" && resource.database_name == "hr" && resource.schema_name == "public" && resource.table_name == "employee" && resource.column_name == "email")'
    );
  });

  test("builds grouped OR conditions for multiple selected resources", async () => {
    const databaseResources: DatabaseResource[] = [
      {
        databaseFullName: "instances/prod/databases/hr",
      },
      {
        databaseFullName: "instances/test/databases/finance",
      },
    ];

    const exemption = await buildMaskingExemption({
      radioValue: "SELECT",
      expr,
      databaseResources,
      memberList: ["user:test@example.com"],
      description: "",
    });

    expect(exemption.condition?.expression).toBe(
      '((resource.instance_id == "prod" && resource.database_name == "hr") || (resource.instance_id == "test" && resource.database_name == "finance"))'
    );
  });

  test("rewrites a database picker membership condition for the backend", () => {
    const expression: ConditionGroupExpr = {
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.Condition,
          operator: "@in",
          args: [
            CEL_ATTRIBUTE_RESOURCE_DATABASE,
            ["instances/prod/databases/hr", "instances/test/databases/finance"],
          ],
        },
        {
          type: ExprType.Condition,
          operator: "@in",
          args: [CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME, ["email", "phone"]],
        },
      ],
    };

    expect(rewriteMaskingExemptionDatabaseConditions(expression)).toEqual({
      type: ExprType.ConditionGroup,
      operator: "_&&_",
      args: [
        {
          type: ExprType.ConditionGroup,
          operator: "_||_",
          args: [
            {
              type: ExprType.ConditionGroup,
              operator: "_&&_",
              args: [
                {
                  type: ExprType.Condition,
                  operator: "_==_",
                  args: [CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID, "prod"],
                },
                {
                  type: ExprType.Condition,
                  operator: "_==_",
                  args: [CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME, "hr"],
                },
              ],
            },
            {
              type: ExprType.ConditionGroup,
              operator: "_&&_",
              args: [
                {
                  type: ExprType.Condition,
                  operator: "_==_",
                  args: [CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID, "test"],
                },
                {
                  type: ExprType.Condition,
                  operator: "_==_",
                  args: [CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME, "finance"],
                },
              ],
            },
          ],
        },
        {
          type: ExprType.Condition,
          operator: "@in",
          args: [CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME, ["email", "phone"]],
        },
      ],
    });
  });
});
