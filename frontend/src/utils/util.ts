import moment from "moment";

export function isDev(): boolean {
  return import.meta.env.DEV;
}

export function isRelease(): boolean {
  return import.meta.env.PROD;
}

// Use VITE_BACKEND as the backendURL if specified. Otherwise, just use the current host
// For dev environment, we usually set VITE_BACKEND to point to different backend.
// For production environment, we won't set VITE_BACKEND because we bundle frontend/backend as a monolithic.
export function backendURL(): string {
  if (import.meta.env.VITE_BACKEND) {
    return import.meta.env.VITE_BACKEND as string;
  }
  return window.location.origin;
}

export function humanizeTs(ts: number): string {
  const time = moment.utc(ts * 1000);
  if (moment().year() == time.year()) {
    if (moment().dayOfYear() == time.dayOfYear()) {
      return time.local().format("HH:mm");
    }
    if (moment().diff(time, "days") < 3) {
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

export function urlfy(str: string): string {
  let result = str.trim();
  if (result.search(/^http[s]?\:\/\//) == -1) {
    result = "http://" + result;
  }
  return result;
}

export function isURL(str: string): boolean {
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
    var k = new_index - arr.length + 1;
    while (k--) {
      arr.push(undefined);
    }
  }
  arr.splice(new_index, 0, arr.splice(old_index, 1)[0]);
}

export function sizeToFit(el: HTMLTextAreaElement) {
  el.style.height = "auto";
  // Extra 2px is to prevent jiggling upon entering the text
  el.style.height = `${el.scrollHeight + 2}px`;
}

export function isValidEmail(email: string) {
  // Rather than using esoteric regex complying RFC 822/2822, we just use a naive but readable version
  // which should work most of the time.
  var re = /\S+@\S+\.\S+/;
  return re.test(email);
}

export function randomString(n?: number): string {
  if (!n) {
    n = 16;
  }
  var result = "";
  var characters =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  var charactersLength = characters.length;
  for (var i = 0; i < n; i++) {
    result += characters.charAt(Math.floor(Math.random() * charactersLength));
  }
  return result;
}

export function getIntCookie(name: string): number | undefined {
  const list = document.cookie.split(";");
  for (var i = 0; i < list.length; i++) {
    const parts = list[i].split("=");
    if (parts[0].trim() == name) {
      return parts.length > 1 ? parseInt(parts[1]) : undefined;
    }
  }

  return undefined;
}

export function getStringCookie(name: string): string {
  const list = document.cookie.split(";");
  for (var i = 0; i < list.length; i++) {
    const parts = list[i].split("=");
    if (parts[0].trim() == name) {
      // For now, just assumes strings are enclosed by quotes
      return parts.length > 1 ? parts[1].slice(1, -1) : "";
    }
  }

  return "";
}

export function removeCookie(name: string) {
  const newList = [];
  const list = document.cookie.split(";");
  for (var i = 0; i < list.length; i++) {
    const parts = list[i].split("=");
    if (parts[0].trim() == name) {
      newList.push(`${name}=;expires=Thu, 01 Jan 1970 00:00:00 GMT`);
    } else {
      newList.push(list[i]);
    }
  }
  document.cookie = newList.join(";");
}
