import type {
  Constant,
  Expr,
} from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";

function stringifyExpr(expr: Expr): string {
  if (expr.exprKind?.case === "constExpr") {
    return stringifyConstant(expr.exprKind.value);
  } else if (expr.exprKind?.case === "identExpr") {
    return expr.exprKind.value.name;
  } else if (expr.exprKind?.case === "selectExpr") {
    const selectExpr = expr.exprKind.value;
    // Check if testOnly flag is used to denote a 'has' operation.
    if (selectExpr.testOnly) {
      return `has(${stringifyExpr(selectExpr.operand!)}.${selectExpr.field})`;
    } else {
      return `${stringifyExpr(selectExpr.operand!)}.${selectExpr.field}`;
    }
  } else if (expr.exprKind?.case === "callExpr") {
    const callExpr = expr.exprKind.value;
    // Remove underscores from function name. e.g. "_&&_" -> "&&"
    // Reference: https://github.com/google/cel-spec/blob/master/doc/langdef.md#list-of-standard-definitions
    const functionName = callExpr.function.replace(/_/g, "");
    if (callExpr.target) {
      const target = stringifyExpr(callExpr.target);
      const args = callExpr.args.map((arg) => stringifyExpr(arg)).join(", ");
      return `${target}.${functionName}(${args})`;
    } else {
      const args = callExpr.args.map((arg) => stringifyExpr(arg));
      if (functionName === "&&" || functionName === "||") {
        return `(${args.join(` ${functionName} `)})`;
      } else {
        return args.join(` ${functionName} `);
      }
    }
  } else if (expr.exprKind?.case === "listExpr") {
    const listExpr = expr.exprKind.value;
    const elements = listExpr.elements
      .map((el) => stringifyExpr(el))
      .join(", ");
    return `[${elements}]`;
  } else if (expr.exprKind?.case === "structExpr") {
    const structExpr = expr.exprKind.value;
    const entries = structExpr.entries
      .map((entry) => {
        const key =
          entry.keyKind?.case === "fieldKey"
            ? entry.keyKind.value
            : entry.keyKind?.case === "mapKey"
              ? stringifyExpr(entry.keyKind.value)
              : "";
        const value = stringifyExpr(entry.value!);
        return `${key}: ${value}`;
      })
      .join(", ");
    return `{${entries}}`;
  } else if (expr.exprKind?.case === "comprehensionExpr") {
    const comprehensionExpr = expr.exprKind.value;
    const iterRange = stringifyExpr(comprehensionExpr.iterRange!);
    const accuInit = stringifyExpr(comprehensionExpr.accuInit!);
    const loopCondition = stringifyExpr(comprehensionExpr.loopCondition!);
    const loopStep = stringifyExpr(comprehensionExpr.loopStep!);
    const result = stringifyExpr(comprehensionExpr.result!);
    return `comprehension(${comprehensionExpr.iterVar} in ${iterRange}; ${comprehensionExpr.accuVar} = ${accuInit}; ${loopCondition}; ${loopStep}; ${result})`;
  } else {
    return "";
  }
}

function stringifyConstant(constant: Constant): string {
  if (constant.constantKind?.case === "nullValue") {
    return "null";
  } else if (constant.constantKind?.case === "boolValue") {
    return constant.constantKind.value.toString();
  } else if (constant.constantKind?.case === "int64Value") {
    return constant.constantKind.value.toString();
  } else if (constant.constantKind?.case === "uint64Value") {
    return constant.constantKind.value.toString();
  } else if (constant.constantKind?.case === "doubleValue") {
    return constant.constantKind.value.toString();
  } else if (constant.constantKind?.case === "stringValue") {
    return `"${constant.constantKind.value}"`;
  } else if (constant.constantKind?.case === "bytesValue") {
    return `b'${constant.constantKind.value.toString()}'`;
  } else if (constant.constantKind?.case === "durationValue") {
    return `duration(${constant.constantKind.value})`;
  } else if (constant.constantKind?.case === "timestampValue") {
    return `timestamp(${constant.constantKind.value})`;
  } else {
    throw new Error("Unknown constant type");
  }
}

export default stringifyExpr;
