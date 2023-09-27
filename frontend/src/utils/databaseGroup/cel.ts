import {
  ConditionGroupExpr,
  SimpleExpr,
  emptySimpleExpr,
  resolveCELExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import { convertCELStringToParsedExpr } from "@/utils";

interface DatabaseGroupExpr {
  environmentId: string;
  conditionGroupExpr: ConditionGroupExpr;
}

export const buildDatabaseGroupExpr = (
  databaseGroupExpr: DatabaseGroupExpr
): SimpleExpr => {
  const { environmentId, conditionGroupExpr } = databaseGroupExpr;
  if (conditionGroupExpr.args.length > 0) {
    return {
      operator: "_&&_",
      args: [
        // Make the environment ID a condition first to avoid confusion when converting from CEL string.
        {
          operator: "_==_",
          args: ["resource.environment_name", environmentId],
        },
        conditionGroupExpr,
      ],
    };
  } else {
    return {
      operator: "_==_",
      args: ["resource.environment_name", environmentId],
    };
  }
};

export const convertCELStringToExpr = async (cel: string) => {
  const celExpr = await convertCELStringToParsedExpr(cel);

  if (!celExpr || !celExpr.expr) {
    return emptySimpleExpr();
  }

  return wrapAsGroup(resolveCELExpr(celExpr.expr));
};

export const getEnvironmentIdAndConditionExpr = (
  expr: SimpleExpr
): [string, ConditionGroupExpr] => {
  if (expr.operator === "_==_") {
    const [left, right] = expr.args;
    if (left === "resource.environment_name") {
      return [right as string, emptySimpleExpr()];
    }
  }
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
