import { capitalize, orderBy } from "lodash-es";

export const MAX_LABEL_VALUE_LENGTH = 63;

export const RESERVED_LABEL_KEYS = ["bb.environment"];

export const PRESET_LABEL_KEYS = ["bb.tenant"];

export const PRESET_DB_NAME_TEMPLATE_PLACEHOLDERS = ["DB_NAME", "TENANT"];

export const PRESET_LABEL_KEY_PLACEHOLDERS = [["TENANT", "bb.tenant"]];

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
  list: { key: string; value: string }[]
) => {
  const labels: Record<string, string> = {};
  list.forEach((kv) => {
    labels[kv.key] = kv.value;
  });
  return labels;
};

export const displayLabelKey = (key: string) => {
  if (key === "bb.environment") {
    return "Environment ID";
  }

  if (key.startsWith("bb.")) {
    const word = key.split("bb.")[1];
    return capitalize(word);
  }
  return key;
};
