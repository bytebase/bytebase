import { describe, expect, test } from "vitest";
import { emptySimpleExpr, wrapAsGroup } from "@/plugins/cel";
import type { DatabaseResource } from "@/types";
import { buildMaskingExemption } from "./maskingExemption";

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
});
