import { orderBy, uniq } from "lodash-es";
import { useEnvironmentV1Store } from "@/store";
import { ComposedDatabase } from "@/types";
import {
  LabelSelector,
  LabelSelectorRequirement,
  OperatorType,
  Schedule,
} from "@/types/proto/v1/project_service";
import { extractEnvironmentResourceName } from "./v1";

export const MAX_LABEL_VALUE_LENGTH = 63;

export const RESERVED_LABEL_KEYS = ["environment"];

export const PRESET_LABEL_KEYS = ["tenant"];

export const PRESET_DB_NAME_TEMPLATE_PLACEHOLDERS = ["DB_NAME", "TENANT"];

export const PRESET_LABEL_KEY_PLACEHOLDERS = [["TENANT", "tenant"]];

export const hidePrefix = (key: string): string => {
  return key.replace(/^bb\./, "");
};

export const validateLabelsWithTemplate = (
  labels: Record<string, string>,
  requiredLabelDict: Set<string>
) => {
  for (const key of requiredLabelDict.values()) {
    const value = labels[key];
    if (!value) return false;
  }
  return true;
};

export const parseLabelListInTemplate = (template: string): string[] => {
  const labelList: string[] = [];

  PRESET_LABEL_KEY_PLACEHOLDERS.forEach(([placeholder, labelKey]) => {
    const pattern = `{{${placeholder}}}`;
    if (template.includes(pattern)) {
      labelList.push(labelKey);
    }
  });

  return labelList;
};

export const validateLabelKey = (key: string) => {
  if (key.length === 0) return false;
  if (key.length >= 64) return false;
  return (
    key.match(/^[a-z0-9A-Z]/) &&
    key.match(/[a-z0-9A-Z]$/) &&
    key.match(/[a-z0-9A-Z-_.]*/)
  );
};

export const isReservedLabel = (key: string) => {
  return RESERVED_LABEL_KEYS.includes(key);
};

export const isPresetLabel = (key: string) => {
  return PRESET_LABEL_KEYS.includes(key);
};

export const getAvailableLabelKeyList = (
  databaseList: ComposedDatabase[],
  withReserved: boolean,
  withPreset: boolean,
  sort = true
) => {
  const keys = uniq(databaseList.flatMap((db) => Object.keys(db.labels)));
  if (withReserved) {
    RESERVED_LABEL_KEYS.forEach((key) => {
      if (!keys.includes(key)) keys.push(key);
    });
  }
  if (withPreset) {
    PRESET_LABEL_KEYS.forEach((key) => {
      if (!keys.includes(key)) keys.push(key);
    });
  }
  if (sort) {
    return orderBy(
      keys,
      [
        (key) => (isReservedLabel(key) ? -1 : 1),
        (key) => (isPresetLabel(key) ? -1 : 1),
        (key) => key,
      ],
      ["asc", "asc", "asc"]
    );
  }
  return keys;
};

export const convertLabelsToKVList = (
  labels: Record<string, string>,
  sort = true
) => {
  const list = Object.keys(labels).map((key) => ({
    key,
    value: labels[key],
  }));

  if (sort) {
    // 1. Preset
    // 2. Others (lexicographical order)
    // ...
    // 3. Hidden
    return orderBy(
      list,
      [
        (kv) => (isReservedLabel(kv.key) ? 1 : -1),
        (kv) => (isPresetLabel(kv.key) ? -1 : 1),
        (kv) => kv.key,
      ],
      ["asc", "asc", "asc"]
    );
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

export const displayLabelKey = (key: string) => {
  if (key === "environment") {
    return "Environment ID";
  }

  return key;
};

export const getPipelineFromDeploymentScheduleV1 = (
  databaseList: ComposedDatabase[],
  schedule: Schedule | undefined
): ComposedDatabase[][] => {
  const stages: ComposedDatabase[][] = [];

  const collectedIds = new Set<string>();
  schedule?.deployments.forEach((deployment) => {
    const dbs: ComposedDatabase[] = [];
    databaseList.forEach((db) => {
      if (collectedIds.has(db.uid)) return;
      if (isDatabaseMatchesSelectorV1(db, deployment.spec?.labelSelector)) {
        dbs.push(db);
        collectedIds.add(db.uid);
      }
    });
    stages.push(dbs);
  });

  return stages;
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

export const isDatabaseMatchesSelectorV1 = (
  database: ComposedDatabase,
  selector: LabelSelector | undefined
): boolean => {
  const rules = selector?.matchExpressions ?? [];
  return rules.every((rule) => {
    switch (rule.operator) {
      case OperatorType.OPERATOR_TYPE_IN:
        return checkLabelIn(database, rule);
      case OperatorType.OPERATOR_TYPE_EXISTS:
        return checkLabelExists(database, rule);
      default:
        // unknown operators are taken as mismatch
        console.warn(`known operator "${rule.operator}"`);
        return false;
    }
  });
};

export const getSemanticLabelValue = (db: ComposedDatabase, key: string) => {
  if (key === "environment") {
    return extractEnvironmentResourceName(db.effectiveEnvironment);
  }
  return db.labels[key] ?? "";
};

const checkLabelIn = (
  db: ComposedDatabase,
  rule: LabelSelectorRequirement
): boolean => {
  const value = getSemanticLabelValue(db, rule.key);
  if (!value) return false;

  return rule.values.includes(value);
};

const checkLabelExists = (
  db: ComposedDatabase,
  rule: LabelSelectorRequirement
): boolean => {
  return !!getSemanticLabelValue(db, rule.key);
};
