import { merge } from "lodash-es";

const localPathPrefix = "../locales/";
type LocaleModule = { default: Record<string, unknown> };

export type LocaleMessageObject = {
  [key: string]: string | LocaleMessageObject;
};

// import i18n resources
// https://vitejs.dev/guide/features.html#glob-import
export const mergedLocalMessage = Object.entries(
  import.meta.glob("../locales/**/*.json", { eager: true })
).reduce<LocaleMessageObject>((map, [key, value]) => {
  const name = key.slice(localPathPrefix.length, -5);
  const sections = name.split("/");
  if (sections.length === 1) {
    map[name] = merge((value as LocaleModule).default, map[name] || {});
  } else {
    const file = sections.slice(-1)[0];
    const sectionsName = sections[0];
    const existed = (map[file] || {}) as LocaleMessageObject;
    map[file] = {
      ...existed,
      [sectionsName]: merge(
        (value as LocaleModule).default,
        existed[sectionsName] || {}
      ),
    };
  }

  return map;
}, {});
