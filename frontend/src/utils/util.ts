import type { Duration } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import dayOfYear from "dayjs/plugin/dayOfYear";
import duration from "dayjs/plugin/duration";
import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";
import DOMPurify from "dompurify";
import { escape as escapeHtml, escapeRegExp, round } from "lodash-es";
import semver from "semver";
import { type Ref, watchEffect } from "vue";

dayjs.extend(dayOfYear);
dayjs.extend(duration);
dayjs.extend(relativeTime);
dayjs.extend(utc);

export function isDev(): boolean {
  return import.meta.env.DEV;
}

export function isRelease(): boolean {
  return import.meta.env.PROD;
}

export function humanizeTs(ts: number): string {
  const time = dayjs.utc(ts * 1000);
  if (dayjs().year() == time.year()) {
    if (dayjs().dayOfYear() == time.dayOfYear()) {
      return time.local().format("HH:mm");
    }
    if (dayjs().diff(time, "days") < 3) {
      return time.local().format("MMM D HH:mm");
    }
    return time.local().format("MMM D");
  }
  return time.local().format("MMM D YYYY");
}

export const humanizeDurationV1 = (duration: Duration | undefined) => {
  if (!duration) return "-";
  const { seconds, nanos } = duration;
  const totalMs = Number(seconds) * 1000 + nanos / 1e6;

  // For durations less than 1 second, show in milliseconds
  if (totalMs < 1000) {
    if (totalMs < 0.01) {
      return "<0.01ms";
    }
    // For sub-10ms, show 2 decimal places for higher precision
    if (totalMs < 10) {
      return `${totalMs.toFixed(2)}ms`;
    }
    // For 10-100ms, show 1 decimal place
    if (totalMs < 100) {
      return `${totalMs.toFixed(1)}ms`;
    }
    // For 100ms-1s, show no decimal places
    return `${totalMs.toFixed(0)}ms`;
  }

  // For durations between 1-60 seconds, show in seconds with 1 decimal
  const totalSeconds = totalMs / 1000;
  if (totalSeconds < 60) {
    return `${totalSeconds.toFixed(1)}s`;
  }

  // For durations between 1-60 minutes, show in minutes and seconds
  const minutes = Math.floor(totalSeconds / 60);
  if (minutes < 60) {
    const remainingSeconds = Math.floor(totalSeconds % 60);
    if (remainingSeconds === 0) {
      return `${minutes}m`;
    }
    return `${minutes}m${remainingSeconds}s`;
  }

  // For durations over 1 hour, show in hours and minutes
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  if (remainingMinutes === 0) {
    return `${hours}h`;
  }
  return `${hours}h${remainingMinutes}m`;
};

export function bytesToString(size: number): string {
  const unitList = ["B", "KB", "MB", "GB", "TB"];
  let i = 0;
  for (i = 0; i < unitList.length; i++) {
    if (size < 1024) {
      break;
    }
    size = size / 1024;
  }

  return round(size, 2) + " " + unitList[i];
}

export function nanosecondsToString(nanoseconds: number): string {
  // dayjs.duration() takes the length of time in milliseconds.
  return dayjs.duration(nanoseconds / 1000000).humanize();
}

export function timezoneString(zoneName: string, offset: number): string {
  let sign = "+";
  if (offset < 0) {
    sign = "-";
  }
  const hour = Math.abs(offset) / 3600;
  const minutes = Math.abs(offset) & (3600 / 60);
  return `${zoneName}${sign}${String(hour).padStart(2, "0")}:${String(
    minutes
  ).padStart(2, "0")}`;
}

export function urlfy(str: string): string {
  let result = str.trim();
  if (result.search(/^http[s]?:\/\//) == -1) {
    result = "http://" + result;
  }
  return result;
}

export function isUrl(str: string): boolean {
  let url;

  try {
    url = new URL(str);
  } catch {
    return false;
  }

  return url.protocol === "http:" || url.protocol === "https:";
}

// Performs inline swap, also handles negative index (counting from the end)
// arraySwap([1, 2, 3, 4], 1, 2) => [1, 3, 2, 4]
// arraySwap([1, 2, 3, 4], -1, -2) => [1, 2, 4, 3]
export function arraySwap(arr: unknown[], oldIndex: number, newIndex: number) {
  while (oldIndex < 0) {
    oldIndex += arr.length;
  }
  while (newIndex < 0) {
    newIndex += arr.length;
  }
  if (newIndex >= arr.length) {
    let k = newIndex - arr.length + 1;
    while (k--) {
      arr.push(undefined);
    }
  }
  arr.splice(newIndex, 0, arr.splice(oldIndex, 1)[0]);
}

export function sizeToFit(
  el: HTMLTextAreaElement | undefined,
  padding = 2,
  max = -1,
  min = -1
) {
  if (!el) return;

  el.style.height = "auto";
  // Extra several pixels are to prevent jiggling upon entering the text
  let height = el.scrollHeight + padding;
  if (max >= 0 && height > max) height = max;
  if (min >= 0 && height < min) height = min;
  el.style.height = `${height}px`;
}

export function isValidEmail(email: string) {
  // Rather than using esoteric regex complying RFC 822/2822, we just use a naive but readable version
  // which should work most of the time.
  const re = /\S+@\S+\.\S+/;
  return re.test(email);
}

export function randomString(
  n: number,
  candidate: string = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
): string {
  if (!n) {
    n = 16;
  }
  let result = "";
  const charactersLength = candidate.length;
  for (let i = 0; i < n; i++) {
    result += candidate.charAt(Math.floor(Math.random() * charactersLength));
  }
  return result;
}

/**
 * Safely highlights search keywords in text by escaping HTML and wrapping matches
 * in <b> tags. Follows OWASP XSS prevention guidelines with defense-in-depth:
 * 1. HTML escape to convert any special characters to entities
 * 2. Apply highlighting with safe markup
 * 3. Sanitize with DOMPurify to ensure only allowed tags/attributes remain
 *
 * This prevents DOM XSS attacks from malicious input in search highlighting.
 *
 * @param s - The text to search in
 * @param k - The keyword to highlight (case-sensitive)
 * @returns HTML-safe string with highlighted keywords
 */
export function getHighlightHTMLByKeyWords(s: string, k: string) {
  // Step 1: Escape HTML entities to prevent XSS (converts <, >, &, ", ', ` to entities)
  const escapedText = escapeHtml(s);
  if (!k) return escapedText;

  // Step 2: Escape the search keyword and apply highlighting
  const escapedKeyword = escapeHtml(k);
  const highlighted = escapedText.replaceAll(
    escapedKeyword,
    `<b class="text-accent">${escapedKeyword}</b>`
  );

  // Step 3: Sanitize with DOMPurify as defense-in-depth (allowlist-based)
  return DOMPurify.sanitize(highlighted, {
    ALLOWED_TAGS: ["b"],
    ALLOWED_ATTR: ["class"],
  });
}

/**
 * Safely highlights text matching a regex pattern by escaping HTML and wrapping
 * matches in <b> tags. Follows OWASP XSS prevention guidelines with defense-in-depth.
 *
 * This prevents DOM XSS attacks from malicious input in search highlighting.
 *
 * @param s - The text to search in
 * @param pattern - String(s) to highlight (converted to escaped regex pattern)
 * @param caseSensitive - Whether the search should be case-sensitive
 * @param className - CSS class for the highlight tag (sanitized by DOMPurify)
 * @returns HTML-safe string with highlighted matches
 */
export function getHighlightHTMLByRegExp(
  target: string,
  pattern: string | string[],
  caseSensitive = false,
  className = "text-accent"
) {
  // Step 1: Escape HTML entities to prevent XSS
  const escapedText = escapeHtml(target);
  if (!pattern || (Array.isArray(pattern) && pattern.length === 0)) {
    return escapedText;
  }

  // Step 2: Build safe regex pattern and apply highlighting
  pattern = Array.isArray(pattern)
    ? pattern.map((kw) => escapeRegExp(kw)).join("|")
    : escapeRegExp(pattern);
  const flags = caseSensitive ? "g" : "gi";
  const re = new RegExp(pattern, flags);
  const highlighted = escapedText.replaceAll(
    re,
    (k) => `<b class="${escapeHtml(className)}">${k}</b>`
  );

  // Step 3: Sanitize with DOMPurify as defense-in-depth (allowlist-based)
  return DOMPurify.sanitize(highlighted, {
    ALLOWED_TAGS: ["b"],
    ALLOWED_ATTR: ["class"],
  });
}

export type Defer<T> = {
  promise: Promise<T>;
  resolve: (value: T | PromiseLike<T>) => void;
  reject: (reason?: unknown) => void;
};
export function defer<T = unknown>() {
  const d = {} as Defer<T>;
  d.promise = new Promise<T>((resolve, reject) => {
    d.resolve = resolve;
    d.reject = reject;
  });
  return d;
}

export const sleep = (ms: number) => {
  return new Promise((resolve) => setTimeout(resolve, ms));
};

/**
 * Wrap a Ref as a Promise, will be resolved when the Ref turns to expectedValue
 * first time
 */
export const wrapRefAsPromise = <T>(r: Ref<T>, expectedValue: T) => {
  return new Promise<void>((resolve) => {
    watchEffect(() => {
      if (r.value === expectedValue) {
        // Need not to care about resolving more than once
        // Since `Promise` will handle this
        resolve();
      }
    });
  });
};

// emitStorageChangedEvent is used to notify the storage changed event
export function emitStorageChangedEvent() {
  const iframeEl = document.createElement("iframe");
  iframeEl.style.display = "none";
  document.body.appendChild(iframeEl);

  iframeEl.contentWindow?.localStorage.setItem("t", Date.now().toString());
  iframeEl.remove();
}

export function removeElementBySelector(selector: string) {
  document.body.querySelectorAll(selector).forEach((e) => e.remove());
}

type CompareFunc = "gt" | "lt" | "eq" | "neq" | "gte" | "lte";
// semverCompare compares version string v1 is greater than v2.
// It should be used to handle the database pseudo semantic version likes "8.0.29-0ubuntu0.20.04.3".
export function semverCompare(
  v1: string,
  v2: string,
  method: CompareFunc = "gt"
) {
  const formattedV1 = semver.coerce(v1);
  const formattedV2 = semver.coerce(v2);
  if (!formattedV1 || !formattedV2) {
    return false;
  }

  return semver[method](formattedV1, formattedV2);
}

export function clearObject(obj: Record<string, unknown>) {
  const keys = Object.keys(obj);
  keys.forEach((key) => delete obj[key]);
  return obj;
}

const MODIFIERS = [
  "cmd",
  "ctrl",
  "cmd_or_ctrl",
  "opt",
  "alt",
  "opt_or_alt",
  "shift",
] as const;
export type ModifierKey = (typeof MODIFIERS)[number];

export const modifierKeyText = (mod: ModifierKey) => {
  const isMac = navigator.userAgent.search("Mac") !== -1;
  if (mod === "cmd" || (mod === "cmd_or_ctrl" && isMac)) {
    return "⌘"; // U+2318
  }
  if (mod === "ctrl" && isMac) {
    return "⌃"; // U+2303
  }
  if ((mod === "ctrl" && !isMac) || (mod === "cmd_or_ctrl" && !isMac)) {
    return "Ctrl";
  }
  if (mod === "opt" || (mod === "opt_or_alt" && isMac)) {
    return "⌥"; // U+2325
  }
  if (mod === "alt" || (mod === "opt_or_alt" && !isMac)) {
    return "Alt";
  }
  if (mod === "shift" && isMac) {
    return "⇧"; // U+21E7
  }
  if (mod === "shift" && !isMac) {
    return "Shift";
  }
  console.assert(false, "should never reach this line");
  return "";
};

export const keyboardShortcutStr = (str: string) => {
  const parts = str.split("+");
  return parts
    .map((part) => {
      const mod = part as ModifierKey;
      if (MODIFIERS.includes(mod)) return modifierKeyText(mod);
      return part;
    })
    .join("+");
};

export const isNullOrUndefined = (value: unknown) => {
  return value === null || value === undefined;
};

export const onlyAllowNumber = (value: string) => {
  return value === "" || /^\d+$/.test(value);
};

export const allEqual = <T>(...values: T[]) => {
  if (values.length <= 1) return true;
  const first = values[0];
  for (let i = 1; i < values.length; i++) {
    if (values[i] !== first) return false;
  }
  return true;
};
