import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import {
  findingCodeTitle,
  RULE_REVIEW_ENGINES,
  standardRuleCodes,
  standardRuleEngines,
} from "./standardRuleCatalog";
import { STANDARD_RULE_TYPES } from "./standardRules";

describe("standardRuleCatalog", () => {
  test("names every reviewed engine by a known enum value", () => {
    expect(RULE_REVIEW_ENGINES).toContain(Engine.POSTGRES);
    for (const engine of RULE_REVIEW_ENGINES) {
      expect(Engine[engine]).toBeDefined();
    }
  });

  test("gives every standard rule its engines and at least one code", () => {
    for (const rule of STANDARD_RULE_TYPES) {
      expect(standardRuleCodes(rule).length).toBeGreaterThan(0);
      const engines = standardRuleEngines(rule);
      expect(engines.length).toBeGreaterThan(0);
      expect(
        engines.every((engine) => RULE_REVIEW_ENGINES.includes(engine))
      ).toBe(true);
    }
  });

  test("leaves out an engine that only reports it lacks the rule's feature", () => {
    expect(standardRuleEngines(ReviewRuleType.ONLINE_MIGRATION)).not.toContain(
      Engine.POSTGRES
    );
    expect(standardRuleEngines(ReviewRuleType.REQUIRE_WHERE)).toContain(
      Engine.POSTGRES
    );
  });

  test("titles a known code and not an unknown one", () => {
    expect(findingCodeTitle("WT101")).toBe("Object doesn't exist");
    expect(findingCodeTitle("ZZ999")).toBeUndefined();
    expect(
      standardRuleCodes(ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED)
    ).toEqual([]);
  });
});
