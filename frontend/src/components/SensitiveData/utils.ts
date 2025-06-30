import type { MaskData } from "@/components/SensitiveData/types";
import type { DatabaseResource } from "@/types";
import type { MaskingExceptionPolicy_MaskingException } from "@/types/proto-es/v1/org_policy_service_pb";
import { extractDatabaseResourceName } from "@/utils";
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
  column: sensitiveColumn.maskData.column,
});

export const isCurrentColumnException = (
  exception: MaskingExceptionPolicy_MaskingException,
  sensitiveColumn: SensitiveColumn
): boolean => {
  const expression = exception.condition?.expression ?? "";
  if (!expression) {
    // no expression means can access all databases.
    return true;
  }
  const databaseExpression = expression
    .split(" && ")
    .filter((expr) => !expr.startsWith("request.time"))
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
    `resource.instance_id == "${instanceName}"`,
    `resource.database_name == "${databaseName}"`,
  ];
  if (databaseResource.schema) {
    expressions.push(`resource.schema_name == "${databaseResource.schema}"`);
  }
  if (databaseResource.table) {
    expressions.push(`resource.table_name == "${databaseResource.table}"`);
  }
  if (databaseResource.column) {
    expressions.push(`resource.column_name == "${databaseResource.column}"`);
  }
  return expressions;
};
