import { App } from "vue";
import { createI18n } from "vue-i18n";
import { useLocalStorage } from "@vueuse/core";

const localPathPrefix = "../locales/";
const validLocaleList = ["en-US", "zh-CN"];

const getValidLocale = () => {
  const storage = useLocalStorage("bytebase_options", {}) as any;

  const params = new URL(window.location.href).searchParams;
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
  if (validLocaleList.includes(locale)) {
    return locale;
  }

  return "en-US";
};

// import i18n resources
// https://vitejs.dev/guide/features.html#glob-import
const mergedLocalMessage = Object.entries(
  import.meta.globEager("../locales/**/*.json")
).reduce((map, [key, value]) => {
  const name = key.slice(localPathPrefix.length, -5);
  const sections = name.split("/");
  if (sections.length === 1) {
    map[name] = value.default;
  } else {
    const file = sections.slice(-1)[0];
    const sectionsName = sections[0];
    const existed = map[file] || {};
    map[file] = {
      ...existed,
      [sectionsName]: {
        ...(existed[sectionsName] || {}),
        ...value.default,
      },
    };
  }

  return map;
}, {} as { [k: string]: any });

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

export const curLocale = i18n.global.locale;

const install = (app: App) => {
  app.use(i18n);
};

export default install;
