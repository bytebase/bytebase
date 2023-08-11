import { cloneDeep, last } from "lodash-es";
import { celServiceClient } from "@/grpcweb";
import { SimpleExpr, resolveCELExpr } from "@/plugins/cel";
import { DatabaseResource } from "@/types";
import { Expr } from "@/types/proto/google/api/expr/v1alpha1/syntax";

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

type DatabaseResourceCondition =
  | DatabaseLevelCondition
  | SchemaLevelCondition
  | TableLevelCondition;

interface ConditionExpression {
  databaseResources?: DatabaseResource[];
  expiredTime?: string;
  statement?: string;
  rowLimit?: number;
  exportFormat?: string;
}

export const stringifyDatabaseResources = (resources: DatabaseResource[]) => {
  const conditionList: DatabaseResourceCondition[] = [];

  for (const resource of resources) {
    if (resource.table === undefined && resource.schema === undefined) {
      // Database level
      conditionList.push({
        database: [resource.databaseName],
      });
    } else if (resource.schema !== undefined && resource.table === undefined) {
      // Schema level
      conditionList.push({
        database: resource.databaseName,
        schema: [resource.schema],
      });
    } else if (resource.schema !== undefined && resource.table !== undefined) {
      // Table level
      conditionList.push({
        database: resource.databaseName,
        schema: resource.schema,
        table: [resource.table],
      });
    } else {
      throw new Error("Invalid database resource");
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
  ).filter((condition) => condition.schema.length > 0);
  const tableLevelConditionList = mergeTableLevelConditions(
    conditionList.filter(
      (condition): condition is TableLevelCondition =>
        typeof condition.database === "string" &&
        typeof (condition as TableLevelCondition).schema === "string" &&
        Array.isArray((condition as TableLevelCondition).table)
    )
  ).filter((condition) => condition.table.length > 0);

  const cel = convertToCELString([
    ...databaseLevelConditionList,
    ...schemaLevelConditionList,
    ...tableLevelConditionList,
  ]);
  return cel;
};

export const stringifyConditionExpression = (
  conditionExpression: ConditionExpression
): string => {
  const expression: string[] = [];
  if (
    conditionExpression.databaseResources !== undefined &&
    conditionExpression.databaseResources.length > 0
  ) {
    expression.push(
      stringifyDatabaseResources(conditionExpression.databaseResources)
    );
  }
  if (conditionExpression.expiredTime !== undefined) {
    expression.push(
      `request.time < timestamp("${conditionExpression.expiredTime}")`
    );
  }
  if (conditionExpression.statement !== undefined) {
    expression.push(
      `request.statement == "${btoa(
        unescape(encodeURIComponent(conditionExpression.statement))
      )}"`
    );
  }
  if (conditionExpression.rowLimit !== undefined) {
    expression.push(`request.row_limit <= ${conditionExpression.rowLimit}`);
  }
  return expression.join(" && ");
};

const convertToCELString = (
  conditions: (
    | DatabaseLevelCondition
    | SchemaLevelCondition
    | TableLevelCondition
  )[]
): string => {
  function buildCondition(
    condition:
      | DatabaseLevelCondition
      | SchemaLevelCondition
      | TableLevelCondition
  ): string {
    if ("table" in condition) {
      return `resource.database == "${
        condition.database
      }" && resource.schema == "${
        condition.schema
      }" && resource.table in ${JSON.stringify(condition.table)}`;
    } else if ("schema" in condition) {
      return `resource.database == "${
        condition.database
      }" && resource.schema in ${JSON.stringify(condition.schema)}`;
    } else {
      return `resource.database in ${JSON.stringify(condition.database)}`;
    }
  }

  function buildGroup(
    conditions: (
      | DatabaseLevelCondition
      | SchemaLevelCondition
      | TableLevelCondition
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
  const { expression: celExpr } = await celServiceClient.parse(
    {
      expression: cel,
    },
    {
      silent: true,
    }
  );

  if (!celExpr || !celExpr.expr) {
    return {};
  }

  const simpleExpr = resolveCELExpr(celExpr.expr);
  const conditionExpression: ConditionExpression = {
    databaseResources: [],
  };

  async function processCondition(expr: SimpleExpr) {
    if (expr.operator === "_&&_" || expr.operator === "_||_") {
      for (const arg of expr.args) {
        await processCondition(arg);
      }
    } else if (expr.operator === "@in") {
      const [property, values] = expr.args;
      if (typeof property === "string" && Array.isArray(values)) {
        if (property === "resource.database") {
          for (const value of values) {
            const databaseResource: DatabaseResource = {
              databaseName: value as string,
            };
            conditionExpression.databaseResources!.push(databaseResource);
          }
        } else if (property === "resource.schema") {
          const databaseResource = conditionExpression.databaseResources?.pop();
          if (databaseResource) {
            for (const value of values) {
              const temp: DatabaseResource = cloneDeep(
                databaseResource
              ) as DatabaseResource;
              temp.schema = value as string;
              conditionExpression.databaseResources!.push(temp);
            }
          }
        } else if (property === "resource.table") {
          const databaseResource = conditionExpression.databaseResources?.pop();
          if (databaseResource) {
            for (const value of values) {
              const temp: DatabaseResource = cloneDeep(
                databaseResource
              ) as DatabaseResource;
              temp.table = value as string;
              conditionExpression.databaseResources!.push(temp);
            }
          }
        }
      }
    } else if (expr.operator === "_==_") {
      const [left, right] = expr.args;
      if (typeof left === "string") {
        if (typeof right === "string") {
          if (left === "resource.database") {
            const databaseResource: DatabaseResource = {
              databaseName: right,
            };
            conditionExpression.databaseResources!.push(databaseResource);
          } else if (left === "resource.schema") {
            const databaseResource = last(
              conditionExpression.databaseResources
            );
            if (databaseResource) {
              databaseResource.schema = right;
            }
          } else if (left === "request.statement") {
            const statement = decodeURIComponent(escape(window.atob(right)));
            conditionExpression.statement = statement;
          }
        } else if (typeof right === "number") {
          if (left === "request.row_limit") {
            conditionExpression.rowLimit = right;
          }
        }
      }
    } else if (expr.operator === "_<_") {
      const [left, right] = expr.args;
      if (left === "request.time") {
        conditionExpression.expiredTime = (right as Date).toISOString();
      }
    }
  }
  await processCondition(simpleExpr);
  return conditionExpression;
};

export const convertFromExpr = (expr: Expr): ConditionExpression => {
  const simpleExpr = resolveCELExpr(expr);
  const conditionExpression: ConditionExpression = {
    databaseResources: [],
  };

  function processCondition(expr: SimpleExpr) {
    if (expr.operator === "_&&_" || expr.operator === "_||_") {
      for (const arg of expr.args) {
        processCondition(arg);
      }
    } else if (expr.operator === "@in") {
      const [property, values] = expr.args;
      if (typeof property === "string" && Array.isArray(values)) {
        if (property === "resource.database") {
          for (const value of values) {
            const databaseResource: DatabaseResource = {
              databaseName: value as string,
            };
            conditionExpression.databaseResources!.push(databaseResource);
          }
        } else if (property === "resource.schema") {
          const databaseResource = conditionExpression.databaseResources?.pop();
          if (databaseResource) {
            for (const value of values) {
              const temp: DatabaseResource = cloneDeep(
                databaseResource
              ) as DatabaseResource;
              temp.schema = value as string;
              conditionExpression.databaseResources!.push(temp);
            }
          }
        } else if (property === "resource.table") {
          const databaseResource = conditionExpression.databaseResources?.pop();
          if (databaseResource) {
            for (const value of values) {
              const temp: DatabaseResource = cloneDeep(
                databaseResource
              ) as DatabaseResource;
              temp.table = value as string;
              conditionExpression.databaseResources!.push(temp);
            }
          }
        }
      }
    } else if (expr.operator === "_==_") {
      const [left, right] = expr.args;
      if (typeof left === "string") {
        if (typeof right === "string") {
          if (left === "resource.database") {
            const databaseResource: DatabaseResource = {
              databaseName: right,
            };
            conditionExpression.databaseResources!.push(databaseResource);
          } else if (left === "resource.schema") {
            const databaseResource = last(
              conditionExpression.databaseResources
            );
            if (databaseResource) {
              databaseResource.schema = right;
            }
          } else if (left === "request.statement") {
            const statement = decodeURIComponent(escape(window.atob(right)));
            conditionExpression.statement = statement;
          }
        } else if (typeof right === "number") {
          // Deprecated. Use _<=_ instead.
          if (left === "request.row_limit") {
            conditionExpression.rowLimit = right;
          }
        }
      }
    } else if (expr.operator === "_<_") {
      const [left, right] = expr.args;
      if (left === "request.time") {
        conditionExpression.expiredTime = (right as Date).toISOString();
      }
    } else if (expr.operator === "_<=_") {
      const [left, right] = expr.args;
      if (left === "request.row_limit" && typeof right === "number") {
        if (left === "request.row_limit") {
          conditionExpression.rowLimit = right;
        }
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

  for (const database in groupedConditions) {
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

  for (const key in groupedConditions) {
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
