import i18next from "i18next";
import { describe, expect, test } from "vitest";
import enUS from "@/react/locales/en-US.json";
import esES from "@/react/locales/es-ES.json";
import jaJP from "@/react/locales/ja-JP.json";
import viVN from "@/react/locales/vi-VN.json";
import zhCN from "@/react/locales/zh-CN.json";

const LOCALES = {
  "en-US": enUS,
  "zh-CN": zhCN,
  "es-ES": esES,
  "ja-JP": jaJP,
  "vi-VN": viVN,
} as const;

describe("Instance form locale interpolation", () => {
  test.each(
    Object.entries(LOCALES)
  )("renders host and proxy placeholders literally in %s", (locale, translation) => {
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

    expect(i18n.t("instance.sentence.host.none-snowflake")).toContain(" | ");
    expect(i18n.t("instance.sentence.host.none-snowflake")).not.toContain("{");
    expect(i18n.t("instance.sentence.proxy.snowflake")).toContain("@");
    expect(i18n.t("instance.sentence.proxy.snowflake")).not.toContain("{");
  });
});
