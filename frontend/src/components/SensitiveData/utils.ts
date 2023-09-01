import {
  MaskData,
  MaskingExceptionPolicy_MaskingException,
} from "@/types/proto/v1/org_policy_service";

export const getMaskDataIdentifier = (maskData: MaskData): string => {
  return `${maskData.schema}.${maskData.table}.${maskData.column}`;
};

export const isCurrentColumnException = (
  exception: MaskingExceptionPolicy_MaskingException,
  maskData: MaskData
): boolean => {
  const expression = exception.condition?.expression ?? "";
  const matches = [
    `resource.table_name == "${maskData.table}"`,
    `resource.column_name == "${maskData.column}"`,
  ];
  if (maskData.schema) {
    matches.push(`resource.schema_name == "${maskData.schema}"`);
  }

  for (const match of matches) {
    if (!expression.includes(match)) {
      return false;
    }
  }
  return true;
};
