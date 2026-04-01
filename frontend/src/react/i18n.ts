import i18next from "i18next";
import { initReactI18next } from "react-i18next";
import enUSDynamic from "@/react/locales/dynamic/en-US.json";
import esESDynamic from "@/react/locales/dynamic/es-ES.json";
import jaJPDynamic from "@/react/locales/dynamic/ja-JP.json";
import viVNDynamic from "@/react/locales/dynamic/vi-VN.json";
import zhCNDynamic from "@/react/locales/dynamic/zh-CN.json";
import enUS from "@/react/locales/en-US.json";
import esES from "@/react/locales/es-ES.json";
import jaJP from "@/react/locales/ja-JP.json";
import viVN from "@/react/locales/vi-VN.json";
import zhCN from "@/react/locales/zh-CN.json";

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

const resources = {
  "en-US": { translation: { ...enUS, dynamic: enUSDynamic } },
  "zh-CN": { translation: { ...zhCN, dynamic: zhCNDynamic } },
  "es-ES": { translation: { ...esES, dynamic: esESDynamic } },
  "ja-JP": { translation: { ...jaJP, dynamic: jaJPDynamic } },
  "vi-VN": { translation: { ...viVN, dynamic: viVNDynamic } },
};

const i18n: import("i18next").i18n = i18next.createInstance();

export const i18nReady = i18n.use(initReactI18next).init({
  resources,
  lng: getLocale(),
  fallbackLng: "en-US",
  interpolation: {
    escapeValue: false,
    prefix: "{",
    suffix: "}",
  },
  initImmediate: false,
});

export default i18n;
