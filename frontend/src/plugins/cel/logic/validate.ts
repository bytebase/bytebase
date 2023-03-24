import { isNumber } from "lodash-es";
import {
  ConditionExpr,
  ConditionGroupExpr,
  isEqualityExpr,
  isNumberFactor,
  isStringFactor,
  SimpleExpr,
} from "../types";
import {
  isCollectionExpr,
  isConditionExpr,
  isConditionGroupExpr,
  isCompareExpr,
  isStringExpr,
} from "../types";

const validateString = (str: any): boolean => {
  if (typeof str !== "string") return false;
  return str.trim().length > 0;
};
const validateNumber = (num: any): boolean => {
  return isNumber(num);
};
const validateArrayValues = (
  array: any,
  predicate: (value: any) => boolean
): boolean => {
  if (!Array.isArray(array)) return false;
  if (array.length === 0) return false;
  return array.every(predicate);
};
const validateStringArray = (array: any) => {
  return validateArrayValues(array, validateString);
};
const validateNumberArray = (array: any) => {
  return validateArrayValues(array, validateNumber);
};

export const validateSimpleExpr = (expr: SimpleExpr): boolean => {
  const validateCondition = (condition: ConditionExpr): boolean => {
    if (condition.args.length !== 2) {
      // All condition expressions' args need to be [factor, value(s)] format.
      return false;
    }
    if (isEqualityExpr(condition)) {
      const [factor, value] = condition.args;
      if (isStringFactor(factor)) return validateString(value);
      if (isNumberFactor(factor)) return validateNumber(value);
    }
    if (isCompareExpr(condition)) {
      const value = condition.args[1];
      return isNumber(value);
    }
    if (isCollectionExpr(condition)) {
      const [factor, values] = condition.args;
      if (isStringFactor(factor)) return validateStringArray(values);
      if (isNumberFactor(factor)) return validateNumberArray(values);
    }
    if (isStringExpr(condition)) {
      const value = condition.args[1];
      return validateString(value);
    }
    // unknown condition expr type
    return false;
  };
  const validateConditionGroup = (group: ConditionGroupExpr): boolean => {
    const { args } = group;
    if (args.length === 0) return false;
    return args.every(validate);
  };
  const validate = (expr: SimpleExpr): boolean => {
    if (isConditionGroupExpr(expr)) return validateConditionGroup(expr);
    if (isConditionExpr(expr)) return validateCondition(expr);
    throw new Error(`unsupported expr '${JSON.stringify(expr)}'`);
  };
  return validate(expr);
};
