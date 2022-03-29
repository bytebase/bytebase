import { ErrorMeta } from ".";

// List should be kept in sync with https://github.com/bytebase/bytebase.com/blob/main/docs/error-code.md
// Bytebase uses this list to display link to the doc site.
export const ERROR_LIST: ErrorMeta[] = [
  {
    code: 0,
    hash: "0---ok",
  },
  {
    code: 1,
    hash: "1---internal",
  },
  {
    code: 101,
    hash: "101---db-connection",
  },
  {
    code: 102,
    hash: "102---statement-syntax",
  },
  {
    code: 103,
    hash: "103---statement-execution",
  },
  {
    code: 201,
    hash: "201---migration-schema-missing",
  },
  {
    code: 202,
    hash: "202---migration-already-applied",
  },
  {
    code: 203,
    hash: "203---migration-out-of-order",
  },
  {
    code: 204,
    hash: "204---migration-baseline-missing",
  },
  {
    code: 10001,
    hash: "10001---drop-database",
  },
  {
    code: 10002,
    hash: "10002---rename-table",
  },
  {
    code: 10003,
    hash: "10003---drop-table",
  },
  {
    code: 10004,
    hash: "10004---rename-column",
  },
  {
    code: 10005,
    hash: "10005---drop-column",
  },
  {
    code: 10006,
    hash: "10006---add-primary-key",
  },
  {
    code: 10007,
    hash: "10007---add-unique-key",
  },
  {
    code: 10008,
    hash: "10008---add-foreign-key",
  },
  {
    code: 10009,
    hash: "10009---add-check",
  },
  {
    code: 10010,
    hash: "10010---alter-check",
  },
  {
    code: 10011,
    hash: "10011---alter-column",
  },
];
