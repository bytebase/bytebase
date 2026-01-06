import type { PickLiteral } from "../utils";
import type { Permission } from "./permission-generated";

export type { Permission };

export type QueryPermission = PickLiteral<
  Permission,
  | "bb.sql.select"
  | "bb.sql.info"
  | "bb.sql.explain"
  | "bb.sql.ddl"
  | "bb.sql.dml"
  | "bb.sql.admin"
>;

export const QueryPermissionQueryOnly: QueryPermission[] = ["bb.sql.select"];
export const QueryPermissionQueryAny: QueryPermission[] = [
  "bb.sql.select",
  "bb.sql.info",
  "bb.sql.explain",
  "bb.sql.ddl",
  "bb.sql.dml",
  "bb.sql.admin",
];
