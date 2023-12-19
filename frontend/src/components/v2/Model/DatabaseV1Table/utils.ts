import { has } from "lodash-es";
import { ComposedDatabase, ComposedDatabaseGroup } from "@/types";

export type Mode =
  | "ALL"
  | "ALL_SHORT"
  | "ALL_TINY"
  | "INSTANCE"
  | "PROJECT"
  | "PROJECT_SHORT";

export const isDatabase = (
  data: ComposedDatabase | ComposedDatabaseGroup
): boolean => {
  return has(data, "uid");
};
