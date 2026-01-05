import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { head } from "lodash-es";
import { celServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import type { Expr as CELExpr } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import {
  ConstantSchema,
  Expr_CallSchema,
  Expr_CreateListSchema,
  Expr_IdentSchema,
  ExprSchema,
} from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import { BatchParseRequestSchema } from "@/types/proto-es/v1/cel_service_pb";
import type { ConditionExpr, ConditionGroupExpr, SimpleExpr } from "../types";
import {
  ExprType,
  isCollectionExpr,
  isCompareExpr,
  isConditionExpr,
  isConditionGroupExpr,
  isEqualityExpr,
  isRawStringExpr,
  isStringExpr,
} from "../types";

const seq = {
  id: 1,
  next() {
    return seq.id++;
  },
};

// Build CEL expr according to simple expr
export const buildCELExpr = async (
  expr: SimpleExpr
): Promise<CELExpr | undefined> => {
  const convert = async (expr: SimpleExpr): Promise<CELExpr | undefined> => {
    if (isConditionGroupExpr(expr)) return convertGroup(expr);
    if (isConditionExpr(expr)) return convertCondition(expr);
    if (isRawStringExpr(expr)) {
      if (!expr.content) {
        return undefined;
      }
      const request = create(BatchParseRequestSchema, {
        expressions: [expr.content],
      });
      const response = await celServiceClientConnect.batchParse(request, {
        contextValues: createContextValues().set(silentContextKey, true),
      });
      return head(response.expressions);
    }
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
      if (operator === "@not_contains") {
        return wrapCallExpr("!_", [
          wrapCallExpr(
            "contains",
            [wrapConstExpr(value)],
            wrapIdentExpr(factor)
          ),
        ]);
      }
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
  const convertGroup = async (
    group: ConditionGroupExpr
  ): Promise<CELExpr | undefined> => {
    const { operator, args } = group;
    if (args.length === 0) {
      return undefined;
    }
    if (args.length === 1) {
      // A dangled Logical Group should be extracted as single condition
      return await convert(args[0]);
    }
    const [left, ...rest] = args;
    return wrapCallExpr(
      operator,
      [
        await convert(left),
        await convertGroup({
          type: ExprType.ConditionGroup,
          operator,
          args: rest,
        }),
      ].filter(Boolean) as CELExpr[]
    );
  };

  seq.id = 1;
  try {
    return await convert(expr);
  } catch (err) {
    console.error(err);
    return undefined;
  }
};

const wrapCELExpr = (object: CELExpr["exprKind"]): CELExpr => {
  return create(ExprSchema, {
    id: BigInt(seq.next()),
    exprKind: object,
  });
};

// Note: We don't need to wrap date type factor right now. Put it here is just for prevent eslint error.
const wrapConstExpr = (value: number | string | Date): CELExpr => {
  if (typeof value === "string") {
    return wrapCELExpr({
      case: "constExpr",
      value: create(ConstantSchema, {
        constantKind: {
          case: "stringValue",
          value,
        },
      }),
    });
  }
  if (typeof value === "number") {
    return wrapCELExpr({
      case: "constExpr",
      value: create(ConstantSchema, {
        constantKind: {
          case: "int64Value",
          value: BigInt(value),
        },
      }),
    });
  }
  throw new Error(`unexpected value "${value}"`);
};

const wrapListExpr = (values: string[] | number[]): CELExpr => {
  return wrapCELExpr({
    case: "listExpr",
    value: create(Expr_CreateListSchema, {
      elements: values.map(wrapConstExpr),
    }),
  });
};

const wrapIdentExpr = (name: string): CELExpr => {
  return wrapCELExpr({
    case: "identExpr",
    value: create(Expr_IdentSchema, {
      name,
    }),
  });
};

const wrapCallExpr = (
  operator: string,
  args: CELExpr[],
  target?: CELExpr
): CELExpr => {
  const object = create(Expr_CallSchema, {
    function: operator,
    args,
  });
  if (target) {
    object.target = target;
  }
  return wrapCELExpr({
    case: "callExpr",
    value: object,
  });
};
