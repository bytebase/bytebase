import { useCallback, useState } from "react";

/**
 * A boolean preference remembered per browser.
 *
 * Only "true" and "false" are honored; anything else — a value another build
 * wrote, a half-written entry — falls back to the default rather than being
 * coerced, so a corrupt entry cannot silently pin the preference off.
 *
 * Reads and writes are guarded because storage throws outright in a locked-down
 * profile, and a preference is never worth failing a render over.
 *
 * `key` must be stable for the life of the component: it is read once, in the
 * initializer, so a key that changes would render the old key's value while
 * writing to the new one.
 */
export function useLocalStorageBoolean(
  key: string,
  defaultValue: boolean
): [boolean, (next: boolean) => void] {
  const [value, setValue] = useState<boolean>(() => {
    try {
      const raw = localStorage.getItem(key);
      if (raw === "true") return true;
      if (raw === "false") return false;
    } catch {
      // ignore
    }
    return defaultValue;
  });
  const update = useCallback(
    (next: boolean) => {
      setValue(next);
      try {
        localStorage.setItem(key, String(next));
      } catch {
        // ignore
      }
    },
    [key]
  );
  return [value, update];
}
