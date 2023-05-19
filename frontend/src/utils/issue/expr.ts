// parse expired time string from expression string for issue grant request paylod.

import { useDatabaseStore } from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import { useInstanceV1Store } from "@/store/modules/v1/instance";
import { DatabaseId, UNKNOWN_ID } from "@/types";

// e.g. timestamp("2021-08-31T00:00:00Z") => "2021-08-31T00:00:00Z"
export const parseExpiredTimeString = (expiredTime: string): string => {
  const regex = /^timestamp\("(.+?)"\)$/;
  const match = expiredTime.match(regex);
  if (!match) {
    throw new Error(`Invalid expired time: ${expiredTime}`);
  }
  return match[1];
};

interface ConditionExpression {
  // Array of database resource name. e.g. `instances/${database.instance.resourceId}/databases/${database.name}`
  databases?: string[];
  expiredTime?: string;
  statement?: string;
  rowLimit?: number;
  exportFormat?: string;
}

export const stringifyConditionExpression = (
  conditionExpression: ConditionExpression
) => {
  const expression: string[] = [];
  if (
    conditionExpression.databases !== undefined &&
    conditionExpression.databases.length > 0
  ) {
    expression.push(
      `resource.database in ${JSON.stringify(conditionExpression.databases)}`
    );
  }
  if (conditionExpression.expiredTime !== undefined) {
    expression.push(
      `request.time < timestamp("${conditionExpression.expiredTime}")`
    );
  }
  if (conditionExpression.statement !== undefined) {
    expression.push(
      `request.statement == "${btoa(conditionExpression.statement)}"`
    );
  }
  if (conditionExpression.rowLimit !== undefined) {
    expression.push(`request.row_limit == ${conditionExpression.rowLimit}`);
  }
  if (conditionExpression.exportFormat !== undefined) {
    expression.push(
      `request.export_format == "${conditionExpression.exportFormat}"`
    );
  }
  return expression.join(" && ");
};

export const parseConditionExpressionString = (
  conditionExpressionString: string
): ConditionExpression => {
  const conditionExpression: ConditionExpression = {};
  const expressionList: string[] = conditionExpressionString.split(" && ");
  for (const expression of expressionList) {
    const fields = expression.split(" ");
    const key = fields[0];
    const value = fields[2];
    if (key === "resource.database") {
      const databases = (JSON.parse(value) as string[]) || [];
      conditionExpression.databases = databases;
    } else if (key === "request.time") {
      conditionExpression.expiredTime = parseExpiredTimeString(value);
    } else if (key === "request.statement") {
      conditionExpression.statement = atob(JSON.parse(value));
    } else if (key === "request.row_limit") {
      conditionExpression.rowLimit = Number(value);
    } else if (key === "request.export_format") {
      conditionExpression.exportFormat = JSON.parse(value);
    }
  }
  return conditionExpression;
};

export const getDatabaseIdByName = async (name: string) => {
  const value = name.split("/");
  const instanceName = value[1] || "";
  const databaseName = value[3] || "";
  const instance = await useInstanceV1Store().getOrFetchInstanceByName(
    instanceNamePrefix + instanceName
  );
  const databaseList =
    await useDatabaseStore().getOrFetchDatabaseListByInstanceId(instance.uid);
  const database = databaseList.find((db) => db.name === databaseName);
  return database?.id || UNKNOWN_ID;
};

export const getDatabaseNameById = (id: DatabaseId) => {
  const database = useDatabaseStore().getDatabaseById(id);
  return `instances/${database.instance.resourceId}/databases/${database.name}`;
};
