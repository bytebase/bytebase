import { describe, expect, test } from "vitest";
import enReact from "@/react/locales/en-US.json";
import esReact from "@/react/locales/es-ES.json";
import jaReact from "@/react/locales/ja-JP.json";
import viReact from "@/react/locales/vi-VN.json";
import zhReact from "@/react/locales/zh-CN.json";

// Locale-format guard for the new role-grant warning strings. Component
// tests use a mock t() that doesn't see the real locale strings, so a typo
// like "kind" instead of "{{kind}}" — or a translator dropping a placeholder
// in a non-EN locale — would slip through. This test loads every React
// locale and asserts placeholder presence directly. Cross-locale key parity
// is enforced separately by frontend/scripts/check-react-i18n.mjs, but key
// parity ≠ placeholder parity.

const locales = {
  "en-US": enReact,
  "zh-CN": zhReact,
  "ja-JP": jaReact,
  "es-ES": esReact,
  "vi-VN": viReact,
};

const projectMembersKindKeys = [
  "ddl-warning",
  "ddl-current-all",
  "ddl-current-some",
  "ddl-current-none",
] as const;

describe("DDLWarningCallout locale strings", () => {
  for (const [locale, data] of Object.entries(locales)) {
    describe(locale, () => {
      test.each(
        projectMembersKindKeys
      )("project.members.%s contains {{kind}} placeholder", (key) => {
        const value = (
          data.project.members as unknown as Record<string, string>
        )[key];
        expect(value).toBeDefined();
        expect(value).toContain("{{kind}}");
      });
    });
  }
});
