import dayjs from "dayjs";
import dayOfYear from "dayjs/plugin/dayOfYear";
import duration from "dayjs/plugin/duration";
import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";
import { escapeRegExp, round } from "lodash-es";
import semver from "semver";
import { watchEffect, type Ref } from "vue";
import type { Duration } from "@/types/proto/google/protobuf/duration";

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

export function humanizeDuration(seconds: number): string {
  if (seconds <= 1) return "Less than 1s";
  return `${seconds}s`;
}

export const humanizeDurationV1 = (
  duration: Duration | undefined,
  brief = true
) => {
  if (!duration) return "-";
  const { seconds, nanos } = duration;
  const total = seconds.toNumber() + nanos / 1e9;
  if (brief && total <= 1) {
    return "Less than 1s";
  }
  return total.toFixed(2) + "s";
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
export function arraySwap(arr: any[], oldIndex: number, newIndex: number) {
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

export function getIntCookie(name: string): number | undefined {
  const list = document.cookie.split(";");
  for (let i = 0; i < list.length; i++) {
    const parts = list[i].split("=");
    if (parts[0].trim() == name) {
      return parts.length > 1 ? parseInt(parts[1]) : undefined;
    }
  }

  return undefined;
}

export function getStringCookie(name: string): string {
  const list = document.cookie.split(";");
  for (let i = 0; i < list.length; i++) {
    const parts = list[i].split("=");
    if (parts[0].trim() == name) {
      // For now, just assumes strings are enclosed by quotes
      return parts.length > 1 ? parts[1].slice(1, -1) : "";
    }
  }

  return "";
}

export function getHighlightHTMLByKeyWords(s: string, k: string) {
  if (!k) return s;
  return s.replaceAll(k, `<b class="text-accent">${k}</b>`);
}

export function getHighlightHTMLByRegExp(
  s: string,
  pattern: string | string[],
  caseSensitive = false,
  className = "text-accent"
) {
  pattern = Array.isArray(pattern)
    ? pattern.map((kw) => escapeRegExp(kw)).join("|")
    : escapeRegExp(pattern);
  const flags = caseSensitive ? "g" : "gi";
  const re = new RegExp(pattern, flags);
  return s.replaceAll(re, (k) => `<b class="${className}">${k}</b>`);
}

export type Defer<T> = {
  promise: Promise<T>;
  resolve: (value: T | PromiseLike<T>) => void;
  reject: (reason?: any) => void;
};
export function defer<T = any>() {
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

export function clearObject(obj: any) {
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

export const isNullOrUndefined = (value: any) => {
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
