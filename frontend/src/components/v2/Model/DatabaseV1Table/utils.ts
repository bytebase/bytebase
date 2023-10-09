import { has } from "lodash-es";
import { ComposedDatabase, ComposedDatabaseGroup } from "@/types";

export const isDatabase = (
  data: ComposedDatabase | ComposedDatabaseGroup
): boolean => {
  return has(data, "uid");
};
