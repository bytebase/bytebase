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

describe("TaskRunLogViewer locale interpolation", () => {
  test.each(
    Object.entries(LOCALES)
  )("interpolates summary and replica labels in %s", (locale, translation) => {
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
      i18n.t("task-run.log-viewer.summary", { sections: 2, entries: 3 })
    ).not.toContain("{");
    expect(i18n.t("task-run.log-viewer.replica-n", { n: 2 })).not.toContain(
      "{"
    );
  });
});
