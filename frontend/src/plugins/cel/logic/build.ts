import { Expr as CELExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import type {
  ConditionExpr,
  ConditionGroupExpr,
  Operator,
  SimpleExpr,
} from "../types";
import {
  isEqualityExpr,
  isStringOperator,
  isCollectionExpr,
  isConditionExpr,
  isConditionGroupExpr,
  isCompareExpr,
  isStringExpr,
} from "../types";

// Build CEL expr according to simple expr
export const buildCELExpr = (expr: SimpleExpr): CELExpr | undefined => {
  const convert = (expr: ConditionGroupExpr | ConditionExpr): CELExpr => {
    if (isConditionGroupExpr(expr)) return convertGroup(expr);
    if (isConditionExpr(expr)) return convertCondition(expr);
    throw new Error(`unexpected type "${String(expr)}"`);
  };
  const convertCondition = (condition: ConditionExpr): CELExpr => {
    if (isEqualityExpr(condition)) {
      const { operator, args } = condition;
      const [factor, value] = args;
      return wrapCallExpr(operator, [
        wrapIdentExpr(factor, operator),
        wrapConstExpr(value),
      ]);
    }
    if (isCompareExpr(condition)) {
      const { operator, args } = condition;
      const [factor, value] = args;
      return wrapCallExpr(operator, [
        wrapIdentExpr(factor, operator),
        wrapConstExpr(value),
      ]);
    }
    if (isStringExpr(condition)) {
      const { operator, args } = condition;
      const [factor, value] = args;
      return wrapCallExpr(
        operator,
        [wrapConstExpr(value)],
        wrapIdentExpr(factor, operator)
      );
    }
    if (isCollectionExpr(condition)) {
      const { operator, args } = condition;
      const [factor, values] = args;
      return wrapCallExpr(operator, [
        wrapIdentExpr(factor, operator),
        wrapListExpr(values),
      ]);
    }
    throw new Error(`unsupported condition '${JSON.stringify(condition)}'`);
  };
  const convertGroup = (group: ConditionGroupExpr): CELExpr => {
    const { operator, args } = group;
    if (args.length === 1) {
      // A dangled Logical Group should be extracted as single condition
      return convert(args[0]);
    }
    return wrapCallExpr(operator, args.map(convert));
  };

  try {
    return convert(expr);
  } catch (err) {
    console.debug(err);
    return undefined;
  }
};

const wrapCELExpr = (object: any): CELExpr => {
  return CELExpr.fromJSON(object);
};

const wrapConstExpr = (value: number | string): CELExpr => {
  if (typeof value === "string") {
    return wrapCELExpr({
      constExpr: {
        stringValue: value,
      },
    });
  }
  if (typeof value === "number") {
    return wrapCELExpr({
      constExpr: {
        int64Value: value,
      },
    });
  }
  throw new Error(`unexpected value "${value}"`);
};
const wrapListExpr = (values: string[] | number[]): CELExpr => {
  return wrapCELExpr({
    listExpr: {
      elements: values.map(wrapConstExpr),
    },
  });
};

const wrapIdentExpr = (name: string, operator: Operator): CELExpr => {
  const factorName = splitFactorName(name, operator);
  return wrapCELExpr({
    identExpr: {
      name: factorName,
    },
  });
};

const wrapCallExpr = (
  operator: string,
  args: CELExpr[],
  target?: CELExpr
): CELExpr => {
  const object: Record<string, any> = {
    function: operator,
    args,
  };
  if (target) {
    object.target = target;
  }
  return wrapCELExpr({
    callExpr: object,
  });
};

const SplitFactors = new Set(["environment", "project"]);
// Split "environment_id"/"environment_name", "project_id"/"project_name"
// according to different operators
const splitFactorName = (factor: string, operator: Operator): string => {
  if (SplitFactors.has(factor)) {
    return isStringOperator(operator) ? `${factor}_name` : `${factor}_id`;
  }
  return factor;
};
