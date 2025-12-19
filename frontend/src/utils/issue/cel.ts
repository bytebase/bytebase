import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { cloneDeep, head } from "lodash-es";
import type { SimpleExpr } from "@/plugins/cel";
import { isRawStringExpr, resolveCELExpr } from "@/plugins/cel";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import type { DatabaseResource } from "@/types";
import type { Expr } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import type { Expr as ConditionExpr } from "@/types/proto-es/google/type/expr_pb";
import { ExprSchema as ConditionExprSchema } from "@/types/proto-es/google/type/expr_pb";
import {
  batchConvertCELStringToParsedExpr,
  displayRoleTitle,
  extractDatabaseResourceName,
} from "@/utils";
import {
  CEL_ATTRIBUTE_REQUEST_TIME,
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
  CEL_ATTRIBUTE_RESOURCE_DATABASE,
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
} from "@/utils/cel-attributes";

interface DatabaseLevelCondition {
  database: string[];
}

interface SchemaLevelCondition {
  database: string;
  schema: string[];
}

interface TableLevelCondition {
  database: string;
  schema: string;
  table: string[];
}

interface ColumnLevelCondition {
  database: string;
  schema: string;
  table: string;
  column: string[];
}

type DatabaseResourceCondition =
  | DatabaseLevelCondition
  | SchemaLevelCondition
  | TableLevelCondition
  | ColumnLevelCondition;

export interface ConditionExpression {
  databaseResources?: DatabaseResource[];
  expiredTime?: string;
  exportFormat?: string;
}

const getDatabaseResourceName = (databaseResource: DatabaseResource) => {
  const { databaseName } = extractDatabaseResourceName(
    databaseResource.databaseFullName
  );
  if (databaseResource.table) {
    if (databaseResource.schema) {
      return `${databaseName}.${databaseResource.schema}.${databaseResource.table}`;
    } else {
      return `${databaseName}.${databaseResource.table}`;
    }
  } else if (databaseResource.schema) {
    return `${databaseName}.${databaseResource.schema}`;
  } else {
    return databaseName;
  }
};

const buildConditionTitle = ({
  role,
  databaseResources,
  expirationTimestampInMS,
}: {
  role: string;
  databaseResources?: DatabaseResource[];
  expirationTimestampInMS?: number;
}): string => {
  const title = [displayRoleTitle(role)];

  let conditionSuffix = "";
  if (databaseResources !== undefined) {
    if (databaseResources.length === 0) {
      conditionSuffix = `All databases`;
    } else if (databaseResources.length <= 3) {
      const databaseResourceNames = databaseResources.map((ds) =>
        getDatabaseResourceName(ds)
      );
      conditionSuffix = `${databaseResourceNames.join(", ")}`;
    } else {
      const firstDatabaseResourceName = getDatabaseResourceName(
        head(databaseResources)!
      );
      conditionSuffix = `${firstDatabaseResourceName} and ${
        databaseResources.length - 1
      } more`;
    }
  }
  if (conditionSuffix) {
    title.push(conditionSuffix);
  }

  if (expirationTimestampInMS) {
    title.push(
      `${dayjs().format("L")}-${dayjs(expirationTimestampInMS).format("L")}`
    );
  }

  return title.join(" ");
};

export const buildConditionExpr = ({
  title,
  role,
  description,
  expirationTimestampInMS,
  databaseResources,
}: {
  title?: string;
  role: string;
  description: string;
  expirationTimestampInMS?: number;
  databaseResources?: DatabaseResource[];
}): ConditionExpr => {
  const expresson = stringifyConditionExpression({
    expirationTimestampInMS,
    databaseResources,
  });
  return create(ConditionExprSchema, {
    title:
      title ||
      buildConditionTitle({
        role,
        databaseResources,
        expirationTimestampInMS,
      }),
    description: description,
    expression: expresson || "",
  });
};

export const stringifyDatabaseResources = (resources: DatabaseResource[]) => {
  const conditionList: DatabaseResourceCondition[] = [];

  for (const resource of resources) {
    if (resource.columns !== undefined) {
      conditionList.push({
        database: resource.databaseFullName,
        schema: resource.schema ?? "",
        table: resource.table ?? "",
        column: [...resource.columns],
      });
    } else if (resource.table !== undefined) {
      conditionList.push({
        database: resource.databaseFullName,
        schema: resource.schema ?? "",
        table: [resource.table],
      });
    } else if (resource.schema !== undefined) {
      conditionList.push({
        database: resource.databaseFullName,
        schema: [resource.schema],
      });
    } else {
      conditionList.push({
        database: [resource.databaseFullName],
      });
    }
  }

  const databaseLevelConditionList = mergeDatabaseLevelConditions(
    conditionList.filter((condition): condition is DatabaseLevelCondition =>
      Array.isArray(condition.database)
    )
  ).filter((condition) => condition.database.length > 0);
  const schemaLevelConditionList = mergeSchemaLevelConditions(
    conditionList.filter(
      (condition): condition is SchemaLevelCondition =>
        typeof condition.database === "string" &&
        Array.isArray((condition as SchemaLevelCondition).schema)
    )
  );
  const tableLevelConditionList = mergeTableLevelConditions(
    conditionList.filter(
      (condition): condition is TableLevelCondition =>
        typeof condition.database === "string" &&
        typeof (condition as TableLevelCondition).schema === "string" &&
        Array.isArray((condition as TableLevelCondition).table)
    )
  );
  const columnLevelConditionList = mergeColumnLevelConditions(
    conditionList.filter(
      (condition): condition is ColumnLevelCondition =>
        typeof condition.database === "string" &&
        typeof (condition as ColumnLevelCondition).schema === "string" &&
        typeof (condition as ColumnLevelCondition).table === "string" &&
        Array.isArray((condition as ColumnLevelCondition).column)
    )
  );

  const cel = convertToCELString([
    ...databaseLevelConditionList,
    ...schemaLevelConditionList,
    ...tableLevelConditionList,
    ...columnLevelConditionList,
  ]);
  return cel;
};

export const stringifyConditionExpression = ({
  expirationTimestampInMS,
  databaseResources,
}: {
  expirationTimestampInMS?: number;
  databaseResources?: DatabaseResource[];
}): string => {
  const expression: string[] = [];
  if (databaseResources !== undefined && databaseResources.length > 0) {
    expression.push(stringifyDatabaseResources(databaseResources));
  }
  if (expirationTimestampInMS) {
    expression.push(
      `${CEL_ATTRIBUTE_REQUEST_TIME} < timestamp("${dayjs(expirationTimestampInMS).toISOString()}")`
    );
  }
  return expression.join(" && ");
};

const convertToCELString = (
  conditions: (
    | DatabaseLevelCondition
    | SchemaLevelCondition
    | TableLevelCondition
    | ColumnLevelCondition
  )[]
): string => {
  if (conditions.length === 0) {
    return "";
  }

  const getArrayExpressionString = (resource: string, arr: string[]) => {
    if (arr.length === 0) {
      return "";
    }
    return `${resource} in ${JSON.stringify(arr)}`;
  };

  const getStringExpressionString = (resource: string, value: string) => {
    if (!value) {
      return "";
    }
    return `${resource} == "${value}"`;
  };

  function buildCondition(
    condition:
      | DatabaseLevelCondition
      | SchemaLevelCondition
      | TableLevelCondition
      | ColumnLevelCondition
  ): string {
    const databaseExpression = `${CEL_ATTRIBUTE_RESOURCE_DATABASE} == "${condition.database}"`;
    if ("column" in condition) {
      return [
        databaseExpression,
        getStringExpressionString(
          CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
          condition.schema
        ),
        getStringExpressionString(
          CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
          condition.table
        ),
        getArrayExpressionString(
          CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
          condition.column
        ),
      ]
        .filter((str) => str)
        .join(" && ");
    } else if ("table" in condition) {
      return [
        databaseExpression,
        getStringExpressionString(
          CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
          condition.schema
        ),
        getArrayExpressionString(
          CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
          condition.table
        ),
      ]
        .filter((str) => str)
        .join(" && ");
    } else if ("schema" in condition) {
      return [
        databaseExpression,
        getArrayExpressionString(
          CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
          condition.schema
        ),
      ]
        .filter((str) => str)
        .join(" && ");
    } else {
      return `${CEL_ATTRIBUTE_RESOURCE_DATABASE} in ${JSON.stringify(condition.database)}`;
    }
  }

  function buildGroup(
    conditions: (
      | DatabaseLevelCondition
      | SchemaLevelCondition
      | TableLevelCondition
      | ColumnLevelCondition
    )[]
  ): string {
    if (conditions.length === 1) {
      return buildCondition(conditions[0]);
    } else {
      const conditionStrings = conditions.map(buildCondition);
      return `${conditionStrings.map((s) => `(${s})`).join(" || ")}`;
    }
  }

  const topLevelCondition = buildGroup(conditions);
  return `(${topLevelCondition})`;
};

export const convertFromCELString = async (
  cel: string
): Promise<ConditionExpression> => {
  let expr: Expr | undefined;
  if (cel) {
    const celExpr = await batchConvertCELStringToParsedExpr([cel]);
    expr = celExpr[0];
  }
  if (!expr) {
    return {};
  }

  return convertFromExpr(expr);
};

export const batchConvertFromCELString = async (
  cels: string[]
): Promise<ConditionExpression[]> => {
  const celExprs = await batchConvertCELStringToParsedExpr(cels);
  const resp: ConditionExpression[] = [];
  for (let i = 0; i < celExprs.length; i++) {
    if (cels[i] === "true" || !celExprs[i]) {
      resp.push({});
    } else {
      resp.push(convertFromExpr(celExprs[i]));
    }
  }
  return resp;
};

export const convertFromExpr = (expr: Expr): ConditionExpression => {
  const simpleExpr = resolveCELExpr(expr);
  const conditionExpression: ConditionExpression = {
    databaseResources: [],
  };

  function processCondition(expr: SimpleExpr) {
    // Do not process raw string expression.
    if (isRawStringExpr(expr)) {
      return;
    }

    if (expr.operator === "_&&_" || expr.operator === "_||_") {
      for (const arg of expr.args) {
        processCondition(arg);
      }
    } else if (expr.operator === "@in") {
      const [property, values] = expr.args;
      if (typeof property === "string" && Array.isArray(values)) {
        switch (property) {
          case CEL_ATTRIBUTE_RESOURCE_DATABASE: {
            for (const value of values) {
              const databaseResource: DatabaseResource = {
                databaseFullName: value as string,
              };
              conditionExpression.databaseResources!.push(databaseResource);
            }
            break;
          }
          case CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME: {
            const databaseResource =
              conditionExpression.databaseResources?.pop();
            if (databaseResource) {
              for (const value of values) {
                const temp: DatabaseResource = cloneDeep(databaseResource);
                temp.schema = value as string;
                conditionExpression.databaseResources!.push(temp);
              }
            }
            break;
          }
          case CEL_ATTRIBUTE_RESOURCE_TABLE_NAME: {
            const databaseResource =
              conditionExpression.databaseResources?.pop();
            if (databaseResource) {
              for (const value of values) {
                const temp: DatabaseResource = cloneDeep(databaseResource);
                temp.table = value as string;
                conditionExpression.databaseResources!.push(temp);
              }
            }
            break;
          }
          case CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME: {
            const databaseResource =
              conditionExpression.databaseResources?.pop();
            if (databaseResource) {
              databaseResource.columns = [];
              for (const value of values) {
                if (value) {
                  databaseResource.columns.push(value as string);
                }
              }
              conditionExpression.databaseResources!.push(databaseResource);
            }
            break;
          }
        }
      }
    } else if (expr.operator === "_==_") {
      const [left, right] = expr.args;
      if (typeof left === "string") {
        if (typeof right === "string") {
          let databaseResource = conditionExpression.databaseResources?.pop();
          if (!databaseResource) {
            databaseResource = {
              databaseFullName: "",
            };
          }
          switch (left) {
            case CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID:
            case CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME:
            case CEL_ATTRIBUTE_RESOURCE_DATABASE: {
              // should parse for next database.
              if (databaseResource.databaseFullName !== "") {
                conditionExpression.databaseResources?.push({
                  ...databaseResource,
                });
                databaseResource = {
                  databaseFullName: "",
                };
              }
            }
          }
          switch (left) {
            case CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID: {
              databaseResource.instanceResourceId = right;
              if (databaseResource.databaseResourceId) {
                databaseResource.databaseFullName = `${instanceNamePrefix}${databaseResource.instanceResourceId}/${databaseNamePrefix}${databaseResource.databaseResourceId}`;
              }
              break;
            }
            case CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME: {
              databaseResource.databaseResourceId = right;
              if (databaseResource.instanceResourceId) {
                databaseResource.databaseFullName = `${instanceNamePrefix}${databaseResource.instanceResourceId}/${databaseNamePrefix}${databaseResource.databaseResourceId}`;
              }
              break;
            }
            case CEL_ATTRIBUTE_RESOURCE_DATABASE: {
              databaseResource.databaseFullName = right;
              break;
            }
            case CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME: {
              databaseResource.schema = right;
              break;
            }
            case CEL_ATTRIBUTE_RESOURCE_TABLE_NAME: {
              databaseResource.table = right;
              break;
            }
            case CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME: {
              if (right) {
                databaseResource.columns = [right];
              }
              break;
            }
          }
          conditionExpression.databaseResources?.push(databaseResource);
        }
      }
    } else if (expr.operator === "_<_") {
      const [left, right] = expr.args;
      if (left === CEL_ATTRIBUTE_REQUEST_TIME) {
        conditionExpression.expiredTime = (right as Date).toISOString();
      }
    }
  }
  processCondition(simpleExpr);
  return conditionExpression;
};

const mergeDatabaseLevelConditions = (
  conditions: DatabaseLevelCondition[]
): DatabaseLevelCondition[] => {
  return [
    {
      database: conditions.map((condition) => condition.database).flat(),
    },
  ];
};

const mergeSchemaLevelConditions = (
  conditions: SchemaLevelCondition[]
): SchemaLevelCondition[] => {
  const groupedConditions: Record<string, string[]> = {};

  for (const condition of conditions) {
    const { database, schema } = condition;

    if (groupedConditions[database]) {
      groupedConditions[database] = [...groupedConditions[database], ...schema];
    } else {
      groupedConditions[database] = [...schema];
    }
  }

  const mergedConditions: SchemaLevelCondition[] = [];

  for (const database of Object.keys(groupedConditions)) {
    const condition: SchemaLevelCondition = {
      database,
      schema: groupedConditions[database],
    };
    mergedConditions.push(condition);
  }

  return mergedConditions;
};

const mergeTableLevelConditions = (
  conditions: TableLevelCondition[]
): TableLevelCondition[] => {
  const groupedConditions: Record<string, string[]> = {};

  for (const condition of conditions) {
    const { database, schema, table } = condition;
    const key = `${database}:${schema}`;

    if (groupedConditions[key]) {
      groupedConditions[key] = [...groupedConditions[key], ...table];
    } else {
      groupedConditions[key] = [...table];
    }
  }

  const mergedConditions: TableLevelCondition[] = [];

  for (const key of Object.keys(groupedConditions)) {
    const [database, schema] = key.split(":");
    const condition: TableLevelCondition = {
      database,
      schema,
      table: groupedConditions[key],
    };
    mergedConditions.push(condition);
  }

  return mergedConditions;
};

const mergeColumnLevelConditions = (
  conditions: ColumnLevelCondition[]
): ColumnLevelCondition[] => {
  const groupedConditions: Record<string, string[]> = {};

  for (const condition of conditions) {
    const { database, schema, table, column } = condition;
    const key = `${database}:${schema}:${table}`;

    if (groupedConditions[key]) {
      groupedConditions[key] = [...groupedConditions[key], ...column];
    } else {
      groupedConditions[key] = [...column];
    }
  }

  const mergedConditions: ColumnLevelCondition[] = [];

  for (const key of Object.keys(groupedConditions)) {
    const [database, schema, table] = key.split(":");
    const condition: ColumnLevelCondition = {
      database,
      schema,
      table,
      column: groupedConditions[key],
    };
    mergedConditions.push(condition);
  }

  return mergedConditions;
};
