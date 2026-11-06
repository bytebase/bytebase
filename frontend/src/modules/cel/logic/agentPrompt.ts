import {
  type Factor,
  isBooleanFactor,
  isListFactor,
  isNumberFactor,
  isStringFactor,
  isTimestampFactor,
} from "../types/factor";
import { getOperatorListByFactor, type Operator } from "../types/operator";

export function buildSelectorCapabilities(
  factorList: Factor[],
  overrideMap?: Map<Factor, Operator[]>
): string {
  const fields = [...new Set(factorList)].map((factor) => {
    const valueType = isListFactor(factor)
      ? "string[]"
      : isBooleanFactor(factor)
        ? "boolean"
        : isNumberFactor(factor)
          ? "integer"
          : isTimestampFactor(factor)
            ? "timestamp"
            : "string";
    const operators =
      overrideMap?.get(factor) ?? getOperatorListByFactor(factor);
    const templates = [...new Set(operators)].flatMap((operator) => {
      if (isListFactor(factor)) {
        if (operator === "@contains") return [`"value" in ${factor}`];
        if (operator === "@not_contains") return [`!("value" in ${factor})`];
        return [];
      }
      if (isTimestampFactor(factor)) return [];
      const literal = isNumberFactor(factor)
        ? "0"
        : isBooleanFactor(factor)
          ? "true"
          : '"value"';
      switch (operator) {
        case "_==_":
          return [`${factor} == ${literal}`];
        case "_!=_":
          return [`${factor} != ${literal}`];
        case "_<_":
        case "_<=_":
        case "_>=_":
        case "_>_":
          return isNumberFactor(factor)
            ? [`${factor} ${operator.slice(1, -1)} ${literal}`]
            : [];
        case "@in":
        case "@not_in":
          if (!isStringFactor(factor) && !isNumberFactor(factor)) return [];
          return [
            operator === "@in"
              ? `${factor} in [${literal}]`
              : `!(${factor} in [${literal}])`,
          ];
        case "contains":
        case "matches":
        case "startsWith":
        case "endsWith":
          return isStringFactor(factor)
            ? [`${factor}.${operator}(${literal})`]
            : [];
        case "@not_contains":
          return isStringFactor(factor)
            ? [`!(${factor}.contains(${literal}))`]
            : [];
        default:
          return [];
      }
    });
    return { name: factor, valueType, templates };
  });
  return JSON.stringify({
    fields,
    grouping: ["&&", "||", "parentheses"],
    unsupported: [
      "transformed-receivers",
      "arbitrary-functions",
      "raw-cel",
      "general-negation",
    ],
  });
}
