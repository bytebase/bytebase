import { merge } from "lodash-es";

const localPathPrefix = "../locales/";

// import i18n resources
// https://vitejs.dev/guide/features.html#glob-import
export const mergedLocalMessage = Object.entries(
  import.meta.glob("../locales/**/*.json", { eager: true })
).reduce((map, [key, value]) => {
  const name = key.slice(localPathPrefix.length, -5);
  const sections = name.split("/");
  if (sections.length === 1) {
    map[name] = merge((value as any).default, map[name] || {});
  } else {
    const file = sections.slice(-1)[0];
    const sectionsName = sections[0];
    const existed = map[file] || {};
    map[file] = {
      ...existed,
      [sectionsName]: merge(
        (value as any).default,
        existed[sectionsName] || {}
      ),
    };
  }

  return map;
}, {} as { [k: string]: any });
