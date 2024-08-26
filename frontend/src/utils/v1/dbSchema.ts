import { head } from "lodash-es";
import type { FunctionMetadata } from "@/types/proto/v1/database_service";

export const keyForFunction = (func: FunctionMetadata) => {
  return JSON.stringify([`functions/${func.name}`, func.definition]);
};

export const extractFunctionName = (key: string) => {
  if (!key) return "";
  try {
    const parts = JSON.parse(key);
    if (!Array.isArray(parts)) return "";
    const name = head(parts);
    if (typeof name !== "string") return "";

    const pattern = /(?:^|\/)functions\/([^/]+)(?:$|\/)/;
    const matches = name.match(pattern);
    return matches?.[1] ?? "";
  } catch {
    return "";
  }
};
