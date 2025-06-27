import type { DependencyColumn } from "@/types/proto-es/v1/database_service_pb";

export const keyForDependencyColumn = (dep: DependencyColumn): string => {
  return [dep.schema, dep.table, dep.column]
    .map((s) => encodeURIComponent(s))
    .join("/");
};
