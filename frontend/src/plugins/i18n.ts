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

const install = (app: App) => {
  const i18n = createI18n({
    legacy: false,
    locale,
    globalInjection: true,
    messages,
  });

  app.use(i18n);
};

export default install;
