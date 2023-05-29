import {
  CollectionExpr,
  CompareExpr,
  ConditionExpr,
  EqualityExpr,
  LogicalExpr,
  NumberFactor,
  Operator,
  SimpleExpr,
  StringExpr,
  StringFactor,
  TimestampFactor,
  isCollectionExpr,
  isCompareExpr,
  isConditionGroupExpr,
  isEqualityExpr,
} from "..";

export function convertToCELString(expr: SimpleExpr): string {
  if (isConditionGroupExpr(expr)) {
    return convertLogicalExprToCELString(expr);
  } else {
    return convertConditionExprToCELString(expr);
  }
}

function convertLogicalExprToCELString(expr: LogicalExpr): string {
  const operator = convertOperatorToCELString(expr.operator);
  const args = expr.args.map((arg) => convertToCELString(arg));
  return `(${args.join(` ${operator} `)})`;
}

function convertConditionExprToCELString(expr: ConditionExpr): string {
  if (isEqualityExpr(expr)) {
    return convertEqualityExprToCELString(expr);
  } else if (isCompareExpr(expr)) {
    return convertCompareExprToCELString(expr);
  } else if (isCollectionExpr(expr)) {
    return convertCollectionExprToCELString(expr);
  } else {
    return convertStringExprToCELString(expr);
  }
}

function convertEqualityExprToCELString(expr: EqualityExpr): string {
  const operator = convertOperatorToCELString(expr.operator);
  const factor = convertFactorToCELString(expr.args[0]);
  const value = convertValueToCELString(expr.args[1]);
  return `${factor} ${operator} ${value}`;
}

function convertCompareExprToCELString(expr: CompareExpr): string {
  const operator = convertOperatorToCELString(expr.operator);
  const factor = convertFactorToCELString(expr.args[0]);
  const value = convertValueToCELString(expr.args[1]);
  return `${factor} ${operator} ${value}`;
}

function convertCollectionExprToCELString(expr: CollectionExpr): string {
  const operator = convertOperatorToCELString(expr.operator);
  const factor = convertFactorToCELString(expr.args[0]);
  const values = expr.args[1].map((value) => convertValueToCELString(value));
  return `${factor} ${operator} [${values.join(", ")}]`;
}

function convertStringExprToCELString(expr: StringExpr): string {
  const operator = convertOperatorToCELString(expr.operator);
  const factor = convertFactorToCELString(expr.args[0]);
  const value = convertValueToCELString(expr.args[1]);
  return `${factor}.${operator}(${value})`;
}

function convertFactorToCELString(
  factor: StringFactor | NumberFactor | TimestampFactor
): string {
  return String(factor);
}

function convertOperatorToCELString(operator: Operator): string {
  switch (operator) {
    case "_&&_":
      return "&&";
    case "_||_":
      return "||";
    case "_==_":
      return "==";
    case "_!=_":
      return "!=";
    case "_>_":
      return ">";
    case "_>=_":
      return ">=";
    case "_<_":
      return "<";
    case "_<=_":
      return "<=";
    case "@in":
      return "in";
    case "contains":
      return "contains";
    case "matches":
      return "matches";
    case "startsWith":
      return "startsWith";
    case "endsWith":
      return "endsWith";
    default:
      throw new Error("Unsupported operator.");
  }
}

function convertValueToCELString(value: string | number | Date): string {
  if (typeof value === "string") {
    return `"${value}"`;
  } else if (typeof value === "number") {
    return value.toString();
  } else if (value instanceof Date) {
    return `timestamp("${value.toISOString()}")`;
  } else {
    throw new Error("Unsupported value type.");
  }
}
