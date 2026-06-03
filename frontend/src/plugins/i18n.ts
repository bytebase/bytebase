// The app's i18n is react-i18next (`@/react/i18n`). This module is a thin
// compatibility layer over that single instance for non-React callers — shared
// `.ts` modules that translate outside React components via `t` / `te` /
// `locale`. React components use `useTranslation()` directly. vue-i18n is gone.
import i18n from "@/react/i18n";
import { mergedLocalMessage } from "./i18n-messages";

// The shared `.ts` callers reference some legacy keys that aren't in the
// React locale files (which are scoped to React-component usage). Merge the
// legacy message set into the same react-i18next instance AT RUNTIME so those
// keys resolve, without duplicating them into the React locale files (keeps
// the React-i18n guard + locale set clean). React-owned keys win on conflict.
// Guarded so tests that mock `@/react/i18n` with a partial instance don't
// break at import time (addResourceBundle is always present on the real one).
if (typeof i18n.addResourceBundle === "function") {
  for (const [lng, messages] of Object.entries(
    mergedLocalMessage as Record<string, Record<string, unknown>>
  )) {
    i18n.addResourceBundle(
      lng,
      "translation",
      messages,
      /* deep */ true,
      /* overwrite */ false
    );
  }
}

export const t = i18n.t.bind(i18n);

export const te = (key: string): boolean => i18n.exists(key);

// Writable locale accessor mirroring the old vue-i18n `WritableComputedRef`
// shape (`.value` get/set) for non-React callers.
export const locale = {
  get value(): string {
    return i18n.language;
  },
  set value(next: string) {
    void i18n.changeLanguage(next);
  },
};

export default i18n;
