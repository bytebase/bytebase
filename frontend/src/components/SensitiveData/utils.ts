import {
  MaskData,
  MaskingExceptionPolicy_MaskingException,
} from "@/types/proto/v1/org_policy_service";
import { extractInstanceResourceName } from "@/utils";
import { SensitiveColumn } from "./types";

export const getMaskDataIdentifier = (maskData: MaskData): string => {
  return `${maskData.schema}.${maskData.table}.${maskData.column}`;
};

export const isCurrentColumnException = (
  exception: MaskingExceptionPolicy_MaskingException,
  sensitiveColumn: SensitiveColumn
): boolean => {
  const expression = exception.condition?.expression ?? "";
  const matches = getExpressionsForSensitiveColumn(sensitiveColumn);
  for (const match of matches) {
    if (!expression.includes(match)) {
      return false;
    }
  }
  return true;
};

export const getExpressionsForSensitiveColumn = (
  sensitiveColumn: SensitiveColumn
): string[] => {
  const expressions = [
    `resource.database_name == "${sensitiveColumn.database.databaseName}"`,
    `resource.instance_id == "${extractInstanceResourceName(
      sensitiveColumn.database.instanceEntity.name
    )}"`,
    `resource.table_name == "${sensitiveColumn.maskData.table}"`,
    `resource.column_name == "${sensitiveColumn.maskData.column}"`,
  ];
  if (sensitiveColumn.maskData.schema) {
    expressions.push(
      `resource.schema_name == "${sensitiveColumn.maskData.schema}"`
    );
  }
  return expressions;
};
