import { App } from "vue";
import { createI18n } from "vue-i18n";
import { useLocalStorage } from "@vueuse/core";

const localPathPrefix = "../locales/";

const storage = useLocalStorage("bytebase_options", {}) as any;

let locale = storage.value?.appearance?.language || navigator.language;
if (locale === "en") {
  // To work with user stored legacy preferences, we switch to en-US
  // here if we got "en" from localStorage
  locale = "en-US";
}

// import i18n resources
// https://vitejs.dev/guide/features.html#glob-import
const messages = Object.fromEntries(
  Object.entries(import.meta.globEager("../locales/*.json")).map(
    ([key, value]) => {
      const name = key.slice(localPathPrefix.length, -5);
      return [name, value.default];
    }
  )
);

const i18n = createI18n({
  legacy: false,
  locale,
  globalInjection: true,
  messages,
  fallbackLocale: "en-US",
});

export const t = i18n.global.t;

export const curLocale = i18n.global.locale;

const install = (app: App) => {
  app.use(i18n);
};

export default install;
