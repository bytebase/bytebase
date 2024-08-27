import {
  DependentColumn,
  FunctionMetadata,
  ProcedureMetadata,
} from "@/types/proto/v1/database_service";

export const keyForFunction = (func: FunctionMetadata): string => {
  return JSON.stringify(FunctionMetadata.toJSON(func));
};

export const extractFunction = (key: string) => {
  if (!key) return FunctionMetadata.fromJSON({});
  try {
    const obj = JSON.parse(key);
    const func = FunctionMetadata.fromJSON(obj);
    return func;
  } catch {
    return FunctionMetadata.fromJSON({});
  }
};

export const keyForProcedure = (procedure: ProcedureMetadata): string => {
  return JSON.stringify(ProcedureMetadata.toJSON(procedure));
};

export const extractProcedure = (key: string) => {
  if (!key) return ProcedureMetadata.fromJSON({});
  try {
    const obj = JSON.parse(key);
    const procedure = ProcedureMetadata.fromJSON(obj);
    return procedure;
  } catch {
    return ProcedureMetadata.fromJSON({});
  }
};

export const keyForDependentColumn = (dep: DependentColumn): string => {
  return [dep.schema, dep.table, dep.column]
    .map((s) => encodeURIComponent(s))
    .join("/");
};

export const extractDependentColumn = (key: string) => {
  if (!key) return DependentColumn.fromJSON({});
  try {
    const parts = key.split("/").map((s) => decodeURIComponent(s));
    return DependentColumn.fromJSON({
      schema: parts[0],
      table: parts[1],
      column: parts[2],
    });
  } catch {
    return DependentColumn.fromJSON({});
  }
};
