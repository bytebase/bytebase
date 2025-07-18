import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { SQLReviewRuleLevel } from "@/types/proto-es/v1/org_policy_service_pb";
import { TEMPLATE_LIST_V2, getRuleLocalizationKey } from "../types/sqlReview";
import { mergedLocalMessage } from "./i18n-messages";

describe("Test i18n messages", () => {
  for (const keyA of Object.keys(mergedLocalMessage)) {
    for (const keyB of Object.keys(mergedLocalMessage)) {
      if (keyA === keyB) {
        continue;
      }

      test(`i18n message for ${keyA} and ${keyB}`, () => {
        const missMatchKey = compareMessages(
          mergedLocalMessage[keyA],
          mergedLocalMessage[keyB]
        );
        let message = "";
        if (missMatchKey !== "") {
          message = `${missMatchKey} not match in ${keyA} and ${keyB}`;
          console.error(message);
        }
        expect(missMatchKey).toBe("");
      });
    }
  }
});

describe("Test i18n for SQL review", () => {
  expect(Object.keys(mergedLocalMessage).length).greaterThan(0);
  const i18nMessage = Object.values(mergedLocalMessage)[0];
  expect(!!i18nMessage["sql-review"]).toBe(true);

  const i18nForSQLReview = i18nMessage["sql-review"];
  expect(!!i18nForSQLReview["template"]).toBe(true);
  expect(!!i18nForSQLReview["rule"]).toBe(true);
  expect(!!i18nForSQLReview["engine"]).toBe(true);
  expect(!!i18nForSQLReview["category"]).toBe(true);

  for (const template of TEMPLATE_LIST_V2) {
    describe(`check i18n for template ${template.id}`, () => {
      const key = `${template.id.split(".").join("-")}`;
      expect(!!i18nForSQLReview["template"][key]).toBe(true);
      expect(!!i18nForSQLReview["template"][`${key}-desc`]).toBe(true);

      for (const rule of template.ruleList) {
        test(`check i18n for rule ${rule.type}`, () => {
          const key = getRuleLocalizationKey(rule.type);
          expect(!!i18nForSQLReview["rule"][key], "rule-key").toBe(true);
          expect(
            !!i18nForSQLReview["rule"][key]["title"],
            "rule-key-title"
          ).toBe(true);
          expect(
            !!i18nForSQLReview["rule"][key]["description"],
            "rule-key-description"
          ).toBe(true);
          expect(
            !!i18nForSQLReview["category"][rule.category.toLowerCase()],
            "category-rule.category"
          ).toBe(true);
          expect(
            !!i18nForSQLReview["level"][
              sqlReviewRuleLevelToString(rule.level).toLowerCase()
            ],
            "level-rule.level"
          ).toBe(true);

          expect(
            !!i18nForSQLReview["engine"][Engine[rule.engine].toLowerCase()],
            "engine.rule-engine"
          ).toBe(true);

          for (const component of rule.componentList) {
            expect(
              !!i18nForSQLReview["rule"][key]["component"],
              "rule-key-component"
            ).toBe(true);
            expect(
              !!i18nForSQLReview["rule"][key]["component"][component.key],
              "rule-key-component-component.key"
            ).toBe(true);
          }
        });
      }
    });
  }
});

// Helper function to convert SQLReviewRuleLevel to string
const sqlReviewRuleLevelToString = (level: SQLReviewRuleLevel): string => {
  switch (level) {
    case SQLReviewRuleLevel.LEVEL_UNSPECIFIED:
      return "LEVEL_UNSPECIFIED";
    case SQLReviewRuleLevel.ERROR:
      return "ERROR";
    case SQLReviewRuleLevel.WARNING:
      return "WARNING";
    case SQLReviewRuleLevel.DISABLED:
      return "DISABLED";
    default:
      return "UNKNOWN";
  }
};

const compareMessages = (
  localA: { [k: string]: any },
  localB: { [k: string]: any }
): string => {
  for (const [key, valA] of Object.entries(localA)) {
    const valB = localB[key];
    if (!valB) {
      return key;
    }
    if (typeof valA === "object") {
      if (typeof valB !== "object") {
        return key;
      }
      // i18n v4 has special body for locale message string.
      if ("type" in valA && "start" in valA && "end" in valA) {
        continue;
      }
      const missMatch = compareMessages(valA, valB);
      if (missMatch) {
        return `${key}.${missMatch}`;
      }
    }
  }

  return "";
};
