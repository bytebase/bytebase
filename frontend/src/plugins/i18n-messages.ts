import { merge } from "lodash-es";

const localPathPrefix = "../locales/";
type LocaleModule = { default: Record<string, unknown> };

export type LocaleMessageObject = {
  [key: string]: string | LocaleMessageObject;
};

const localeModules = import.meta.glob("../locales/**/*.json", { eager: true });
const rootMessages: Record<string, Record<string, unknown>> = {};
const sectionMessages: Record<string, Record<string, unknown>> = {};

// import i18n resources
// https://vitejs.dev/guide/features.html#glob-import
for (const [key, value] of Object.entries(localeModules)) {
  const name = key.slice(localPathPrefix.length, -5);
  const sections = name.split("/");
  const messages = (value as LocaleModule).default;
  if (sections.length === 1) {
    rootMessages[name] = messages;
  } else {
    const file = sections.at(-1) ?? "";
    const sectionsName = sections[0];
    sectionMessages[file] = sectionMessages[file] ?? {};
    sectionMessages[file][sectionsName] = merge(
      sectionMessages[file][sectionsName] ?? {},
      messages
    );
  }
}

export const mergedLocalMessage = Object.fromEntries(
  Object.keys({ ...sectionMessages, ...rootMessages }).map((locale) => [
    locale,
    merge({}, sectionMessages[locale] ?? {}, rootMessages[locale] ?? {}),
  ])
) as LocaleMessageObject;
