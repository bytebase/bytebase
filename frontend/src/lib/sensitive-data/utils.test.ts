import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import { MaskingExemptionPolicy_ExemptionSchema } from "@/types/proto-es/v1/org_policy_service_pb";
import type { SensitiveColumn } from "./types";
import { isCurrentColumnException } from "./utils";

const sensitiveColumn: SensitiveColumn = {
  database: {
    name: "instances/prod/databases/hr",
  } as SensitiveColumn["database"],
  maskData: {
    schema: "public",
    table: "employee",
    column: "email",
    semanticTypeId: "",
    classificationId: "",
    target: {} as SensitiveColumn["maskData"]["target"],
  },
};

const makeException = (expression: string) =>
  create(MaskingExemptionPolicy_ExemptionSchema, {
    condition: create(ExprSchema, { expression }),
  });

describe("isCurrentColumnException", () => {
  test("matches a parenthesized resource condition with expiration", () => {
    const exception = makeException(
      '(resource.instance_id == "prod" && resource.database_name == "hr" && resource.schema_name == "public" && resource.table_name == "employee" && resource.column_name == "email") && request.time < timestamp("2026-04-15T00:00:00Z")'
    );

    expect(isCurrentColumnException(exception, sensitiveColumn)).toBe(true);
  });

  test("matches a parenthesized resource condition with classification level", () => {
    const exception = makeException(
      '(resource.instance_id == "prod" && resource.database_name == "hr" && resource.schema_name == "public" && resource.table_name == "employee" && resource.column_name == "email") && resource.classification_level <= 2'
    );

    expect(isCurrentColumnException(exception, sensitiveColumn)).toBe(true);
  });

  test("matches a parenthesized resource and classification condition group", () => {
    const exception = makeException(
      '(resource.instance_id == "prod" && resource.database_name == "hr" && resource.classification_level <= 2)'
    );

    expect(isCurrentColumnException(exception, sensitiveColumn)).toBe(true);
  });

  test("matches a resource", () => {
    expect(
      isCurrentColumnException(
        makeException(
          '(resource.instance_id == "prod") && request.time < timestamp("2026-04-15T00:00:00Z")'
        ),
        sensitiveColumn
      )
    ).toBe(true);

    expect(
      isCurrentColumnException(
        makeException(
          '((resource.instance_id == "prod") && request.time < timestamp("2026-04-15T00:00:00Z"))'
        ),
        sensitiveColumn
      )
    ).toBe(true);

    expect(
      isCurrentColumnException(
        makeException(
          '(resource.instance_id == "prod" && request.time < timestamp("2026-04-15T00:00:00Z"))'
        ),
        sensitiveColumn
      )
    ).toBe(true);

    expect(
      isCurrentColumnException(
        makeException(
          'resource.instance_id == "prod" && request.time < timestamp("2026-04-15T00:00:00Z")'
        ),
        sensitiveColumn
      )
    ).toBe(true);

    expect(
      isCurrentColumnException(
        makeException(
          'resource.instance_id == "prod" && resource.database_name == "hr"'
        ),
        sensitiveColumn
      )
    ).toBe(true);

    expect(
      isCurrentColumnException(
        makeException('(resource.instance_id == "prod")'),
        sensitiveColumn
      )
    ).toBe(true);
  });

  test("matches one resource from a grouped OR resource condition", () => {
    expect(
      isCurrentColumnException(
        makeException(
          '((resource.instance_id == "prod" && resource.database_name == "hr") || (resource.instance_id == "test" && resource.database_name == "finance"))'
        ),
        sensitiveColumn
      )
    ).toBe(true);
  });

  test("matches one resource from a grouped OR resource condition with expiration", () => {
    expect(
      isCurrentColumnException(
        makeException(
          'request.time < timestamp("2026-04-15T00:00:00Z") && ((resource.instance_id == "prod" && resource.database_name == "hr") || (resource.instance_id == "test" && resource.database_name == "finance"))'
        ),
        sensitiveColumn
      )
    ).toBe(true);
  });
});
