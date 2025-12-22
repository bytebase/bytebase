import { isEqual } from "lodash-es";
import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { SQLReviewRule_Level } from "@/types/proto-es/v1/review_config_service_pb";
import {
  getRuleLocalizationKey,
  ruleTypeToString,
  TEMPLATE_LIST_V2,
} from "../types/sqlReview";
import { type LocaleMessageObject, mergedLocalMessage } from "./i18n-messages";

// Helper to safely access nested locale objects
const getNestedObject = (
  obj: LocaleMessageObject,
  key: string
): LocaleMessageObject => {
  const value = obj[key];
  if (typeof value === "string") {
    throw new Error(`Expected object at key "${key}", got string`);
  }
  return value;
};

describe("Test i18n messages", () => {
  for (const keyA of Object.keys(mergedLocalMessage)) {
    for (const keyB of Object.keys(mergedLocalMessage)) {
      if (keyA === keyB) {
        continue;
      }
      if (
        typeof mergedLocalMessage[keyA] === "string" ||
        typeof mergedLocalMessage[keyB] === "string"
      ) {
        if (!isEqual(mergedLocalMessage[keyA], mergedLocalMessage[keyB])) {
          throw Error(`${keyA} and ${keyB} not match`);
        }
        continue;
      }

      test(`i18n message for ${keyA} and ${keyB}`, () => {
        const missMatchKey = compareMessages(
          mergedLocalMessage[keyA] as LocaleMessageObject,
          mergedLocalMessage[keyB] as LocaleMessageObject
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
  const i18nMessage = Object.values(
    mergedLocalMessage
  )[0] as LocaleMessageObject;
  expect(!!i18nMessage["sql-review"]).toBe(true);

  const i18nForSQLReview = getNestedObject(i18nMessage, "sql-review");
  const templateMessages = getNestedObject(i18nForSQLReview, "template");
  const ruleMessages = getNestedObject(i18nForSQLReview, "rule");
  const categoryMessages = getNestedObject(i18nForSQLReview, "category");
  const levelMessages = getNestedObject(i18nForSQLReview, "level");
  const engineMessages = getNestedObject(i18nForSQLReview, "engine");

  expect(!!templateMessages).toBe(true);
  expect(!!ruleMessages).toBe(true);
  expect(!!engineMessages).toBe(true);
  expect(!!categoryMessages).toBe(true);

  for (const template of TEMPLATE_LIST_V2) {
    describe(`check i18n for template ${template.id}`, () => {
      const key = `${template.id.split(".").join("-")}`;
      expect(!!templateMessages[key]).toBe(true);
      expect(!!templateMessages[`${key}-desc`]).toBe(true);

      for (const rule of template.ruleList) {
        test(`check i18n for rule ${ruleTypeToString(rule.type)}`, () => {
          const key = getRuleLocalizationKey(ruleTypeToString(rule.type));
          const ruleMessage = getNestedObject(ruleMessages, key);
          expect(!!ruleMessage, "rule-key").toBe(true);
          expect(!!ruleMessage["title"], "rule-key-title").toBe(true);
          expect(!!ruleMessage["description"], "rule-key-description").toBe(
            true
          );
          expect(
            !!categoryMessages[rule.category.toLowerCase()],
            "category-rule.category"
          ).toBe(true);
          expect(
            !!levelMessages[
              sqlReviewRuleLevelToString(rule.level).toLowerCase()
            ],
            "level-rule.level"
          ).toBe(true);

          expect(
            !!engineMessages[Engine[rule.engine].toLowerCase()],
            "engine.rule-engine"
          ).toBe(true);

          for (const component of rule.componentList) {
            const componentMessages = getNestedObject(ruleMessage, "component");
            expect(!!componentMessages, "rule-key-component").toBe(true);
            expect(
              !!componentMessages[component.key],
              "rule-key-component-component.key"
            ).toBe(true);
          }
        });
      }
    });
  }
});

// Helper function to convert SQLReviewRule_Level to string
const sqlReviewRuleLevelToString = (level: SQLReviewRule_Level): string => {
  switch (level) {
    case SQLReviewRule_Level.LEVEL_UNSPECIFIED:
      return "LEVEL_UNSPECIFIED";
    case SQLReviewRule_Level.ERROR:
      return "ERROR";
    case SQLReviewRule_Level.WARNING:
      return "WARNING";
    default:
      return "UNKNOWN";
  }
};

const compareMessages = (
  localA: LocaleMessageObject,
  localB: LocaleMessageObject
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
