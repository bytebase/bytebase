import { FunctionMetadata } from "@/types/proto/v1/database_service";

export const keyForFunction = (func: FunctionMetadata): string => {
  // return JSON.stringify([`functions/${func.name}`, func.definition]);
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
