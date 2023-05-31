// parse expired time string from expression string for issue grant request paylod.

import { useDatabaseV1Store } from "@/store";
import { instanceNamePrefix } from "@/store/modules/v1/common";
import { useInstanceV1Store } from "@/store/modules/v1/instance";
import { UNKNOWN_ID } from "@/types";

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
  const databaseList = await useDatabaseV1Store().fetchDatabaseList({
    parent: instance.name,
  });
  const database = databaseList.find((db) => db.databaseName === databaseName);
  return database?.uid || String(UNKNOWN_ID);
};

export const getDatabaseNameById = (id: string) => {
  const database = useDatabaseV1Store().getDatabaseByUID(id);
  return database.name;
};
