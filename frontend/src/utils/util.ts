import dayjs from "dayjs";
import dayOfYear from "dayjs/plugin/dayOfYear";
import duration from "dayjs/plugin/duration";
import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";

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

export function bytesToString(size: number): string {
  const unitList = ["B", "KB", "MB", "GB", "TB"];
  let i = 0;
  for (i = 0; i < unitList.length; i++) {
    if (size < 1024) {
      break;
    }
    size = size / 1024;
  }
  return size.toString() + " " + unitList[i];
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
  } catch (_) {
    return false;
  }

  return url.protocol === "http:" || url.protocol === "https:";
}

// Performs inline swap, also handles negative index (counting from the end)
// array_swap([1, 2, 3, 4], 1, 2) => [1, 3, 2, 4]
// array_swap([1, 2, 3, 4], -1, -2) => [1, 2, 4, 3]
export function array_swap(arr: any[], old_index: number, new_index: number) {
  while (old_index < 0) {
    old_index += arr.length;
  }
  while (new_index < 0) {
    new_index += arr.length;
  }
  if (new_index >= arr.length) {
    let k = new_index - arr.length + 1;
    while (k--) {
      arr.push(undefined);
    }
  }
  arr.splice(new_index, 0, arr.splice(old_index, 1)[0]);
}

export function sizeToFit(el: HTMLTextAreaElement | undefined) {
  if (!el) return;

  el.style.height = "auto";
  // Extra 2px is to prevent jiggling upon entering the text
  el.style.height = `${el.scrollHeight + 2}px`;
}

export function isValidEmail(email: string) {
  // Rather than using esoteric regex complying RFC 822/2822, we just use a naive but readable version
  // which should work most of the time.
  const re = /\S+@\S+\.\S+/;
  return re.test(email);
}

export function randomString(n?: number): string {
  if (!n) {
    n = 16;
  }
  let result = "";
  const characters =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  const charactersLength = characters.length;
  for (let i = 0; i < n; i++) {
    result += characters.charAt(Math.floor(Math.random() * charactersLength));
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
  return s.replaceAll(k, `<b class="text-accent">${k}</b>`);
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
