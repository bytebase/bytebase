import { isEqual, snakeCase } from "lodash-es";
import { humanizeTs } from "../util";

export const calcUpdateMask = (
  a: Record<string, unknown>,
  b: Record<string, unknown>,
  toSnakeCase = false
) => {
  const updateMask = new Set<string>();
  const aKeys = new Set(Object.keys(a));
  const bKeys = new Set(Object.keys(b));

  for (const key of aKeys) {
    if (!isEqual(a[key], b[key])) {
      updateMask.add(key);
    }
    bKeys.delete(key);
  }
  const keys = [...updateMask.values(), ...bKeys.values()];
  return toSnakeCase ? keys.map(snakeCase) : keys;
};

export const humanizeDate = (date: Date | undefined) => {
  return humanizeTs(Math.floor((date?.getTime() ?? 0) / 1000));
};

// The list of supported encodings.
// Reference: https://developer.mozilla.org/en-US/docs/Web/API/Encoding_API/Encodings
export const ENCODINGS = [
  "utf-8",
  "ibm866",
  "iso-8859-2",
  "iso-8859-3",
  "iso-8859-4",
  "iso-8859-5",
  "iso-8859-6",
  "iso-8859-7",
  "iso-8859-8",
  "iso-8859-8i",
  "iso-8859-10",
  "iso-8859-13",
  "iso-8859-14",
  "iso-8859-15",
  "iso-8859-16",
  "koi8-r",
  "koi8-u",
  "macintosh",
  "windows-874",
  "windows-1250",
  "windows-1251",
  "windows-1252",
  "windows-1253",
  "windows-1254",
  "windows-1255",
  "windows-1256",
  "windows-1257",
  "windows-1258",
  "x-mac-cyrillic",
  "gbk",
  "gb18030",
  "hz-gb-2312",
  "big5",
  "euc-jp",
  "iso-2022-jp",
  "shift-jis",
  "euc-kr",
  "iso-2022-kr",
  "utf-16be",
  "utf-16le",
];

export type Encoding = (typeof ENCODINGS)[number];
