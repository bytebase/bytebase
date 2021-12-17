import { useI18n } from "vue-i18n";
import { useLocalStorage } from "@vueuse/core";

/**
 * Language hook for i18n
 * @returns
 */
const useLanguage = () => {
  const { availableLocales, locale } = useI18n();
  const storage = useLocalStorage("bytebase_options", {}) as any;

  const setLocale = (lang: string) => {
    locale.value = lang;
    storage.value = {
      appearance: {
        language: lang,
      },
    };
  };

  const toggleLocales = () => {
    const locales = availableLocales;
    const nextLocale =
      locales[(locales.indexOf(locale.value) + 1) % locales.length];
    setLocale(nextLocale);
  };

  return {
    locale,
    availableLocales,
    setLocale,
    toggleLocales,
  };
};

export { useLanguage };
