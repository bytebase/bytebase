import i18next from "i18next";
import { describe, expect, test } from "vitest";
import enUS from "@/locales/en-US.json";
import esES from "@/locales/es-ES.json";
import jaJP from "@/locales/ja-JP.json";
import viVN from "@/locales/vi-VN.json";
import zhCN from "@/locales/zh-CN.json";

const LOCALES = {
  "en-US": enUS,
  "zh-CN": zhCN,
  "es-ES": esES,
  "ja-JP": jaJP,
  "vi-VN": viVN,
} as const;

describe("IssueTable locale interpolation", () => {
  test.each(Object.entries(LOCALES))(
    "interpolates the disabled batch action in %s",
    (locale, translation) => {
      const i18n = i18next.createInstance();

      void i18n.init({
        resources: {
          [locale]: { translation },
        },
        lng: locale,
        interpolation: {
          escapeValue: false,
        },
        initAsync: false,
      });

      expect(
        i18n.t("issue.batch-transition.not-allowed-tips", {
          operation: "TRANSITION",
        })
      ).toContain("TRANSITION");
    }
  );
});
