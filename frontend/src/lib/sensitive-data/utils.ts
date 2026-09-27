import type { DatabaseResource } from "@/types";
import type { MaskingExemptionPolicy_Exemption } from "@/types/proto-es/v1/org_policy_service_pb";
import { extractDatabaseResourceName } from "@/utils";
import {
  CEL_ATTRIBUTE_REQUEST_TIME,
  CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
} from "@/utils/cel-attributes";
import { celString, celStringList } from "@/utils/v1/celLiteral";
import type { MaskData, SensitiveColumn } from "./types";

export const getMaskDataIdentifier = (maskData: MaskData): string => {
  return `${maskData.schema}.${maskData.table}.${maskData.column}`;
};

export const convertSensitiveColumnToDatabaseResource = (
  sensitiveColumn: SensitiveColumn
): DatabaseResource => ({
  databaseFullName: sensitiveColumn.database.name,
  schema: sensitiveColumn.maskData.schema,
  table: sensitiveColumn.maskData.table,
  columns: [sensitiveColumn.maskData.column].filter((c) => c),
});

export const isCurrentColumnException = (
  exception: MaskingExemptionPolicy_Exemption,
  sensitiveColumn: SensitiveColumn
): boolean => {
  const expression = exception.condition?.expression ?? "";
  if (!expression) {
    // no expression means can access all databases.
    return true;
  }
  const databaseExpressions = expression.split(" || ").map((part) =>
    part
      .split(" && ")
      .map(trimParentheses)
      .filter(
        (expr) =>
          !expr.startsWith(CEL_ATTRIBUTE_REQUEST_TIME) &&
          !expr.startsWith(CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL)
      )
      .join(" && ")
  );
  const matches = getExpressionsForDatabaseResource(
    convertSensitiveColumnToDatabaseResource(sensitiveColumn)
  );
  const currentColumnExpression = matches.join(" && ");
  return databaseExpressions.some((databaseExpression) =>
    currentColumnExpression.includes(databaseExpression)
  );
};

const trimParentheses = (expression: string): string => {
  let result = expression.trim();
  while (result.startsWith("(")) {
    result = result.slice(1).trimStart();
  }
  while (result.endsWith(")")) {
    result = result.slice(0, -1).trimEnd();
  }
  return result;
};
export const getExpressionsForDatabaseResource = (
  databaseResource: DatabaseResource
): string[] => {
  const { instanceName, databaseName } = extractDatabaseResourceName(
    databaseResource.databaseFullName
  );
  const expressions = [
    `${CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID} == ${celString(instanceName)}`,
    `${CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME} == ${celString(databaseName)}`,
  ];
  if (databaseResource.schema) {
    expressions.push(
      `${CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME} == ${celString(databaseResource.schema)}`
    );
  }
  if (databaseResource.table) {
    expressions.push(
      `${CEL_ATTRIBUTE_RESOURCE_TABLE_NAME} == ${celString(databaseResource.table)}`
    );
  }
  if (databaseResource.columns && databaseResource.columns.length > 0) {
    if (databaseResource.columns.length === 1) {
      expressions.push(
        `${CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME} == ${celString(databaseResource.columns[0])}`
      );
    } else {
      expressions.push(
        `${CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME} in ${celStringList(databaseResource.columns)}`
      );
    }
  }
  return expressions;
};
