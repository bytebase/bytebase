import {
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
