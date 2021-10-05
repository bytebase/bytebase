import { ErrorMeta } from ".";

// List should be kept in sync with https://github.com/bytebase/bytebase.com/blob/main/pages/doc/errorList.ts
export const ERROR_LIST: ErrorMeta[] = [
  {
    code: 0,
    hash: "0-ok",
  },
  {
    code: 101,
    hash: "101-db-connection",
  },
  {
    code: 10001,
    hash: "10001-drop-database",
  },
];
