import { readdirSync, readFileSync } from "fs";
import { isEqual } from "lodash-es";
import { dirname, resolve } from "path";
import { fileURLToPath } from "url";
import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { SQLReviewRule_Level } from "@/types/proto-es/v1/review_config_service_pb";
import {
  getRuleLocalizationKey,
  ruleTypeToString,
  TEMPLATE_LIST_V2,
} from "@/types/sqlReview";
import { type LocaleMessageObject, mergedLocalMessage } from "./messages";

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

      if (template.ruleList.length === 0) {
        test("has template localization even without rules", () => {
          expect(!!templateMessages[key]).toBe(true);
          expect(!!templateMessages[`${key}-desc`]).toBe(true);
        });
      }

      for (const rule of template.ruleList) {
        test(`check i18n for rule ${ruleTypeToString(rule.type)}`, () => {
          const key = getRuleLocalizationKey(ruleTypeToString(rule.type));
          const ruleMessage = getNestedObject(ruleMessages, key);
          expect(!!ruleMessage, "rule-key").toBe(true);
          expect("title" in ruleMessage, "rule-key-title").toBe(true);
          expect("description" in ruleMessage, "rule-key-description").toBe(
            true
          );
          expect(
            rule.category.toLowerCase() in categoryMessages,
            "category-rule.category"
          );
          expect(
            sqlReviewRuleLevelToString(rule.level).toLowerCase() in
              levelMessages,
            "level-rule.level"
          ).toBe(true);

          expect(
            Engine[rule.engine].toLowerCase() in engineMessages,
            "engine.rule-engine"
          ).toBe(true);

          for (const component of rule.componentList) {
            const componentMessages = getNestedObject(ruleMessage, "component");
            expect(!!componentMessages, "rule-key-component").toBe(true);
            expect(
              component.key in componentMessages,
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

const __dirname = dirname(fileURLToPath(import.meta.url));

const checkKeysSorted = (
  obj: Record<string, unknown>,
  path: string
): string[] => {
  const keys = Object.keys(obj);
  const sorted = [...keys].sort();
  const errors: string[] = [];
  for (let i = 0; i < keys.length; i++) {
    if (keys[i] !== sorted[i]) {
      errors.push(
        `${path}: key "${keys[i]}" is out of order (expected "${sorted[i]}")`
      );
      break;
    }
  }
  for (const key of keys) {
    const val = obj[key];
    if (val && typeof val === "object" && !Array.isArray(val)) {
      errors.push(
        ...checkKeysSorted(val as Record<string, unknown>, `${path}.${key}`)
      );
    }
  }
  return errors;
};

describe("Locale keys are sorted alphabetically", () => {
  const localesDirs = [resolve(__dirname, "../../locales")];

  for (const dir of localesDirs) {
    const files = readdirSync(dir).filter((f) => f.endsWith(".json"));
    for (const file of files) {
      test(`locales/${file}`, () => {
        const content = JSON.parse(readFileSync(resolve(dir, file), "utf-8"));
        const errors = checkKeysSorted(content, file);
        expect(errors, errors.join("\n")).toHaveLength(0);
      });
    }
  }
});

const dig = (messages: LocaleMessageObject, path: string): string => {
  let cur: string | LocaleMessageObject = messages;
  for (const part of path.split(".")) {
    expect(typeof cur, path).toBe("object");
    cur = (cur as LocaleMessageObject)[part];
  }
  expect(typeof cur, path).toBe("string");
  return cur as string;
};

// The workspace data-export setting's description quotes the "Request
// export" button label verbatim so admins can connect the policy to the
// affordance users see. Nothing else ties the two strings together, and
// the label has a history of renames (#21081, #21200) — assert the quote
// tracks the label in every locale. Scoped to these two keys on purpose:
// a repo-wide grep would false-positive on prose that coincidentally
// contains the label (e.g. ja role.project-viewer.description).
describe("data-export description quotes the Request export label", () => {
  for (const [locale, messages] of Object.entries(mergedLocalMessage)) {
    test(locale, () => {
      const label = dig(
        messages as LocaleMessageObject,
        "sql-editor.request-export"
      );
      const description = dig(
        messages as LocaleMessageObject,
        "settings.general.workspace.data-export.description"
      );
      expect(description).toContain(label);
    });
  }
});

// The Masking Exemptions FEATURE (admin-granted, resource-scoped policy on
// the project's Masking Exemptions page) and the access-grant "unmask"
// capability (user-requested, statement-bound) are different mechanisms.
// zh once shipped the feature's name (脱敏豁免) on the capability's CTA,
// sending users into the wrong mental model in the largest zh market —
// lock the separation in every locale: the capability strings must never
// reuse the feature's name.
describe("unmask capability wording stays distinct from the Masking Exemptions feature", () => {
  for (const [locale, messages] of Object.entries(mergedLocalMessage)) {
    test(locale, () => {
      const featureName = dig(
        messages as LocaleMessageObject,
        "project.masking-exemption.self"
      );
      for (const key of [
        "sql-editor.request-unmask",
        "sql-editor.grant-type-unmask",
      ]) {
        const value = dig(messages as LocaleMessageObject, key);
        expect(
          value.toLowerCase().includes(featureName.toLowerCase()),
          `${locale} ${key} ("${value}") must not reuse the Masking Exemptions feature name ("${featureName}")`
        ).toBe(false);
      }
    });
  }
});

const compareMessages = (
  localA: LocaleMessageObject,
  localB: LocaleMessageObject
): string => {
  for (const [key, valA] of Object.entries(localA)) {
    const valB = localB[key];
    if (!(key in localB)) {
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
