import { useEnvironmentStore } from "@/store";
import { countBy, groupBy, uniq, uniqBy } from "lodash-es";
import {
  Database,
  DatabaseLabel,
  Label,
  LabelKeyType,
  LabelValueType,
} from "../types";

export const MAX_LABEL_VALUE_LENGTH = 63;

export const RESERVED_LABEL_KEYS = ["bb.environment"];

export const PRESET_LABEL_KEYS = ["bb.tenant"];

export const PRESET_DB_NAME_TEMPLATE_PLACEHOLDERS = ["DB_NAME", "TENANT"];

export const PRESET_LABEL_KEY_PLACEHOLDERS = [["TENANT", "bb.tenant"]];

export const LABEL_VALUE_EMPTY = "";

export const isReservedLabel = (label: Label): boolean => {
  return RESERVED_LABEL_KEYS.includes(label.key);
};

export const isReservedDatabaseLabel = (label: DatabaseLabel): boolean => {
  return RESERVED_LABEL_KEYS.includes(label.key);
};

export const hidePrefix = (key: LabelKeyType): LabelKeyType => {
  return key.replace(/^bb\./, "");
};

export const getLabelValueFromLabelList = (
  labels: DatabaseLabel[],
  key: LabelKeyType
): LabelValueType => {
  const label = labels.find((target) => target.key === key);
  if (!label) return LABEL_VALUE_EMPTY;
  return label.value;
};

export const getLabelValue = (
  db: Database,
  key: LabelKeyType
): LabelValueType => {
  return getLabelValueFromLabelList(db.labels, key);
};

export const setLabelValue = (
  labels: DatabaseLabel[],
  key: LabelKeyType,
  value: LabelValueType
) => {
  const index = labels.findIndex((label) => label.key === key);

  if (index < 0) {
    if (value) {
      // push new value
      labels.push({ key, value });
    }
  } else {
    if (value) {
      labels[index].value = value;
    } else {
      // remove empty value from the list
      labels.splice(index, 1);
    }
  }
};

export const groupingDatabaseListByLabelKey = (
  databaseList: Database[],
  key: LabelKeyType,
  emptyValue: LabelValueType = LABEL_VALUE_EMPTY
): Array<{ labelValue: LabelValueType; databaseList: Database[] }> => {
  const dict = groupBy(databaseList, (db) => {
    const label = db.labels.find((target) => target.key === key);
    if (!label) return emptyValue;
    return label.value;
  });
  return Object.keys(dict).map((value) => ({
    labelValue: value,
    databaseList: dict[value],
  }));
};

export const validateLabels = (labels: DatabaseLabel[]): string | undefined => {
  for (let i = 0; i < labels.length; i++) {
    const label = labels[i];
    if (!label.key) return "label.error.key-necessary";
    if (!label.value) return "label.error.value-necessary";
  }
  if (labels.length !== uniqBy(labels, "key").length) {
    return "label.error.key-duplicated";
  }
  return undefined;
};

export const validateLabelsWithTemplate = (
  labelList: DatabaseLabel[],
  requiredLabelDict: Set<LabelKeyType>
) => {
  for (const key of requiredLabelDict.values()) {
    const value = labelList.find((label) => label.key === key)?.value;
    if (!value) return false;
  }
  return true;
};

export const findDefaultGroupByLabel = (
  databaseList: Database[],
  excludedKeyList: LabelKeyType[] = []
): string => {
  // concat all databases' keys into one array
  const databaseLabelKeys = databaseList.flatMap((db) =>
    db.labels
      .map((label) => label.key)
      .filter((key) => !excludedKeyList.includes(key))
  );
  if (databaseLabelKeys.length > 0) {
    // counting up the keys' frequency
    const countsDict = countBy(databaseLabelKeys);
    const countsList = Object.keys(countsDict).map((key) => ({
      key,
      count: countsDict[key],
    }));
    // return the most frequent used key
    countsList.sort((a, b) => b.count - a.count);
    return countsList[0].key;
  }
  // Fallback to bb.environment if all databases have no labels and values.
  return "bb.environment";
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

export const getLabelValuesFromDatabaseList = (
  key: string,
  databaseList: Database[],
  withEmptyValue = false
): string[] => {
  if (key === "bb.environment") {
    const environmentList = useEnvironmentStore().getEnvironmentList();
    return environmentList.map((env) => env.resourceId);
  }

  const labelList = databaseList.flatMap((db) =>
    db.labels.filter((label) => label.key === key)
  );
  // Select all distinct database label values of {{key}}
  const distinctValueList = uniq(labelList.map((label) => label.value));

  if (withEmptyValue) {
    // plus one more "<empty value>" if needed
    distinctValueList.push(LABEL_VALUE_EMPTY);
  }

  return distinctValueList;
};
