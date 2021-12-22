import { App } from "vue";
import { createI18n } from "vue-i18n";
import { useLocalStorage } from "@vueuse/core";

const localPathPrefix = "../locales/";

const storage = useLocalStorage("bytebase_options", {}) as any;
const locale = storage.value?.appearance?.language || navigator.language;

// import i18n resources
// https://vitejs.dev/guide/features.html#glob-import
const messages = Object.fromEntries(
  Object.entries(import.meta.globEager("../locales/*.y(a)?ml")).map(
    ([key, value]) => {
      const yaml = key.endsWith(".yaml");
      return [key.slice(localPathPrefix.length, yaml ? -5 : -4), value.default];
    }
  )
);

const i18n = createI18n({
  legacy: false,
  locale,
  globalInjection: true,
  messages,
});

export const t = i18n.global.t;

export const curLocale = i18n.global.locale;

const install = (app: App) => {
  app.use(i18n);
};

export default install;
