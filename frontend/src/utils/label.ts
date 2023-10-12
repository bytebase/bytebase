import { orderBy, uniq } from "lodash-es";
import { useEnvironmentV1Store } from "@/store";
import { ComposedDatabase } from "@/types";
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

export const getLabelValuesFromDatabaseV1List = (
  key: string,
  databaseList: ComposedDatabase[],
  withEmptyValue = false
): string[] => {
  if (key === "environment") {
    const environmentList = useEnvironmentV1Store().getEnvironmentList();
    return environmentList.map((env) =>
      extractEnvironmentResourceName(env.name)
    );
  }

  const valueList = databaseList.flatMap((db) => {
    if (key in db.labels) {
      return getSemanticLabelValue(db, key);
    }
    return [];
  });
  // Select all distinct database label values of {{key}}
  const distinctValueList = uniq(valueList);

  if (withEmptyValue) {
    // plus one more "<empty value>" if needed
    distinctValueList.push("");
  }

  return distinctValueList;
};

export const getSemanticLabelValue = (db: ComposedDatabase, key: string) => {
  if (key === "environment") {
    return extractEnvironmentResourceName(db.effectiveEnvironment);
  }
  return db.labels[key] ?? "";
};
