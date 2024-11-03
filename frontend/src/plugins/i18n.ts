import { useLocalStorage } from "@vueuse/core";
import { createI18n } from "vue-i18n";
import { mergedLocalMessage } from "./i18n-messages";

export interface LanguageStorage {
  appearance: {
    language: string;
  };
}

const validLocaleList = ["en-US", "zh-CN", "es-ES", "ja-JP"];

const getValidLocale = () => {
  const storage = useLocalStorage<LanguageStorage>("bytebase_options", {
    appearance: {
      language: "",
    },
  });

  const params = new URL(globalThis.location.href).searchParams;
  let locale = params.get("locale") || "";
  if (validLocaleList.includes(locale)) {
    storage.value = {
      appearance: {
        language: locale,
      },
    };
  }

  locale = storage.value?.appearance?.language || "";
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
  if (validLocaleList.includes(locale)) {
    return locale;
  }

  return "en-US";
};

const i18n = createI18n({
  legacy: false,
  locale: getValidLocale(),
  globalInjection: true,
  messages: mergedLocalMessage,
  fallbackLocale: "en-US",
});

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore
export const t = i18n.global.t;

export const te = i18n.global.te;

export const locale = i18n.global.locale;

export default i18n;
