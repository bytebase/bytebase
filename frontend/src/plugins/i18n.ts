import { useLocalStorage } from "@vueuse/core";
import type { WritableComputedRef } from "vue";
import { type Composer, createI18n } from "vue-i18n";
import { STORAGE_KEY_LANGUAGE } from "@/utils/storage-keys";
import { mergedLocalMessage } from "./i18n-messages";

const validLocaleList = ["en-US", "zh-CN", "es-ES", "ja-JP", "vi-VN"];

const getValidLocale = () => {
  const storage = useLocalStorage<string>(STORAGE_KEY_LANGUAGE, "");

  const params = new URL(globalThis.location.href).searchParams;
  let locale = params.get("locale") || "";
  if (validLocaleList.includes(locale)) {
    storage.value = locale;
  }

  locale = storage.value || "";
  if (validLocaleList.includes(locale)) {
    return locale;
  }

  locale = navigator.language;
  if (locale === "en") {
    // To work with user stored legacy preferences, we switch to en-US
    // here if we got "en" from localStorage
    locale = "en-US";
  }
  if (locale === "ja") {
    locale = "ja-JP";
  }
  if (locale === "es") {
    locale = "es-ES";
  }
  if (locale === "vi") {
    locale = "vi-VN";
  }
  if (validLocaleList.includes(locale)) {
    return locale;
  }

  return "en-US";
};

const dtfOptions = {
  full: {
    month: "short" as const,
    day: "numeric" as const,
    year: "numeric" as const,
    hour: "numeric" as const,
    minute: "2-digit" as const,
    timeZoneName: "short" as const,
  },
  date: {
    month: "short" as const,
    day: "numeric" as const,
    year: "numeric" as const,
  },
  dateShort: {
    month: "short" as const,
    day: "numeric" as const,
  },
};

const datetimeFormats = Object.fromEntries(
  validLocaleList.map((l) => [l, dtfOptions])
);

const i18n = createI18n({
  legacy: false,
  locale: getValidLocale(),
  globalInjection: true,
  messages: mergedLocalMessage as Record<string, Record<string, string>>,
  datetimeFormats,
  fallbackLocale: "en-US",
});

export const t = i18n.global.t as Composer["t"];

export const te = i18n.global.te as Composer["te"];

export const locale = i18n.global.locale as WritableComputedRef<string>;

export default i18n;
