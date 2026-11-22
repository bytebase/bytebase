// @vitest-environment node
import { expect, test } from "vitest";
import enUS from "@/locales/en-US.json";
import esES from "@/locales/es-ES.json";
import jaJP from "@/locales/ja-JP.json";
import viVN from "@/locales/vi-VN.json";
import zhCN from "@/locales/zh-CN.json";
import { isCanonicalQueryMaskingQuestion } from "./queryFollowUpQuestions";

const keys = [
  "query-mask",
  "query-mask-database",
  "query-unmask",
  "query-unmask-database",
] as const;
const locales = [
  { name: "en-US", locale: enUS },
  { name: "es-ES", locale: esES },
  { name: "ja-JP", locale: jaJP },
  { name: "vi-VN", locale: viVN },
  { name: "zh-CN", locale: zhCN },
];

test.each(
  locales.flatMap(({ name, locale }) =>
    keys.map((key) => ({
      name,
      key,
      question: locale.agent["follow-up"][key].replace(
        "{{database}}",
        "instances/example/databases/app"
      ),
    }))
  )
)("recognizes the canonical $name $key question", ({ question }) => {
  expect(isCanonicalQueryMaskingQuestion(question)).toBe(true);
  expect(isCanonicalQueryMaskingQuestion(`  ${question.toUpperCase()}  `)).toBe(
    true
  );
});

test.each([
  "How many records have masked set to true?",
  "Can you summarize tax exemptions by category?",
  "Can you summarize these rows?",
  "When does my masking exemption expire?",
  'The status is "How can I mask these columns?"?',
  "How can I mask columns in ?",
  "How can I request unmasked access to ?",
  "How can I mask columns in instances/example/\ndatabases/app?",
])(
  "does not classify noncanonical wording as a canonical question: %s",
  (question) => {
    expect(isCanonicalQueryMaskingQuestion(question)).toBe(false);
  }
);
