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
