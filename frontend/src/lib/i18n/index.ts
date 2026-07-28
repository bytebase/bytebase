import i18next from "i18next";
import { initReactI18next } from "react-i18next";
import enUSDynamic from "@/locales/dynamic/en-US.json";
import esESDynamic from "@/locales/dynamic/es-ES.json";
import jaJPDynamic from "@/locales/dynamic/ja-JP.json";
import viVNDynamic from "@/locales/dynamic/vi-VN.json";
import zhCNDynamic from "@/locales/dynamic/zh-CN.json";
import enUS from "@/locales/en-US.json";
import esES from "@/locales/es-ES.json";
import jaJP from "@/locales/ja-JP.json";
import enUSSQLReview from "@/locales/sql-review/en-US.json";
import esESSQLReview from "@/locales/sql-review/es-ES.json";
import jaJPSQLReview from "@/locales/sql-review/ja-JP.json";
import viVNSQLReview from "@/locales/sql-review/vi-VN.json";
import zhCNSQLReview from "@/locales/sql-review/zh-CN.json";
import enUSSubscription from "@/locales/subscription/en-US.json";
import esESSubscription from "@/locales/subscription/es-ES.json";
import jaJPSubscription from "@/locales/subscription/ja-JP.json";
import viVNSubscription from "@/locales/subscription/vi-VN.json";
import zhCNSubscription from "@/locales/subscription/zh-CN.json";
import viVN from "@/locales/vi-VN.json";
import zhCN from "@/locales/zh-CN.json";

const STORAGE_KEY_LANGUAGE = "bb.language";

function getLocale(): string {
  const stored = localStorage.getItem(STORAGE_KEY_LANGUAGE) ?? "";
  if (stored) {
    try {
      const parsed = JSON.parse(stored);
      if (typeof parsed === "string" && parsed) return parsed;
    } catch {
      if (stored) return stored;
    }
  }
  const nav = navigator.language;
  const mapping: Record<string, string> = {
    en: "en-US",
    ja: "ja-JP",
    es: "es-ES",
    vi: "vi-VN",
  };
  return mapping[nav] ?? (nav.includes("-") ? nav : "en-US");
}

function mergeMessages(
  base: Record<string, unknown>,
  override: Record<string, unknown>
): Record<string, unknown> {
  const result = { ...base };
  for (const [key, value] of Object.entries(override)) {
    const existing = result[key];
    if (
      existing &&
      typeof existing === "object" &&
      !Array.isArray(existing) &&
      value &&
      typeof value === "object" &&
      !Array.isArray(value)
    ) {
      result[key] = mergeMessages(
        existing as Record<string, unknown>,
        value as Record<string, unknown>
      );
    } else {
      result[key] = value;
    }
  }
  return result;
}

function buildTranslation({
  main,
  dynamic,
  sqlReview,
  subscription,
}: {
  main: Record<string, unknown>;
  dynamic: Record<string, unknown>;
  sqlReview: Record<string, unknown>;
  subscription: Record<string, unknown>;
}) {
  return mergeMessages(
    {
      dynamic,
      "sql-review": sqlReview,
      subscription,
    },
    main
  );
}

const resources = {
  "en-US": {
    translation: buildTranslation({
      main: enUS,
      dynamic: enUSDynamic,
      sqlReview: enUSSQLReview,
      subscription: enUSSubscription,
    }),
  },
  "zh-CN": {
    translation: buildTranslation({
      main: zhCN,
      dynamic: zhCNDynamic,
      sqlReview: zhCNSQLReview,
      subscription: zhCNSubscription,
    }),
  },
  "es-ES": {
    translation: buildTranslation({
      main: esES,
      dynamic: esESDynamic,
      sqlReview: esESSQLReview,
      subscription: esESSubscription,
    }),
  },
  "ja-JP": {
    translation: buildTranslation({
      main: jaJP,
      dynamic: jaJPDynamic,
      sqlReview: jaJPSQLReview,
      subscription: jaJPSubscription,
    }),
  },
  "vi-VN": {
    translation: buildTranslation({
      main: viVN,
      dynamic: viVNDynamic,
      sqlReview: viVNSQLReview,
      subscription: viVNSubscription,
    }),
  },
};

const i18n: import("i18next").i18n = i18next.createInstance();

export const i18nReady = i18n.use(initReactI18next).init({
  resources,
  lng: getLocale(),
  fallbackLng: "en-US",
  interpolation: {
    escapeValue: false,
  },
  initAsync: false,
});

export default i18n;
