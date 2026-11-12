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
  reviewers: { rules: boolean; ai: boolean },
  input: ReturnType<typeof plan>,
  sampleSize = 0,
  matched: Record<string, string[]> = {}
) =>
  reviewScope(input, {
    reviewers,
    sampleSize,
    matchedDatabases: (group) => matched[group],
    engineOf: (database) => engines[database],
  });

const RULES = { rules: true, ai: false };
const AI = { rules: false, ai: true };
const BOTH = { rules: true, ai: true };

describe("reviewScope", () => {
  test("the standard rules cover their engines and leave the rest to the review policy", () => {
    expect(scope(RULES, plan([db("pg1"), db("bq")]))).toEqual({
      databases: [db("pg1")],
      engines: [Engine.POSTGRES],
      total: 1,
    });
  });

  test("the AI review reads every engine", () => {
    for (const reviewers of [AI, BOTH]) {
      expect(scope(reviewers, plan([db("pg1"), db("bq")])).databases).toEqual([
        db("pg1"),
        db("bq"),
      ]);
    }
  });

  test("the review takes the first targets of each spec when the project samples", () => {
    for (const reviewers of [RULES, AI, BOTH]) {
      const result = scope(
        reviewers,
        plan([db("pg1"), db("pg2")], [db("pg2")]),
        1
      );
      expect(result.databases).toEqual([db("pg1"), db("pg2")]);
      expect(result.total).toBe(2);
    }
    const result = scope(BOTH, plan([db("pg1"), db("pg2"), db("bq")]), 1);
    expect(result.databases).toEqual([db("pg1")]);
    expect(result.total).toBe(3);
  });

  test("the sample is taken before the standard rules leave an engine out", () => {
    const result = scope(RULES, plan([db("bq"), db("pg1"), db("pg2")]), 2);
    expect(result.databases).toEqual([db("pg1")]);
    expect(result.total).toBe(2);
  });

  test("a review with no reviewer on covers nothing", () => {
    expect(
      scope({ rules: false, ai: false }, plan([db("pg1")])).databases
    ).toEqual([]);
  });

  test("a group counts its matched databases once loaded, and dedupes across specs", () => {
    const input = plan([GROUP], [db("pg1")]);
    expect(scope(RULES, input).databases).toEqual([db("pg1")]);
    expect(
      scope(RULES, input, 0, { [GROUP]: [db("pg1"), db("pg2")] }).databases
    ).toEqual([db("pg1"), db("pg2")]);
  });
});
