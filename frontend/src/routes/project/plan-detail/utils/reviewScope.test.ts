// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { reviewScope } from "./reviewScope";

const db = (name: string) => `instances/i/databases/${name}`;
const GROUP = "projects/p/databaseGroups/g";
const engines: Record<string, Engine> = {
  [db("pg1")]: Engine.POSTGRES,
  [db("pg2")]: Engine.POSTGRES,
  [db("bq")]: Engine.BIGQUERY,
};

const plan = (...specs: string[][]) =>
  create(PlanSchema, {
    specs: specs.map((targets, index) => ({
      id: `spec-${index}`,
      config: {
        case: "changeDatabaseConfig" as const,
        value: create(Plan_ChangeDatabaseConfigSchema, { targets }),
      },
    })),
  });

const scope = (
  reviewer: "rule" | "ai",
  input: ReturnType<typeof plan>,
  sampleSize = 0,
  matched: Record<string, string[]> = {}
) =>
  reviewScope(input, {
    reviewer,
    sampleSize,
    matchedDatabases: (group) => matched[group],
    engineOf: (database) => engines[database],
  });

describe("reviewScope", () => {
  test("the rule review covers its engines and leaves the rest to the review policy", () => {
    expect(scope("rule", plan([db("pg1"), db("bq")]))).toEqual({
      databases: [db("pg1")],
      engines: [Engine.POSTGRES],
      total: 1,
    });
  });

  test("the AI review takes the first targets of each spec when it samples", () => {
    const result = scope("ai", plan([db("pg1"), db("pg2"), db("bq")]), 1);
    expect(result.databases).toEqual([db("pg1")]);
    expect(result.total).toBe(3);
  });

  test("a group counts its matched databases once loaded, and dedupes across specs", () => {
    const input = plan([GROUP], [db("pg1")]);
    expect(scope("rule", input).databases).toEqual([db("pg1")]);
    expect(
      scope("rule", input, 0, { [GROUP]: [db("pg1"), db("pg2")] }).databases
    ).toEqual([db("pg1"), db("pg2")]);
  });
});
