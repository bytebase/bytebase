import { DependencyColumn } from "@/types/proto/api/v1alpha/database_service";

export const keyForDependencyColumn = (dep: DependencyColumn): string => {
  return [dep.schema, dep.table, dep.column]
    .map((s) => encodeURIComponent(s))
    .join("/");
};
