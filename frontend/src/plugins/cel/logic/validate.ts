import { isNumber } from "lodash-es";
import type {
  ConditionExpr,
  ConditionGroupExpr,
  RawStringExpr,
  SimpleExpr,
} from "../types";
import {
  isCollectionExpr,
  isCompareExpr,
  isConditionExpr,
  isConditionGroupExpr,
  isEqualityExpr,
  isNumberFactor,
  isRawStringExpr,
  isStringExpr,
  isStringFactor,
} from "../types";

const validateString = (str: unknown): boolean => {
  if (typeof str !== "string") return false;
  return str.trim().length > 0;
};
const validateNumber = (num: unknown): boolean => {
  return isNumber(num);
};
const validateArrayValues = (
  array: unknown,
  predicate: (value: unknown) => boolean
): boolean => {
  if (!Array.isArray(array)) return false;
  if (array.length === 0) return false;
  return array.every(predicate);
};
const validateStringArray = (array: unknown) => {
  return validateArrayValues(array, validateString);
};
const validateNumberArray = (array: unknown) => {
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
  const validateRawString = (rawString: RawStringExpr): boolean => {
    return Boolean(rawString.content);
  };
  const validate = (expr: SimpleExpr): boolean => {
    if (isConditionGroupExpr(expr)) return validateConditionGroup(expr);
    if (isConditionExpr(expr)) return validateCondition(expr);
    if (isRawStringExpr(expr)) return validateRawString(expr);
    throw new Error(`unsupported expr '${JSON.stringify(expr)}'`);
  };
  return validate(expr);
};
