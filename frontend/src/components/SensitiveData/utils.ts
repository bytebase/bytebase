import type { MaskData } from "@/components/SensitiveData/types";
import type { DatabaseResource } from "@/types";
import type { MaskingExemptionPolicy_Exemption } from "@/types/proto-es/v1/org_policy_service_pb";
import { extractDatabaseResourceName } from "@/utils";
import {
  CEL_ATTRIBUTE_REQUEST_TIME,
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
} from "@/utils/cel-attributes";
import type { SensitiveColumn } from "./types";

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
  const databaseExpression = expression
    .split(" && ")
    .filter((expr) => !expr.startsWith(CEL_ATTRIBUTE_REQUEST_TIME))
    .join(" && ");
  const matches = getExpressionsForDatabaseResource(
    convertSensitiveColumnToDatabaseResource(sensitiveColumn)
  );
  return matches.join(" && ").includes(databaseExpression);
};

export const getExpressionsForDatabaseResource = (
  databaseResource: DatabaseResource
): string[] => {
  const { instanceName, databaseName } = extractDatabaseResourceName(
    databaseResource.databaseFullName
  );
  const expressions = [
    `${CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID} == "${instanceName}"`,
    `${CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME} == "${databaseName}"`,
  ];
  if (databaseResource.schema) {
    expressions.push(
      `${CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME} == "${databaseResource.schema}"`
    );
  }
  if (databaseResource.table) {
    expressions.push(
      `${CEL_ATTRIBUTE_RESOURCE_TABLE_NAME} == "${databaseResource.table}"`
    );
  }
  if (databaseResource.columns && databaseResource.columns.length > 0) {
    if (databaseResource.columns.length === 1) {
      expressions.push(
        `${CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME} == "${databaseResource.columns[0]}"`
      );
    } else {
      expressions.push(
        `${CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME} in [${databaseResource.columns.map((c) => `"${c}"`)}]`
      );
    }
  }
  return expressions;
};
