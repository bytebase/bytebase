import { DependentColumn } from "@/types/proto/v1/database_service";

export const keyForDependentColumn = (dep: DependentColumn): string => {
  return [dep.schema, dep.table, dep.column]
    .map((s) => encodeURIComponent(s))
    .join("/");
};
