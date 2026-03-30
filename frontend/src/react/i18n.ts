// Virtual module that reads raw JSON at build time, bypassing vue-i18n AST compilation
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore -- virtual module provided by vite.config.ts react-raw-locales plugin
import rawLocales from "virtual:react-locales";
import i18next from "i18next";
import { merge } from "lodash-es";
import { initReactI18next } from "react-i18next";

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

function buildResources() {
  const resources: Record<string, { translation: Record<string, unknown> }> =
    {};
  for (const [locale, data] of Object.entries(
    rawLocales as Record<string, { main: string; sub: string }>
  )) {
    const main = JSON.parse(data.main);
    const sub = JSON.parse(data.sub);
    resources[locale] = {
      translation: merge({}, main, { subscription: sub }),
    };
  }
  return resources;
}

const i18n: import("i18next").i18n = i18next.createInstance();

export const i18nReady = i18n.use(initReactI18next).init({
  resources: buildResources(),
  lng: getLocale(),
  fallbackLng: "en-US",
  interpolation: {
    escapeValue: false,
  },
  initImmediate: false,
});

export default i18n;
