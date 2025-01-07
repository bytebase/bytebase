import { DependencyColumn } from "@/types/proto/v1/database_service";

export const keyForDependencyColumn = (dep: DependencyColumn): string => {
  return [dep.schema, dep.table, dep.column]
    .map((s) => encodeURIComponent(s))
    .join("/");
};
