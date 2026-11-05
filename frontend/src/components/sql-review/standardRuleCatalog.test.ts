import { describe, expect, test } from "vitest";
import enUS from "@/locales/sql-review/en-US.json";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import {
  findingCodeTitleKey,
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

  test("keys a known code's title and not an unknown one", () => {
    expect(findingCodeTitleKey("WT101")).toBe("sql-review.finding-code.wt101");
    expect(findingCodeTitleKey("ZZ999")).toBeUndefined();
    expect(
      standardRuleCodes(ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED)
    ).toEqual([]);
  });

  // A code omni adds or retitles reaches the catalog through the backend's
  // sync test, then fails here until en-US carries it; check-i18n holds the
  // other locales to en-US's keys.
  test("gives every code an en-US title equal to omni's", () => {
    const titles: Record<string, string | undefined> = enUS["finding-code"];
    for (const rule of STANDARD_RULE_TYPES) {
      for (const { code, title } of standardRuleCodes(rule)) {
        expect(titles[code.toLowerCase()], code).toBe(title);
      }
    }
  });
});
