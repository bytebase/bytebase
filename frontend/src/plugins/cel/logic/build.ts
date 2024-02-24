import { Expr as CELExpr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import type { ConditionExpr, ConditionGroupExpr, SimpleExpr } from "../types";
import {
  isEqualityExpr,
  isCollectionExpr,
  isConditionExpr,
  isConditionGroupExpr,
  isCompareExpr,
  isStringExpr,
} from "../types";

const seq = {
  id: 1,
  next() {
    return seq.id++;
  },
};

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
        wrapIdentExpr(factor),
        wrapConstExpr(value),
      ]);
    }
    if (isCompareExpr(condition)) {
      const { operator, args } = condition;
      const [factor, value] = args;
      return wrapCallExpr(operator, [
        wrapIdentExpr(factor),
        wrapConstExpr(value),
      ]);
    }
    if (isStringExpr(condition)) {
      const { operator, args } = condition;
      const [factor, value] = args;
      return wrapCallExpr(
        operator,
        [wrapConstExpr(value)],
        wrapIdentExpr(factor)
      );
    }
    if (isCollectionExpr(condition)) {
      const { operator, args } = condition;
      const [factor, values] = args;
      if (operator === "@not_in") {
        return wrapCallExpr("!_", [
          wrapCallExpr("@in", [wrapIdentExpr(factor), wrapListExpr(values)]),
        ]);
      }
      return wrapCallExpr(operator, [
        wrapIdentExpr(factor),
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
    const [left, ...rest] = args;
    const _args = [
      convert(left),
      convertGroup({
        operator,
        args: rest,
      }),
    ];
    // return createCallExpr(operator, args);
    return wrapCallExpr(operator, _args);
  };

  seq.id = 1;
  try {
    return convert(expr);
  } catch (err) {
    console.debug(err);
    return undefined;
  }
};

const wrapCELExpr = (object: any): CELExpr => {
  return CELExpr.fromJSON({
    id: seq.next(),
    ...object,
  });
};

// Note: We don't need to wrap date type factor right now. Put it here is just for prevent eslint error.
const wrapConstExpr = (value: number | string | Date): CELExpr => {
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

const wrapIdentExpr = (name: string): CELExpr => {
  return wrapCELExpr({
    identExpr: {
      name,
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
