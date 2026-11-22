import type { Expr } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";

// Selector resolution can coerce constants; ignore only IDs, direct factor
// notation, and binary AND/OR associativity when comparing roundtrips.
export function getMaskingDraftCELShape(
  expr: Expr,
  selector: "rule" | "exemption"
): unknown {
  const shape = (expr: Expr): unknown =>
    getMaskingDraftCELShape(expr, selector);
  const kind = expr.exprKind;
  switch (kind.case) {
    case "identExpr":
      return ["factor", kind.value.name];
    case "selectExpr":
      if (
        !kind.value.testOnly &&
        kind.value.operand?.exprKind.case === "identExpr"
      )
        return [
          "factor",
          `${kind.value.operand.exprKind.value.name}.${kind.value.field}`,
        ];
      break;
    case "constExpr": {
      const constant = kind.value.constantKind;
      return ["constant", constant.case, String(constant.value)];
    }
    case "listExpr":
      return [
        "list",
        kind.value.elements.map(shape),
        kind.value.optionalIndices,
      ];
    case "callExpr": {
      const call = kind.value;
      if (
        !call.target &&
        (call.function === "_&&_" || call.function === "_||_")
      ) {
        const flatten = (node: Expr): unknown[] => {
          const nested = node.exprKind;
          if (
            nested.case === "callExpr" &&
            !nested.value.target &&
            nested.value.function === call.function &&
            nested.value.args.length === 2
          )
            return nested.value.args.flatMap(flatten);
          return [shape(node)];
        };
        if (call.args.length !== 2) break;
        return [call.function, call.args.flatMap(flatten)];
      }
      return [
        "call",
        call.function,
        call.target ? shape(call.target) : undefined,
        call.args.map(shape),
      ];
    }
  }
  throw new Error(
    `The condition cannot be represented by the ${selector} selector.`
  );
}
