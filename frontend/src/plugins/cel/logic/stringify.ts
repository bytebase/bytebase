import type {
  Constant,
  Expr,
} from "@/types/proto/google/api/expr/v1alpha1/syntax";

function stringifyExpr(expr: Expr): string {
  if (expr.constExpr) {
    return stringifyConstant(expr.constExpr);
  } else if (expr.identExpr) {
    return expr.identExpr.name;
  } else if (expr.selectExpr) {
    // Check if testOnly flag is used to denote a 'has' operation.
    if (expr.selectExpr.testOnly) {
      return `has(${stringifyExpr(expr.selectExpr.operand!)}.${expr.selectExpr.field})`;
    } else {
      return `${stringifyExpr(expr.selectExpr.operand!)}.${expr.selectExpr.field}`;
    }
  } else if (expr.callExpr) {
    // Remove underscores from function name. e.g. "_&&_" -> "&&"
    // Reference: https://github.com/google/cel-spec/blob/master/doc/langdef.md#list-of-standard-definitions
    const functionName = expr.callExpr.function.replace(/_/g, "");
    if (expr.callExpr.target) {
      const target = stringifyExpr(expr.callExpr.target);
      const args = expr.callExpr.args
        .map((arg) => stringifyExpr(arg))
        .join(", ");
      return `${target}.${functionName}(${args})`;
    } else {
      const args = expr.callExpr.args.map((arg) => stringifyExpr(arg));
      if (functionName === "&&" || functionName === "||") {
        return `(${args.join(` ${functionName} `)})`;
      } else {
        return args.join(` ${functionName} `);
      }
    }
  } else if (expr.listExpr) {
    const elements = expr.listExpr.elements
      .map((el) => stringifyExpr(el))
      .join(", ");
    return `[${elements}]`;
  } else if (expr.structExpr) {
    const entries = expr.structExpr.entries
      .map((entry) => {
        const key = entry.fieldKey
          ? entry.fieldKey
          : stringifyExpr(entry.mapKey!);
        const value = stringifyExpr(entry.value!);
        return `${key}: ${value}`;
      })
      .join(", ");
    return `{${entries}}`;
  } else if (expr.comprehensionExpr) {
    const iterRange = stringifyExpr(expr.comprehensionExpr.iterRange!);
    const accuInit = stringifyExpr(expr.comprehensionExpr.accuInit!);
    const loopCondition = stringifyExpr(expr.comprehensionExpr.loopCondition!);
    const loopStep = stringifyExpr(expr.comprehensionExpr.loopStep!);
    const result = stringifyExpr(expr.comprehensionExpr.result!);
    return `comprehension(${expr.comprehensionExpr.iterVar} in ${iterRange}; ${expr.comprehensionExpr.accuVar} = ${accuInit}; ${loopCondition}; ${loopStep}; ${result})`;
  } else {
    return "";
  }
}

function stringifyConstant(constant: Constant): string {
  if (constant.nullValue !== undefined) {
    return "null";
  } else if (constant.boolValue !== undefined) {
    return constant.boolValue.toString();
  } else if (constant.int64Value !== undefined) {
    return constant.int64Value.toString();
  } else if (constant.uint64Value !== undefined) {
    return constant.uint64Value.toString();
  } else if (constant.doubleValue !== undefined) {
    return constant.doubleValue.toString();
  } else if (constant.stringValue !== undefined) {
    return `"${constant.stringValue}"`;
  } else if (constant.bytesValue !== undefined) {
    return `b'${constant.bytesValue.toString()}'`;
  } else if (constant.durationValue !== undefined) {
    return `duration(${constant.durationValue})`;
  } else if (constant.timestampValue !== undefined) {
    return `timestamp(${constant.timestampValue})`;
  } else {
    throw new Error("Unknown constant type");
  }
}

export default stringifyExpr;
