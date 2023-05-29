import { celServiceClient } from "@/grpcweb";
import {
  ConditionGroupExpr,
  SimpleExpr,
  convertToCELString,
  emptySimpleExpr,
  resolveCELExpr,
  wrapAsGroup,
} from "@/plugins/cel";

interface DatabaseGroupExpr {
  environmentId: string;
  conditionGroupExpr: ConditionGroupExpr;
}

export const stringifyDatabaseGroupExpr = (
  databaseGroupExpr: DatabaseGroupExpr
): string => {
  const { environmentId, conditionGroupExpr } = databaseGroupExpr;
  return convertToCELString({
    operator: "_&&_",
    args: [
      // Make the environment ID a condition first to avoid confusion when converting from CEL string.
      {
        operator: "_==_",
        args: ["resource.environment_id", environmentId],
      },
      conditionGroupExpr,
    ],
  });
};

export const convertDatabaseGroupExprFromCEL = async (
  cel: string
): Promise<DatabaseGroupExpr> => {
  const { expression: celExpr } = await celServiceClient.parse({
    expression: cel,
  });

  if (!celExpr || !celExpr.expr) {
    throw new Error("Invalid CEL expression");
  }

  const simpleExpr = resolveCELExpr(celExpr.expr);
  const [environmentId, ...conditionGroupExpr] =
    getEnvironmentIdAndConditionExpr(simpleExpr);
  if (!environmentId) {
    throw new Error("Invalid CEL expression");
  }

  return {
    environmentId,
    conditionGroupExpr: wrapAsGroup(...conditionGroupExpr),
  };
};

const getEnvironmentIdAndConditionExpr = (
  expr: SimpleExpr
): [string, ConditionGroupExpr] => {
  if (expr.operator !== "_&&_") {
    throw ["", emptySimpleExpr()];
  }

  const [left, ...right] = expr.args;
  const environmentName = left.args[1] as string;
  if (Array.isArray(right)) {
    return [
      environmentName,
      {
        operator: "_&&_",
        args: right,
      },
    ];
  } else {
    return [environmentName, right];
  }
};
