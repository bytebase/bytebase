import { orderBy } from "lodash-es";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { extractEnvironmentResourceName } from "./v1";

export const MAX_LABEL_VALUE_LENGTH = 63;

export const convertLabelsToKVList = (
  labels: Record<string, string>,
  sort = true
) => {
  const list = Object.keys(labels).map((key) => ({
    key,
    value: labels[key],
  }));

  if (sort) {
    return orderBy(list, (kv) => kv.key, "asc");
  }
  return list;
};

export const convertKVListToLabels = (
  list: { key: string; value: string }[],
  omitEmpty = true // true to omit empty values in the returned kv object
) => {
  const labels: Record<string, string> = {};
  for (const kv of list) {
    const { key, value } = kv;
    if (!value && omitEmpty) continue;
    labels[key] = value;
  }
  return labels;
};

export const getSemanticLabelValue = (db: Database, key: string) => {
  if (key === "environment") {
    return extractEnvironmentResourceName(db.effectiveEnvironment ?? "");
  }
  return db.labels[key] ?? "";
};
