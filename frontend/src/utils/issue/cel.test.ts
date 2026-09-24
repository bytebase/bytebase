import { describe, expect, test } from "vitest";
import { buildCELExpr, ExprType, type SimpleExpr } from "@/modules/cel";
import {
  CEL_ATTRIBUTE_RESOURCE_DATABASE,
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
} from "@/utils/cel-attributes";
import { convertFromExpr, stringifyConditionExpression } from "./cel";

const envIn = (ids: string[]): SimpleExpr => ({
  type: ExprType.Condition,
  operator: "@in",
  args: [CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID, ids],
});
const dbIn = (names: string[]): SimpleExpr => ({
  type: ExprType.Condition,
  operator: "@in",
  args: [CEL_ATTRIBUTE_RESOURCE_DATABASE, names],
});
const group = (operator: "_&&_" | "_||_", args: SimpleExpr[]): SimpleExpr => ({
  type: ExprType.ConditionGroup,
  operator,
  args,
});

const decode = async (expr: SimpleExpr) => {
  const built = await buildCELExpr(expr);
  if (!built) throw new Error("expected a built expression");
  return convertFromExpr(built);
};

describe("convertFromExpr recognizes only what the grant forms emit", () => {
  test("an AND of database and environment clauses is recognized", async () => {
    const decoded = await decode(
      group("_&&_", [
        dbIn(["instances/pg/databases/app"]),
        envIn(["staging", "prod"]),
      ])
    );
    expect(decoded.unrecognized).toBeUndefined();
    expect(decoded.environments).toEqual([
      "environments/staging",
      "environments/prod",
    ]);
    expect(decoded.databaseResources).toEqual([
      { databaseFullName: "instances/pg/databases/app" },
    ]);
  });

  test("the empty list — the switch off — decodes to an empty, recognized list", async () => {
    const decoded = await decode(envIn([]));
    expect(decoded.unrecognized).toBeUndefined();
    expect(decoded.environments).toEqual([]);
  });

  test("an OR is unrecognized even though its clauses are decoded", async () => {
    const decoded = await decode(
      group("_||_", [envIn(["staging"]), dbIn(["instances/pg/databases/app"])])
    );
    expect(decoded.unrecognized).toBe(true);
    expect(decoded.environments).toEqual(["environments/staging"]);
  });

  test("a negated membership is unrecognized", async () => {
    const decoded = await decode({
      type: ExprType.Condition,
      operator: "@not_in",
      args: [CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID, ["prod"]],
    });
    expect(decoded.unrecognized).toBe(true);
  });

  test("an inequality is unrecognized", async () => {
    const decoded = await decode(
      group("_&&_", [
        envIn(["staging"]),
        {
          type: ExprType.Condition,
          operator: "_!=_",
          args: [CEL_ATTRIBUTE_RESOURCE_DATABASE, "instances/pg/databases/x"],
        },
      ])
    );
    expect(decoded.unrecognized).toBe(true);
  });
});

describe("stringifyConditionExpression", () => {
  test("the switch off is byte-for-byte the empty picker's clause", () => {
    expect(stringifyConditionExpression({ environments: [] })).toBe(
      "resource.environment_id in []"
    );
    expect(stringifyConditionExpression({ environments: undefined })).toBe("");
    expect(
      stringifyConditionExpression({
        environments: ["environments/staging", "environments/prod"],
      })
    ).toBe('resource.environment_id in ["staging", "prod"]');
  });
});
