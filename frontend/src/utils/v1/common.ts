import { snakeCase } from "lodash-es";
import { humanizeTs } from "../util";

export const calcUpdateMask = (a: any, b: any, toSnakeCase = false) => {
  const updateMask = new Set<string>();
  const aKeys = new Set(Object.keys(a));
  const bKeys = new Set(Object.keys(b));

  for (const key of aKeys) {
    if (a[key] !== b[key]) {
      updateMask.add(key);
    }
    bKeys.delete(key);
  }
  const keys = [...updateMask.values(), ...bKeys.values()];
  return toSnakeCase ? keys.map(snakeCase) : keys;
};

export const humanizeDate = (date: Date | undefined) => {
  return humanizeTs(Math.floor((date?.getTime() ?? 0) / 1000));
};
